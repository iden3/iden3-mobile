package iden3mobile

import (
	"errors"
	"io/ioutil"
	"testing"
	"time"

	"github.com/iden3/go-iden3-core/db"
	log "github.com/sirupsen/logrus"

	"github.com/stretchr/testify/require"
)

func eventHandler(ev *Event) {
	log.Info("Test event received. Id: ", ev.TicketId)
	_err := ""
	if ev.Err != nil {
		_err = ev.Err.Error()
	}
	receivedEvents[ev.TicketId] = testEvent{
		Typ:  ev.Type,
		Id:   ev.TicketId,
		Data: ev.Data,
		Err:  _err,
	}
}

type testEvent struct {
	Typ  string
	Id   string
	Data string
	Err  string
}

var expectedEvents map[string]testEvent
var receivedEvents map[string]testEvent

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
		expectedEvents[ticketId] = testEvent{
			Typ:  TicketTypeTest,
			Id:   ticketId,
			Data: expectedData,
			Err:  err,
		}
	}
	return ticket
}

func testGetEventWithTimeOut(em *EventManager, idx uint32, nAtempts int, period time.Duration) *Event {
	var ev *Event
	var err error
	i := 0
	for ; i < nAtempts; i++ {
		ev, err = em.GetEvent(idx)
		if err == nil {
			break
		}
		time.Sleep(period)
	}
	if i == nAtempts {
		panic("Event not received: " + err.Error())
	}
	return ev
}

func newTestTicketSystem(dir string, firstTime bool) (*Tickets, *EventManager, chan Event, chan bool, *db.Storage, error) {
	storage, err := loadStorage(dir)
	if err != nil {
		return nil, nil, nil, nil, nil, err
	}
	eventCh := make(chan Event)
	stopTs := make(chan bool)
	em := NewEventManager(storage, eventCh, &testEventHandler{})
	ts := NewTickets(storage.WithPrefix([]byte(ticketPrefix)))
	if firstTime {
		if err := em.Init(); err != nil {
			return nil, nil, nil, nil, nil, err
		}
		if err := ts.Init(); err != nil {
			return nil, nil, nil, nil, nil, err
		}
	}
	em.Start()
	return ts, em, eventCh, stopTs, &storage, nil
}

func startTestTicketSystem(ts *Tickets, eventCh chan Event, stopTs chan bool) {
	go ts.CheckPending(nil, eventCh, time.Duration(c.HolderTicketPeriod)*time.Millisecond, stopTs)
}

func stopTestTicketSystem(em *EventManager, stopTs chan bool, storage db.Storage) {
	stopTs <- true
	em.Stop()
	storage.Close()
}

func TestTicketSystem(t *testing.T) {
	expectedEvents = make(map[string]testEvent)
	receivedEvents = make(map[string]testEvent)
	// Init two new ticket systems
	dir1, err := ioutil.TempDir("", "ticketSystem1")
	require.Nil(t, err)
	rmDirs = append(rmDirs, dir1)
	dir2, err := ioutil.TempDir("", "ticketSystem2")
	require.Nil(t, err)
	rmDirs = append(rmDirs, dir2)

	ts1, em1, eventCh1, stopTs1, storage1, err := newTestTicketSystem(dir1, true)
	require.Nil(t, err)
	startTestTicketSystem(ts1, eventCh1, stopTs1)
	ts2, em2, eventCh2, stopTs2, storage2, err := newTestTicketSystem(dir2, true)
	require.Nil(t, err)
	startTestTicketSystem(ts2, eventCh2, stopTs2)

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
	nAtempts := 4
	period := time.Duration(c.HolderTicketPeriod) * time.Millisecond
	// Cancel ticket remove1
	require.Nil(t, ts1.CancelTicket("ts1 - remove1"))
	require.Nil(t, ts2.CancelTicket("ts2 - remove1"))
	// Get "before" events
	eventHandler(testGetEventWithTimeOut(em1, 0, nAtempts, period))
	eventHandler(testGetEventWithTimeOut(em1, 1, nAtempts, period))
	eventHandler(testGetEventWithTimeOut(em2, 0, nAtempts, period))
	eventHandler(testGetEventWithTimeOut(em2, 1, nAtempts, period))

	// Stop ticket system, and create new ones *without starting*
	stopTestTicketSystem(em1, stopTs1, *storage1)
	ts1, em1, eventCh1, stopTs1, storage1, err = newTestTicketSystem(dir1, false)
	require.Nil(t, err)
	stopTestTicketSystem(em2, stopTs2, *storage2)
	ts2, em2, eventCh2, stopTs2, storage2, err = newTestTicketSystem(dir2, false)
	require.Nil(t, err)

	// Make "after" tickets finish
	id1After1.handler = &testTicketHandler{SayImDone: true, Err: ""}
	id1After2.handler = &testTicketHandler{SayImDone: true, Err: "Something went wrong"}
	require.Nil(t, ts1.Add([]Ticket{id1After1, id1After2}))
	id2After1.handler = &testTicketHandler{SayImDone: true, Err: ""}
	id2After2.handler = &testTicketHandler{SayImDone: true, Err: "Something went wrong"}
	require.Nil(t, ts2.Add([]Ticket{id2After1, id2After2}))

	// Start ticket system
	startTestTicketSystem(ts1, eventCh1, stopTs1)
	startTestTicketSystem(ts2, eventCh2, stopTs2)

	// Get "after" events
	eventHandler(testGetEventWithTimeOut(em1, 2, nAtempts, period))
	eventHandler(testGetEventWithTimeOut(em1, 3, nAtempts, period))
	eventHandler(testGetEventWithTimeOut(em2, 2, nAtempts, period))
	eventHandler(testGetEventWithTimeOut(em2, 3, nAtempts, period))

	// Cancel ticket remove2 and remove3
	require.Nil(t, ts1.CancelTicket("ts1 - remove2"))
	require.Nil(t, ts2.CancelTicket("ts2 - remove2"))
	require.Nil(t, ts1.CancelTicket("ts1 - remove3"))
	require.Nil(t, ts2.CancelTicket("ts2 - remove3"))

	// Compare received events vs expected events
	require.Equal(t, expectedEvents, receivedEvents)
	// Check that there are no pending tickets, give time for cancellation to be effective
	time.Sleep(period * 2)
	iterFn := func(t *Ticket) (bool, error) {
		if t.Status == TicketStatusPending {
			return false, errors.New("There should not be any pending ticket. Pending ticket ID: " + t.Id)
		}
		return true, nil
	}
	if err := ts1.Iterate_(iterFn); err != nil {
		require.Nil(t, err)
	}
	if err := ts2.Iterate_(iterFn); err != nil {
		require.Nil(t, err)
	}
	// Stop ticket system
	stopTestTicketSystem(em1, stopTs1, *storage1)
	stopTestTicketSystem(em2, stopTs2, *storage2)
}
