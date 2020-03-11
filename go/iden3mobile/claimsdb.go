package iden3mobile

import (
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"sync"

	"github.com/iden3/go-iden3-core/common"
	"github.com/iden3/go-iden3-core/core/claims"
	"github.com/iden3/go-iden3-core/core/proof"
	"github.com/iden3/go-iden3-core/db"
	log "github.com/sirupsen/logrus"
)

type ClaimDB struct {
	storage db.Storage
	m       sync.Mutex
}

func NewClaimDB(storage db.Storage) *ClaimDB {
	return &ClaimDB{storage: storage}
}

// AddCredentialExistance adds a credential existence to the ClaimDB and
// returns the id of the entry in the DB.
func (cdb *ClaimDB) AddCredentialExistance(cred *proof.CredentialExistence) ([]byte, error) {
	cdb.m.Lock()
	defer cdb.m.Unlock()
	tx, err := cdb.storage.NewTx()
	if err != nil {
		return nil, err
	}
	id := sha256.Sum256(append(cred.Id[:], cred.Claim.Bytes()...))
	if _, err := tx.Get(id[:]); err == nil {
		return nil, fmt.Errorf("Credentail already exsits in the ClaimDB")
	}
	if err := db.StoreJSON(tx, id[:], cred); err != nil {
		return nil, err
	}
	if err := tx.Commit(); err != nil {
		return nil, err
	}
	log.WithField("key", common.Hex(id[:])).Info("Stored new existence credential")
	return id[:], nil
}

// GetReceivedClaim returns the requested claim
func (cdb *ClaimDB) GetReceivedClaim(id []byte) ([]byte, error) {
	cred, err := cdb.GetReceivedCredential(id)
	if err != nil {
		return nil, err
	}
	// TODO: return something nicer than bytes (metadata)
	return cred.Claim.Bytes(), nil
}

func (cdb *ClaimDB) GetReceivedCredential(id []byte) (*proof.CredentialExistence, error) {
	log.WithField("key", common.Hex(id)).Info("Loading existence credential")
	var cred proof.CredentialExistence
	if err := db.LoadJSON(cdb.storage, id, &cred); err != nil {
		return nil, err
	}
	return &cred, nil
}

func (cdb *ClaimDB) Iterate_(fn func([]byte, *proof.CredentialExistence) (bool, error)) error {
	if err := cdb.storage.Iterate(
		func(key, value []byte) (bool, error) {
			var cred proof.CredentialExistence
			if err := db.LoadJSON(cdb.storage, key, &cred); err != nil {
				return false, err
			}
			return fn(key, &cred)
		},
	); err != nil {
		return err
	}
	return nil
}

func (cdb *ClaimDB) IterateCredExistJSON_(fn func([]byte, string) (bool, error)) error {
	return cdb.Iterate_(
		func(id []byte, cred *proof.CredentialExistence) (bool, error) {
			credJSON, err := json.Marshal(cred)
			if err != nil {
				return false, err
			}
			return fn(id, string(credJSON))
		},
	)
}

func (cdb *ClaimDB) IterateClaimsJSON_(fn func([]byte, string) (bool, error)) error {
	return cdb.Iterate_(
		func(id []byte, cred *proof.CredentialExistence) (bool, error) {
			var claimData interface{}
			var err error
			claimData, err = claims.NewClaimFromEntry(cred.Claim)
			if err == claims.ErrInvalidClaimType {
				claimData = cred.Claim
			}
			var claim struct {
				Metadata claims.Metadata
				Data     interface{}
			}
			claim.Metadata.Unmarshal(cred.Claim)
			claim.Data = claimData
			claimJSON, err := json.Marshal(claim)
			if err != nil {
				return false, err
			}
			return fn(id, string(claimJSON))
		},
	)
}

type ClaimDBIterFner interface {
	Fn([]byte, string) (bool, error)
}

func (cdb *ClaimDB) IterateCredExistJSON(iterFn ClaimDBIterFner) error {
	return cdb.IterateCredExistJSON_(iterFn.Fn)
}

func (cdb *ClaimDB) IterateClaimsJSON(iterFn ClaimDBIterFner) error {
	return cdb.IterateClaimsJSON_(iterFn.Fn)
}
