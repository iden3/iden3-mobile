package iden3mobile

import (
	"errors"

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
	// TODO: implement
	return errors.New("NOT IMPLEMENTED")
}

//
func (i *Identity) Import(filePath string) error {
	// TODO: implement
	return errors.New("NOT IMPLEMENTED")
}
