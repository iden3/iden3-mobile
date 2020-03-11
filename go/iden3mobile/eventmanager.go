package iden3mobile

import (
	"errors"
	"strconv"
	"sync"
	"time"

	"github.com/iden3/go-iden3-core/db"
	log "github.com/sirupsen/logrus"
)

type eventToStore struct {
	Ev  Event
	Err string
}

type EventManager struct {
	m            sync.Mutex
	storage      db.Storage
	eventQueue   chan Event
	next         sync.WaitGroup
	unreadEvents *db.StorageList
	stopCh       chan bool
	waiting      bool
}

func NewEventManager(storage db.Storage, eventQueue chan Event) (*EventManager, error) {
	sl := db.NewStorageList([]byte(eventsStorKey))
	tx, err := storage.NewTx()
	if err != nil {
		return nil, err
	}
	sl.Init(tx)
	return &EventManager{
		storage:      storage,
		eventQueue:   eventQueue,
		unreadEvents: sl,
		stopCh:       make(chan bool),
	}, tx.Commit()
}

// Initalize the event manager. Call this function only when creating an Identity
func (em *EventManager) Init() error {
	tx, err := em.storage.NewTx()
	if err != nil {
		return err
	}
	tx.Put([]byte(nextEventIdxKey), []byte(strconv.Itoa(0)))
	return tx.Commit()
}

func (em *EventManager) Start() {
	go em.storeNextEvent()
}

func (em *EventManager) Stop() {
	em.stopCh <- true
}

func (em *EventManager) storeNextEvent() {
	isLocked := false
	for {
		// Check if there are new events to send
		if !isLocked {
			em.m.Lock()
			isLocked = true
		}
		tx, err := em.storage.NewTx()
		if err != nil {
			log.Error("Error storing event: ", err)
			continue
		}
		totalEvents, nextEventToSendIdx, err := em.getPendingStatus(tx)
		if err != nil {
			log.Error("error geting pending status")
		}
		if totalEvents-nextEventToSendIdx > 0 && isLocked {
			// Unlock GetNextEvent
			em.m.Unlock()
			isLocked = false
		}
		select {
		case ev := <-em.eventQueue:
			// Store new event
			if !isLocked {
				em.m.Lock()
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
			if err := em.unreadEvents.Append(tx, []byte(ev.TicketId), evToSTore); err != nil {
				log.Error("Error storing event: ", err)
			} else if err := tx.Commit(); err != nil {
				log.Error("Error storing event: ", err)
			}
			if em.waiting {
				em.next.Done()
			}
			isLocked = false
			em.m.Unlock()
			continue
		case <-em.stopCh:
			// Stop loop
			tx.Close()
			if isLocked {
				em.m.Unlock()
			}
			return
		}
	}
}

func (em *EventManager) GetNextEvent() (*Event, error) {
	var idx uint32
	tx, err := em.storage.NewTx()
	if err != nil {
		return nil, err
	}
	for {
		// Check if there are new events to send
		em.m.Lock()
		totalEvents, nextEventToSendIdx, err := em.getPendingStatus(tx)
		if err != nil {
			log.Error("Error geting event pending status: ", err)
			em.m.Unlock()
			return nil, err
		}
		if totalEvents-nextEventToSendIdx > 0 {
			// There are new events, break the loop without unlocking
			idx = nextEventToSendIdx
			break
		}
		// give time to storeNextEvent so it can lock before next loop,
		// and until there is a new event
		em.m.Unlock()
		time.Sleep(1 * time.Second)
	}
	defer em.m.Unlock()
	// Read next event
	ev := &eventToStore{}
	if _, err := em.unreadEvents.GetByIdx(tx, idx, ev); err != nil {
		return nil, err
	}
	// Remove event (Increase idx)
	tx.Put([]byte(nextEventIdxKey), []byte(strconv.Itoa(int(idx)+1)))
	if err := tx.Commit(); err != nil {
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

// getPendingStatus returns tha ammount of stored events, and the next event to send index
func (em *EventManager) getPendingStatus(tx db.Tx) (uint32, uint32, error) {
	encodedIdx, err := em.storage.Get([]byte(nextEventIdxKey))
	if err != nil {
		return 0, 0, err
	}
	idxInt, err := strconv.Atoi(string(encodedIdx))
	if err != nil {
		return 0, 0, err
	}
	idx := uint32(idxInt)
	length, err := em.unreadEvents.Length(tx)
	return length, idx, err
}
