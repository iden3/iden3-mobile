package iden3mobile

import (
	"strconv"

	"github.com/iden3/go-iden3-core/core/proof"
	"github.com/iden3/go-iden3-core/db"
	log "github.com/sirupsen/logrus"
)

const (
	credExisPrefix = "credExis"
	credCounterKey = "credCounter"
)

func (i *Identity) addCredentialExistance(cred proof.CredentialExistence) error {
	// TODO: make it thread safe
	// TODO: USe hash of (id + claim) as key
	credDB := i.storage.WithPrefix([]byte(credExisPrefix))
	tx, err := credDB.NewTx()
	if err != nil {
		return err
	}
	counterStr, err := tx.Get([]byte(credCounterKey))
	if err != nil {
		return err
	}
	counter, err := strconv.Atoi(string(counterStr))
	if err != nil {
		return err
	}
	if err := db.StoreJSON(tx, []byte(strconv.Itoa(counter)), cred); err != nil {
		return err
	}
	counter++
	tx.Put([]byte(credCounterKey), []byte(strconv.Itoa(counter)))
	if err := tx.Commit(); err != nil {
		return err
	}
	log.Info("Stored new existence credential, with key = ", strconv.Itoa(counter-1))
	return nil
}

// GetReceivedClaimsLen return the amount of received claims by the identity
func (i *Identity) GetReceivedClaimsLen() (int, error) {
	credDB := i.storage.WithPrefix([]byte(credExisPrefix))
	tx, err := credDB.NewTx()
	if err != nil {
		return 0, err
	}
	counterStr, err := tx.Get([]byte(credCounterKey))
	if err != nil {
		return 0, err
	}
	counter, err := strconv.Atoi(string(counterStr))
	if err != nil {
		return 0, err
	}
	return counter, nil
}

// GetReceivedClaim returns the requested claim
func (i *Identity) GetReceivedClaim(pos int) ([]byte, error) {
	cred, err := i.getReceivedCredential(pos)
	if err != nil {
		return nil, err
	}
	// TODO: return something nicer than bytes (metadata)
	return cred.Claim.Bytes(), nil
}

func (i *Identity) getReceivedCredential(pos int) (proof.CredentialExistence, error) {
	var cred proof.CredentialExistence
	log.Info("Loading existence credential, with key = ", strconv.Itoa(pos))
	credDB := i.storage.WithPrefix([]byte(credExisPrefix))
	if err := db.LoadJSON(credDB, []byte(strconv.Itoa(pos)), &cred); err != nil {
		return cred, err
	}
	return cred, nil
}
