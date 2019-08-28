package gomobile

import (
	"encoding/json"
	// "fmt"

	"github.com/iden3/go-iden3-core/core"
	"github.com/iden3/go-iden3-core/merkletree"
	"go-iden3-light-wallet/identityprovider"
)

type ID []byte

func (i *ID) toCore() (*core.ID, error) {
	id, err := core.IDFromBytes((*i)[:])
	return &id, err
}

func IDFromCore(id *core.ID) ID {
	return ID(id[:])
}

type Entry []byte

func (en *Entry) toCore() (*merkletree.Entry, error) {
	return merkletree.NewEntryFromBytes((*en)[:])
}

func EntryFromCore(e *merkletree.Entry) Entry {
	return Entry(e.Bytes())
}

type ProofClaim []byte

func (pc *ProofClaim) toCore() (*core.ProofClaim, error) {
	proofClaim := &core.ProofClaim{}
	err := json.Unmarshal((*pc)[:], proofClaim)
	return proofClaim, err
}

func ProofClaimFromCore(_proofClaim *core.ProofClaim) (ProofClaim, error) {
	bs, err := json.Marshal(_proofClaim)
	return ProofClaim(bs), err
}

type BytesArray struct {
	array [][]byte
}

func NewBytesArray() *BytesArray {
	return &BytesArray{
		array: make([][]byte, 0),
	}
}

func (ba *BytesArray) Len() int {
	return len(ba.array)
}

func (ba *BytesArray) Get(i int) []byte {
	return ba.array[i]
}

func (ba *BytesArray) Append(bs []byte) {
	ba.array = append(ba.array, bs)
}

func (ba *BytesArray) toEntries() ([]*merkletree.Entry, error) {
	claims := []*merkletree.Entry{}
	for i := 0; i < ba.Len(); i++ {
		_claim := Entry(ba.Get(i))
		claim, err := _claim.toCore()
		if err != nil {
			return nil, err
		}
		claims = append(claims, claim)
	}
	return claims, nil
}

type HttpProvider struct {
	provider identityprovider.HttpProvider
}

type Identity struct {
	iden identityprovider.Identity
}

func (i *Identity) Export(exportFilePath string, exportParams identityprovider.ExportParams) error {
	return i.iden.Export(exportFilePath, exportParams)
}

func (i *Identity) Import(importFilePath string, importParams identityprovider.ImportParams) error {
	return i.iden.Import(importFilePath, importParams)
}

func (i *Identity) ID() ID {
	return IDFromCore(i.iden.ID())
}

func (i *Identity) AddClaim(_claim Entry) error {
	claim, err := _claim.toCore()
	if err != nil {
		return err
	}
	return i.iden.AddClaim(claim)
}

func (i *Identity) AddClaims(_claims *BytesArray) error {
	claims, err := _claims.toEntries()
	if err != nil {
		return err
	}
	return i.iden.AddClaims(claims)
}

func (i *Identity) GenProofClaim(_claim Entry) (ProofClaim, error) {
	claim, err := _claim.toCore()
	if err != nil {
		return nil, err
	}
	proofClaim, err := i.iden.GenProofClaim(claim)
	if err != nil {
		return nil, err
	}
	return ProofClaimFromCore(proofClaim)
}

func (i *Identity) GenProofClaims(_claims *BytesArray) (*BytesArray, error) {
	claims, err := _claims.toEntries()
	if err != nil {
		return nil, err
	}
	proofs, err := i.iden.GenProofClaims(claims)
	if err != nil {
		return nil, err
	}
	_proofs := NewBytesArray()
	for _, proof := range proofs {
		_proof, err := ProofClaimFromCore(&proof)
		if err != nil {
			return nil, err
		}
		_proofs.Append(_proof[:])
	}
	return _proofs, nil
}

func (i *Identity) EmittedClaims() (*BytesArray, error) {
	claims, err := i.iden.EmittedClaims()
	if err != nil {
		return nil, err
	}
	_claims := NewBytesArray()
	for _, claim := range claims {
		_claims.Append(EntryFromCore(claim)[:])
	}
	return _claims, nil
}

func (i *Identity) ReceivedClaims() (*BytesArray, error) {
	claims, err := i.iden.ReceivedClaims()
	if err != nil {
		return nil, err
	}
	_claims := NewBytesArray()
	for _, claim := range claims {
		_claims.Append(EntryFromCore(claim)[:])
	}
	return _claims, nil
}

func (i *Identity) VerifyProofClaim(_proof ProofClaim) (bool, error) {
	proof, err := _proof.toCore()
	if err != nil {
		return false, err
	}
	return i.iden.VerifyProofClaim(proof)
}
