package identityserver

import (
	"testing"

	"github.com/iden3/go-iden3-crypto/babyjub"

	"github.com/stretchr/testify/assert"
)

func TestNewIdentity(t *testing.T) {
	providerParams := make(map[string]string)
	providerParams["url"] = "http://127.0.0.1:25000/api/unstable"
	provider := Provider{
		Type:   "remote",
		Params: providerParams,
	}

	kOpStr := "0x117f0a278b32db7380b078cdb451b509a2ed591664d1bac464e8c35a90646796"
	var kOpComp babyjub.PublicKeyComp
	err := kOpComp.UnmarshalText([]byte(kOpStr))
	assert.Nil(t, err)
	kOpPub, err := kOpComp.Decompress()

	identity, err := provider.NewIdentity(kOpPub, nil)
	assert.Nil(t, err)
	assert.Equal(t, "119h9u2nXbtg5TmPsMm8W5bDkmVZhdS6TgKMvNWPU3", identity.ID.String())
}
