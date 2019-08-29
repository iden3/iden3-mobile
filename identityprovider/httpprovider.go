package identityprovider

import (
	"fmt"

	"github.com/dghubble/sling"
	"github.com/iden3/go-iden3-core/core"
	"github.com/iden3/go-iden3-core/keystore"
	"github.com/iden3/go-iden3-core/merkletree"
	"github.com/iden3/go-iden3-core/services/claimsrv"
	"github.com/iden3/go-iden3-core/services/identityagentsrv"
	"github.com/iden3/go-iden3-crypto/babyjub"
	"gopkg.in/go-playground/validator.v9"
)

type KeyStorer interface {
	SignBaby(pk *babyjub.PublicKeyComp, prefix keystore.PrefixType, msg []byte) (*babyjub.SignatureComp, int64, error)
}

type KeyStore struct {
	*keystore.KeyStore
}

func (ks *KeyStore) SignBaby(pk *babyjub.PublicKeyComp, prefix keystore.PrefixType, msg []byte) (*babyjub.SignatureComp, int64, error) {
	return ks.Sign(pk, prefix, msg)
}

type ExportParams struct {
	Passphrase string
}

type ImportParams struct {
	Passphrase string
}

type Identity interface {
	Export(exportFilePath string, exportParams ExportParams) error
	Import(importFilePath string, importParams ImportParams) error
	ID() *core.ID
	AddClaim(claim *merkletree.Entry) error
	AddClaims(claims []*merkletree.Entry) error
	GenProofClaim(claim *merkletree.Entry) (*core.ProofClaim, error)
	GenProofClaims(claims []*merkletree.Entry) ([]core.ProofClaim, error)
	EmittedClaims() ([]*merkletree.Entry, error)
	ReceivedClaims() ([]*merkletree.Entry, error)
	// DataObjects() ([]Data, error)
	VerifyProofClaim(proof *core.ProofClaim) (bool, error)
}

type ServerError struct {
	Err string `json:"error"`
}

func (e ServerError) Error() string {
	return fmt.Sprintf("server: %v", e.Err)
}

type HttpProvider struct {
	Url      string
	Username string
	Password string
	_client  *sling.Sling
	validate *validator.Validate
}

func (p *HttpProvider) client() *sling.Sling {
	return p._client.New()
}

type HttpProviderParams struct {
	Url      string
	Username string
	Password string
}

func NewHttpProvider(params HttpProviderParams) *HttpProvider {
	url := params.Url
	if url[len(url)-1] != '/' {
		url += "/"
	}
	client := sling.New().Base(url)
	return &HttpProvider{Url: url, Username: params.Username, Password: params.Password,
		_client: client, validate: validator.New()}
}

func (p *HttpProvider) request(s *sling.Sling, res interface{}) error {
	var serverError ServerError
	resp, err := s.Receive(res, &serverError)
	if err == nil {
		defer resp.Body.Close()
		if !(200 <= resp.StatusCode && resp.StatusCode < 300) {
			err = serverError
		} else if res != nil {
			err = p.validate.Struct(res)
		}
	}
	return err
}

type CreateIdentityReq struct {
	ClaimAuthKOp       *merkletree.Entry   `json:"claimAuthKOp" binding:"required"`
	ExtraGenesisClaims []*merkletree.Entry `json:"extraGenesisClaims"`
}

func (p *HttpProvider) CreateIdentity(keyStore KeyStorer, kOp *babyjub.PublicKey,
	extraGenesisClaims []*merkletree.Entry) (*core.ID, *core.ProofClaimGenesis, error) {

	claimAuthKOp := core.NewClaimAuthorizeKSignBabyJub(kOp)
	createIdentityReq := CreateIdentityReq{
		ClaimAuthKOp:       claimAuthKOp.Entry(),
		ExtraGenesisClaims: extraGenesisClaims,
	}

	var createIdentityRes struct {
		Id       *core.ID                `json:"id" validate:"required"`
		ProofKOp *core.ProofClaimGenesis `json:"proofKOp" validate:"required"`
	}
	err := p.request(p.client().Path("identity").Post("").BodyJSON(createIdentityReq), &createIdentityRes)
	if err != nil {
		return nil, nil, err
	}

	return createIdentityRes.Id, createIdentityRes.ProofKOp, nil
}

