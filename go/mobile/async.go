package iden3mobile

import (
	"fmt"
	"sync"
	"time"
)

type (
	Event interface {
		OnIssuerResponse(string, string, []byte, error)
	}

	Callback interface {
		VerifierResponse(bool, error)
	}

	ticketInterface interface {
		isDone(string) bool
		// TODO: add Cancel method
	}

	Ticket struct {
		Id          string
		LastChecked int64
		Type        string
		handler     ticketInterface
	}
)

const checkPendingTicketsPeriod time.Duration = 5 * time.Second

var mutex = &sync.Mutex{}

func getTicketEndpoint(url string) (string, error) {
	body, err := httpGet(url)
	// WARNING: right now the only needed thing from the response is the future url, so it's the only returned thing
	return string(body), err
}

func (i *Identity) addTicket(t *Ticket) {
	mutex.Lock()
	i.Tickets.m[t.Id] = t
	mutex.Unlock()
}

// TODO: give more control to native host (check now, ...): Impl ctx context, Check out futures @ rust for inspiration.
func (i *Identity) checkPendingTickets() {
	for {
		fmt.Println("Checking pending tickets")
		for _, t := range i.Tickets.m {
			go func(t *Ticket) {
				if t.handler.isDone(t.Id) {
					// DELETE TICKET
					mutex.Lock()
					delete(i.Tickets.m, t.Id)
					mutex.Unlock()
				} else {
					// UPDATE last checked time
					mutex.Lock()
					t.LastChecked = int64(time.Now().Unix())
					mutex.Unlock()
				}
			}(t)
		}
		time.Sleep(checkPendingTicketsPeriod)
	}
}
