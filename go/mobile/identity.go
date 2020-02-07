package iden3mobile

import (
	"encoding/json"
	"io/ioutil"

	"github.com/google/uuid"
)

type (
	Identity struct {
		Id             string
		ReceivedClaims []string
	}
)

// CreateIdentity creates a new identity
func (i *Identity) CreateIdentity() error {
	i.Id = uuid.New().String()
	return nil
}

//
func (i *Identity) Export(filePath string) error {
	j, err := json.Marshal(i)
	if err != nil {
		return err
	}
	if err := ioutil.WriteFile(filePath, j, 0644); err != nil {
		return err
	}
	return nil
}

//
func (i *Identity) Import(filePath string) error {
	j, err := ioutil.ReadFile(filePath)
	if err != nil {
		return err
	}
	return json.Unmarshal(j, i)
}
