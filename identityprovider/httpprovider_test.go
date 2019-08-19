package identityprovider

import (
	"testing"

	"github.com/iden3/go-iden3-core/merkletree"
	"github.com/iden3/go-iden3-crypto/babyjub"

	"github.com/stretchr/testify/assert"
)

func TestCreateIdentity(t *testing.T) {
	params := HttpProviderParams{Url: "http://127.0.0.1:25000/api/unstable"}
	provider := NewHttpProvider(params)

	kOpStr := "0x117f0a278b32db7380b078cdb451b509a2ed591664d1bac464e8c35a90646796"
	var kOp babyjub.PublicKey
	err := kOp.UnmarshalText([]byte(kOpStr))
	assert.Nil(t, err)

	keyStore := 0
	id, err := provider.CreateIdentity(keyStore, &kOp, []*merkletree.Entry{})
	assert.Nil(t, err)
	assert.Equal(t, "119h9u2nXbtg5TmPsMm8W5bDkmVZhdS6TgKMvNWPU3", id.String())
}
