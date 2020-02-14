package iden3mobile

import (
	"errors"

	"github.com/iden3/go-iden3-core/core/proof"
	"github.com/iden3/go-iden3-core/db"
	"github.com/iden3/go-iden3-core/identity"
	"github.com/iden3/go-iden3-core/identity/issuer"
	"github.com/iden3/go-iden3-core/keystore"
)

type Identity struct {
	id                  identity.Issuer
	receivedCredentials []proof.CredentialExistence
	Tickets             *TicketsMap
	eventSender         Event
}

// NewIdentity creates a new identity
// this funciton is mapped as a constructor in Java
func NewIdentity(storePath, pass string, extraGenesisClaims *BytesArray, e Event) (*Identity, error) {
	// TODO: make db & ksStorage persistent (using param: storePath string)
	id := &Identity{}
	_extraGenesisClaims, err := extraGenesisClaims.toEntriers()
	if err != nil {
		return id, err
	}
	storage := db.NewMemoryStorage()
	cfg := issuer.ConfigDefault
	ksStorage := keystore.MemStorage([]byte{})
	keyStore, err := keystore.NewKeyStore(&ksStorage, keystore.LightKeyStoreParams)
	if err != nil {
		return id, err
	}
	kOp, err := keyStore.NewKey([]byte(pass))
	if err != nil {
		return id, err
	}
	is, err := issuer.New(cfg, kOp, _extraGenesisClaims, storage, keyStore, nil)
	if err != nil {
		return id, err
	}
	id.id = is
	id.Tickets = &TicketsMap{
		m: make(map[string]*Ticket),
	}
	go id.checkPendingTickets()
	id.eventSender = e
	return id, nil
}

// NewIdentityLoad loads an already created identity
// this funciton is mapped as a constructor in Java
func NewIdentityLoad(storePath string, e Event) (*Identity, error) {
	return &Identity{}, errors.New("NOT IMPLEMENTED")
}
