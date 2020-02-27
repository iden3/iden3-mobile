package iden3mobile

import (
	"io/ioutil"
	"sync"
	"testing"
	"time"

	log "github.com/sirupsen/logrus"

	"github.com/stretchr/testify/require"
)

type asyncTestEvent struct{}

func (e *asyncTestEvent) EventHandler(typ string, id, data string, err error) {
	defer wgAsyncTest.Done()
	log.Info("Test event received. Id: ", id)
	_err := ""
	if err != nil {
		_err = err.Error()
	}
	receivedEvents.Lock()
	defer receivedEvents.Unlock()
	receivedEvents.Map[id] = event{
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

var expectedEvents eventsMap
var receivedEvents eventsMap

type eventsMap struct {
	sync.Mutex
	Map map[string]event
}

var wgAsyncTest sync.WaitGroup

func addTestTicket(id *Identity, ticketId, err, expectedData string, sayImDone, addToExpected bool) Ticket {
	// Succes ticket before stop
	wgAsyncTest.Add(1)
	hdlr := &testTicketHandler{
		SayImDone: sayImDone,
		Err:       err,
	}
	ticket := Ticket{
		Id:      ticketId,
		Type:    TicketTypeTest,
		Status:  TicketStatusPending,
		handler: hdlr,
	}
	if err := id.addTickets([]Ticket{ticket}); err != nil {
		panic(err)
	}
	if addToExpected {
		expectedEvents.Lock()
		defer expectedEvents.Unlock()
		expectedEvents.Map[ticketId] = event{
			Typ:  TicketTypeTest,
			Id:   ticketId,
			Data: expectedData,
			Err:  err,
		}
	}
	return ticket
}

type forEacher struct{}

var pendingTicketsCounter = 0

func (fe *forEacher) Iterate(t *Ticket) (bool, error) {
	if t.Status == TicketStatusPending {
		pendingTicketsCounter++
	}
	return true, nil
}

func TestTicketSystem(t *testing.T) {
	expectedEvents = eventsMap{
		Map: make(map[string]event),
	}
	receivedEvents = eventsMap{
		Map: make(map[string]event),
	}
	// Two new identities without extra claims
	dir1, err := ioutil.TempDir("", "ticketSystem1")
	require.Nil(t, err)
	rmDirs = append(rmDirs, dir1)
	dir2, err := ioutil.TempDir("", "ticketSystem2")
	require.Nil(t, err)
	rmDirs = append(rmDirs, dir2)

	eventHandler := &asyncTestEvent{}
	id1, err := NewIdentity(dir1, "pass_TestTicketSystem1", c.Web3Url, c.HolderTicketPeriod, NewBytesArray(), eventHandler)
	require.Nil(t, err)
	id2, err := NewIdentity(dir2, "pass_TestTicketSystem2", c.Web3Url, c.HolderTicketPeriod, NewBytesArray(), eventHandler)
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
	id2After1 := addTestTicket(id2, "id2 - Succes ticket after stop", "", `{"SayImDone":true,"Err":""}`, false, true)
	// Fail ticket after stop
	id1After2 := addTestTicket(id1, "id1 - Fail ticket after stop", "Something went wrong", `{}`, false, true)
	id2After2 := addTestTicket(id2, "id2 - Fail ticket after stop", "Something went wrong", `{}`, false, true)
	// Add tickets that will be deleted before being solved
	addTestTicket(id1, "id1 - remove1", "Something went wrong", `{}`, false, false)
	addTestTicket(id2, "id2 - remove1", "Something went wrong", `{}`, false, false)
	addTestTicket(id1, "id1 - remove2", "Something went wrong", `{}`, false, false)
	addTestTicket(id2, "id2 - remove2", "Something went wrong", `{}`, false, false)
	addTestTicket(id1, "id1 - remove3", "Something went wrong", `{}`, false, false)
	addTestTicket(id2, "id2 - remove3", "Something went wrong", `{}`, false, false)
	// Give time to process tickets for the first time.
	time.Sleep(time.Duration(2 * time.Millisecond))
	// At this point tickets "Succes ticket before stop" and "Fail ticket before stop" should be resolved
	// Cancel ticket remove1
	require.Nil(t, id1.CancelTicket("id1 - remove1"))
	wgAsyncTest.Done()
	require.Nil(t, id2.CancelTicket("id2 - remove1"))
	wgAsyncTest.Done()
	id1.Stop()
	id2.Stop()
	// Load identity
	id1, err = NewIdentityLoad(dir1, "pass_TestTicketSystem1", c.Web3Url, c.HolderTicketPeriod, eventHandler)
	require.Nil(t, err)
	id2, err = NewIdentityLoad(dir2, "pass_TestTicketSystem2", c.Web3Url, c.HolderTicketPeriod, eventHandler)
	require.Nil(t, err)
	// Make after stop tickets finish
	id1After1.handler = &testTicketHandler{SayImDone: true, Err: ""}
	id2After1.handler = &testTicketHandler{SayImDone: true, Err: ""}
	id1After2.handler = &testTicketHandler{SayImDone: true, Err: "Something went wrong"}
	id2After2.handler = &testTicketHandler{SayImDone: true, Err: "Something went wrong"}
	require.Nil(t, id1.addTickets([]Ticket{id1After1, id1After2}))
	require.Nil(t, id2.addTickets([]Ticket{id2After1, id2After2}))
	// After loading identity, tickets "Succes ticket after stop" and "Fail ticket after stop" will get resolved
	// Cancel ticket remove2
	require.Nil(t, id1.CancelTicket("id1 - remove2"))
	wgAsyncTest.Done()
	// Randomize goroutines excution
	time.Sleep(time.Duration(1 * time.Millisecond))
	require.Nil(t, id2.CancelTicket("id2 - remove2"))
	wgAsyncTest.Done()
	// Randomize goroutines excution
	time.Sleep(time.Duration(1 * time.Millisecond))
	// Cancel ticket remove3
	require.Nil(t, id1.CancelTicket("id1 - remove3"))
	wgAsyncTest.Done()
	require.Nil(t, id2.CancelTicket("id2 - remove3"))
	wgAsyncTest.Done()
	// Wait for all tickets to produce events
	wgAsyncTest.Wait()
	// Compare received events vs expected events
	require.Equal(t, expectedEvents.Map, receivedEvents.Map)
	// Check that there are no pending tickets
	if err := id1.IterateTickets(&forEacher{}); err != nil {
		require.Nil(t, err)
	}
	if err := id2.IterateTickets(&forEacher{}); err != nil {
		require.Nil(t, err)
	}
	require.Equal(t, 0, pendingTicketsCounter)
	id1.Stop()
	id2.Stop()
}
