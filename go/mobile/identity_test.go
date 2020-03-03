package iden3mobile

import (
	"io/ioutil"
	"os"
	"testing"

	"github.com/stretchr/testify/require"
	"gopkg.in/yaml.v2"
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

func TestMain(m *testing.M) {
	// Load config file
	dat, err := ioutil.ReadFile("./config.yml")
	if err != nil {
		panic(err)
	}
	if err := yaml.Unmarshal(dat, &c); err != nil {
		panic(err)
	}
	// Create a tmp directory to store test files
	// Run tests
	result := m.Run()
	// Remove tmp directory
	for _, dir := range rmDirs {
		os.RemoveAll(dir)
	}
	os.Exit(result)
}

func TestNewIdentity(t *testing.T) {
	// New identity without extra claims
	dir1, err := ioutil.TempDir("", "identityTest")
	rmDirs = append(rmDirs, dir1)
	require.Nil(t, err)
	id, err := NewIdentity(dir1, "pass_TestNewIdentity", c.Web3Url, c.HolderTicketPeriod, NewBytesArray())
	require.Nil(t, err)
	// Stop identity
	id.Stop()
	// Error when creating new identity on a non empty dir
	_, err = NewIdentity(dir1, "pass_TestNewIdentity", c.Web3Url, c.HolderTicketPeriod, NewBytesArray())
	require.Error(t, err)
	// Load identity
	id, err = NewIdentityLoad(dir1, "pass_TestNewIdentity", c.Web3Url, c.HolderTicketPeriod)
	require.Nil(t, err)
	// Stop identity
	id.Stop()
}
