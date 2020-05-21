package iden3mobile

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"math"
	"net/http"
	"os"
	"strconv"
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
	// go func() {
	// 	for {
	// 		log.Info("idenPubOnChain.Sync()")
	// 		timeBlock.AddTime(10)
	// 		timeBlock.AddBlock(1)
	// 		idenPubOnChain.Sync()
	// 		time.Sleep(2 * time.Second)
	// 	}
	// }()

	// // Start mockup server
	// server := mockupserver.Serve(t, &mockupserver.Conf{
	// 	IP:                "127.0.0.1",
	// 	Port:              "1234",
	// 	TimeToAproveClaim: 1 * time.Second,
	// 	TimeToPublish:     2 * time.Second,
	// },
	// 	idenPubOnChain,
	// )
	// time.Sleep(1 * time.Second)

	expectedEvents = make(map[string]testEvent)
	// Create two new identities without extra claims
	dir1, err := ioutil.TempDir("", "holderTest1")
	require.Nil(t, err)
	rmDirs = append(rmDirs, dir1)
	dir2, err := ioutil.TempDir("", "holderTest2")
	require.Nil(t, err)
	rmDirs = append(rmDirs, dir2)

	id1, err := NewIdentityTest(dir1, "pass_TestHolder_1", c.Web3Url, c.HolderTicketPeriod, NewBytesArray(), nil)
	require.Nil(t, err)
	id2, err := NewIdentityTest(dir2, "pass_TestHolder_2", c.Web3Url, c.HolderTicketPeriod, NewBytesArray(), nil)
	require.Nil(t, err)
	// Request claim
	t1, err := id1.RequestClaim(c.IssuerUrl, randomBase64String(80))
	require.Nil(t, err)
	expectedEvents[t1.Id] = testEvent{Typ: t1.Type}

	t2, err := id2.RequestClaim(c.IssuerUrl, randomBase64String(80))
	require.Nil(t, err)
	expectedEvents[t2.Id] = testEvent{Typ: t2.Type}
	// Test that tickets are persisted by reloading identities
	id1.Stop()
	id2.Stop()
	id1, err = NewIdentityTestLoad(dir1, "pass_TestHolder_1", c.Web3Url, c.HolderTicketPeriod, nil)
	require.Nil(t, err)
	id2, err = NewIdentityTestLoad(dir2, "pass_TestHolder_2", c.Web3Url, c.HolderTicketPeriod, nil)
	require.Nil(t, err)
	// Wait for the events that will get triggered on issuer response
	nAtempts := 1000 // TODO: go back to 10 atempts
	period := time.Duration(c.HolderTicketPeriod) * time.Millisecond
	eventFromId = 1
	holderEventHandler(testGetEventWithTimeOut(id1.eventMan, 0, nAtempts, period))
	eventFromId = 2
	holderEventHandler(testGetEventWithTimeOut(id2.eventMan, 0, nAtempts, period))
	// Prove Claims
	isSuccess, err := id1.ProveClaim(c.VerifierUrl, id1ClaimID[:])
	require.True(t, isSuccess)
	require.NoError(t, err)
	isSuccess, err = id2.ProveClaim(c.VerifierUrl, id2ClaimID[:])
	require.True(t, isSuccess)
	require.NoError(t, err)
	// Prove Claims with ZK
	isSuccess, err = id1.ProveClaimZK(c.VerifierUrl, id1ClaimID[:])
	require.True(t, isSuccess)
	require.NoError(t, err)
	isSuccess, err = id2.ProveClaimZK(c.VerifierUrl, id2ClaimID[:])
	require.True(t, isSuccess)
	require.NoError(t, err)
	// Stop identities
	id1.Stop()
	id2.Stop()

	// err = server.Shutdown(context.Background())
	// require.Nil(t, err)
}

