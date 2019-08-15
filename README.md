# go-iden3-light-wallet
**iden3 light wallet client** library implementation in Go for desktop & smartphone wallets (using GoMobile).


## Test
- this library connects with the [IdenityServer](https://github.com/iden3/go-iden3-servers), so will need a running IdentityServer

```
go test ./...
```

## Usage

```go
// define a provider
providerParams := make(map[string]string)
providerParams["url"] = "http://127.0.0.1:25000/api/unstable"
provider := Provider{
        Type:   "remote",
        Params: providerParams,
}

// new BabyJubJub public key
kOpStr := "0x117f0a278b32db7380b078cdb451b509a2ed591664d1bac464e8c35a90646796"
var kOpComp babyjub.PublicKeyComp
err := kOpComp.UnmarshalText([]byte(kOpStr))
assert.Nil(t, err)
kOpPub, err := kOpComp.Decompress()

// create new identity
identity, err := provider.NewIdentity(kOpPub, nil)
assert.Nil(t, err)
assert.Equal(t, "119h9u2nXbtg5TmPsMm8W5bDkmVZhdS6TgKMvNWPU3", identity.ID.String())

// [WIP]
```