type HttpIdentity struct {
	provider *HttpProvider
	kOp      *babyjub.PublicKeyComp
	proofKOp *core.ProofClaimGenesis
	keyStore KeyStorer
	id       *core.ID
	_client  *sling.Sling
}

func (p *HttpProvider) LoadIdentity(id *core.ID, kOp *babyjub.PublicKey,
	proofKOp *core.ProofClaimGenesis, keyStore KeyStorer) (*HttpIdentity, error) {
	client := p.client().Path(fmt.Sprintf("id/%s/", id.String()))
	kOpComp := kOp.Compress()
	return &HttpIdentity{provider: p, kOp: &kOpComp, proofKOp: proofKOp,
		keyStore: keyStore, id: id, _client: client}, nil
}

func (i *HttpIdentity) client() *sling.Sling {
	return i._client.New()
}

func (i *HttpIdentity) ID() *core.ID {
	return i.id
}

func (i *HttpIdentity) AddClaim(claim *merkletree.Entry) error {
	return i.AddClaims([]*merkletree.Entry{claim})
}

type ClaimsReq struct {
	Claims []*merkletree.Entry `json:"claims" binding:"required"`
}

func (i *HttpIdentity) AddClaims(claims []*merkletree.Entry) error {
	// 1. send claims to IdentityAgent
	claimsReq := ClaimsReq{Claims: claims}
	err := i.provider.request(i.client().Path("claims").Post("").BodyJSON(claimsReq), nil)
	if err != nil {
		return err
	}

	// 2. getCurrentRoot from IdentityAgent
	currentRoot := identityagentsrv.CurrentRoot{}
	err = i.provider.request(i.client().Path("root").Get(""), &currentRoot)
	if err != nil {
		return err
	}

	// 3. sign UpdateRootMsg (containing Local (new) & Published (old) roots)
	msg := append(currentRoot.Published[:], currentRoot.Local[:]...) // concatenate old + new root
	sig, date, err := i.keyStore.SignBaby(i.kOp, keystore.PrefixMinorUpdate, msg)
	if err != nil {
		return err
	}

	// 4. send UpdateRootMsg signed to the IdentityAgent
	setRootReq := claimsrv.SetRoot0Req{
		OldRoot:   currentRoot.Published,
		NewRoot:   currentRoot.Local,
		ProofKOp:  i.proofKOp,
		Date:      date,
		Signature: sig,
	}
	err = i.provider.request(i.client().Path("root").Post("").BodyJSON(setRootReq), nil)
	if err != nil {
		return err
	}

	return nil
}

func (i *HttpIdentity) GenProofClaim(claim *merkletree.Entry) (*core.ProofClaim, error) {
	return nil, fmt.Errorf("TODO")
}

func (i *HttpIdentity) GenProofClaims(claims []*merkletree.Entry) ([]core.ProofClaim, error) {
	return nil, fmt.Errorf("TODO")
}

func (i *HttpIdentity) EmittedClaims() ([]*merkletree.Entry, error) {
	var emittedClaims struct {
		Claims []*merkletree.Entry `json:"emittedClaims" binding:"required"`
	}
	err := i.provider.request(i.client().Path("claims/emitted").Get(""), &emittedClaims)
	return emittedClaims.Claims, err
}

func (i *HttpIdentity) ReceivedClaims() ([]*merkletree.Entry, error) {
	var receivedClaims struct {
		Claims []*merkletree.Entry `json:"receivedClaims" binding:"required"`
	}
	err := i.provider.request(i.client().Path("claims/received").Get(""), &receivedClaims)
	return receivedClaims.Claims, err
}

func (i *HttpIdentity) VerifyProofClaim(proof *core.ProofClaim) (bool, error) {
	return false, fmt.Errorf("TODO")
}
