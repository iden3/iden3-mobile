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

## Gomobile

Using go1.12.7 linux/amd64

```
go mod vendor
gomobile init
ln -s ~/git/iden3/go-iden3-light-wallet ~/go/src/github.com/iden3/
GO111MODULE=off go get github.com/ethereum/go-ethereum
ln -s $PWD ~/go/src/github.com/iden3/
cp -r \
  "${GOPATH}/src/github.com/ethereum/go-ethereum/crypto/secp256k1/libsecp256k1" \
  "vendor/github.com/ethereum/go-ethereum/crypto/secp256k1/"
GO111MODULE=off gomobile bind -target=android github.com/iden3/go-iden3-light-wallet/identityprovider
```
