package iden3mobile

import (
	"encoding/json"
	"io/ioutil"
	"sync"
	"testing"
	"time"

	log "github.com/sirupsen/logrus"
	"github.com/stretchr/testify/require"
)

type holderTestEvent struct{}

type callback struct{}
type counter struct {
	sync.Mutex
	n int
}

func (c *callback) VerifierResHandler(success bool, err error) {
	defer wgCallbackTest.Done()
	verificationCounter.Lock()
	defer verificationCounter.Unlock()
	log.Info("--- TEST LOG: Callback VerifierResHandler. Successs: ", success, ". Error: ", err)
	if !success || err != nil {
		return
	}
	verificationCounter.n++
}

func (c *callback) RequestClaimResHandler(ticket *Ticket, err error) {
	defer wgCallbackTest.Done()
	log.Info("--- TEST LOG: Callback RequestClaimResHandler. Ticket: ", ticket, ". Error: ", err)
	if err != nil {
		panic("Callback with error: " + err.Error())
	} else {
		expectedEvents.Lock()
		defer expectedEvents.Unlock()
		expectedEvents.Map[ticket.Id] = event{Typ: ticket.Type}
		wgHolderTest.Add(1)
	}
}

func (e *holderTestEvent) EventHandler(typ string, id, data string, err error) {
	defer wgHolderTest.Done()
	if err != nil {
		panic("Test event received with error: " + err.Error())
	}
	log.Info("--- TEST LOG: Test event received. Type: ", typ, ". Id: ", id, ". Data: ", data)
	// Check if the event was expected
	expectedEvents.Lock()
	defer expectedEvents.Unlock()
	if ev, ok := expectedEvents.Map[id]; !ok || ev.Typ != typ {
		panic("Unexpected event")
	} else {
		delete(expectedEvents.Map, id)
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
		expectedEvents.Map[d.CredentialTicket.Id] = event{
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
	expectedEvents = eventsMap{
		Map: make(map[string]event),
	}
	// Create two new identities without extra claims
	dir1, err := ioutil.TempDir("", "holderTest1")
	require.Nil(t, err)
	rmDirs = append(rmDirs, dir1)
	dir2, err := ioutil.TempDir("", "holderTest2")
	require.Nil(t, err)
	rmDirs = append(rmDirs, dir2)

	eventHandler := &holderTestEvent{}
	id1, err := NewIdentity(dir1, "pass_TestHolder1", c.Web3Url, c.HolderTicketPeriod, NewBytesArray(), eventHandler)
	require.Nil(t, err)
	id2, err := NewIdentity(dir2, "pass_TestHolder2", c.Web3Url, c.HolderTicketPeriod, NewBytesArray(), eventHandler)
	require.Nil(t, err)
	// Request claim
	cllbck := &callback{}
	wgCallbackTest.Add(2)
	id1.RequestClaim(c.IssuerUrl, id1.id.ID().String(), cllbck)
	id2.RequestClaim(c.IssuerUrl, id2.id.ID().String(), cllbck)
	// Wait for callback response
	wgCallbackTest.Wait()
	// Test that tickets are persisted by reloading identities
	id1.Stop()
	id2.Stop()
	id1, err = NewIdentityLoad(dir1, "pass_TestHolder1", c.Web3Url, c.HolderTicketPeriod, eventHandler)
	require.Nil(t, err)
	id2, err = NewIdentityLoad(dir2, "pass_TestHolder2", c.Web3Url, c.HolderTicketPeriod, eventHandler)
	require.Nil(t, err)
	// Wait for the events that will get triggered on issuer response
	wgHolderTest.Wait()
	log.Info("--- TEST LOG: Claims received!")
	// If events don't cause panic everything went as expected. Check that identities have one claim.
	// Do it after reload identities to test claim persistance
	id1.Stop()
	id2.Stop()
	id1, err = NewIdentityLoad(dir1, "pass_TestHolder1", c.Web3Url, c.HolderTicketPeriod, eventHandler)
	require.Nil(t, err)
	id2, err = NewIdentityLoad(dir2, "pass_TestHolder2", c.Web3Url, c.HolderTicketPeriod, eventHandler)
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
