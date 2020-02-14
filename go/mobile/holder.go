package iden3mobile

import (
	"encoding/json"
	"errors"
	"io/ioutil"
	"net/http"
	"strconv"

	"github.com/go-playground/validator/v10"

	"github.com/iden3/go-iden3-core/core/proof"

	"github.com/google/uuid"
)

type (
	claimRequestHandler struct {
		holder   *Identity
		endpoint string
	}

	verifierResponse struct {
		Success bool `validate:"required"`
	}

	issuerResponse struct {
		Done       bool                      `validate:"required"`
		Success    bool                      `validate:"required"`
		Credential proof.CredentialExistence `validate:"required"`
	}
)

// RequestClaim sends a petition to issue a claim to an issuer.
// This function will eventually trigger the registered event "OnIssuerResponse"
// The reurned string will match the identifier of the event,
// and potentially the identifier of a ticket that will be added to the identit
// TODO: add context
func (i *Identity) RequestClaim(endpoint, data string) *Ticket {
	id := uuid.New().String()
	t := &Ticket{
		Id:   id,
		Type: "claimRequestHandler",
	}
	go func() {
		futureEndpoint, err := getTicketEndpoint(endpoint + "?data=" + data)
		if err != nil {
			i.eventSender.OnIssuerResponse(id, i.id.ID().String(), []byte{}, err)
			return
		}
		t.handler = &claimRequestHandler{
			holder:   i,
			endpoint: futureEndpoint,
		}
		i.addTicket(t)
	}()

	return t
}

//
func (h *claimRequestHandler) isDone(id string) bool {
	body, err := httpGet(h.endpoint)
	if err != nil {
		h.holder.eventSender.OnIssuerResponse(id, h.holder.id.ID().String(), []byte{}, err)
		return true
	}

	var veredict issuerResponse
	err = json.Unmarshal(body, &veredict)
	if err != nil {
		h.holder.eventSender.OnIssuerResponse(id, h.holder.id.ID().String(), []byte{}, err)
		return true
	}

	if veredict.Done {
		if veredict.Success {
			// VALIDATE RESPONSE
			validate := validator.New()
			err = validate.Struct(veredict)
			if err != nil {
				h.holder.eventSender.OnIssuerResponse(id, h.holder.id.ID().String(), []byte{}, errors.New("Invalid response from the issuer: "+err.Error()))
				return true
			}
			// SEND EVENT WITH CREDENTIAL
			h.holder.addCredentialExistance(veredict.Credential)
			h.holder.eventSender.OnIssuerResponse(id, h.holder.id.ID().String(), veredict.Credential.Claim.Bytes(), nil)
		} else {
			// ISSUER DECIDE TO NOT SEND THE CREDENTIAL
			h.holder.eventSender.OnIssuerResponse(id, h.holder.id.ID().String(), []byte{}, errors.New("Issuer did not send the claim"))
		}
		return true
	} else {
		return false
	}
}

// ProveCredential sends a credentialValidity build from the given credentialExistance to a verifier
// the callback is used to check if the verifier has accepted the credential as valid
// TODO: add context
func (i *Identity) ProveClaim(endpoint string, credIndex int, c Callback) {
	go func() {
		if credIndex < 0 || len(i.receivedCredentials) <= credIndex {
			c.VerifierResponse(false, errors.New("Credential not found in the DB"))
			return
		}
		// TODO: build validity credential from credentialExistance
		credExis := i.receivedCredentials[credIndex]
		j, err := json.Marshal(credExis)
		if err != nil {
			c.VerifierResponse(false, err)
			return
		}
		body, err := httpGet(endpoint + "?proof=" + string(j))
		if err != nil {
			c.VerifierResponse(false, err)
			return
		}
		// Check response
		var veredict verifierResponse
		err = json.Unmarshal(body, &veredict)
		if err != nil {
			// Bad formed response
			c.VerifierResponse(false, err)
		} else {
			// Callback
			c.VerifierResponse(veredict.Success, nil)
		}
	}()
}

func httpGet(endpoint string) ([]byte, error) {
	res, err := http.Get(endpoint)
	if err != nil {
		return []byte{}, err
	}
	defer res.Body.Close()
	if res.StatusCode != 200 {
		return []byte{}, errors.New("Invalid response: " + strconv.Itoa(res.StatusCode))
	}
	return ioutil.ReadAll(res.Body)
}

func (i *Identity) addCredentialExistance(cred proof.CredentialExistence) {
	i.receivedCredentials = append(i.receivedCredentials, cred)
}

// GetReceivedClaimsLen return the amount of received claims by the identity
func (i *Identity) GetReceivedClaimsLen() int {
	return len(i.receivedCredentials)
}

// TODO: return something nicer than bytes
// GetReceivedClaim returns the requested claim
func (i *Identity) GetReceivedClaim(pos int) ([]byte, error) {
	if pos < 0 || len(i.receivedCredentials) <= pos {
		return []byte{}, errors.New("Invalid position")
	}
	return i.receivedCredentials[pos].Claim.Bytes(), nil
}
