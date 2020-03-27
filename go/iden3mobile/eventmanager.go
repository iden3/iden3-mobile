package iden3mobile

import (
	"errors"
	"sync"

	"github.com/iden3/go-iden3-core/db"
	log "github.com/sirupsen/logrus"
)

type eventToStore struct {
	Ev  Event
	Err string
}

type Sender interface {
	Send(*Event)
}
type EventManager struct {
	storage       db.Storage
	eventsStorage *db.StorageList
	eventChIn     chan Event
	stopCh        chan bool
	eventSend     Sender
	m             sync.RWMutex
}

func NewEventManager(storage db.Storage, eventQueue chan Event, s Sender) *EventManager {
	sl := db.NewStorageList([]byte(eventsStorKey))

	return &EventManager{
		storage:       storage,
		eventChIn:     eventQueue,
		eventsStorage: sl,
		stopCh:        make(chan bool),
		eventSend:     s,
	}
}

func (em *EventManager) Init() error {
	tx, err := em.storage.NewTx()
	if err != nil {
		return err
	}
	em.eventsStorage.Init(tx)
	return tx.Commit()
}

func (em *EventManager) Start() {
	go em.controller()
}

func (em *EventManager) Stop() {
	em.stopCh <- true
}

func (em *EventManager) EventLength() (uint32, error) {
	tx, err := em.storage.NewTx()
	if err != nil {
		return 0, err
	}
	return em.eventsStorage.Length(tx)
}

func (em *EventManager) GetEvent(idx uint32) (*Event, error) {
	em.m.RLock()
	defer em.m.RUnlock()
	tx, err := em.storage.NewTx()
	if err != nil {
		return nil, err
	}
	ev := &eventToStore{}
	if _, err := em.eventsStorage.GetByIdx(tx, idx, ev); err != nil {
		return nil, err
	}
	if ev.Err == "" {
		return &ev.Ev, nil
	}
	return &Event{
		Type:     ev.Ev.Type,
		TicketId: ev.Ev.TicketId,
		Data:     ev.Ev.Data,
		Err:      errors.New(ev.Err),
	}, nil
}

func (em *EventManager) controller() {
	// nPendingRequests := 0
	for {
		select {
		case <-em.stopCh:
			// Stop loop
			return
		case ev := <-em.eventChIn:
			// Store event
			if err := em.storeEvent(ev); err != nil {
				log.Error("Error storing event: ", err)
			}
			// Send event
			em.eventSend.Send(&ev)
		}
	}
}

func (em *EventManager) storeEvent(ev Event) error {
	em.m.Lock()
	defer em.m.Unlock()
	tx, err := em.storage.NewTx()
	if err != nil {
		return err
	}
	var errToStore string
	if ev.Err != nil {
		errToStore = ev.Err.Error()
		ev.Err = nil
	}
	evToSTore := eventToStore{
		Ev:  ev,
		Err: errToStore,
	}
	if err := em.eventsStorage.Append(tx, []byte(ev.TicketId), evToSTore); err != nil {
		return err
	}
	return tx.Commit()
}
