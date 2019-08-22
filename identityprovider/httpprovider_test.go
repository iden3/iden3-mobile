// Run integration tests with:
// TEST=int go test -v -count=1 ./... -run=TestInt

package identityprovider

import (
	"os"
	"testing"

	common3 "github.com/iden3/go-iden3-core/common"
	"github.com/iden3/go-iden3-core/core"
	"github.com/iden3/go-iden3-core/keystore"
	"github.com/iden3/go-iden3-core/merkletree"
	"github.com/iden3/go-iden3-crypto/babyjub"

	"github.com/ethereum/go-ethereum/common"
	"github.com/stretchr/testify/require"
)

var integration bool

func init() {
	if os.Getenv("TEST") == "int" {
		integration = true
	}
}

var keyStore *KeyStore
var provider *HttpProvider
var kOp *babyjub.PublicKey
var kOpComp *babyjub.PublicKeyComp
var id *core.ID
var identity *HttpIdentity

func setup() {
	params := HttpProviderParams{Url: "http://127.0.0.1:25000/api/unstable"}
	provider = NewHttpProvider(params)

	skHex := []byte("febbb6a0693ed405b1ae48c014ccac333234fcf11e5744a1409b02c32b9a44e5")
	var sk babyjub.PrivateKey
	if err := common3.HexDecodeInto(sk[:], skHex); err != nil {
		panic(err)
	}

	storage := keystore.MemStorage{}
	ks, err := keystore.NewKeyStore(&storage, keystore.LightKeyStoreParams)
	if err != nil {
		panic(err)
	}
	pass := []byte("passphrase")
	kOpComp, err = ks.ImportKey(sk, pass)
	if err != nil {
		panic(err)
	}
	kOp, err = kOpComp.Decompress()
	if err != nil {
		panic(err)
	}
	ks.UnlockKey(kOpComp, pass)
	keyStore = &KeyStore{ks}
}

func teardown() {

}

func TestIntNotificationService(t *testing.T) {
	if !integration {
		t.Skip()
	}
	setup()
	defer teardown()

	require.True(t, t.Run("testCreateIdentity", testCreateIdentity))
	require.True(t, t.Run("testLoadIdentity", testLoadIdentity))
	require.True(t, t.Run("testAddClaim", testAddClaim))
	require.True(t, t.Run("testAddClaims", testAddClaims))
	require.True(t, t.Run("testEmittedClaims", testEmittedClaims))
	require.True(t, t.Run("testReceivedClaims", testReceivedClaims))
}

func testCreateIdentity(t *testing.T) {
	var err error
	id, err = provider.CreateIdentity(keyStore, kOp, nil)
	require.Nil(t, err)
	require.Equal(t, "11AzwiXcYzf6S7AJd41iutEvPC8xup7BFfzjYB2HU", id.String())
}

func testLoadIdentity(t *testing.T) {
	var err error
	identity, err = provider.LoadIdentity(id, kOp, keyStore)
	require.Nil(t, err)
}

func testAddClaim(t *testing.T) {
	// create claim to be added
	ethKey := common.HexToAddress("0xe0fbce58cfaa72812103f003adce3f284fe5fc7c")
	ethKeyType := core.EthKeyTypeUpgrade
	c0 := core.NewClaimAuthEthKey(ethKey, ethKeyType).Entry()

	err := identity.AddClaim(c0)
	require.Nil(t, err)

	// Adding repeated claim should fail
	err = identity.AddClaim(c0)
	require.NotNil(t, err)
}

func testAddClaims(t *testing.T) {
	// create claims to be added
	ethKey := common.HexToAddress("0x9e74a48149BB01BFfC8cbF06A8246539bDA135B1")
	ethKeyType := core.EthKeyTypeUpgrade
	c0 := core.NewClaimAuthEthKey(ethKey, ethKeyType).Entry()
	ethKey = common.HexToAddress("0x3d380182Cd261CdcD413e4B8D17c89c943c39b1A")
	ethKeyType = core.EthKeyTypeUpgrade
	c1 := core.NewClaimAuthEthKey(ethKey, ethKeyType).Entry()

	err := identity.AddClaims([]*merkletree.Entry{c0, c1})
	require.Nil(t, err)
}

func testEmittedClaims(t *testing.T) {

	ethKey := common.HexToAddress("0xc17c2155f197F5Ea395e17E21a5d0b91D81E989E")
	ethKeyType := core.EthKeyTypeUpgrade
	c := core.NewClaimAuthEthKey(ethKey, ethKeyType).Entry()
	err := identity.AddClaim(c)
	require.Nil(t, err)

	claims, err := identity.EmittedClaims()
	require.Nil(t, err)
	require.Equal(t, 5, len(claims))

	// Find the claim we just added
	found := false
	for _, claim := range claims {
		if c.Equal(claim) {
			found = true
		}
	}
	require.True(t, found)
}

func testReceivedClaims(t *testing.T) {
	claims, err := identity.ReceivedClaims()
	require.Nil(t, err)
	require.Equal(t, 0, len(claims))
}
