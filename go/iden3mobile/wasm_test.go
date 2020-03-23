package iden3mobile

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestWasm(t *testing.T) {
	i := Identity{}
	require.Nil(t, i.CallWasm("simple.wasm"))
}
