package iden3mobile

import (
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"math"
	"sync"
	"testing"
	"time"

	"github.com/iden3/go-iden3-core/core/proof"
	"github.com/iden3/iden3-mobile/go/mockupserver"
	log "github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

var id1ClaimID string
var id2ClaimID string
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
			id1ClaimID = d.CredID
		} else if eventFromId == 2 {
			id2ClaimID = d.CredID
		} else {
			panic("Event from unexpected identity")
		}
		return
	default:
		panic("Unexpected event")
	}
}

func TestHolderHandlers(t *testing.T) {
	// Sync idenPubOnChain every 2 seconds
	go func() {
		for {
			log.Info("idenPubOnChain.Sync()")
			timeNow += 10
			blockNow += 1
			idenPubOnChain.Sync()
			time.Sleep(2 * time.Second)
		}
	}()

	// Start mockup server
	go func() {
		err := mockupserver.Serve(t, &mockupserver.Conf{
			IP:                "127.0.0.1",
			Port:              "1234",
			TimeToAproveClaim: 1 * time.Second,
			TimeToPublish:     2 * time.Second,
		},
			idenPubOnChain,
		)
		require.Nil(t, err)
	}()
	time.Sleep(1 * time.Second)

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

func randomBase64String(l int) string {
	buff := make([]byte, int(math.Round(float64(l)/float64(1.33333333333))))
	rand.Read(buff)
	str := base64.RawURLEncoding.EncodeToString(buff)
	return str[:l] // strip 1 extra character we get from odd length results
}

func TestStressIdentity(t *testing.T) {
	// Sync idenPubOnChain every 2 seconds
	go func() {
		for {
			log.Info("idenPubOnChain.Sync()")
			timeNow += 10
			blockNow += 1
			idenPubOnChain.Sync()
			time.Sleep(2 * time.Second)
		}
	}()

	n := 16
	m := 4

	// Start mockup servers
	for i := 0; i < n; i++ {
		go func() {
			err := mockupserver.Serve(t, &mockupserver.Conf{
				IP:                "127.0.0.1",
				Port:              fmt.Sprintf("9%03d", i),
				TimeToAproveClaim: 1 * time.Second,
				TimeToPublish:     2 * time.Second,
			},
				idenPubOnChain,
			)
			require.Nil(t, err)
		}()
		time.Sleep(200 * time.Millisecond)
	}

	dir1, err := ioutil.TempDir("", "holderStressTest")
	require.Nil(t, err)
	rmDirs = append(rmDirs, dir1)
	iden, err := NewIdentityTest(dir1, "pass_TestHolder1", c.Web3Url, c.HolderTicketPeriod, NewBytesArray())
	require.Nil(t, err)

	// Request claims
	tickets := make(map[string]*Ticket)
	mutex := sync.Mutex{}
	for i := 0; i < n; i++ {
		for j := 0; j < m; j++ {
			go func() {
				t1, err := iden.RequestClaim(fmt.Sprintf("http://127.0.0.1:9%03d/", i), randomBase64String(80))
				require.Nil(t, err)
				log.WithField("TicketId", t1.Id).WithField("i", i).WithField("j", j).Info("--- Request claim ---")
				mutex.Lock()
				tickets[t1.Id] = t1
				mutex.Unlock()
			}()
		}
		if i >= 8 {
			time.Sleep(200 * time.Millisecond)
		}
	}

	// Test that tickets are persisted by reloading identities
	// iden.Stop()
	// id2.Stop()
	// iden, err = NewIdentityTestLoad(dir1, "pass_TestHolder1", c.Web3Url, c.HolderTicketPeriod)
	// require.Nil(t, err)
	// id2, err = NewIdentityTestLoad(dir2, "pass_TestHolder2", c.Web3Url, c.HolderTicketPeriod)
	// require.Nil(t, err)

	claimsLen := n * m

	// Wait for the events that will get triggered on issuer response
	for {
		ev, err := iden.GetNextEvent()
		require.Nil(t, err)
		log.WithField("TicketId", ev.TicketId).Info("--- Get Event ---")
		assert.Nil(t, ev.Err)
		mutex.Lock()
		delete(tickets, ev.TicketId)
		if len(tickets) == 0 {
			mutex.Unlock()
			break
		}
		mutex.Unlock()
	}

	claimIds := make([]string, 0)
	iden.ClaimDB.Iterate_(func(id string, _ *proof.CredentialExistence) (bool, error) {
		claimIds = append(claimIds, id)
		return true, nil
	})

	require.Equal(t, claimsLen, len(claimIds))

	// Prove claim
	var wg sync.WaitGroup
	for k, id := range claimIds {
		wg.Add(1)
		go func() {
			i := 0
			for ; i < c.VerifierAttempts; i++ {
				success1, err := iden.ProveClaim("http://127.0.0.1:9000", id)
				if err != nil {
					log.Error("Error proving claim: ", err)
				}
				if success1 {
					wg.Done()
					break
				}
				time.Sleep(time.Duration(c.VerifierRetryPeriod) * time.Second)
			}
			if i == c.VerifierAttempts {
				panic(fmt.Errorf("Reached maximum number of loops for iden.ProveClaim"))
			}
		}()
		if k >= 8 {
			time.Sleep(200 * time.Millisecond)
		}
	}
	wg.Wait()
	iden.Stop()
}
