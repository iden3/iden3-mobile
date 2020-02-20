package iden3mobile

import (
	"fmt"
	"io/ioutil"
	"os"
	"testing"

	"github.com/stretchr/testify/require"
	"gopkg.in/yaml.v2"
)

type config struct {
	Web3Url string `yaml:"web3Url"`
}

const dir = "./tmp"

var c config

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
	os.RemoveAll(dir)
	if err := os.Mkdir(dir, 0777); err != nil {
		panic(err)
	}
	// Run tests
	result := m.Run()
	// Remove tmp directory
	if err := os.RemoveAll(dir); err != nil {
		panic(err)
	}
	os.Exit(result)
}

func TestNewIdentity(t *testing.T) {
	// New identity without extra claims
	if err := os.Mkdir(dir+"/TestNewIdentity", 0777); err != nil {
		panic(err)
	}
	fmt.Println(c.Web3Url)
	id, err := NewIdentity(dir+"/TestNewIdentity", "pass_TestNewIdentity", c.Web3Url, 100, NewBytesArray(), nil)
	require.Nil(t, err)
	// Stop identity
	id.Stop()
	// Load identity
	id, err = NewIdentityLoad(dir+"/TestNewIdentity", "pass_TestNewIdentity", c.Web3Url, 100, nil)
	require.Nil(t, err)
	// Stop identity
	id.Stop()
}
