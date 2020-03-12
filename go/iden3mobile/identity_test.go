package iden3mobile

import (
	"io/ioutil"
	"os"
	"testing"
	"time"

	idenpubonchainlocal "github.com/iden3/go-iden3-core/components/idenpubonchain/local"
	"github.com/stretchr/testify/require"
)

type config struct {
	Web3Url             string `yaml:"web3Url"`
	IssuerUrl           string `yaml:"issuerUrl"`
	VerifierUrl         string `yaml:"verifierUrl"`
	VerifierAttempts    int    `yaml:"verifierAttempts"`
	VerifierRetryPeriod int    `yaml:"verifierRetryPeriod"`
	HolderTicketPeriod  int    `yaml:"holderTicketPeriod"`
}

var c config
var rmDirs []string
var idenPubOnChain *idenpubonchainlocal.IdenPubOnChain
var timeNow int64
var blockNow uint64

func TestMain(m *testing.M) {
	// Load config file
	// dat, err := ioutil.ReadFile("./config.yml")
	// if err != nil {
	// 	panic(err)
	// }
	// if err := yaml.Unmarshal(dat, &c); err != nil {
	// 	panic(err)
	// }
	c = config{
		Web3Url:             "https://foo.bar",
		IssuerUrl:           "http://127.0.0.1:1234/",
		VerifierUrl:         "http://127.0.0.1:1234/",
		VerifierAttempts:    5,
		VerifierRetryPeriod: 6,
		HolderTicketPeriod:  1000,
	}
	idenPubOnChain = idenpubonchainlocal.New(
		func() time.Time { return time.Unix(timeNow, 0) },
		func() uint64 { return blockNow },
	)
	// Create a tmp directory to store test files
	// Run tests
	result := m.Run()
	// Remove tmp directory
	for _, dir := range rmDirs {
		os.RemoveAll(dir)
	}
	os.Exit(result)
}

// NewIdentityTest is like NewIdentity but uses a local implementation of the smart contract in idenPubOnChain
func NewIdentityTest(storePath, pass, web3Url string, checkTicketsPeriodMilis int, extraGenesisClaims *BytesArray) (*Identity, error) {
	return newIdentity(storePath, pass, idenPubOnChain, checkTicketsPeriodMilis, extraGenesisClaims)
}

// NewIdentityTestLoad is like NewIdentityLoad but uses a local implementation of the smart contract in idenPubOnChain
func NewIdentityTestLoad(storePath, pass, web3Url string, checkTicketsPeriodMilis int) (*Identity, error) {
	return newIdentityLoad(storePath, pass, idenPubOnChain, checkTicketsPeriodMilis)
}

func TestNewIdentity(t *testing.T) {
	// New identity without extra claims
	dir1, err := ioutil.TempDir("", "identityTest")
	rmDirs = append(rmDirs, dir1)
	require.Nil(t, err)
	id, err := NewIdentityTest(dir1, "pass_TestNewIdentity", c.Web3Url, c.HolderTicketPeriod, NewBytesArray())
	require.Nil(t, err)
	// Stop identity
	id.Stop()
	// Error when creating new identity on a non empty dir
	_, err = NewIdentityTest(dir1, "pass_TestNewIdentity", c.Web3Url, c.HolderTicketPeriod, NewBytesArray())
	require.Error(t, err)
	// Load identity
	id, err = NewIdentityTestLoad(dir1, "pass_TestNewIdentity", c.Web3Url, c.HolderTicketPeriod)
	require.Nil(t, err)
	// Stop identity
	id.Stop()
}
