package iden3mobile

import (
	"encoding/json"
	"errors"
	"sync"
	"time"

	"github.com/iden3/go-iden3-core/db"
	log "github.com/sirupsen/logrus"
)

type Event interface {
	EventHandler(string, string, string, error) // Type, Ticket ID, Data(json), error
	// EventHandler(TicketType, string, string, error) // Type, Ticket ID, Data(json), error
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

func (i *Identity) addTickets(tickets []Ticket) error {
	if len(tickets) == 0 {
		return errors.New("tickets is empty!")
	}
	ticketsDB := i.storage.WithPrefix([]byte(ticketPrefix))
	tx, err := ticketsDB.NewTx()
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

func (i *Identity) checkPendingTickets(checkPendingTicketsPeriod time.Duration) {
	// TODO: give more control to native host (check now, ...): Impl ctx context, Check out futures @ rust for inspiration.
	for {
		// Should stop?
		select {
		case <-i.stopTickets:
			log.Info("Stopping check pending tickets routine")
			return
		default:
		}
		tickets, err := i.getPendingTickets()
		if err != nil {
			// Should cause panic instead?
			log.Error("Error loading pending tickets", err)
			time.Sleep(checkPendingTicketsPeriod)
			continue
		}
		var wg sync.WaitGroup
		nPendingTickets := len(tickets)
		finished := make(chan Ticket, nPendingTickets)
		log.Info("Checking ", nPendingTickets, " pending tickets")
		for _, ticket := range tickets {
			// Check ticket
			wg.Add(1)
			go func(t Ticket) {
				defer wg.Done()
				isDone, data, err := t.handler.isDone(i)
				if isDone {
					// Resolve ticket
					log.Info("Sending event for ticket: " + t.Id)
					// TODO: should event sending be serialized? what if it takes too long to come back?
					i.eventSender.EventHandler(t.Type, t.Id, data, err)
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
			if err := i.addTickets(finishedTickets); err != nil {
				log.Error("Error updating tickets that have been resolved. Will check them next iteration.")
			}
		}
		// Should stop?
		select {
		case <-i.stopTickets:
			log.Info("Stopping check pending tickets routine")
			return
		default:
		}
		// Sleep
		time.Sleep(checkPendingTicketsPeriod)
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

func (i *Identity) getPendingTickets() ([]*Ticket, error) {
	var tickets []*Ticket
	ticketsDB := i.storage.WithPrefix([]byte(ticketPrefix))
	if err := ticketsDB.Iterate(func(key, value []byte) (bool, error) {
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

func (i *Identity) IterateTickets(handler TicketOperator) error {
	if err := i.storage.WithPrefix([]byte(ticketPrefix)).Iterate(
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

func (i *Identity) CancelTicket(id string) error {
	ticketsDB := i.storage.WithPrefix([]byte(ticketPrefix))
	ticket := Ticket{}
	if err := db.LoadJSON(ticketsDB, []byte(id), &ticket); err != nil {
		return err
	}
	ticket.Status = TicketStatusCancel
	return i.addTickets([]Ticket{ticket})
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
