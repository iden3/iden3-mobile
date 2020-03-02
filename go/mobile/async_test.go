package iden3mobile

import (
	"io/ioutil"
	"sync"
	"testing"
	"time"

	"github.com/iden3/go-iden3-core/db"
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

func addTestTicket(ts *Tickets, ticketId, err, expectedData string, sayImDone, addToExpected bool) Ticket {
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
	if err := ts.Add([]Ticket{ticket}); err != nil {
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

	storage1, err := db.NewLevelDbStorage(dir1, false)
	require.Nil(t, err)
	storage2, err := db.NewLevelDbStorage(dir2, false)
	require.Nil(t, err)

	eventHandler := &asyncTestEvent{}
	stopTs1 := make(chan bool)
	ts1 := NewTickets(storage1)
	go ts1.CheckPending(nil, eventHandler, time.Duration(c.HolderTicketPeriod)*time.Millisecond, stopTs1)
	stopTs2 := make(chan bool)
	ts2 := NewTickets(storage2)
	go ts2.CheckPending(nil, eventHandler, time.Duration(c.HolderTicketPeriod)*time.Millisecond, stopTs2)
	// Add tickets
	// Succes ticket before stop
	addTestTicket(ts1, "ts1 - Succes ticket before stop", "", `{"SayImDone":true,"Err":""}`, true, true)
	addTestTicket(ts2, "ts2 - Succes ticket before stop", "", `{"SayImDone":true,"Err":""}`, true, true)
	// Fail ticket before stop
	addTestTicket(ts1, "ts1 - Fail ticket before stop", "Something went wrong", `{}`, true, true)
	addTestTicket(ts2, "ts2 - Fail ticket before stop", "Something went wrong", `{}`, true, true)
	// Success ticket after stop
	id1After1 := addTestTicket(ts1, "ts1 - Succes ticket after stop", "", `{"SayImDone":true,"Err":""}`, false, true)
	id2After1 := addTestTicket(ts2, "ts2 - Succes ticket after stop", "", `{"SayImDone":true,"Err":""}`, false, true)
	// Fail ticket after stop
	id1After2 := addTestTicket(ts1, "ts1 - Fail ticket after stop", "Something went wrong", `{}`, false, true)
	id2After2 := addTestTicket(ts2, "ts2 - Fail ticket after stop", "Something went wrong", `{}`, false, true)
	// Add tickets that will be deleted before being solved
	addTestTicket(ts1, "ts1 - remove1", "Something went wrong", `{}`, false, false)
	addTestTicket(ts2, "ts2 - remove1", "Something went wrong", `{}`, false, false)
	addTestTicket(ts1, "ts1 - remove2", "Something went wrong", `{}`, false, false)
	addTestTicket(ts2, "ts2 - remove2", "Something went wrong", `{}`, false, false)
	addTestTicket(ts1, "ts1 - remove3", "Something went wrong", `{}`, false, false)
	addTestTicket(ts2, "ts2 - remove3", "Something went wrong", `{}`, false, false)
	// Give time to process tickets for the first time.
	time.Sleep(time.Duration(2 * time.Millisecond))
	// At this point tickets "Succes ticket before stop" and "Fail ticket before stop" should be resolved
	// Cancel ticket remove1
	require.Nil(t, ts1.Cancel("ts1 - remove1"))
	wgAsyncTest.Done()
	require.Nil(t, ts2.Cancel("ts2 - remove1"))
	wgAsyncTest.Done()
	stopTs1 <- true
	stopTs2 <- true
	// Load tickets
	stopTs1 = make(chan bool)
	ts1 = NewTickets(storage1)
	go ts1.CheckPending(nil, eventHandler, time.Duration(c.HolderTicketPeriod)*time.Millisecond, stopTs1)
	stopTs2 = make(chan bool)
	ts2 = NewTickets(storage2)
	go ts2.CheckPending(nil, eventHandler, time.Duration(c.HolderTicketPeriod)*time.Millisecond, stopTs2)
	// Make after stop tickets finish
	id1After1.handler = &testTicketHandler{SayImDone: true, Err: ""}
	id2After1.handler = &testTicketHandler{SayImDone: true, Err: ""}
	id1After2.handler = &testTicketHandler{SayImDone: true, Err: "Something went wrong"}
	id2After2.handler = &testTicketHandler{SayImDone: true, Err: "Something went wrong"}
	require.Nil(t, ts1.Add([]Ticket{id1After1, id1After2}))
	require.Nil(t, ts2.Add([]Ticket{id2After1, id2After2}))
	// After loading identity, tickets "Succes ticket after stop" and "Fail ticket after stop" will get resolved
	// Cancel ticket remove2
	require.Nil(t, ts1.Cancel("ts1 - remove2"))
	wgAsyncTest.Done()
	// Randomize goroutines excution
	time.Sleep(time.Duration(1 * time.Millisecond))
	require.Nil(t, ts2.Cancel("ts2 - remove2"))
	wgAsyncTest.Done()
	// Randomize goroutines excution
	time.Sleep(time.Duration(1 * time.Millisecond))
	// Cancel ticket remove3
	require.Nil(t, ts1.Cancel("ts1 - remove3"))
	wgAsyncTest.Done()
	require.Nil(t, ts2.Cancel("ts2 - remove3"))
	wgAsyncTest.Done()
	// Wait for all tickets to produce events
	wgAsyncTest.Wait()
	// Compare received events vs expected events
	require.Equal(t, expectedEvents.Map, receivedEvents.Map)
	// Check that there are no pending tickets
	if err := ts1.Iterate(&forEacher{}); err != nil {
		require.Nil(t, err)
	}
	if err := ts2.Iterate(&forEacher{}); err != nil {
		require.Nil(t, err)
	}
	require.Equal(t, 0, pendingTicketsCounter)
	stopTs1 <- true
	stopTs2 <- true
}
