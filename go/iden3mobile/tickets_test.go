package iden3mobile

import (
	"io/ioutil"
	"testing"
	"time"

	"github.com/iden3/go-iden3-core/db"
	log "github.com/sirupsen/logrus"

	"github.com/stretchr/testify/require"
)

func eventHnadler(ev *Event) {
	log.Info("Test event received. Id: ", ev.TicketId)
	_err := ""
	if ev.Err != nil {
		_err = ev.Err.Error()
	}
	receivedEvents[ev.TicketId] = event{
		Typ:  ev.Type,
		Id:   ev.TicketId,
		Data: ev.Data,
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

func addTestTicket(ts *Tickets, ticketId, err, expectedData string, sayImDone, addToExpected bool) Ticket {
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
		expectedEvents[ticketId] = event{
			Typ:  TicketTypeTest,
			Id:   ticketId,
			Data: expectedData,
			Err:  err,
		}
	}
	return ticket
}

func TestTicketSystem(t *testing.T) {
	expectedEvents = make(map[string]event)
	receivedEvents = make(map[string]event)
	// Init two new ticket systems
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
	eventCh1 := make(chan Event)
	stopTs1 := make(chan bool)
	em1, err := NewEventManager(storage1, eventCh1)
	require.Nil(t, em1.Init())
	em1.Start()
	defer em1.Stop()
	ts1 := NewTickets(storage1)
	require.Nil(t, err)
	require.Nil(t, ts1.Init())
	go ts1.CheckPending(nil, eventCh1, time.Duration(c.HolderTicketPeriod)*time.Millisecond, stopTs1)
	eventCh2 := make(chan Event)
	stopTs2 := make(chan bool)
	em2, err := NewEventManager(storage2, eventCh2)
	require.Nil(t, em2.Init())
	em2.Start()
	defer em2.Stop()
	ts2 := NewTickets(storage2)
	require.Nil(t, err)
	require.Nil(t, ts2.Init())
	go ts2.CheckPending(nil, eventCh2, time.Duration(c.HolderTicketPeriod)*time.Millisecond, stopTs2)

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
	time.Sleep(time.Duration(c.HolderTicketPeriod*2) * time.Millisecond)
	// Cancel ticket remove1
	require.Nil(t, ts1.CancelTicket("ts1 - remove1"))
	require.Nil(t, ts2.CancelTicket("ts2 - remove1"))

	// Get "before" events
	ev, err := em1.GetNextEvent() // ts1 - Succes ticket before stop || ts1 - Fail ticket before stop
	require.Nil(t, err)
	eventHnadler(ev)
	ev, err = em2.GetNextEvent() // ts2 - Succes ticket before stop || ts2 - Fail ticket before stop
	require.Nil(t, err)
	eventHnadler(ev)
	ev, err = em1.GetNextEvent() // ts1 - Succes ticket before stop || ts1 - Fail ticket before stop
	require.Nil(t, err)
	eventHnadler(ev)
	ev, err = em2.GetNextEvent() // ts2 - Succes ticket before stop || ts2 - Fail ticket before stop
	require.Nil(t, err)
	eventHnadler(ev)

	// Stop ticket system
	stopTs1 <- true
	stopTs2 <- true
	// Make after stop tickets finish
	id1After1.handler = &testTicketHandler{SayImDone: true, Err: ""}
	id2After1.handler = &testTicketHandler{SayImDone: true, Err: ""}
	id1After2.handler = &testTicketHandler{SayImDone: true, Err: "Something went wrong"}
	id2After2.handler = &testTicketHandler{SayImDone: true, Err: "Something went wrong"}
	require.Nil(t, ts1.Add([]Ticket{id1After1, id1After2}))
	require.Nil(t, ts2.Add([]Ticket{id2After1, id2After2}))

	// Restart ticket sistem
	stopTs1 = make(chan bool)
	ts1 = NewTickets(storage1)
	go ts1.CheckPending(nil, eventCh1, time.Duration(c.HolderTicketPeriod)*time.Millisecond, stopTs1)
	stopTs2 = make(chan bool)
	ts2 = NewTickets(storage2)
	go ts2.CheckPending(nil, eventCh2, time.Duration(c.HolderTicketPeriod)*time.Millisecond, stopTs2)
	// Get "after" events
	ev, err = em1.GetNextEvent() // ts1 - Succes ticket after stop || ts1 - Fail ticket after stop
	require.Nil(t, err)
	eventHnadler(ev)
	ev, err = em2.GetNextEvent() // ts2 - Succes ticket after stop || ts2 - Fail ticket after stop
	require.Nil(t, err)
	eventHnadler(ev)
	ev, err = em1.GetNextEvent() // ts1 - Succes ticket after stop || ts1 - Fail ticket after stop
	require.Nil(t, err)
	eventHnadler(ev)
	ev, err = em2.GetNextEvent() // ts2 - Succes ticket after stop || ts2 - Fail ticket after stop
	require.Nil(t, err)
	eventHnadler(ev)

	// Cancel ticket remove2
	require.Nil(t, ts1.CancelTicket("ts1 - remove2"))
	// Randomize goroutines excution
	time.Sleep(time.Duration(c.HolderTicketPeriod*2) * time.Millisecond)
	require.Nil(t, ts2.CancelTicket("ts2 - remove2"))
	// Randomize goroutines excution
	time.Sleep(time.Duration(1 * time.Millisecond))
	// Cancel ticket remove3
	require.Nil(t, ts1.CancelTicket("ts1 - remove3"))
	require.Nil(t, ts2.CancelTicket("ts2 - remove3"))

	// Compare received events vs expected events
	require.Equal(t, expectedEvents, receivedEvents)
	// Check that there are no pending tickets, give time for cancellation to be effective
	time.Sleep(time.Duration(c.HolderTicketPeriod*2) * time.Millisecond)
	pendingTicketsCounter := 0
	iterFn := func(t *Ticket) (bool, error) {
		if t.Status == TicketStatusPending {
			pendingTicketsCounter++
		}
		return true, nil
	}
	if err := ts1.Iterate_(iterFn); err != nil {
		require.Nil(t, err)
	}
	if err := ts2.Iterate_(iterFn); err != nil {
		require.Nil(t, err)
	}
	require.Equal(t, 0, pendingTicketsCounter)
	stopTs1 <- true
	stopTs2 <- true
}
