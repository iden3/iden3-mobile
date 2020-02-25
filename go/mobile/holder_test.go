package iden3mobile

import (
	"encoding/json"
	"os"
	"sync"
	"testing"
	"time"

	log "github.com/sirupsen/logrus"
	"github.com/stretchr/testify/require"
)

type holderTestEventHandler struct{}

type callbacker struct{}
type counter struct {
	sync.Mutex
	n int
}

func (c *callbacker) VerifierResponse(success bool, err error) {
	defer wgCallbackTest.Done()
	verificationCounter.Lock()
	defer verificationCounter.Unlock()
	log.Info("--- TEST LOG: Callback VerifierResponse. Successs: ", success, ". Error: ", err)
	if !success || err != nil {
		return
	}
	verificationCounter.n++
}

func (c *callbacker) RequestClaimResponse(ticket *Ticket, err error) {
	defer wgCallbackTest.Done()
	log.Info("--- TEST LOG: Callback RequestClaimResponse. Ticket: ", ticket, ". Error: ", err)
	if err != nil {
		panic("Callback with error: " + err.Error())
	} else {
		expectedEvents[ticket.Id] = event{Typ: ticket.Type}
		wgHolderTest.Add(1)
	}
}

func (e *holderTestEventHandler) OnEvent(typ, id, data string, err error) {
	defer wgHolderTest.Done()
	if err != nil {
		panic("Test event received with error: " + err.Error())
	}
	log.Info("--- TEST LOG: Test event received. Type: ", typ, ". Id: ", id, ". Data: ", data)
	// Check if the event was expected
	if ev, ok := expectedEvents[id]; !ok || ev.Typ != typ {
		panic("Unexpected event")
	} else {
		delete(expectedEvents, id)
	}
	// Evaluate event
	switch typ {
	case "RequestClaimStatus":
		d := &reqClaimStatusEvent{}
		if err := json.Unmarshal([]byte(data), d); err != nil {
			panic(err)
		}
		// Check received data
		if d.CredentialTicket.Type != "RequestClaimCredential" {
			panic("Unexpected CredentialTicket type")
		}
		expectedEvents[d.CredentialTicket.Id] = event{
			Id:  d.CredentialTicket.Id,
			Typ: d.CredentialTicket.Type,
		}
		// Wait for "RequestClaimCredential" event
		wgHolderTest.Add(1)
		return
	case "RequestClaimCredential":
		if data != `{"success":true}` {
			panic("Validity credential not received")
		}
		return
	default:
		panic("Unexpected event")
	}
}

var wgHolderTest sync.WaitGroup
var wgCallbackTest sync.WaitGroup
var verificationCounter = counter{n: 0}

func TestHolder(t *testing.T) {
	expectedEvents = make(map[string]event)
	// Create two new identities without extra claims
	if err := os.Mkdir(dir+"/TestHolderIdentity1", 0777); err != nil {
		panic(err)
	}
	if err := os.Mkdir(dir+"/TestHolderIdentity2", 0777); err != nil {
		panic(err)
	}
	eventHandler := &holderTestEventHandler{}
	id1, err := NewIdentity(dir+"/TestHolderIdentity1", "pass_TestHolder1", c.Web3Url, c.HolderTicketPeriod, NewBytesArray(), eventHandler)
	require.Nil(t, err)
	id2, err := NewIdentity(dir+"/TestHolderIdentity2", "pass_TestHolder2", c.Web3Url, c.HolderTicketPeriod, NewBytesArray(), eventHandler)
	require.Nil(t, err)
	// Request claim
	cllbck := &callbacker{}
	wgCallbackTest.Add(2)
	id1.RequestClaim(c.IssuerUrl, id1.id.ID().String(), cllbck)
	id2.RequestClaim(c.IssuerUrl, id2.id.ID().String(), cllbck)
	// Wait for callback response
	wgCallbackTest.Wait()
	// Test that tickets are persisted by reloading identities
	id1.Stop()
	id2.Stop()
	id1, err = NewIdentityLoad(dir+"/TestHolderIdentity1", "pass_TestHolder1", c.Web3Url, c.HolderTicketPeriod, eventHandler)
	require.Nil(t, err)
	id2, err = NewIdentityLoad(dir+"/TestHolderIdentity2", "pass_TestHolder2", c.Web3Url, c.HolderTicketPeriod, eventHandler)
	require.Nil(t, err)
	// Wait for the events that will get triggered on issuer response
	wgHolderTest.Wait()
	log.Info("--- TEST LOG: Claims received!")
	// If events don't cause panic everything went as expected. Check that identities have one claim.
	// Do it after reload identities to test claim persistance
	id1.Stop()
	id2.Stop()
	id1, err = NewIdentityLoad(dir+"/TestHolderIdentity1", "pass_TestHolder1", c.Web3Url, c.HolderTicketPeriod, eventHandler)
	require.Nil(t, err)
	id2, err = NewIdentityLoad(dir+"/TestHolderIdentity2", "pass_TestHolder2", c.Web3Url, c.HolderTicketPeriod, eventHandler)
	require.Nil(t, err)
	_, err = id1.GetReceivedClaim(0)
	require.Nil(t, err)
	_, err = id2.GetReceivedClaim(0)
	require.Nil(t, err)
	// Prove claim
	for i := 0; i < c.VerifierAttempts; i++ {
		wgCallbackTest.Add(2)
		id1.ProveClaim(c.VerifierUrl, 0, cllbck)
		id2.ProveClaim(c.VerifierUrl, 0, cllbck)
		wgCallbackTest.Wait()
		if verificationCounter.n == 2 {
			break
		}
		time.Sleep(time.Duration(c.VerifierRetryPeriod) * time.Second)
	}
	// Wait for the callback response.
	id1.Stop()
	id2.Stop()
	require.Equal(t, 2, verificationCounter.n)
}
