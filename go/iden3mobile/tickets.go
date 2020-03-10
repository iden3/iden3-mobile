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

type ticketInterface interface {
	isDone(*Identity) (bool, string, error)
}

type TicketType string
type TicketStatus string

const (
	TicketTypeClaimReq    = "RequestClaim"
	TicketTypeTest        = "test ticket"
	TicketStatusDone      = "Done"
	TicketStatusDoneError = "Done with error"
	TicketStatusPending   = "Pending"
	TicketStatusCancel    = "Canceled"
	ticketsToCancelKey    = "ticketsToCancelKey"
	ticketsKey            = "ticketsKey"
)

type Ticket struct {
	Id          string
	LastChecked int64
	Type        string
	Status      string
	handler     ticketInterface
	HandlerJSON json.RawMessage
}

const ticketPrefix = "tickets"

type Tickets struct {
	storage       db.Storage
	ticketStorage db.Storage
	m             sync.Mutex
}

func NewTickets(storage db.Storage) *Tickets {
	return &Tickets{
		storage:       storage,
		ticketStorage: storage.WithPrefix([]byte(ticketsKey)),
	}
}

func (ts *Tickets) Init() error {
	ts.m.Lock()
	defer ts.m.Unlock()
	return ts.storeTicketsToCancel([]string{})
}

func (ts *Tickets) Add(tickets []Ticket) error {
	ts.m.Lock()
	defer ts.m.Unlock()
	if len(tickets) == 0 {
		return errors.New("tickets is empty!")
	}
	tx, err := ts.ticketStorage.NewTx()
	if err != nil {
		return err
	}
	log.Debug("Adding / Updating ", len(tickets), " tickets")
	for _, ticket := range tickets {
		if err := db.StoreJSON(tx, []byte(ticket.Id), ticket); err != nil {
			return err
		}
	}
	return tx.Commit()
}

func (ts *Tickets) CancelTicket(id string) error {
	ts.m.Lock()
	defer ts.m.Unlock()
	ticketsToCancel, err := ts.getTicketsToCancel()
	if err != nil {
		return err
	}
	ticketsToCancel = append(ticketsToCancel, id)
	return ts.storeTicketsToCancel(ticketsToCancel)
}

func (ts *Tickets) storeTicketsToCancel(ids []string) error {
	tx, err := ts.storage.NewTx()
	if err != nil {
		return err
	}
	if err := db.StoreJSON(tx, []byte(ticketsToCancelKey), ids); err != nil {
		return err
	}
	return tx.Commit()
}

func (ts *Tickets) getTicketsToCancel() ([]string, error) {
	ticketsToCancel := &[]string{}
	err := db.LoadJSON(ts.storage, []byte(ticketsToCancelKey), ticketsToCancel)
	return *ticketsToCancel, err
}

func (ts *Tickets) removeTicketsToCancel(cancelledIds []string) error {
	ts.m.Lock()
	defer ts.m.Unlock()
	// get current tickets
	ticketsToCancel, err := ts.getTicketsToCancel()
	if err != nil {
		return err
	}
	// filter tickets that appear on the provided list
	ticketsToCancelCleaned := []string{}
	for _, toCancelId := range ticketsToCancel {
		isCancelled := false
		for _, cancelledId := range cancelledIds {
			if toCancelId == cancelledId {
				isCancelled = true
				// continue
			}
		}
		if !isCancelled {
			ticketsToCancelCleaned = append(ticketsToCancelCleaned, toCancelId)
		}
	}
	// overwrite with the filtered list
	return ts.storeTicketsToCancel(ticketsToCancelCleaned)
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
		// Check if ticket has been canceled
		ts.m.Lock()
		idsToCancel, err := ts.getTicketsToCancel()
		ts.m.Unlock()
		if err != nil {
			log.Error("Error geting cancelled ids: ", err)
			time.Sleep(checkPendingPeriod)
			continue
		}
		tickets, err := ts.GetPending()
		if err != nil {
			// Should cause panic instead?
			log.Error("Error loading pending tickets: ", err)
			time.Sleep(checkPendingPeriod)
			continue
		}
		var wg sync.WaitGroup
		nPendingTickets := len(tickets)
		checkedTickets := make(chan Ticket, nPendingTickets)
		events := make(chan Event, nPendingTickets)
		log.Debug("Checking ", nPendingTickets, " pending tickets")
		for _, ticket := range tickets {
			// If cancelled, cancel
			isCancelled := false
			for _, idToCancel := range idsToCancel {
				if idToCancel == ticket.Id {
					ticket.Status = TicketStatusCancel
					isCancelled = true
					checkedTickets <- *ticket
					continue
				}
			}
			if isCancelled {
				// cehck next ticket, this one is cancelled
				continue
			}
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
					// Update ticket status
					if err != nil {
						t.Status = TicketStatusDoneError
					} else {
						t.Status = TicketStatusDone
					}
					checkedTickets <- t
				} else {
					if err != nil {
						log.Error("Error handling ticket: "+t.Id, err)
					}
					// Update ticket last checked time
					t.LastChecked = int64(time.Now().Unix())
					checkedTickets <- t
				}
			}(*ticket)
		}
		wg.Wait()
		// Check if ticket has been canceled
		ts.m.Lock()
		idsToCancel, err = ts.getTicketsToCancel()
		ts.m.Unlock()
		if err != nil {
			log.Error("Error geting cancelled ids: ", err)
		}
		// Update tickets
		close(checkedTickets)
		var ticketsToUpdate []Ticket
		var canceledTicketIds []string
		nResolvedTickets := 0
		for ticket := range checkedTickets {
			for _, idToCancel := range idsToCancel {
				if idToCancel == ticket.Id {
					if ticket.Status == TicketStatusPending {
						ticket.Status = TicketStatusCancel
					}
					canceledTicketIds = append(canceledTicketIds, ticket.Id)
					continue
				}
			}
			ticketsToUpdate = append(ticketsToUpdate, ticket)
			if ticket.Status == TicketStatusDone {
				nResolvedTickets++
			}
		}
		log.Debug("Done checking tickets. ", nResolvedTickets, " / ", nPendingTickets, " pending tickets has been resolved.")
		if len(ticketsToUpdate) > 0 {
			if err := ts.Add(ticketsToUpdate); err != nil {
				log.Error("Error updating tickets. Will check them next iteration.")
			}
		}
		if len(canceledTicketIds) > 0 {
			log.Debug(len(canceledTicketIds), " tickets have been cancelled.")
			if err := ts.removeTicketsToCancel(canceledTicketIds); err != nil {
				log.Error("Error updating tickets to cancel: ", err)
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
	case TicketTypeClaimReq:
		t.handler = &reqClaimHandler{}
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
	if err := ts.ticketStorage.Iterate(func(key, value []byte) (bool, error) {
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

func (ts *Tickets) Iterate_(fn func(*Ticket) (bool, error)) error {
	if err := ts.ticketStorage.Iterate(
		func(key, value []byte) (bool, error) {
			// load ticket
			var ticket Ticket
			if err := ticket.UnmarshalJSON(value); err != nil {
				return false, err
			}
			return fn(&ticket)
		},
	); err != nil {
		return err
	}
	return nil
}

type TicketOperator interface {
	Iterate(*Ticket) (bool, error)
}

func (ts *Tickets) Iterate(handler TicketOperator) error { return ts.Iterate_(handler.Iterate) }

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
