package iden3mobile

import (
	"errors"
	"fmt"
	"io"
	"os"
	"time"

	"github.com/google/uuid"
	"github.com/iden3/go-iden3-core/components/idenpuboffchain/readerhttp"
	"github.com/iden3/go-iden3-core/db"
	"github.com/iden3/go-iden3-core/identity/holder"
	"github.com/iden3/go-iden3-crypto/babyjub"
	issuerMsg "github.com/iden3/go-iden3-servers-demo/servers/issuerdemo/messages"
	verifierMsg "github.com/iden3/go-iden3-servers-demo/servers/verifier/messages"
	log "github.com/sirupsen/logrus"
)

type Identity struct {
	id          *holder.Holder
	storage     db.Storage
	ClaimDB     *ClaimDB
	Tickets     *Tickets
	stopTickets chan bool
	eventMan    *EventManager
}

const (
	kOpStorKey           = "kOpComp"
	eventsStorKey        = "eventsKey"
	nextEventIdxKey      = "nextEventIdxKey"
	storageSubPath       = "/idStore"
	keyStorageSubPath    = "/idKeyStore"
	smartContractAddress = "0xF6a014Ac66bcdc1BF51ac0fa68DF3f17f4b3e574"
	credExisPrefix       = "credExis"
)

func isEmpty(path string) (bool, error) {
	f, err := os.Open(path)
	if err != nil {
		return false, err
	}
	defer f.Close()

	_, err = f.Readdirnames(1)
	if err == io.EOF {
		return true, nil
	}
	return false, err
}

// NewIdentity creates a new identity
// this funciton is mapped as a constructor in Java.
// NOTE: The storePath must be unique per Identity.
// NOTE: Right now the extraGenesisClaims is useless.
func NewIdentity(storePath, pass, web3Url string, checkTicketsPeriodMilis int, extraGenesisClaims *BytesArray) (*Identity, error) {
	// Check that storePath points to an empty dir
	if dirIsEmpty, err := isEmpty(storePath); !dirIsEmpty || err != nil {
		if err == nil {
			err = errors.New("Directory is not empty")
		}
		return nil, err
	}
	_, keyStore, storage, err := loadComponents(storePath, web3Url)
	if err != nil {
		return nil, err
	}
	resourcesAreClosed := false
	defer func() {
		if !resourcesAreClosed {
			keyStore.Close()
			storage.Close()
		}
	}()
	// Create babyjub keys
	kOpComp, err := keyStore.NewKey([]byte(pass))
	if err != nil {
		return nil, err
	}
	if err = keyStore.UnlockKey(kOpComp, []byte(pass)); err != nil {
		return nil, err
	}
	// Store kOpComp
	tx, err := storage.NewTx()
	if err != nil {
		return nil, err
	}
	if err := db.StoreJSON(tx, []byte(kOpStorKey), kOpComp); err != nil {
		return nil, err
	}
	if err := tx.Commit(); err != nil {
		return nil, err
	}
	// TODO: Parse extra genesis claims. Call toClaimers once it's implemented
	// _extraGenesisClaims, err := extraGenesisClaims.toClaimers()
	if err != nil {
		return nil, err
	}
	em, err := NewEventManager(storage, nil)
	if err != nil {
		return nil, err
	}
	if err := em.Init(); err != nil {
		return nil, err
	}
	// Create new Identity (holder)
	if _, err = holder.Create(
		holder.ConfigDefault,
		kOpComp,
		nil,
		storage,
		keyStore,
	); err != nil {
		return nil, err
	}
	// Init tickets
	ts := NewTickets(storage.WithPrefix([]byte(ticketPrefix)))
	if err := ts.Init(); err != nil {
		return nil, err
	}
	// Verify that the Identity can be loaded successfully
	keyStore.Close()
	storage.Close()
	resourcesAreClosed = true
	return NewIdentityLoad(storePath, pass, web3Url, checkTicketsPeriodMilis)
}

