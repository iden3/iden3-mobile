package iden3mobile

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"sync"

	"github.com/iden3/go-iden3-core/core/claims"
	"github.com/iden3/go-iden3-core/core/proof"
	"github.com/iden3/go-iden3-core/db"
	"github.com/iden3/go-iden3-core/merkletree"
	log "github.com/sirupsen/logrus"
)

func claim2JSON(e *merkletree.Entry) ([]byte, error) {
	var claimData interface{}
	var err error
	claimData, err = claims.NewClaimFromEntry(e)
	if err == claims.ErrInvalidClaimType {
		claimData = e
	}
	var claim struct {
		Metadata claims.Metadata
		Data     interface{}
	}
	claim.Metadata.Unmarshal(e)
	claim.Data = claimData
	claimJSON, err := json.Marshal(claim)
	if err != nil {
		return nil, err
	}
	return claimJSON, nil
}

type ClaimDB struct {
	storage db.Storage
	m       sync.Mutex
}

func NewClaimDB(storage db.Storage) *ClaimDB {
	return &ClaimDB{storage: storage}
}

// AddCredentialExistance adds a credential existence to the ClaimDB and
// returns the id of the entry in the DB.
func (cdb *ClaimDB) AddCredentialExistance(cred *proof.CredentialExistence) (string, error) {
	cdb.m.Lock()
	defer cdb.m.Unlock()
	tx, err := cdb.storage.NewTx()
	if err != nil {
		return "", err
	}
	h := sha256.Sum256(append(cred.Id[:], cred.Claim.Bytes()...))
	id := hex.EncodeToString(h[:160/8]) // Take 160 bits of the sha256 and encode them in hex
	if _, err := tx.Get([]byte(id)); err == nil {
		return "", fmt.Errorf("Credentail already exsits in the ClaimDB")
	}
	if err := db.StoreJSON(tx, []byte(id), cred); err != nil {
		return "", err
	}
	if err := tx.Commit(); err != nil {
		return "", err
	}
	log.WithField("key", id).Info("Stored new existence credential")
	return id, nil
}

// GetCredExistJSON returns the requested claim
func (cdb *ClaimDB) GetCredExistJSON(id string) (string, error) {
	cred, err := cdb.GetCredExist(id)
	if err != nil {
		return "", err
	}
	credJSON, err := json.Marshal(cred)
	if err != nil {
		return "", err
	}
	return string(credJSON), nil
}

func (cdb *ClaimDB) GetClaimJSON(id string) (string, error) {
	cred, err := cdb.GetCredExist(id)
	if err != nil {
		return "", err
	}
	claimJSON, err := claim2JSON(cred.Claim)
	if err != nil {
		return "", err
	}
	return string(claimJSON), nil
}

func (cdb *ClaimDB) GetCredExist(id string) (*proof.CredentialExistence, error) {
	log.WithField("key", id).Info("Loading existence credential")
	var cred proof.CredentialExistence
	if err := db.LoadJSON(cdb.storage, []byte(id), &cred); err != nil {
		return nil, err
	}
	return &cred, nil
}

func (cdb *ClaimDB) Iterate_(fn func(string, *proof.CredentialExistence) (bool, error)) error {
	if err := cdb.storage.Iterate(
		func(key, value []byte) (bool, error) {
			var cred proof.CredentialExistence
			if err := db.LoadJSON(cdb.storage, key, &cred); err != nil {
				return false, err
			}
			return fn(string(key), &cred)
		},
	); err != nil {
		return err
	}
	return nil
}

func (cdb *ClaimDB) IterateCredExistJSON_(fn func(string, string) (bool, error)) error {
	return cdb.Iterate_(
		func(id string, cred *proof.CredentialExistence) (bool, error) {
			credJSON, err := json.Marshal(cred)
			if err != nil {
				return false, err
			}
			return fn(id, string(credJSON))
		},
	)
}

func (cdb *ClaimDB) IterateClaimsJSON_(fn func(string, string) (bool, error)) error {
	return cdb.Iterate_(
		func(id string, cred *proof.CredentialExistence) (bool, error) {
			claimJSON, err := claim2JSON(cred.Claim)
			if err != nil {
				return false, err
			}
			return fn(id, string(claimJSON))
		},
	)
}

type ClaimDBIterFner interface {
	Fn(string, string) (bool, error)
}

func (cdb *ClaimDB) IterateCredExistJSON(iterFn ClaimDBIterFner) error {
	return cdb.IterateCredExistJSON_(iterFn.Fn)
}

func (cdb *ClaimDB) IterateClaimsJSON(iterFn ClaimDBIterFner) error {
	return cdb.IterateClaimsJSON_(iterFn.Fn)
}
