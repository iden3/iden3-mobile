package iden3mobile

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"math/big"
	"os"
	"path"
	"time"

	"github.com/google/uuid"
	"github.com/iden3/go-iden3-core/components/idenpuboffchain/readerhttp"
	"github.com/iden3/go-iden3-core/components/idenpubonchain"
	"github.com/iden3/go-iden3-core/core/claims"
	"github.com/iden3/go-iden3-core/core/proof"
	"github.com/iden3/go-iden3-core/db"
	"github.com/iden3/go-iden3-core/identity/holder"
	babykeystore "github.com/iden3/go-iden3-core/keystore"
	"github.com/iden3/go-iden3-core/merkletree"
	zkutils "github.com/iden3/go-iden3-core/utils/zk"
	"github.com/iden3/go-iden3-crypto/babyjub"
	issuerMsg "github.com/iden3/go-iden3-servers-demo/servers/issuerdemo/messages"
	verifierMsg "github.com/iden3/go-iden3-servers-demo/servers/verifier/messages"
	log "github.com/sirupsen/logrus"
)

type Identity struct {
	id              *holder.Holder
	sharedStorePath string
	storage         db.Storage
	keyStore        *babykeystore.KeyStore
	ClaimDB         *ClaimDB
	Tickets         *Tickets
	stopTickets     chan bool
	eventMan        *EventManager
}

