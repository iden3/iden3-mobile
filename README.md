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

// create new keystore
ks := NewKeyStore()

// import key to keystore
err = ks.ImportKeyBabyJub(kOpPub)

// create new identity
id, proofKOp, err = provider.CreateIdentity(keyStore, kOp, nil)

// load identity
identity, err = provider.LoadIdentity(id, kOp, proofKOp, keyStore)

// add claims
err := identity.AddClaims([]*merkletree.Entry{c0, c1})

// get emitted claims
claims, err := identity.EmittedClaims()

// get received claims
claims, err := identity.ReceivedClaims()

// [WIP]
```

## Gomobile

Using go1.12.7 linux/amd64

### First time

```
go mod vendor
gomobile init
GO111MODULE=off go get github.com/ethereum/go-ethereum
ln -s $PWD ~/go/src/
cp -r \
  "${GOPATH}/src/github.com/ethereum/go-ethereum/crypto/secp256k1/libsecp256k1" \
  "vendor/github.com/ethereum/go-ethereum/crypto/secp256k1/"
GO111MODULE=off gomobile bind -target=android go-iden3-light-wallet/identityprovider
```
