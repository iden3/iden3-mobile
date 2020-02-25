package iden3mobile

import (
	"encoding/json"
	"errors"
	"fmt"
	"strconv"

	"github.com/google/uuid"
	"github.com/iden3/go-iden3-core/components/httpclient"
	"github.com/iden3/go-iden3-core/core/proof"
	"github.com/iden3/go-iden3-core/db"
	"github.com/iden3/go-iden3-core/merkletree"
	issuerMsg "github.com/iden3/go-iden3-servers-demo/servers/issuerdemo/messages"
	verifierMsg "github.com/iden3/go-iden3-servers-demo/servers/verifier/messages"
	log "github.com/sirupsen/logrus"
)

type (
	reqClaimStatusHandler struct {
		Id      int
		BaseUrl string
	}

	reqClaimStatusEvent struct {
		Claim            *merkletree.Entry
		CredentialTicket *Ticket
	}

	reqClaimCredentialHandler struct {
		Claim   *merkletree.Entry
		BaseUrl string
	}
)

// RequestClaim sends a petition to issue a claim to an issuer.
// This function will eventually trigger an event,
// the returned ticket can be used to reference the event
func (i *Identity) RequestClaim(baseUrl, data string, c Callback) {
	go func() {
		id := uuid.New().String()
		t := &Ticket{
			Id:   id,
			Type: "RequestClaimStatus",
		}
		httpClient := httpclient.NewHttpClient(baseUrl)
		res := issuerMsg.ResClaimRequest{}
		if err := httpClient.DoRequest(httpClient.NewRequest().Path(
			"claim/request").Post("").BodyJSON(&issuerMsg.ReqClaimRequest{
			Value: data,
		}), &res); err != nil {
			c.RequestClaimResponse(nil, err)
			return
		}
		t.handler = &reqClaimStatusHandler{
			Id:      res.Id,
			BaseUrl: baseUrl,
		}
		i.addTicket(t)
		c.RequestClaimResponse(t, nil)
	}()
}

//
func (h *reqClaimStatusHandler) isDone(id *Identity) (bool, string, error) {
	httpClient := httpclient.NewHttpClient(h.BaseUrl)
	var res issuerMsg.ResClaimStatus
	// it's ok to remove ticket on a network error?
	// TODO: impl error counter and remove ticket after limit
	if err := httpClient.DoRequest(httpClient.NewRequest().Path(
		fmt.Sprintf("claim/status/%v", h.Id)).Get(""), &res); err != nil {
		return true, "{}", err
	}
	switch res.Status {
	case issuerMsg.RequestStatusPending:
		return false, "", nil
	case issuerMsg.RequestStatusRejected:
		j, err := json.Marshal(res)
		if err != nil {
			return true, "{}", err
		}
		return true, string(j), nil
	case issuerMsg.RequestStatusApproved:
		// Create new ticket to handle credential request
		ticket := &Ticket{
			Id:   uuid.New().String(),
			Type: "RequestClaimCredential",
			handler: &reqClaimCredentialHandler{
				Claim:   res.Claim,
				BaseUrl: h.BaseUrl,
			},
		}
		// Add credential request ticket
		go id.addTicket(ticket)
		// Send event with received claim and credential request ticket
		event := reqClaimStatusEvent{
			Claim:            res.Claim,
			CredentialTicket: ticket,
		}
		j, err := json.Marshal(event)
		if err != nil {
			return true, "{}", err
		}
		return true, string(j), nil
	default:
		return true, "{}", errors.New("Unexpected response from issuer")
	}
}

func (h *reqClaimStatusHandler) marshal() ([]byte, error) {
	return json.Marshal(h)
}

func (h *reqClaimStatusHandler) unmarshal(data []byte) error {
	return json.Unmarshal(data, &h)
}

