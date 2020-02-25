package iden3mobile

import (
	"os"
	"sync"
	"testing"
	"time"

	log "github.com/sirupsen/logrus"

	"github.com/stretchr/testify/require"
)

type asyncTestEventHandler struct{}

func (e *asyncTestEventHandler) OnEvent(typ, id, data string, err error) {
	defer wgAsyncTest.Done()
	log.Info("Test event received. Id: ", id)
	_err := ""
	if err != nil {
		_err = err.Error()
	}
	receivedEvents[id] = event{
		Typ:  typ,
		Id:   id,
		Data: data,
		Err:  _err,
	}
}

type event struct {
	Typ  string
	Id   string
	Data string
	Err  string
}

var expectedEvents map[string]event
var receivedEvents map[string]event
var wgAsyncTest sync.WaitGroup

func addTestTicket(id *Identity, ticketId, err, expectedData string, sayImDone, addToExpected bool) *testTicketHandler {
	const typ = "test ticket"
	// Succes ticket before stop
	wgAsyncTest.Add(1)
	hdlr := &testTicketHandler{
		SayImDone: sayImDone,
		Err:       err,
	}
	id.addTicket(&Ticket{
		Id:      ticketId,
		Type:    typ,
		handler: hdlr,
	})
	if addToExpected {
		expectedEvents[ticketId] = event{
			Typ:  typ,
			Id:   ticketId,
			Data: expectedData,
			Err:  err,
		}
	}
	return hdlr
}

type forEacher struct{}

var pendingTicketsCounter = 0

func (fe *forEacher) F(t *Ticket) error {
	log.Error("Found pending test ticket. Id: ", t.Id)
	pendingTicketsCounter++
	return nil
}

func TestTicketSystem(t *testing.T) {
	expectedEvents = make(map[string]event)
	receivedEvents = make(map[string]event)
	// Two new identities without extra claims
	if err := os.Mkdir(dir+"/TestTicketSystemIdentity1", 0777); err != nil {
		panic(err)
	}
	if err := os.Mkdir(dir+"/TestTicketSystemIdentity2", 0777); err != nil {
		panic(err)
	}
	eventHandler := &asyncTestEventHandler{}
	id1, err := NewIdentity(dir+"/TestTicketSystemIdentity1", "pass_TestTicketSystem1", c.Web3Url, 1, NewBytesArray(), eventHandler)
	require.Nil(t, err)
	id2, err := NewIdentity(dir+"/TestTicketSystemIdentity2", "pass_TestTicketSystem2", c.Web3Url, 1, NewBytesArray(), eventHandler)
	require.Nil(t, err)
	// Add tickets
	// Succes ticket before stop
	addTestTicket(id1, "id1 - Succes ticket before stop", "", `{"SayImDone":true,"Err":""}`, true, true)
	addTestTicket(id2, "id2 - Succes ticket before stop", "", `{"SayImDone":true,"Err":""}`, true, true)
	// Fail ticket before stop
	addTestTicket(id1, "id1 - Fail ticket before stop", "Something went wrong", `{}`, true, true)
	addTestTicket(id2, "id2 - Fail ticket before stop", "Something went wrong", `{}`, true, true)
	// Success ticket after stop
	id1After1 := addTestTicket(id1, "id1 - Succes ticket after stop", "", `{"SayImDone":true,"Err":""}`, false, true)
	id2After1 := addTestTicket(id2, "id1 - Succes ticket after stop", "", `{"SayImDone":true,"Err":""}`, false, true)
	// Fail ticket after stop
	id1After2 := addTestTicket(id1, "id1 - Fail ticket after stop", "Something went wrong", `{}`, false, true)
	id2After2 := addTestTicket(id2, "id1 - Fail ticket after stop", "Something went wrong", `{}`, false, true)
	// Add tickets that will be deleted before being solved
	addTestTicket(id1, "id1 - remove1", "Something went wrong", `{}`, false, false)
	addTestTicket(id2, "id2 - remove1", "Something went wrong", `{}`, false, false)
	addTestTicket(id1, "id1 - remove2", "Something went wrong", `{}`, false, false)
	addTestTicket(id2, "id2 - remove2", "Something went wrong", `{}`, false, false)
	addTestTicket(id1, "id1 - remove3", "Something went wrong", `{}`, false, false)
	addTestTicket(id2, "id2 - remove3", "Something went wrong", `{}`, false, false)
	// Give time to process tickets for the first time, Stop identity, Give time to stop ticket loop (2 * checkTicketsPeriodMilis)
	time.Sleep(time.Duration(2 * time.Millisecond))
	// At this point tickets "Succes ticket before stop" and "Fail ticket before stop" should be resolved

	// Make after stop tickets finish
	// It needs to be done before stop, because after the reference of the identity tickets will change (they will be recovered from storage).
	// This is a hacky trick to avoid overcomplicating the logic of the testing tickets
	id1After1.SayImDone = true
	id2After1.SayImDone = true
	id1After2.SayImDone = true
	id2After2.SayImDone = true
	// Cancel ticket remove1
	require.Nil(t, id1.Tickets.Cancel("id1 - remove1"))
	wgAsyncTest.Done()
	require.Nil(t, id2.Tickets.Cancel("id2 - remove1"))
	wgAsyncTest.Done()
	id1.Stop()
	id2.Stop()
	// Load identity
	id1, err = NewIdentityLoad(dir+"/TestTicketSystemIdentity1", "pass_TestTicketSystem1", c.Web3Url, 1, eventHandler)
	require.Nil(t, err)
	id2, err = NewIdentityLoad(dir+"/TestTicketSystemIdentity2", "pass_TestTicketSystem2", c.Web3Url, 1, eventHandler)
	require.Nil(t, err)
	// After loading identity, tickets "Succes ticket after stop" and "Fail ticket after stop" will get resolved
	// Cancel ticket remove2
	require.Nil(t, id1.Tickets.Cancel("id1 - remove2"))
	wgAsyncTest.Done()
	// Randomize goroutines excution
	time.Sleep(time.Duration(1 * time.Millisecond))
	require.Nil(t, id2.Tickets.Cancel("id2 - remove2"))
	wgAsyncTest.Done()
	// Randomize goroutines excution
	time.Sleep(time.Duration(1 * time.Millisecond))
	// Cancel ticket remove3
	require.Nil(t, id1.Tickets.Cancel("id1 - remove3"))
	wgAsyncTest.Done()
	require.Nil(t, id2.Tickets.Cancel("id2 - remove3"))
	wgAsyncTest.Done()
	// Wait for all tickets to produce events
	wgAsyncTest.Wait()
	id1.Stop()
	id2.Stop()
	// Compare received events vs expected events
	require.Equal(t, expectedEvents, receivedEvents)
	// Check that there are no pending tickets
	if err := id1.Tickets.ForEach(&forEacher{}); err != nil {
		require.Nil(t, err)
	}
	if err := id2.Tickets.ForEach(&forEacher{}); err != nil {
		require.Nil(t, err)
	}
	require.Equal(t, 0, pendingTicketsCounter)
}
