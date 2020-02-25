package iden3mobile

import (
	"encoding/json"
	"errors"
	"sync"
	"time"

	"github.com/iden3/go-iden3-core/db"
	log "github.com/sirupsen/logrus"
)

type (
	Event interface {
		OnEvent(string, string, string, error) // Type, Ticket ID, Data(json), error
	}

	Callback interface {
		VerifierResponse(bool, error)
		RequestClaimResponse(*Ticket, error)
	}

	ticketInterface interface {
		isDone(*Identity) (bool, string, error)
		marshal() ([]byte, error)
		unmarshal([]byte) error
	}

	Ticket struct {
		Id                   string
		LastChecked          int64
		Type                 string
		handler              ticketInterface
		HandlerMarshaledData []byte
	}
)

func (i *Identity) addTicket(t *Ticket) {
	i.Tickets.Lock()
	i.Tickets.m[t.Id] = t
	i.Tickets.Unlock()
	log.Info("Ticket added: Type: ", t.Type, ". ID: ", t.Id)
}

func (i *Identity) checkPendingTickets(checkPendingTicketsPeriod time.Duration) {
	// TODO: give more control to native host (check now, ...): Impl ctx context, Check out futures @ rust for inspiration.
	for {
		var wg sync.WaitGroup
		i.Tickets.Lock()
		// Stop the loop?
		if i.Tickets.shouldStop {
			log.Info("Stopping check pending tickets routine")
			i.Tickets.Unlock()
			return
		}
		// Check tickets
		log.Info("Checking ", len(i.Tickets.m), " pending tickets")
		finishedTickets := make([]string, len(i.Tickets.m))
		index := 0
		for _, t := range i.Tickets.m {
			// Check ticket
			wg.Add(1)
			go func(t *Ticket, index int) {
				defer wg.Done()
				isDone, data, err := t.handler.isDone(i)
				if isDone {
					// Resolve ticket
					log.Info("Sending event for ticket: " + t.Id)
					finishedTickets[index] = t.Id
					i.eventSender.OnEvent(t.Type, t.Id, data, err)
				} else {
					if err != nil {
						log.Error("Error handling ticket: "+t.Id, err)
					}
					// Update last checked time
					t.LastChecked = int64(time.Now().Unix())
				}
			}(t, index)
			index++
		}
		// Wait until all tickets have been checked
		wg.Wait()
		// Delete resolved tickets
		finishedTicketsCounter := 0
		totalTickets := len(i.Tickets.m)
		for _, key := range finishedTickets {
			if key != "" {
				delete(i.Tickets.m, key)
				finishedTicketsCounter++
			}
		}
		i.Tickets.Unlock()
		log.Info("Done checking pending tickets: ", finishedTicketsCounter, "/", totalTickets, " resolved")
		// Stop the loop?
		if i.Tickets.shouldStop {
			log.Info("Stopping check pending tickets routine")
			return
		}
		// Go to sleep
		log.Info("Sleeping before checking tickets again")
		time.Sleep(checkPendingTicketsPeriod)
	}
}

func (t *Ticket) marshal() ([]byte, error) {
	hdlrData, err := t.handler.marshal()
	if err != nil {
		return nil, err
	}
	t.HandlerMarshaledData = hdlrData
	return json.Marshal(t)
}

func unmarshalTicket(data []byte) (*Ticket, error) {
	t := &Ticket{}
	if err := json.Unmarshal(data, t); err != nil {
		return nil, err
	}
	var handler ticketInterface
	switch t.Type {
	case "RequestClaimStatus":
		handler = &reqClaimStatusHandler{}
	case "RequestClaimCredential":
		handler = &reqClaimCredentialHandler{}
	case "test ticket":
		handler = &testTicketHandler{}
	default:
		return t, errors.New("Wrong ticket type")
	}
	if err := handler.unmarshal(t.HandlerMarshaledData); err != nil {
		return t, err
	}
	t.handler = handler
	return t, nil
}

func (i *Identity) loadTickets() {
	// Load pending tickets
	tickets := make(map[string][]byte)
	err := db.LoadJSON(i.storage, []byte("pendingTickets"), &tickets)
	if err == nil {
		log.Info("Loading ", len(tickets), " pending tickets")
		for _, t := range tickets {
			ticket, err := unmarshalTicket(t)
			if err != nil {
				log.Error("Error loading ticket: ", err)
			} else {
				i.Tickets.m[ticket.Id] = ticket
			}
		}
	} else {
		log.Error("Error loading pending tickets: ", err)
	}
}

func (i *Identity) storeTickets() {
	// If the pending tickets loop is running, wait
	i.Tickets.RLock()
	defer i.Tickets.RUnlock()
	// Marshal tickets
	marshaledTickets := make(map[string][]byte)
	log.Info("Storing ", len(i.Tickets.m), " pending tickets")
	for _, t := range i.Tickets.m {
		// Marshal ticket
		data, err := t.marshal()
		if err != nil {
			log.Error("Error storing ticket", t.Id, err)
		} else {
			marshaledTickets[t.Id] = data
		}
	}
	// Store pending tickets
	tx, err := i.storage.NewTx()
	if err != nil {
		log.Error("Error storing ALL tickets", err)
		return
	}
	if err := db.StoreJSON(tx, []byte("pendingTickets"), marshaledTickets); err != nil {
		log.Error("Error storing ALL tickets", err)
		return
	}
	if err := tx.Commit(); err != nil {
		log.Error("Error storing ALL tickets", err)
	}
}

// TODO: Move this code to test
type testTicketHandler struct {
	SayImDone bool
	Err       string
}

func (h *testTicketHandler) isDone(id *Identity) (bool, string, error) {
	if h.SayImDone {
		data, err := h.marshal()
		if err != nil {
			return true, "{}", err
		}
		if h.Err != "" {
			return true, "{}", errors.New(h.Err)
		}
		return true, string(data), err
	} else {
		return false, "", nil
	}
}

func (h *testTicketHandler) marshal() ([]byte, error) {
	return json.Marshal(h)
}

func (h *testTicketHandler) unmarshal(data []byte) error {
	return json.Unmarshal(data, h)
}
