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
	Tickets     *TicketsMap
	eventSender Event
	storage     db.Storage
}

// TODO:
// store on changes vs store on stop

// NewIdentity creates a new identity
// this funciton is mapped as a constructor in Java.
// NOTE: The storePath must be unique per Identity
func NewIdentity(storePath, pass, web3Url string, checkTicketsPeriodMilis int, extraGenesisClaims *BytesArray, e Event) (*Identity, error) {
	id := &Identity{}
	idenPubOnChain, keyStore, storage, err := loadComponents(storePath, web3Url)
	if err != nil {
		return id, err
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
		return id, err
	}
	if err = keyStore.UnlockKey(kOpComp, []byte(pass)); err != nil {
		return id, err
	}
	// Store kOpComp
	tx, err := storage.NewTx()
	if err != nil {
		return id, err
	}
	if err := db.StoreJSON(tx, []byte("kOpComp"), kOpComp); err != nil {
		return id, err
	}
	// Init claim DB
	tx.Put([]byte("receivedCredentialsCounter"), []byte("0"))
	if err := tx.Commit(); err != nil {
		return id, err
	}
	// Parse extra genesis claims
	// TODO: Call toClaimers once it's implemented
	// _extraGenesisClaims, err := extraGenesisClaims.toClaimers()
	if err != nil {
		return id, err
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
		return id, err
	}
	// Verify that the Identity can be loaded successfully
	keyStore.Close()
	storage.Close()
	resourcesAreClosed = true
	return NewIdentityLoad(storePath, pass, web3Url, checkTicketsPeriodMilis, e)
}

// NewIdentityLoad loads an already created identity
// this funciton is mapped as a constructor in Java
func NewIdentityLoad(storePath, pass, web3Url string, checkTicketsPeriodMilis int, e Event) (*Identity, error) {
	// TODO: figure out how to diferentiate the two constructors from Java: https://github.com/iden3/iden3-mobile/issues/17#issuecomment-587374644
	id := &Identity{}
	idenPubOnChain, keyStore, storage, err := loadComponents(storePath, web3Url)
	if err != nil {
		return id, err
	}
	defer keyStore.Close()
	// Unlock key store
	kOpComp := &babyjub.PublicKeyComp{}
	if err := db.LoadJSON(storage, []byte("kOpComp"), kOpComp); err != nil {
		return id, err
	}
	if err := keyStore.UnlockKey(kOpComp, []byte(pass)); err != nil {
		return nil, fmt.Errorf("Error unlocking babyjub key from keystore: %w", err)
	}
	// Load existing Identity (holder)
	holdr, err := holder.Load(storage, keyStore, idenPubOnChain, nil, readerhttp.NewIdenPubOffChainHttp())
	if err != nil {
		return id, err
	}
	// Init Identity
	id.id = holdr
	id.Tickets = &TicketsMap{
		m: make(map[string]*Ticket),
	}
	id.eventSender = e
	id.storage = storage
	id.loadTickets()
	go id.checkPendingTickets(time.Duration(checkTicketsPeriodMilis * 1000000))
	return id, nil
}

// TODO: update marshal functions (s *Struct) ==> (s Struct)

// Stop close all the open resources of the Identity
func (i *Identity) Stop() {
	log.Info("Stopping identity: ", i.id.ID())
	defer i.storage.Close()
	// Send "singnal" to stop the pending tickets loop
	i.Tickets.shouldStop = true
	i.storeTickets()
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
	storagePath := baseStorePath + "/idStore"
	// Open database
	storage, err := db.NewLevelDbStorage(storagePath, false)
	if err != nil {
		return nil, fmt.Errorf("Error opening leveldb storage: %w", err)
	}
	log.WithField("path", storagePath).Info("Storage opened")
	return storage, nil
}

func loadKeyStoreBabyJub(baseStorePath string) (*babykeystore.KeyStore, error) {
	storagePath := baseStorePath + "/keyStore"
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
		IdenStates: common.HexToAddress("0xF6a014Ac66bcdc1BF51ac0fa68DF3f17f4b3e574"), // TODO: hardcode the address
	}
	return idenpubonchain.New(client, addresses), nil
}