func randomBase64String(l int) string {
	buff := make([]byte, int(math.Round(float64(l)/float64(1.33333333333))))
	_, err := rand.Read(buff)
	if err != nil {
		panic(err)
	}
	str := base64.RawURLEncoding.EncodeToString(buff)
	return str[:l] // strip 1 extra character we get from odd length results
}

type testStressIdentityEventHandler struct {
	tickets map[string]*Ticket
	m       *sync.Mutex
	t       *testing.T
}

var testStressIdentityWg sync.WaitGroup

func (ha *testStressIdentityEventHandler) Send(ev *Event) {
	log.WithField("TicketId", ev.TicketId).Info("--- Get Event ---")
	assert.Nil(ha.t, ev.Err)
	ha.m.Lock()
	delete(ha.tickets, ev.TicketId)
	ha.m.Unlock()
	testStressIdentityWg.Done()
}

func TestStressIdentity(t *testing.T) {
	// Sync idenPubOnChain every 2 seconds
	go func() {
		for {
			log.Info("idenPubOnChain.Sync()")
			timeBlock.AddTime(10)
			timeBlock.AddBlock(1)
			idenPubOnChain.Sync()
			time.Sleep(700 * time.Millisecond)
		}
	}()

	n := 16
	m := 4

	if val := os.Getenv("N"); val != "" {
		_n, err := strconv.ParseInt(val, 10, 0)
		n = int(_n)
		if err != nil {
			panic(err)
		}
	}
	if val := os.Getenv("M"); val != "" {
		_m, err := strconv.ParseInt(val, 10, 0)
		m = int(_m)
		if err != nil {
			panic(err)
		}
	}

	log.WithField("n", n).WithField("m", m).Info("-- Stress Identity")

	// Start mockup servers
	servers := make([]*http.Server, n)
	for i := 0; i < n; i++ {
		servers[i] = mockupserver.Serve(t, &mockupserver.Conf{
			IP:                "127.0.0.1",
			Port:              fmt.Sprintf("9%03d", i),
			TimeToAproveClaim: 1 * time.Second,
			TimeToPublish:     500 * time.Millisecond,
		},
			idenPubOnChain,
		)
		time.Sleep(200 * time.Millisecond)
	}

	dir1, err := ioutil.TempDir("", "holderStressTest")
	require.Nil(t, err)
	rmDirs = append(rmDirs, dir1)
	tickets := make(map[string]*Ticket)
	mutex := sync.Mutex{}
	ha := &testStressIdentityEventHandler{
		tickets: tickets,
		t:       t,
		m:       &mutex,
	}
	claimsLen := n * m
	testStressIdentityWg.Add(claimsLen)
	iden, err := NewIdentityTest(dir1, "pass_TestHolder_1", c.Web3Url, 400, NewBytesArray(), ha)
	require.Nil(t, err)

	// Request claims
	for _i := 0; _i < n; _i++ {
		i := _i
		for _j := 0; _j < m; _j++ {
			j := _j
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

	// Wait for the events that will get triggered on issuer response
	testStressIdentityWg.Wait()
	claimIds := make([]string, 0)
	err = iden.ClaimDB.Iterate_(func(id string, _ *proof.CredentialExistence) (bool, error) {
		claimIds = append(claimIds, id)
		return true, nil
	})
	require.Nil(t, err)

	require.Equal(t, claimsLen, len(claimIds))

	// Prove claim
	var wg sync.WaitGroup
	for k, _id := range claimIds {
		wg.Add(1)
		id := _id
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
				time.Sleep(time.Duration(400 * time.Millisecond))
			}
			if i == c.VerifierAttempts {
				panic(fmt.Errorf("Reached maximum number of loops for iden.ProveClaim"))
			}
		}()
		if k >= 16 {
			time.Sleep(200 * time.Millisecond)
		}
	}
	wg.Wait()
	iden.Stop()

	for _, server := range servers {
		err = server.Shutdown(context.Background())
		require.Nil(t, err)
	}
}
