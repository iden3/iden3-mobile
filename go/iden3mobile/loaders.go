package iden3mobile

import (
	"fmt"
	"strings"

	"github.com/ethereum/go-ethereum/accounts"
	ethkeystore "github.com/ethereum/go-ethereum/accounts/keystore"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/iden3/go-iden3-core/components/idenpubonchain"
	"github.com/iden3/go-iden3-core/db"
	"github.com/iden3/go-iden3-core/eth"
	babykeystore "github.com/iden3/go-iden3-core/keystore"
	log "github.com/sirupsen/logrus"
)

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
func loadEthClient(ks *ethkeystore.KeyStore, acc *accounts.Account, web3Url string) (*eth.Client, error) {
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
	return eth.NewClient(client, acc, ks), nil
}

func loadIdenPubOnChain(web3Url string) (idenpubonchain.IdenPubOnChainer, error) {
	client, err := loadEthClient(nil, nil, web3Url)
	if err != nil {
		return nil, err
	}
	addresses := idenpubonchain.ContractAddresses{
		IdenStates: common.HexToAddress(smartContractAddress), // TODO: hardcode the address
	}
	return idenpubonchain.New(client, addresses), nil
}
