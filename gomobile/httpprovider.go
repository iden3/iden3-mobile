package gomobile

import (
	"encoding/json"
	"fmt"

	"github.com/iden3/go-iden3-core/core"
	"github.com/iden3/go-iden3-core/merkletree"
	"github.com/iden3/go-iden3-crypto/babyjub"
	"github.com/iden3/go-iden3-light-wallet/identityprovider"
)

type ExportParams identityprovider.ExportParams
type ImportParams identityprovider.ImportParams
type HttpProviderParams identityprovider.HttpProviderParams

// Interface that is gomobile-friendly
type KeyStorerGoMobile interface {
	SignBaby(pk []byte, msg []byte) ([]byte, error)
}

// Implementation of the indentityprovider.KeyStore that wraps a struct that implements KeyStorerGoMobile
type KeyStoreFromGoMobile struct {
	keyStore KeyStorerGoMobile
}

func NewKeyStoreFromGoMobile(keyStore KeyStorerGoMobile) *KeyStoreFromGoMobile {
	return &KeyStoreFromGoMobile{
		keyStore: keyStore,
	}
}

// func (ks *KeyStoreGoMobile) SignBaby(_pk []byte, msg []byte) ([]byte, error) {
// 	__pk := PublicKey(_pk)
// 	pk, err := __pk.toCore()
// 	sig, err := ks.keyStore.SignBaby(pk)
// 	return sig[:], err
// }
func (ks *KeyStoreFromGoMobile) SignBaby(pk *babyjub.PublicKeyComp, msg []byte) (*babyjub.SignatureComp, error) {
	_sig, err := ks.keyStore.SignBaby(pk[:], msg)
	if err != nil {
		return nil, err
	}
	if len(_sig) != 32 {
		return nil, fmt.Errorf("Compressed Signature must be 32 bytes")
	}
	sig := babyjub.SignatureComp{}
	copy(sig[:], _sig)
	return &sig, nil
}

// type KeyStore struct {
// 	keyStore identityprovider.KeyStorer
// }
//
// func NewKeyStore() {
//
// }

type PublicKey []byte

func (pk *PublicKey) toCore() (*babyjub.PublicKey, error) {
	pkc := babyjub.PublicKeyComp{}
	if len((*pk)[:]) != 32 {
		return nil, fmt.Errorf("Compressed Public Key must be 32 bytes")
	}
	copy(pkc[:], (*pk)[:])
	return pkc.Decompress()
}

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
	provider *identityprovider.HttpProvider
}

func NewHttpProvider(params *HttpProviderParams) *HttpProvider {
	return &HttpProvider{
		provider: identityprovider.NewHttpProvider(identityprovider.HttpProviderParams(*params)),
	}
}

func (p *HttpProvider) CreateIdentity(_keyStore KeyStorerGoMobile, _kOp []byte,
	_extraGenesisClaims *BytesArray) ([]byte, error) {
	extraGenesisClaims, err := _extraGenesisClaims.toEntries()
	if err != nil {
		return nil, err
	}
	__kOp := PublicKey(_kOp)
	kOp, err := __kOp.toCore()
	keyStore := NewKeyStoreFromGoMobile(_keyStore)
	id, err := p.provider.CreateIdentity(keyStore, kOp, extraGenesisClaims)
	if err != nil {
		return nil, err
	}
	_id := IDFromCore(id)
	return _id, nil
}

type Identity struct {
	iden identityprovider.Identity
}

func (i *Identity) Export(exportFilePath string, exportParams *ExportParams) error {
	return i.iden.Export(exportFilePath, identityprovider.ExportParams(*exportParams))
}

func (i *Identity) Import(importFilePath string, importParams *ImportParams) error {
	return i.iden.Import(importFilePath, identityprovider.ImportParams(*importParams))
}

func (i *Identity) ID() []byte {
	return IDFromCore(i.iden.ID())
}

func (i *Identity) AddClaim(_claim []byte) error {
	__claim := Entry(_claim)
	claim, err := __claim.toCore()
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

func (i *Identity) GenProofClaim(_claim []byte) ([]byte, error) {
	__claim := Entry(_claim)
	claim, err := __claim.toCore()
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

func (i *Identity) VerifyProofClaim(_proof []byte) (bool, error) {
	__proof := ProofClaim(_proof)
	proof, err := __proof.toCore()
	if err != nil {
		return false, err
	}
	return i.iden.VerifyProofClaim(proof)
}
