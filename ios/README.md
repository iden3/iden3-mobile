# Iden3 - iOS

Iden3 is an iOS library for using iden3 identity system.

## Installation


Use the package manager [Cocoapods](https://guides.cocoapods.org/using/getting-started.html) to install Iden3.

Add to your podfile the following line
```bash
pod 'iden3'
```
and run in the folder where is the podfile the following command in terminal
```bash
pod install
```

## Usage

```swift
import iden3

let iden3IdentityFactory = Iden3IdentityFactory.sharedInstance

// Initialize identity factory
iden3IdentityFactory.initialize(with: web3Url, storePath: storePath, checkTicketsPeriod: 10000)

// Create identity
do {
	let iden3Identity = try iden3IdentityFactory.createIdentity(alias: "alias", password:"password", eventDelegate: nil)
} catch let error {
    error.localizedDescription
}

// Request claim
iden3Identity?.requestClaim(issuerUrl: issuerUrl, data: data, ticketDelegate: nil)

// Prove claim
iden3Identity?.proveClaim(verifierUrl: verifierUrl, credentialId: credentialId, withZKProof: true, proveClaimDelegate: nil)
```

## Contributing
Pull requests are welcome. For major changes, please open an issue first to discuss what you would like to change.

Please make sure to update tests as appropriate.

## License
[GPL-3.0](https://choosealicense.com/licenses/gpl-3.0/)
