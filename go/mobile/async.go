package iden3mobile

import (
	"encoding/json"
	"errors"
	"sync"
	"time"

	log "github.com/sirupsen/logrus"
)

type (
	Event interface {
		OnEvent(string, string, string, error) // Type, Ticket ID, Data(json), error
	}

	Callback interface {
		VerifierResponse(bool, error)
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

func getTicketEndpoint(url string) (string, error) {
	// TODO: use go-iden3-servers/httpClient
	body, err := httpGet(url)
	return string(body), err
}

func (i *Identity) addTicket(t *Ticket) {
	i.Tickets.Lock()
	i.Tickets.m[t.Id] = t
	i.Tickets.Unlock()
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
	switch t.Type {
	case "RequestClaim":
		hdlr := &claimRequestHandler{}
		if err := hdlr.unmarshal(t.HandlerMarshaledData); err != nil {
			return t, err
		}
		t.handler = hdlr
		return t, nil
	case "test ticket":
		hdlr := &testTicketHandler{}
		if err := hdlr.unmarshal(t.HandlerMarshaledData); err != nil {
			return t, err
		}
		t.handler = hdlr
		return t, nil
	default:
		return t, errors.New("Wrong ticket type")
	}
}
