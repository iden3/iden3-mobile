package iden3mobile

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"testing"
	"time"

	"github.com/iden3/go-iden3-core/core"
	"github.com/iden3/go-iden3-core/merkletree"
	"github.com/iden3/iden3-mobile/go/mockupserver"
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
	if evn, ok := expectedEvents[ev.TicketId]; !ok || evn.Typ != ev.Type {
		panic("Unexpected event")
	} else {
		delete(expectedEvents, ev.TicketId)
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
	// Start mockup server
	go mockupserver.Serve(&mockupserver.Conf{
		IP:                "127.0.0.1",
		TimeToAproveClaim: time.Duration(1) * time.Second,
		TimeToVerify:      time.Duration(1) * time.Second,
	})
	time.Sleep(1 * time.Second)

	// Load test vector values into idenPubOnChain
	id, err := core.IDFromString("114HNY4C7NrKMQ3XZ7GPLdaQqAQ2TjxgFtLEq312nf")
	require.Nil(t, err)
	var state merkletree.Hash
	err = state.UnmarshalText([]byte("0xaaada0c31752c0e794b64cd65260d8d7506e3fe19ed7e0341cc925d1bb85530e"))
	require.Nil(t, err)
	timeNow = 1583931881
	blockNow = 2326694
	idenPubOnChain.InitState(&id, nil, &state, nil, nil, nil)
	idenPubOnChain.Sync()

	expectedEvents = make(map[string]event)
	// Create two new identities without extra claims
	dir1, err := ioutil.TempDir("", "holderTest1")
	require.Nil(t, err)
	rmDirs = append(rmDirs, dir1)
	dir2, err := ioutil.TempDir("", "holderTest2")
	require.Nil(t, err)
	rmDirs = append(rmDirs, dir2)

	id1, err := NewIdentityTest(dir1, "pass_TestHolder1", c.Web3Url, c.HolderTicketPeriod, NewBytesArray())
	require.Nil(t, err)
	id2, err := NewIdentityTest(dir2, "pass_TestHolder2", c.Web3Url, c.HolderTicketPeriod, NewBytesArray())
	require.Nil(t, err)
	// Request claim
	t1, err := id1.RequestClaim(c.IssuerUrl, id1.id.ID().String())
	require.Nil(t, err)
	expectedEvents[t1.Id] = event{Typ: t1.Type}

	t2, err := id2.RequestClaim(c.IssuerUrl, id2.id.ID().String())
	require.Nil(t, err)
	expectedEvents[t2.Id] = event{Typ: t2.Type}
	// Test that tickets are persisted by reloading identities
	id1.Stop()
	id2.Stop()
	id1, err = NewIdentityTestLoad(dir1, "pass_TestHolder1", c.Web3Url, c.HolderTicketPeriod)
	require.Nil(t, err)
	id2, err = NewIdentityTestLoad(dir2, "pass_TestHolder2", c.Web3Url, c.HolderTicketPeriod)
	require.Nil(t, err)
	// Wait for the events that will get triggered on issuer response
	eventFromId = 1
	ev1, err := id1.GetNextEvent()
	require.Nil(t, err)
	holderEventHandler(ev1) // Wait until the issuer response produce event
	eventFromId = 2
	ev2, err := id2.GetNextEvent()
	holderEventHandler(ev2) // Wait until the issuer response produce event
	require.Nil(t, err)
	// Prove claim
	i := 0
	for ; i < c.VerifierAttempts; i++ {
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
	if i == c.VerifierAttempts {
		panic(fmt.Errorf("Reached maximum number of loops for id{1,2}.ProveClaim"))
	}
	id1.Stop()
	id2.Stop()
}
