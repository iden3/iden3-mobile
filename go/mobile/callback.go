package iden3mobile

import (
	"fmt"
	"time"
	"sync"
)

type (
	Callback interface {
		OnIssuerResponse(string, string, string, error)
		OnVerifierResponse(string, string, bool, error)
	}

	ticketInterface interface {
		isDone() bool
	}
)

var callback Callback

var pendingTickets []ticketInterface
var isPendingTicketsBeingChecked bool = false

const checkPendingTicketsPeriod time.Duration = 10 * time.Second

var mutex = &sync.Mutex{}

// SetCallbackHandler stores a Callback interface
func (i *Identity) SetCallbackHandler(c Callback) {
	callback = c
}

func getTicketEndpoint(url string) (string, error) {
	body, err := httpGet(url)
	// WARNING: right now the only needed thing from the response is the future url, so it's the only returned thing
	return string(body), err
}

func addTicket(t ticketInterface) {
	// block pending tickets and add the new one
	mutex.Lock()
	pendingTickets = append(pendingTickets, t)
	mutex.Unlock()
	// if the check loop hasn't been triggered, do it!
	if !isPendingTicketsBeingChecked {
		isPendingTicketsBeingChecked = true
		go checkPendingTickets()
	}
}

func checkPendingTickets() {
	var ticketsToCheck []ticketInterface
	for {
		// block pending tickets and dump it into ticketsToCheck
		mutex.Lock()
		ticketsToCheck = append(ticketsToCheck, pendingTickets...)
		pendingTickets = []ticketInterface{}
		mutex.Unlock()
		// initialize unsolved tickets
		var unresolvedTickets []ticketInterface
		for _, ticket := range ticketsToCheck {
			if !ticket.isDone() {
				unresolvedTickets = append(unresolvedTickets, ticket)
			}
		}
		// keep the unresolved tickets for next iteration and go to sleep
		ticketsToCheck = unresolvedTickets
		// append
		fmt.Println("Sleeping before checking tickets again")
		time.Sleep(checkPendingTicketsPeriod)
	}
}