// NewIdentityLoad loads an already created identity
// this funciton is mapped as a constructor in Java
func NewIdentityLoad(storePath, pass, web3Url string, checkTicketsPeriodMilis int) (*Identity, error) {
	// TODO: figure out how to diferentiate the two constructors from Java: https://github.com/iden3/iden3-mobile/issues/17#issuecomment-587374644
	idenPubOnChain, keyStore, storage, err := loadComponents(storePath, web3Url)
	if err != nil {
		return nil, err
	}
	defer keyStore.Close()
	// Unlock key store
	kOpComp := &babyjub.PublicKeyComp{}
	if err := db.LoadJSON(storage, []byte(kOpStorKey), kOpComp); err != nil {
		return nil, err
	}
	if err := keyStore.UnlockKey(kOpComp, []byte(pass)); err != nil {
		return nil, fmt.Errorf("Error unlocking babyjub key from keystore: %w", err)
	}
	// Load existing Identity (holder)
	holdr, err := holder.Load(storage, keyStore, idenPubOnChain, nil, readerhttp.NewIdenPubOffChainHttp())
	if err != nil {
		return nil, err
	}
	// Init event manager
	eventQueue := make(chan Event, 10)
	em, err := NewEventManager(storage, eventQueue)
	if err != nil {
		return nil, err
	}
	em.Start()

	// Init Identity
	iden := &Identity{
		id:          holdr,
		storage:     storage,
		Tickets:     NewTickets(storage.WithPrefix([]byte(ticketPrefix))),
		stopTickets: make(chan bool),
		eventMan:    em,
		ClaimDB:     NewClaimDB(storage.WithPrefix([]byte(credExisPrefix))),
	}
	go iden.Tickets.CheckPending(iden, em.eventQueue, time.Duration(checkTicketsPeriodMilis)*time.Millisecond, iden.stopTickets)
	return iden, nil
}

// GetNextEvent returns the oldest event that has been generated.
// Note that each event can only be retireved once.
// Note that this function is blocking and potentially for a very long time.
func (i *Identity) GetNextEvent() (*Event, error) {
	return i.eventMan.GetNextEvent()
}

// Stop close all the open resources of the Identity
func (i *Identity) Stop() {
	log.Info("Stopping identity: ", i.id.ID())
	defer i.storage.Close()
	i.stopTickets <- true
	i.eventMan.Stop()
}

// RequestClaim sends a petition to issue a claim to an issuer.
// This function will eventually trigger an event,
// the returned ticket can be used to reference the event
func (i *Identity) RequestClaim(baseUrl, data string) (*Ticket, error) {
	id := uuid.New().String()
	t := &Ticket{
		Id:     id,
		Type:   TicketTypeClaimReq,
		Status: TicketStatusPending,
	}
	httpClient := NewHttpClient(baseUrl)
	res := issuerMsg.ResClaimRequest{}
	if err := httpClient.DoRequest(httpClient.NewRequest().Path(
		"claim/request").Post("").BodyJSON(&issuerMsg.ReqClaimRequest{
		Value: data,
	}), &res); err != nil {
		return nil, err
	}
	t.handler = &reqClaimHandler{
		Id:      res.Id,
		BaseUrl: baseUrl,
		Status:  string(issuerMsg.RequestStatusPending),
	}
	err := i.Tickets.Add([]Ticket{*t})
	return t, err
}

type CallbackRequestClaim interface {
	Fn(*Ticket, error)
}

func (i *Identity) RequestClaimWithCb(baseUrl, data string, c CallbackRequestClaim) {
	go func() { c.Fn(i.RequestClaim(baseUrl, data)) }()
}

// ProveCredential sends a credentialValidity build from the given credentialExistance to a verifier
// the callback is used to check if the verifier has accepted the credential as valid
func (i *Identity) ProveClaim(baseUrl string, credId []byte) (bool, error) {
	// TODO: add context
	// Get credential existance
	credExis, err := i.ClaimDB.GetReceivedCredential(credId)
	if err != nil {
		return false, err
	}
	// Build credential validity
	credVal, err := i.id.HolderGetCredentialValidity(credExis)
	if err != nil {
		return false, err
	}
	// Send credential to verifier
	httpClient := NewHttpClient(baseUrl)
	if err := httpClient.DoRequest(httpClient.NewRequest().Path(
		"verify").Post("").BodyJSON(verifierMsg.ReqVerify{
		CredentialValidity: credVal,
	}), nil); err != nil {
		// Credential declined / error
		return false, err
	}
	// Success
	return true, nil
}

type CallbackProveClaim interface {
	Fn(bool, error)
}

func (i *Identity) ProveClaimWithCb(baseUrl string, credId []byte, c CallbackProveClaim) {
	go func() { c.Fn(i.ProveClaim(baseUrl, credId)) }()
}
