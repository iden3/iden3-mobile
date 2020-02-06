package iden3mobile

import (
	"encoding/json"
	"errors"
	"io/ioutil"
	"net/http"

	"github.com/google/uuid"
)

type (
	issueTicket struct {
		id       string
		holder   *Identity
		endpoint string
		callback func(string, string, string, error)
	}

	verifyTicket struct {
		id       string
		holder   *Identity
		endpoint string
		callback func(string, string, bool, error)
	}

	verifierResponse struct {
		Done    bool
		Success bool
	}

	issuerResponse struct {
		Done    bool
		Success bool
		Claim   string
	}
)

//
func (i *Identity) RequestClaim(url string) string {
	id := uuid.New().String()
	go func() {
		futureURL, err := getTicketEndpoint(url)
		if err != nil {
			callback.OnIssuerResponse(id, i.Id, "", err)
			return
		}
		addTicket(&issueTicket{
			id:       id,
			holder:   i,
			endpoint: futureURL,
			callback: callback.OnIssuerResponse,
		})
	}()

	return id
}

func (t *issueTicket) isDone() bool {
	body, err := httpGet(t.endpoint)
	if err != nil {
		t.callback(t.id, t.holder.Id, "", err)
		return true
	}

	var veredict issuerResponse
	err = json.Unmarshal(body, &veredict)
	if err != nil {
		t.callback(t.id, t.holder.Id, "", err)
		return true
	}
	if veredict.Done {
		if veredict.Success {
			err = t.holder.addIssuedClaim(veredict.Claim)
			t.callback(t.id, t.holder.Id, veredict.Claim, err)
		} else {
			t.callback(t.id, t.holder.Id, veredict.Claim, errors.New("Issuer did not send the claim"))
		}
		return true
	} else {
		return false
	}
}

// TODO: use received claim
func (i *Identity) RequestVerification(url string, claimIndex []byte) string {
	id := uuid.New().String()
	go func() {
		futureURL, err := getTicketEndpoint(url + "?proof=" + string(claimIndex))
		if err != nil {
			callback.OnVerifierResponse(id, i.Id, false, err)
			return
		}
		addTicket(&verifyTicket{
			id:       id,
			holder:   i,
			endpoint: futureURL,
			callback: callback.OnVerifierResponse,
		})
	}()
	return id
}



func (t *verifyTicket) isDone() bool {
	body, err := httpGet(t.endpoint)
	if err != nil {
		t.callback(t.id, t.holder.Id, false, err)
		return true
	}
	var veredict verifierResponse
	err = json.Unmarshal(body, &veredict)
	if err != nil {
		t.callback(t.id, t.holder.Id, false, err)
		return true
	}
	if veredict.Done {
		t.callback(t.id, t.holder.Id, veredict.Success, nil)
		return true
	} else {
		return false
	}
}

func httpGet(url string) ([]byte, error) {
	res, err := http.Get(url)
	if err != nil {
		return []byte{}, err
	}
	defer res.Body.Close()
	if res.StatusCode != 200 {
		return []byte{}, errors.New("Invalid response")
	}
	return ioutil.ReadAll(res.Body)
}

func (i *Identity) addIssuedClaim(claim string) error {
	i.ReceivedClaims = append(i.ReceivedClaims, claim)
	return nil
}
