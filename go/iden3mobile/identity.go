package iden3mobile

import (
	"errors"
	"fmt"
	"io"
	"os"
	"path"
	"time"

	"github.com/google/uuid"
	"github.com/iden3/go-iden3-core/components/idenpuboffchain/readerhttp"
	"github.com/iden3/go-iden3-core/components/idenpubonchain"
	"github.com/iden3/go-iden3-core/db"
	"github.com/iden3/go-iden3-core/identity/holder"
	babykeystore "github.com/iden3/go-iden3-core/keystore"
	"github.com/iden3/go-iden3-crypto/babyjub"
	issuerMsg "github.com/iden3/go-iden3-servers-demo/servers/issuerdemo/messages"
	verifierMsg "github.com/iden3/go-iden3-servers-demo/servers/verifier/messages"
	log "github.com/sirupsen/logrus"
)

type Identity struct {
	id          *holder.Holder
	storage     db.Storage
	keyStore    *babykeystore.KeyStore
	ClaimDB     *ClaimDB
	Tickets     *Tickets
	stopTickets chan bool
	eventMan    *EventManager
}

const (
	kOpStorKey           = "kOpComp"
	eventsStorKey        = "eventsKey"
	storageSubPath       = "/idStore"
	keyStorageSubPath    = "/idKeyStore"
	smartContractAddress = "0x09561a45339910894705419af321c69a8832eab4"
	credExisPrefix       = "credExis"
	folderStore          = "store"
	folderKeyStore       = "keystore"
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
func NewIdentity(storePath, pass, web3Url string, checkTicketsPeriodMilis int, extraGenesisClaims *BytesArray, eventHandler Sender) (*Identity, error) {
	idenPubOnChain, err := loadIdenPubOnChain(web3Url)
	if err != nil {
		return nil, err
	}
	return newIdentity(storePath, pass, idenPubOnChain, checkTicketsPeriodMilis, extraGenesisClaims, eventHandler)
}

func newIdentity(storePath, pass string, idenPubOnChain idenpubonchain.IdenPubOnChainer,
	checkTicketsPeriodMilis int, extraGenesisClaims *BytesArray, eventHandler Sender) (*Identity, error) {
	// Check that storePath points to an empty dir
	if dirIsEmpty, err := isEmpty(storePath); !dirIsEmpty || err != nil {
		if err == nil {
			err = errors.New("Directory is not empty")
		}
		return nil, err
	}
	storagePath := path.Join(storePath, folderStore)
	if err := os.Mkdir(storagePath, 0700); err != nil {
		return nil, err
	}
	storage, err := loadStorage(storagePath)
	if err != nil {
		return nil, err
	}
	keyStoreStoragePath := path.Join(storePath, folderKeyStore)
	if err := os.Mkdir(keyStoreStoragePath, 0700); err != nil {
		return nil, err
	}
	keyStore, err := loadKeyStoreBabyJub(keyStoreStoragePath)
	if err != nil {
		return nil, err
	}
	resourcesAreClosed := false
	defer func() {
		if !resourcesAreClosed {
			if err := keyStore.Close(); err != nil {
				log.WithError(err).Error("keyStore.Close()")
			}
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
	// Init event mannager
	if err := NewEventManager(storage, nil, nil).Init(); err != nil {
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
	if err := keyStore.Close(); err != nil {
		log.WithError(err).Error("keyStore.Close()")
	}
	storage.Close()
	resourcesAreClosed = true
	return newIdentityLoad(storePath, pass, idenPubOnChain, checkTicketsPeriodMilis, eventHandler)
}

// NewIdentityLoad loads an already created identity
// this funciton is mapped as a constructor in Java
func NewIdentityLoad(storePath, pass, web3Url string, checkTicketsPeriodMilis int, eventHandler Sender) (*Identity, error) {
	idenPubOnChain, err := loadIdenPubOnChain(web3Url)
	if err != nil {
		return nil, err
	}
	return newIdentityLoad(storePath, pass, idenPubOnChain, checkTicketsPeriodMilis, eventHandler)
}

func newIdentityLoad(storePath, pass string, idenPubOnChain idenpubonchain.IdenPubOnChainer, checkTicketsPeriodMilis int, eventHandler Sender) (*Identity, error) {
	// TODO: figure out how to diferentiate the two constructors from Java: https://github.com/iden3/iden3-mobile/issues/17#issuecomment-587374644
	storage, err := loadStorage(path.Join(storePath, folderStore))
	if err != nil {
		return nil, err
	}
	keyStore, err := loadKeyStoreBabyJub(path.Join(storePath, folderKeyStore))
	if err != nil {
		return nil, err
	}
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
	eventQueue := make(chan Event, 16)
	em := NewEventManager(storage, eventQueue, eventHandler)
	em.Start()

	// Init Identity
	iden := &Identity{
		id:          holdr,
		storage:     storage,
		keyStore:    keyStore,
		Tickets:     NewTickets(storage.WithPrefix([]byte(ticketPrefix))),
		stopTickets: make(chan bool),
		eventMan:    em,
		ClaimDB:     NewClaimDB(storage.WithPrefix([]byte(credExisPrefix))),
	}
	go iden.Tickets.CheckPending(iden, eventQueue, time.Duration(checkTicketsPeriodMilis)*time.Millisecond, iden.stopTickets)
	return iden, nil
}

// Export storage and keystore
func (i Identity) Export(pass []byte) (storage db.Storage, keyStore *babykeystore.KeyStore) {

	// Unlock key store
	kOpComp := &babyjub.PublicKeyComp{}
	if err := db.LoadJSON(i.storage, []byte(kOpStorKey), kOpComp); err != nil {
		return nil, nil
	}
	if err := i.keyStore.UnlockKey(kOpComp, []byte(pass)); err != nil {
		return nil, nil
	}
        return i.storage, i.keyStore
}

// Stop close all the open resources of the Identity
func (i *Identity) Stop() {
	log.Info("Stopping identity: ", i.id.ID())
	defer i.storage.Close()
	defer i.keyStore.Close()
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
func (i *Identity) ProveClaim(baseUrl string, credID string) (bool, error) {
	// TODO: add context
	// Get credential existance
	credExis, err := i.ClaimDB.GetCredExist(credID)
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

func (i *Identity) ProveClaimWithCb(baseUrl string, credID string, c CallbackProveClaim) {
	go func() { c.Fn(i.ProveClaim(baseUrl, credID)) }()
}
