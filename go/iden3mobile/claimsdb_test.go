package iden3mobile

import (
	"encoding/json"
	"fmt"
	"testing"

	"github.com/iden3/go-iden3-core/common"
	"github.com/iden3/go-iden3-core/core/proof"
	"github.com/iden3/go-iden3-core/db"
	"github.com/stretchr/testify/require"
)

const cred1JSON = `
{
	"Id": "118NZoexLLTgiApGVod8cGXRTeae1a9RqvYaJM5cq4",
	"IdenStateData": {
		"BlockTs": 1582637420,
		"BlockN": 2240421,
		"IdenState": "0x6e7c6798c63a9f168ac705ce6cafd1f0076798cb258bac1d3509c0b8c3c7ad01"
	},
	"MtpClaim": "0x00050000000000000000000000000000000000000000000000000000000000179242e667845d558d6d9061b7d406f9b2fecb9dd98ad0f205b4576b8b2336a92cd9b65acbed05b78bb199ec5de3225281e38ccbd8add32c65f715603862c68c0476c47edbbc50e3c510182b7b4ed49f5968bf644978cc4d8a2b0218f7ff1f461b5c4fa26c1eac035b2eea47a401f953e62d2bbad53800a1e390937b083b168124",
	"Claim": "0x0000000000000000000000003131347674793255664d6b74714e50645a687300567151673948594b67546e7a507058644537796b655146000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000700000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000",
	"RevocationsTreeRoot": "0x0000000000000000000000000000000000000000000000000000000000000000",
	"RootsTreeRoot": "0x8641ba923381e34f94be83b444a5cb7bff0a470d92a2d7cf99f5ca0f98405b21",
	"IdenPubUrl": "http://foo.bar"
}
`

const cred2JSON = `
{
	"Id": "1N7d2qVEJeqnYAWVi5Cq6PLj6GwxaW6FYcfmY2Xh6",
	"IdenStateData": {
		"BlockTs": 1582637420,
		"BlockN": 2240421,
		"IdenState": "0x6e7c6798c63a9f168ac705ce6cafd1f0076798cb258bac1d3509c0b8c3c7ad01"
	},
	"MtpClaim": "0x00050000000000000000000000000000000000000000000000000000000000179242e667845d558d6d9061b7d406f9b2fecb9dd98ad0f205b4576b8b2336a92cd9b65acbed05b78bb199ec5de3225281e38ccbd8add32c65f715603862c68c0476c47edbbc50e3c510182b7b4ed49f5968bf644978cc4d8a2b0218f7ff1f461b5c4fa26c1eac035b2eea47a401f953e62d2bbad53800a1e390937b083b168124",
	"Claim": "0x0000000000000000000000003131347674793255664d6b74714e50645a687300567151673948594b87546eaa5a7058644537796b655146000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000700000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000",
	"RevocationsTreeRoot": "0x0000000000000000000000000000000000000000000000000000000000000000",
	"RootsTreeRoot": "0x8641ba923381e34f94be83b444a5cb7bff0a470d92a2d7cf99f5ca0f98405b21",
	"IdenPubUrl": "http://foo.bar"
}
`

func TestClaimDB(t *testing.T) {
	var cred1 proof.CredentialExistence
	err := json.Unmarshal([]byte(cred1JSON), &cred1)
	require.Nil(t, err)
	var cred2 proof.CredentialExistence
	err = json.Unmarshal([]byte(cred2JSON), &cred2)
	require.Nil(t, err)

	storage := db.NewMemoryStorage()
	cdb := NewClaimDB(storage)

	id1, err := cdb.AddCredentialExistance(&cred1)
	require.Nil(t, err)
	id2, err := cdb.AddCredentialExistance(&cred2)
	require.Nil(t, err)

	cred1Cpy, err := cdb.GetCredExist(id1)
	require.Nil(t, err)
	require.Equal(t, cred1.Id, cred1Cpy.Id)
	require.Equal(t, cred1.Claim.Data, cred1Cpy.Claim.Data)

	cred2Cpy, err := cdb.GetCredExist(id2)
	require.Nil(t, err)
	require.Equal(t, cred2.Id, cred2Cpy.Id)
	require.Equal(t, cred2.Claim.Data, cred2Cpy.Claim.Data)

	credNoExist, err := cdb.GetCredExist("")
	require.Error(t, err)
	require.Nil(t, credNoExist)

	creds := make(map[string]*proof.CredentialExistence)
	err = cdb.Iterate_(func(id string, cred *proof.CredentialExistence) (bool, error) {
		creds[id] = cred
		return true, nil
	})
	require.Nil(t, err)
	require.Equal(t, &cred1, creds[id1])
	require.Equal(t, &cred2, creds[id2])

	credsJSON := make(map[string]string)
	err = cdb.IterateCredExistJSON_(func(id string, cred string) (bool, error) {
		credsJSON[id] = cred
		return true, nil
	})
	require.Nil(t, err)
	require.Equal(t, 2, len(credsJSON))
	for k, v := range credsJSON {
		fmt.Printf("credJSON %v: %v\n", common.Hex(k[:]), v)
	}

	claimsJSON := make(map[string]string)
	err = cdb.IterateClaimsJSON_(func(id string, claim string) (bool, error) {
		claimsJSON[id] = claim
		return true, nil
	})
	require.Nil(t, err)
	require.Equal(t, 2, len(claimsJSON))
	for k, v := range claimsJSON {
		fmt.Printf("claimsJSON %v: %v\n", common.Hex(k[:]), v)
	}
}
