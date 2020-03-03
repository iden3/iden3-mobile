package iden3mobile

import (
	"encoding/json"
	"errors"
	"sync"
	"time"

	"github.com/iden3/go-iden3-core/db"
	log "github.com/sirupsen/logrus"
)

type Event struct {
	Type     string
	TicketId string
	Data     string
	Err      error
}

type Callback interface {
	VerifierResHandler(bool, error)
	RequestClaimResHandler(*Ticket, error)
}

type ticketInterface interface {
	isDone(*Identity) (bool, string, error)
}

type TicketType string
type TicketStatus string

const (
	TicketTypeClaimStatus = "RequestClaimStatus"
	TicketTypeClaimCred   = "RequestClaimCredential"
	TicketTypeTest        = "test ticket"
	TicketStatusDone      = "Done"
	TicketStatusDoneError = "Done with error"
	TicketStatusPending   = "Pending"
	TicketStatusCancel    = "Canceled"
)

type Ticket struct {
	Id          string
	LastChecked int64
	Type        string
	Status      string
	handler     ticketInterface
	HandlerJSON json.RawMessage
}

type TicketOperator interface {
	Iterate(*Ticket) (bool, error)
}

const ticketPrefix = "tickets"

type Tickets struct {
	storage db.Storage
}

func NewTickets(storage db.Storage) *Tickets {
	return &Tickets{storage: storage}
}

func (ts *Tickets) Add(tickets []Ticket) error {
	if len(tickets) == 0 {
		return errors.New("tickets is empty!")
	}
	tx, err := ts.storage.NewTx()
	if err != nil {
		return err
	}
	log.Info("Adding / Updating ", len(tickets), " tickets")
	for _, ticket := range tickets {
		if err := db.StoreJSON(tx, []byte(ticket.Id), ticket); err != nil {
			return err
		}
	}
	return tx.Commit()
}

func (ts *Tickets) CheckPending(id *Identity, eventCh chan Event, checkPendingPeriod time.Duration, stopCh chan bool) {
	// TODO: give more control to native host (check now, ...): Impl ctx context, Check out futures @ rust for inspiration.
	for {
		// Should stop?
		select {
		case <-stopCh:
			log.Info("Stopping check pending tickets routine")
			return
		default:
		}
		tickets, err := ts.GetPending()
		if err != nil {
			// Should cause panic instead?
			log.Error("Error loading pending tickets", err)
			time.Sleep(checkPendingPeriod)
			continue
		}
		var wg sync.WaitGroup
		nPendingTickets := len(tickets)
		finished := make(chan Ticket, nPendingTickets)
		events := make(chan Event, nPendingTickets)
		log.Info("Checking ", nPendingTickets, " pending tickets")
		for _, ticket := range tickets {
			// Check ticket
			wg.Add(1)
			go func(t Ticket) {
				defer wg.Done()
				isDone, data, err := t.handler.isDone(id)
				if isDone {
					// Resolve ticket
					log.Info("Sending event for ticket: " + t.Id)
					events <- Event{
						Type:     t.Type,
						TicketId: t.Id,
						Data:     data,
						Err:      err,
					}
					if err != nil {
						t.Status = TicketStatusDoneError
					} else {
						t.Status = TicketStatusDone
					}
					finished <- t
				} else {
					if err != nil {
						log.Error("Error handling ticket: "+t.Id, err)
					}
					// Update last checked time
					t.LastChecked = int64(time.Now().Unix())
				}
			}(*ticket)
		}
		wg.Wait()
		// Update tickets that are done
		close(finished)
		var finishedTickets []Ticket
		nResolvedTickets := 0
		for ticket := range finished {
			finishedTickets = append(finishedTickets, ticket)
			nResolvedTickets++
		}
		log.Info("Done checking tickets. ", nResolvedTickets, " / ", nPendingTickets, " pending tickets has been resolved.")
		if len(finishedTickets) > 0 {
			if err := ts.Add(finishedTickets); err != nil {
				log.Error("Error updating tickets that have been resolved. Will check them next iteration.")
			}
		}
		close(events)
		for ev := range events {
			eventCh <- ev
		}
		// Should stop?
		select {
		case <-stopCh:
			log.Info("Stopping check pending tickets routine")
			return
		default:
		}
		// Sleep
		time.Sleep(checkPendingPeriod)
	}
}

func (t Ticket) MarshalJSON() ([]byte, error) {
	handlerJSON, err := json.Marshal(t.handler)
	if err != nil {
		return nil, err
	}
	t.HandlerJSON = handlerJSON
	type TicketAlias Ticket
	return json.Marshal((*TicketAlias)(&t))
}

func (t *Ticket) UnmarshalJSON(b []byte) error {
	type TicketAlias Ticket
	if err := json.Unmarshal(b, (*TicketAlias)(t)); err != nil {
		return err
	}
	switch t.Type {
	case TicketTypeClaimStatus:
		t.handler = &reqClaimStatusHandler{}
	case TicketTypeClaimCred:
		t.handler = &reqClaimCredentialHandler{}
	case TicketTypeTest:
		t.handler = &testTicketHandler{}
	default:
		return errors.New("Wrong ticket type")
	}
	if err := json.Unmarshal(t.HandlerJSON, t.handler); err != nil {
		return err
	}
	return nil
}

func (ts *Tickets) GetPending() ([]*Ticket, error) {
	var tickets []*Ticket
	// ticketsDB := i.storage.WithPrefix([]byte(ticketPrefix))
	if err := ts.storage.Iterate(func(key, value []byte) (bool, error) {
		// load ticket
		var ticket Ticket
		if err := ticket.UnmarshalJSON(value); err != nil {
			return false, err
		}
		// only keep the ones that are not done
		if ticket.Status == TicketStatusPending {
			tickets = append(tickets, &ticket)
			return true, nil
		}
		return true, nil
	}); err != nil {
		return nil, err
	}
	return tickets, nil
}

func (ts *Tickets) Iterate(handler TicketOperator) error {
	if err := ts.storage.Iterate(
		func(key, value []byte) (bool, error) {
			// load ticket
			var ticket Ticket
			if err := ticket.UnmarshalJSON(value); err != nil {
				return false, err
			}
			return handler.Iterate(&ticket)
		},
	); err != nil {
		return err
	}
	return nil
}

func (ts *Tickets) Cancel(id string) error {
	var ticket Ticket
	if err := db.LoadJSON(ts.storage, []byte(id), &ticket); err != nil {
		return err
	}
	ticket.Status = TicketStatusCancel
	return ts.Add([]Ticket{ticket})
}

// TODO: Move this code to test
type testTicketHandler struct {
	SayImDone bool
	Err       string
}

func (h *testTicketHandler) isDone(id *Identity) (bool, string, error) {
	if h.SayImDone {
		data, err := json.Marshal(h)
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
