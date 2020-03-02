# iden3-mobile
**iden3 mobile** is a set of libraries that implement iden3 protocol in mobile platforms. This repository contain a set of components o do so:
* go: go code that implements all the logic, and have gomobile friendly wrappers
* [To do] android: android library that wrapper the binded go code
* [To do] iOS: iOS library that wrapper the binded go code
* [To do] flutter: flutter plugin that calls the native iOS and android libraries


## Test
### Go
1. Rename go/mobile/config-example.yml to go/mobile/config.yml, and change the value of `web3Url` for a valid web3 provider.
2. `cd` into `go/mockupServer/main` and run `go run . -ip 127.0.0.1 -verifytime 1 -aprovetime 1`
3. In a new terminal, `cd` into `go/mobile` and run `go test ./...`
