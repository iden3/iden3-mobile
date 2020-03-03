package iden3mobile

import (
	"fmt"
	"strings"
	"time"

	"github.com/ethereum/go-ethereum/accounts"
	ethkeystore "github.com/ethereum/go-ethereum/accounts/keystore"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/iden3/go-iden3-core/components/idenpuboffchain/readerhttp"
	"github.com/iden3/go-iden3-core/components/idenpubonchain"
	"github.com/iden3/go-iden3-core/db"
	"github.com/iden3/go-iden3-core/eth"
	"github.com/iden3/go-iden3-core/identity/holder"
	babykeystore "github.com/iden3/go-iden3-core/keystore"
	"github.com/iden3/go-iden3-crypto/babyjub"
	log "github.com/sirupsen/logrus"
)

type Identity struct {
	id          *holder.Holder
	storage     db.Storage
	tickets     *Tickets
	ClaimDB     *ClaimDB
	stopTickets chan bool
	eventQueue  chan Event
}

const (
	kOpStorKey           = "kOpComp"
	storageSubPath       = "/idStore"
	keyStorageSubPath    = "/idKeyStore"
	smartContractAddress = "0xF6a014Ac66bcdc1BF51ac0fa68DF3f17f4b3e574"
)

// NewIdentity creates a new identity
// this funciton is mapped as a constructor in Java.
// NOTE: The storePath must be unique per Identity.
// NOTE: Right now the extraGenesisClaims is useless.
func NewIdentity(storePath, pass, web3Url string, checkTicketsPeriodMilis int, extraGenesisClaims *BytesArray) (*Identity, error) {
	idenPubOnChain, keyStore, storage, err := loadComponents(storePath, web3Url)
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
	// Create new Identity (holder)
	_, err = holder.New(
		holder.ConfigDefault,
		kOpComp,
		nil,
		storage,
		keyStore,
		idenPubOnChain,
		nil,
		readerhttp.NewIdenPubOffChainHttp(),
	)
	if err != nil {
		return nil, err
	}
	// Init claim DB
	// cdb := NewClaimDB(storage.WithPrefix([]byte(credExisPrefix)))
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
	// Init Identity
	iden := &Identity{
		id:          holdr,
		storage:     storage,
		tickets:     NewTickets(storage.WithPrefix([]byte(ticketPrefix))),
		stopTickets: make(chan bool),
		eventQueue:  make(chan Event, 10),
		ClaimDB:     NewClaimDB(storage.WithPrefix([]byte(credExisPrefix))),
	}
	go iden.tickets.CheckPending(iden, iden.eventQueue, time.Duration(checkTicketsPeriodMilis)*time.Millisecond, iden.stopTickets)
	return iden, nil
}

// GetNextEvent returns the oldest event that has been generated.
// Note that each event can only be retireved once.
// Note that this function is blocking and potentially for a very long time.
func (i *Identity) GetNextEvent() *Event {
	ev := <-i.eventQueue
	return &ev
}

// Stop close all the open resources of the Identity
func (i *Identity) Stop() {
	log.Info("Stopping identity: ", i.id.ID())
	defer i.storage.Close()
	i.stopTickets <- true
}

func loadComponents(storePath, web3Url string) (idenpubonchain.IdenPubOnChainer, *babykeystore.KeyStore, db.Storage, error) {
	iPub, err := loadIdenPubOnChain(web3Url)
	if err != nil {
		return nil, nil, nil, err
	}
	storage, err := loadStorage(storePath)
	if err != nil {
		return nil, nil, nil, err
	}
	ks, err := loadKeyStoreBabyJub(storePath)
	if err != nil {
		return nil, nil, nil, err
	}
	return iPub, ks, storage, nil
}

func loadStorage(baseStorePath string) (db.Storage, error) {
	storagePath := baseStorePath + storageSubPath
	// Open database
	storage, err := db.NewLevelDbStorage(storagePath, false)
	if err != nil {
		return nil, fmt.Errorf("Error opening leveldb storage: %w", err)
	}
	log.WithField("path", storagePath).Info("Storage opened")
	return storage, nil
}

func loadKeyStoreBabyJub(baseStorePath string) (*babykeystore.KeyStore, error) {
	storagePath := baseStorePath + keyStorageSubPath
	// Open keystore
	storage := babykeystore.NewFileStorage(storagePath)
	ks, err := babykeystore.NewKeyStore(storage, babykeystore.StandardKeyStoreParams)
	if err != nil {
		return nil, fmt.Errorf("Error creating/opening babyjub keystore: %w", err)
	}
	return ks, nil
}

// WARNING: THIS CODE IS COPIED FROM go-iden3-servers/loaders/loaders.go
func loadEthClient2(ks *ethkeystore.KeyStore, acc *accounts.Account, web3Url string) (*eth.Client2, error) {
	// TODO: Handle the hidden: thing with a custon configuration type
	hidden := strings.HasPrefix(web3Url, "hidden:")
	if hidden {
		web3Url = web3Url[len("hidden:"):]
	}
	client, err := ethclient.Dial(web3Url)
	if err != nil {
		return nil, fmt.Errorf("Error dialing with ethclient: %w", err)
	}
	if hidden {
		log.WithField("url", "(hidden)").Info("Connection to web3 server opened")
	} else {
		log.WithField("url", web3Url).Info("Connection to web3 server opened")
	}
	return eth.NewClient2(client, acc, ks), nil
}

func loadIdenPubOnChain(web3Url string) (idenpubonchain.IdenPubOnChainer, error) {
	client, err := loadEthClient2(nil, nil, web3Url)
	if err != nil {
		return nil, err
	}
	addresses := idenpubonchain.ContractAddresses{
		IdenStates: common.HexToAddress(smartContractAddress), // TODO: hardcode the address
	}
	return idenpubonchain.New(client, addresses), nil
}
