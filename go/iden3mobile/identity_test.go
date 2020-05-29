package iden3mobile

import (
	"io/ioutil"
	"os"
	"sync"
	"testing"
	"time"

	idenpubonchainlocal "github.com/iden3/go-iden3-core/components/idenpubonchain/local"
	zkutils "github.com/iden3/go-iden3-core/utils/zk"
	log "github.com/sirupsen/logrus"
	"github.com/stretchr/testify/require"
)

type config struct {
	IssuerUrl           string `yaml:"issuerUrl"`
	VerifierUrl         string `yaml:"verifierUrl"`
	VerifierAttempts    int    `yaml:"verifierAttempts"`
	VerifierRetryPeriod int    `yaml:"verifierRetryPeriod"`
	HolderTicketPeriod  int    `yaml:"holderTicketPeriod"`
}

var c config
var rmDirs []string
var idenPubOnChain *idenpubonchainlocal.IdenPubOnChain

type TimeBlock struct {
	timeNow  int64
	blockNow uint64
	rw       sync.RWMutex
}

func (tb *TimeBlock) SetTime(t int64) {
	tb.rw.Lock()
	defer tb.rw.Unlock()
	tb.timeNow = t
}

func (tb *TimeBlock) SetBlock(n uint64) {
	tb.rw.Lock()
	defer tb.rw.Unlock()
	tb.blockNow = n
}

func (tb *TimeBlock) AddTime(t int64) {
	tb.rw.Lock()
	defer tb.rw.Unlock()
	tb.timeNow += t
}

func (tb *TimeBlock) AddBlock(n uint64) {
	tb.rw.Lock()
	defer tb.rw.Unlock()
	tb.blockNow += n
}

func (tb *TimeBlock) Time() time.Time {
	tb.rw.RLock()
	defer tb.rw.RUnlock()
	return time.Unix(tb.timeNow, 0)
}

func (tb *TimeBlock) Block() uint64 {
	tb.rw.RLock()
	defer tb.rw.RUnlock()
	return tb.blockNow
}

var timeBlock TimeBlock
var zkFilesIdenState *zkutils.ZkFiles
var zkFilesCredential *zkutils.ZkFiles

func TestMain(m *testing.M) {
	log.SetLevel(log.DebugLevel)

	zkFilesIdenState = zkutils.NewZkFiles(
		"http://161.35.72.58:9000/circuit-idstate/", "/tmp/iden3-test/idenstatezk-issuer",
		zkutils.ProvingKeyFormatGoBin,
		zkutils.ZkFilesHashes{
			ProvingKey:      "37b6b3addd52faf9357f1496312e6a86af4f5c41c557cda9931468809d32c03c",
			VerificationKey: "473952ff80aef85403005eb12d1e78a3f66b1cc11e7bd55d6bfe94e0b5577640",
			WitnessCalcWASM: "8eafd9314c4d2664a23bf98a4f42cd0c29984960ae3544747ba5fbd60905c41f",
		}, true)
	if err := zkFilesIdenState.DownloadAll(); err != nil {
		panic(err)
	}

	vk, err := zkFilesIdenState.VerificationKey()
	if err != nil {
		panic(err)
	}

	zkFilesCredential = zkutils.NewZkFiles(
		"http://161.35.72.58:9000/credentialDemoWrapper", "/tmp/iden3-test/credentialzk",
		zkutils.ProvingKeyFormatGoBin,
		zkutils.ZkFilesHashes{
			ProvingKey:      "bdefc89d07d1dfab75c43f09aedb9da876496c5c3967383337482e4c5ae4f7d3",
			VerificationKey: "12a730890e85e33d8bf0f2e54db41dcff875c2dc49011d7e2a283185f47ac0de",
			WitnessCalcWASM: "6b3c28c4842e04129674eb71dc84d76dd8b290c84987929d54d890b7b8bed211",
		}, true)
	if err := zkFilesCredential.DownloadAll(); err != nil {
		panic(err)
	}

	c = config{
		IssuerUrl:           "http://127.0.0.1:1234/",
		VerifierUrl:         "http://127.0.0.1:1234/",
		VerifierAttempts:    5,
		VerifierRetryPeriod: 6,
		HolderTicketPeriod:  1000,
	}
	idenPubOnChain = idenpubonchainlocal.New(
		timeBlock.Time,
		timeBlock.Block,
		vk,
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

type testEventHandler struct{}

func (teh *testEventHandler) Send(ev *Event) {
	log.Info("Event received: ", ev.TicketId)
}

// NewIdentityTest is like NewIdentity but uses a local implementation of the smart contract in idenPubOnChain
func NewIdentityTest(storePath, sharedStorePath, pass string,
	idenPubOnChain *idenpubonchainlocal.IdenPubOnChain, checkTicketsPeriodMilis int,
	extraGenesisClaims *BytesArray, s Sender) (*Identity, error) {
	if s == nil {
		s = &testEventHandler{}
	}
	return newIdentity(storePath, sharedStorePath, pass, idenPubOnChain, checkTicketsPeriodMilis,
		extraGenesisClaims, s)
}

// NewIdentityTestLoad is like NewIdentityLoad but uses a local implementation of the smart contract in idenPubOnChain
func NewIdentityTestLoad(storePath, sharedStorePath, pass string,
	idenPubOnChain *idenpubonchainlocal.IdenPubOnChain, checkTicketsPeriodMilis int,
	s Sender) (*Identity, error) {
	if s == nil {
		s = &testEventHandler{}
	}
	return newIdentityLoad(storePath, sharedStorePath, pass, idenPubOnChain, checkTicketsPeriodMilis, s)
}

func TestNewIdentity(t *testing.T) {
	// New identity without extra claims
	sharedDir, err := ioutil.TempDir("", "shared")
	require.Nil(t, err)
	rmDirs = append(rmDirs, sharedDir)
	dir1, err := ioutil.TempDir("", "identityTest")
	rmDirs = append(rmDirs, dir1)
	require.Nil(t, err)
	id, err := NewIdentityTest(dir1, sharedDir, "pass_TestNewIdentity", idenPubOnChain,
		c.HolderTicketPeriod, NewBytesArray(), nil)
	require.Nil(t, err)
	// Stop identity
	id.Stop()
	// Error when creating new identity on a non empty dir
	_, err = NewIdentityTest(dir1, sharedDir, "pass_TestNewIdentity", idenPubOnChain,
		c.HolderTicketPeriod, NewBytesArray(), nil)
	require.Error(t, err)
	// Load identity
	id, err = NewIdentityTestLoad(dir1, sharedDir, "pass_TestNewIdentity", idenPubOnChain,
		c.HolderTicketPeriod, nil)
	require.Nil(t, err)
	// Stop identity
	id.Stop()
}
