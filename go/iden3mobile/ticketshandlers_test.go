package iden3mobile

import (
	"encoding/json"
	"io/ioutil"
	"testing"
	"time"

	log "github.com/sirupsen/logrus"
	"github.com/stretchr/testify/require"
)

var id1ClaimID [32]byte
var id2ClaimID [32]byte
var eventFromId = 0

func holderEventHandler(ev *Event) {
	if ev.Err != nil {
		panic("Test event received with error: " + ev.Err.Error())
	}
	log.Info("--- TEST LOG: Test event received. Type: ", ev.Type, ". Id: ", ev.TicketId, ". Data: ", ev.Data)
	// Check if the event was expected
	expectedEvents.Lock()
	defer expectedEvents.Unlock()
	if evn, ok := expectedEvents.Map[ev.TicketId]; !ok || evn.Typ != ev.Type {
		panic("Unexpected event")
	} else {
		delete(expectedEvents.Map, ev.TicketId)
	}
	// Evaluate event
	switch ev.Type {
	case TicketTypeClaimReq:
		d := &eventReqClaim{}
		if err := json.Unmarshal([]byte(ev.Data), d); err != nil {
			panic(err)
		}
		if eventFromId == 1 {
			copy(id1ClaimID[:], d.DBkey)
		} else if eventFromId == 2 {
			copy(id2ClaimID[:], d.DBkey)
		} else {
			panic("Event from unexpected identity")
		}
		return
	default:
		panic("Unexpected event")
	}
}

func TestHolderHandlers(t *testing.T) {
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

	id1, err := NewIdentity(dir1, "pass_TestHolder1", c.Web3Url, c.HolderTicketPeriod, NewBytesArray())
	require.Nil(t, err)
	id2, err := NewIdentity(dir2, "pass_TestHolder2", c.Web3Url, c.HolderTicketPeriod, NewBytesArray())
	require.Nil(t, err)
	// Request claim
	t1, err := id1.RequestClaim(c.IssuerUrl, id1.id.ID().String())
	require.Nil(t, err)
	expectedEvents.Map[t1.Id] = event{Typ: t1.Type}

	t2, err := id2.RequestClaim(c.IssuerUrl, id2.id.ID().String())
	require.Nil(t, err)
	expectedEvents.Map[t2.Id] = event{Typ: t2.Type}
	// Test that tickets are persisted by reloading identities
	id1.Stop()
	id2.Stop()
	id1, err = NewIdentityLoad(dir1, "pass_TestHolder1", c.Web3Url, c.HolderTicketPeriod)
	require.Nil(t, err)
	id2, err = NewIdentityLoad(dir2, "pass_TestHolder2", c.Web3Url, c.HolderTicketPeriod)
	require.Nil(t, err)
	// Wait for the events that will get triggered on issuer response
	eventFromId = 1
	holderEventHandler(id1.GetNextEvent()) // Wait until the issuer response produce event
	eventFromId = 2
	holderEventHandler(id2.GetNextEvent()) // Wait until the issuer response produce event
	// Prove claim
	for i := 0; i < c.VerifierAttempts; i++ {
		success1, err := id1.ProveClaim(c.VerifierUrl, id1ClaimID[:])
		if err != nil {
			log.Error("Error proving claim: ", err)
		}
		success2, err := id2.ProveClaim(c.VerifierUrl, id2ClaimID[:])
		if err != nil {
			log.Error("Error proving claim: ", err)
		}
		if success1 && success2 {
			break
		}
		time.Sleep(time.Duration(c.VerifierRetryPeriod) * time.Second)
	}
	id1.Stop()
	id2.Stop()
}
