package identityserver

import (
	"bytes"
	"encoding/hex"
	"encoding/json"
	"net/http"

	"github.com/iden3/go-iden3-core/core"
	"github.com/iden3/go-iden3-core/merkletree"
	"github.com/iden3/go-iden3-crypto/babyjub"
)

type Provider struct {
	Type   string            // "local", "remote"
	Params map[string]string // if "remote" => "url", "auth", ...
}

type NewIdentityMsg struct {
	ClaimAuthKOp       string   `json:"claimAuthKOp"`
	GenesisExtraClaims []string `json:"extraGenesisClaims"`
}
type NewIdentityRes struct {
	Id string `json:"id"`
	// ProofOpKey string `json:"proofOpKey"`
}

func (p *Provider) NewIdentity(kOp *babyjub.PublicKey, genesisExtraClaims []merkletree.Claim) (*Identity, error) {
	var hexGenesisExtraClaims []string
	for _, gc := range genesisExtraClaims {
		hexGenesisExtraClaims = append(hexGenesisExtraClaims, hex.EncodeToString(gc.Entry().Bytes()))
	}
	claimAuthKOp := core.NewClaimAuthorizeKSignBabyJub(kOp)
	newIdentityMsg := NewIdentityMsg{
		ClaimAuthKOp:       hex.EncodeToString(claimAuthKOp.Entry().Bytes()),
		GenesisExtraClaims: hexGenesisExtraClaims,
	}
	newIdentityMsgJson, err := json.Marshal(newIdentityMsg)
	if err != nil {
		return nil, err
	}
	res, err := http.Post(p.Params["url"]+"/identity", "application/json", bytes.NewBuffer(newIdentityMsgJson))
	if err != nil {
		return nil, err
	}
	defer res.Body.Close()

	var newIdentityRes NewIdentityRes
	decoder := json.NewDecoder(res.Body)
	err = decoder.Decode(&newIdentityRes)
	if err != nil {
		return nil, err
	}

	id, err := core.IDFromString(newIdentityRes.Id)
	if err != nil {
		return nil, err
	}
	identity := &Identity{
		Provider: p,
		ID:       id,
	}
	return identity, nil
}

type Identity struct {
	Provider *Provider
	ID       core.ID
}