const (
	kOpStorKey           = "kOpComp"
	eventsStorKey        = "eventsKey"
	storageSubPath       = "/idStore"
	keyStorageSubPath    = "/idKeyStore"
	smartContractAddress = "0x4cd72fcedf61937ffc8995d7c0839c976f3cc129"
	credExistPrefix      = "credExist"
	folderStore          = "store"
	folderKeyStore       = "keystore"
	folderZKArtifacts    = "ZKArtifacts"
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
func NewIdentity(storePath, sharedStorePath, pass, web3Url string, checkTicketsPeriodMilis int, extraGenesisClaims *BytesArray, eventHandler Sender) (*Identity, error) {
	idenPubOnChain, err := loadIdenPubOnChain(web3Url)
	if err != nil {
		return nil, err
	}
	return newIdentity(storePath, sharedStorePath, pass, idenPubOnChain, checkTicketsPeriodMilis, extraGenesisClaims, eventHandler)
}

func newIdentity(storePath, sharedStorePath, pass string, idenPubOnChain idenpubonchain.IdenPubOnChainer,
	checkTicketsPeriodMilis int, extraGenesisClaims *BytesArray, eventHandler Sender) (*Identity, error) {
	// Check that storePath points to an empty dir
	if dirIsEmpty, err := isEmpty(storePath); !dirIsEmpty || err != nil {
		if err == nil {
			err = errors.New("Directory is not empty")
		}
		return nil, err
	}
	ZKPath := path.Join(sharedStorePath, folderZKArtifacts)
	if err := os.MkdirAll(ZKPath, 0700); err != nil {
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
	return newIdentityLoad(storePath, sharedStorePath, pass, idenPubOnChain, checkTicketsPeriodMilis, eventHandler)
}

// NewIdentityLoad loads an already created identity
// this funciton is mapped as a constructor in Java
func NewIdentityLoad(storePath, sharedStorePath, pass, web3Url string, checkTicketsPeriodMilis int, eventHandler Sender) (*Identity, error) {
	idenPubOnChain, err := loadIdenPubOnChain(web3Url)
	if err != nil {
		return nil, err
	}
	return newIdentityLoad(storePath, sharedStorePath, pass, idenPubOnChain, checkTicketsPeriodMilis, eventHandler)
}

func newIdentityLoad(storePath, sharedStorePath, pass string, idenPubOnChain idenpubonchain.IdenPubOnChainer, checkTicketsPeriodMilis int, eventHandler Sender) (*Identity, error) {
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
	holdr, err := holder.Load(
		storage,
		keyStore,
		idenPubOnChain,
		nil,
		nil,
		readerhttp.NewIdenPubOffChainHttp(),
	)
	if err != nil {
		return nil, err
	}
	// Init event manager
	eventQueue := make(chan Event, 16)
	em := NewEventManager(storage, eventQueue, eventHandler)
	em.Start()

	// Init Identity
	iden := &Identity{
		id:              holdr,
		storage:         storage,
		sharedStorePath: sharedStorePath,
		keyStore:        keyStore,
		Tickets:         NewTickets(storage.WithPrefix([]byte(ticketPrefix))),
		stopTickets:     make(chan bool),
		eventMan:        em,
		ClaimDB:         NewClaimDB(storage.WithPrefix([]byte(credExistPrefix))),
	}
	go iden.Tickets.CheckPending(iden, eventQueue, time.Duration(checkTicketsPeriodMilis)*time.Millisecond, iden.stopTickets)
	return iden, nil
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
	// Warning: This only applies to the current used claim!
	if len(data) > 16 {
		return nil, errors.New("The data string cannot be longer than 16 chars")
	}
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
		Value:    data,
		Index:    data,
		HolderID: i.id.ID(),
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

// ProveClaim sends a credentialValidity build from the given credentialExistance to a verifier.
// The response should be true if the verified accepted the prove as valid
func (i *Identity) ProveClaim(baseUrl string, credID string) (bool, error) {
	// Build credential validity
	credVal, err := i.getCredentialValidity(credID)
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

func (i *Identity) getCredentialValidity(credID string) (*proof.CredentialValidity, error) {
	// Get credential existance
	credExist, err := i.ClaimDB.GetCredExist(credID)
	if err != nil {
		return nil, err
	}
	// Build credential validity
	return i.id.HolderGetCredentialValidity(credExist)
}

// CallbackProveClaim is a interface used to get an asynchronous response from
// ProveClaimWithCb and ProveClaimZKWithCb
type CallbackProveClaim interface {
	Fn(bool, error)
}

// ProveClaimWithCb sends a credentialValidity build from the given credentialExistance to a verifier.
// The callback is used to check if the verifier has accepted the credential as valid in an async maner
func (i *Identity) ProveClaimWithCb(baseUrl string, credID string, c CallbackProveClaim) {
	go func() { c.Fn(i.ProveClaim(baseUrl, credID)) }()
}

// ProveClaimZK sends a credentialValidity build from the given credentialExistance to a verifier.
// This method will generate a zero knowledge proof so the verifier can't see the content of the claim.
// The response should be true if the verified accepted the prove as valid.
func (i *Identity) ProveClaimZK(baseUrl string, credID string) (bool, error) {
	// Get credential existance
	credExist, err := i.ClaimDB.GetCredExist(credID)
	if err != nil {
		return false, err
	}

	// DBG BEGIN
	// claimHex := make([]string, 8)
	// for i := 0; i < 8; i++ {
	// 	claimHex[i] = hex.EncodeToString(credExist.Claim.Data[i][:])
	// }
	// log.WithField("claim", claimHex).Debug("ProveClaimZK")
	// DBG END

	// Build credential ownership zk proof
	// WARNING: this is a hardcoded proof generation for a specific claim/circuit.
	// In the future we will add some mechanism that can deduce how to generate an arbitrary proof.
	proofName := "claimDemo"
	addInputs := func(claim *merkletree.Entry) func(inputs map[string]interface{}) error {
		return func(inputs map[string]interface{}) error {
			var metadata claims.Metadata
			metadata.Unmarshal(claim)
			data := claim.Data
			inputs["claimI2_3"] = []*big.Int{data[0*4+2].BigInt(), data[0*4+3].BigInt()}
			inputs["claimV1_3"] = []*big.Int{data[1*4+1].BigInt(), data[1*4+2].BigInt(), data[1*4+3].BigInt()}
			inputs["id"] = i.id.ID().BigInt()
			inputs["revNonce"] = new(big.Int).SetUint64(uint64(metadata.RevNonce))

			// DBG BEGIN
			in, err := zkutils.InputsToMapStrings(inputs)
			if err != nil {
				return err
			}
			inJSON, err := json.MarshalIndent(in, "", "  ")
			if err != nil {
				return err
			}
			fmt.Println(string(inJSON))
			// DBG END

			return nil
		}
	}
	ZKPath := path.Join(i.sharedStorePath, folderZKArtifacts, proofName)
	if err := os.MkdirAll(ZKPath, 0700); err != nil {
		return false, err
	}
	zkProofCredOut, err := i.id.HolderGenZkProofCredential(
		credExist,
		addInputs(credExist.Claim),
		4,
		16,
		zkutils.NewZkFiles(
			baseUrl+"credentialDemo/artifacts",
			ZKPath,
			zkutils.ProvingKeyFormatGoBin,
			zkutils.ZkFilesHashes{
				ProvingKey:      "bdefc89d07d1dfab75c43f09aedb9da876496c5c3967383337482e4c5ae4f7d3",
				VerificationKey: "12a730890e85e33d8bf0f2e54db41dcff875c2dc49011d7e2a283185f47ac0de",
				WitnessCalcWASM: "6b3c28c4842e04129674eb71dc84d76dd8b290c84987929d54d890b7b8bed211",
			},
			false,
		),
	)
	if err != nil {
		return false, err
	}

	// Send the CredentialValidity proof to Verifier
	httpClient := NewHttpClient(baseUrl)
	reqVerifyZkp := verifierMsg.ReqVerifyZkp{
		ZkProof:         &zkProofCredOut.ZkProofOut.Proof,
		PubSignals:      zkProofCredOut.ZkProofOut.PubSignals,
		IssuerID:        zkProofCredOut.IssuerID,
		IdenStateBlockN: zkProofCredOut.IdenStateBlockN,
	}
	if err := httpClient.DoRequest(httpClient.NewRequest().Path(
		"credentialDemo/verifyzkp").Post("").BodyJSON(&reqVerifyZkp), nil); err != nil {
		return false, err
	}
	return true, nil
}

// ProveClaimZKWithCb sends a credentialValidity build from the given credentialExistance to a verifier.
// This method will generate a zero knowledge proof so the verifier can't see the content of the claim.
// The callback is used to check if the verifier has accepted the credential as valid in an async maner
func (i *Identity) ProveClaimZKWithCb(baseUrl string, credID string, c CallbackProveClaim) {
	go func() { c.Fn(i.ProveClaimZK(baseUrl, credID)) }()
}
