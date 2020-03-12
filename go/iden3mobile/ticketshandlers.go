package iden3mobile

import (
	"encoding/json"
	"errors"
	"fmt"

	"github.com/iden3/go-iden3-core/merkletree"
	issuerMsg "github.com/iden3/go-iden3-servers-demo/servers/issuerdemo/messages"
	log "github.com/sirupsen/logrus"
)

// TODO: async wrappers (with and without callback)

type reqClaimHandler struct {
	Id      int
	BaseUrl string
	Claim   *merkletree.Entry
	CredID  string
	Status  string
}

type eventReqClaim struct {
	Claim  *merkletree.Entry
	CredID string
}

func (h *reqClaimHandler) isDone(id *Identity) (bool, string, error) {
	if h.Status == string(issuerMsg.RequestStatusPending) {
		return h.checkClaimStatus(id)
	} else if h.Status == string(issuerMsg.ClaimtStatusNotYet) {
		return h.checkClaimCredential(id)
	}
	return true, "", errors.New("Unexpected status, aborting claim request.")
}

//
func (h *reqClaimHandler) checkClaimStatus(id *Identity) (bool, string, error) {
	httpClient := NewHttpClient(h.BaseUrl)
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
		event := eventReqClaim{
			Claim:  nil,
			CredID: "",
		}
		j, err := json.Marshal(event)
		if err != nil {
			return true, "{}", err
		}
		return true, string(j), nil
	case issuerMsg.RequestStatusApproved:
		// Create new ticket to handle credential request
		h.Claim = res.Claim
		h.Status = string(issuerMsg.ClaimtStatusNotYet)
		return false, "", nil
	default:
		return true, "{}", errors.New("Unexpected response from issuer")
	}
}

func (h *reqClaimHandler) checkClaimCredential(id *Identity) (bool, string, error) {
	httpClient := NewHttpClient(h.BaseUrl)
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
			return true, "{}", err
		}
		// Add credential to the identity
		credID, err := id.ClaimDB.AddCredentialExistance(res.Credential)
		if err != nil {
			log.Error("Error storing credential existance", err)
			return true, "{}", err
		}
		// Send event with success
		h.Status = string(issuerMsg.ClaimtStatusReady)
		j, err := json.Marshal(eventReqClaim{
			Claim:  h.Claim,
			CredID: credID,
		})
		if err != nil {
			return true, "{}", err
		}
		return true, string(j), nil
	default:
		return true, "{}", errors.New("Unexpected response from issuer")
	}
}
