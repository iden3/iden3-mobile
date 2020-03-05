package iden3mobile

import (
	"encoding/json"
	"errors"
	"fmt"

	"github.com/google/uuid"
	"github.com/iden3/go-iden3-core/components/httpclient"
	"github.com/iden3/go-iden3-core/merkletree"
	issuerMsg "github.com/iden3/go-iden3-servers-demo/servers/issuerdemo/messages"
	log "github.com/sirupsen/logrus"
)

// TODO: async wrappers (with and without callback)

type resClaimStatusHandler struct {
	Claim            *merkletree.Entry
	CredentialTicket *Ticket
	Status           string
}

type reqClaimStatusHandler struct {
	Id      int
	BaseUrl string
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
		event := resClaimStatusHandler{
			Claim:            res.Claim,
			CredentialTicket: nil,
			Status:           string(res.Status),
		}
		j, err := json.Marshal(event)
		if err != nil {
			return true, "{}", err
		}
		return true, string(j), nil
	case issuerMsg.RequestStatusApproved:
		// Create new ticket to handle credential request
		ticket := &Ticket{
			Id:     uuid.New().String(),
			Type:   TicketTypeClaimCred,
			Status: TicketStatusPending,
			handler: &reqClaimCredentialHandler{
				Claim:   res.Claim,
				BaseUrl: h.BaseUrl,
			},
		}
		// Add credential request ticket
		if err := id.Tickets.Add([]Ticket{*ticket}); err != nil {
			return true, "{}", err
		}
		// Send event with received claim and credential request ticket
		event := resClaimStatusHandler{
			Claim:            res.Claim,
			CredentialTicket: ticket,
			Status:           string(res.Status),
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

type reqClaimCredentialHandler struct {
	Claim   *merkletree.Entry
	BaseUrl string
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
		if _, err := id.ClaimDB.AddCredentialExistance(res.Credential); err != nil {
			log.Error("Error storing credential existance", err)
			return true, "", err
		}
		// Send event with success
		// TODO: return db key
		return true, `{"success":true}`, nil
	default:
		return true, "{}", errors.New("Unexpected response from issuer")
	}
}
