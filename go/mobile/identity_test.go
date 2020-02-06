package iden3mobile

import (
	"testing"

	"github.com/stretchr/testify/require"
)

var i Identity

func TestCreateIdentity(t *testing.T) {
	// TODO: once mockup implementation is replaced by a true one, this test will need some love
	i = Identity{}
	err := i.CreateIdentity()
	require.Nil(t, err)
}

func TestImport(t *testing.T) {
	// TODO: once implementation is rdone, this test will need some love
	err := i.Import("not implemented")
	require.Nil(t, err)
}

func TestExport(t *testing.T) {
	// TODO: once implementation is rdone, this test will need some love
	err := i.Export("not implemented")
	require.Nil(t, err)
}