func (h *reqClaimCredentialHandler) isDone(id *Identity) (bool, string, error) {
	httpClient := httpclient.NewHttpClient(h.BaseUrl)
	res := issuerMsg.ResClaimCredential{}
	// it's ok to remove ticket on a network error?
	// TODO: impl error counter and remove ticket after limit
	if err := httpClient.DoRequest(httpClient.NewRequest().Path(
		"claim/credential").Post("").BodyJSON(issuerMsg.ReqClaimCredential{
		Claim: h.Claim,
	}), &res); err != nil {
		return true, "{}", err
	}
	switch res.Status {
	case issuerMsg.ClaimtStatusNotYet:
		return false, "", nil
	case issuerMsg.ClaimtStatusReady:
		// Check that credential match the issued claim
		if !h.Claim.Equal(res.Credential.Claim) {
			err := errors.New("The received credential doesn't match the issued claim")
			log.Error(err)
			return true, "", err
		}
		// Add credential to the identity
		if err := id.addCredentialExistance(*res.Credential); err != nil {
			log.Error("Error storing credential existance", err)
			return true, "", err
		}
		// Send event with success
		return true, `{"success":true}`, nil
	default:
		return true, "{}", errors.New("Unexpected response from issuer")
	}
}

func (h *reqClaimCredentialHandler) marshal() ([]byte, error) {
	return json.Marshal(h)
}

func (h *reqClaimCredentialHandler) unmarshal(data []byte) error {
	return json.Unmarshal(data, &h)
}

// ProveCredential sends a credentialValidity build from the given credentialExistance to a verifier
// the callback is used to check if the verifier has accepted the credential as valid
func (i *Identity) ProveClaim(baseUrl string, credIndex int, c Callback) {
	// TODO: add context
	go func() {
		// Get credential existance
		credExis, err := i.getReceivedCredential(credIndex)
		if err != nil {
			c.VerifierResponse(false, err)
			return
		}
		// Build credential validity
		credVal, err := i.id.HolderGetCredentialValidity(&credExis)
		if err != nil {
			c.VerifierResponse(false, err)
			return
		}
		// Send credential to verifier
		httpClient := httpclient.NewHttpClient(baseUrl)
		if err := httpClient.DoRequest(httpClient.NewRequest().Path(
			"verify").Post("").BodyJSON(verifierMsg.ReqVerify{
			CredentialValidity: credVal,
		}), nil); err != nil {
			// Credential declined / error
			c.VerifierResponse(false, err)
			return
		}
		// Success
		c.VerifierResponse(true, nil)
	}()
}

func (i *Identity) addCredentialExistance(cred proof.CredentialExistence) error {
	tx, err := i.storage.NewTx()
	if err != nil {
		return err
	}
	byteCounter, err := tx.Get([]byte("receivedCredentialsCounter"))
	if err != nil {
		return err
	}
	counter, err := strconv.Atoi(string(byteCounter))
	if err != nil {
		return err
	}
	if err := db.StoreJSON(tx, []byte("receivedCredential_"+strconv.Itoa(counter)), cred); err != nil {
		return err
	}
	counter++
	tx.Put([]byte("receivedCredentialsCounter"), []byte(strconv.Itoa(counter)))
	if err := tx.Commit(); err != nil {
		return err
	}
	log.Info("Stored new existence credential, with key = ", "receivedCredential_"+strconv.Itoa(counter-1))
	return nil
}

// GetReceivedClaimsLen return the amount of received claims by the identity
func (i *Identity) GetReceivedClaimsLen() (int, error) {
	tx, err := i.storage.NewTx()
	// get current amount of claims
	if err != nil {
		return 0, err
	}
	byteCounter, err := tx.Get([]byte("receivedCredentialsCounter"))
	if err != nil {
		return 0, err
	}
	counter, err := strconv.Atoi(string(byteCounter))
	if err != nil {
		return 0, err
	}
	return counter, nil
}

// GetReceivedClaim returns the requested claim
func (i *Identity) GetReceivedClaim(pos int) ([]byte, error) {
	cred, err := i.getReceivedCredential(pos)
	if err != nil {
		return nil, err
	}
	// TODO: return something nicer than bytes (metadata)
	return cred.Claim.Bytes(), nil
}

func (i *Identity) getReceivedCredential(pos int) (proof.CredentialExistence, error) {
	var cred proof.CredentialExistence
	log.Info("Loading existence credential, with key = ", "receivedCredential_"+strconv.Itoa(pos))
	if err := db.LoadJSON(i.storage, []byte("receivedCredential_"+strconv.Itoa(pos)), &cred); err != nil {
		return cred, err
	}
	return cred, nil
}
