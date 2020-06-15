//
//  Iden3IdentityFactory.swift
//  iden3CoreSDK
//
//  Created by Iden3 on 12/06/2020.
//  Copyright © 2020 Iden3. All rights reserved.
//

import Foundation
import Iden3mobile

/// Main class to initialize and setup Iden3CoreSDK framework.

public final class Iden3IdentityFactory {
    
    // MARK: - Singleton
    
    public static let sharedInstance = Iden3IdentityFactory()
    
    // MARK: - Properties
    
    private var web3Url: String = ""
    private var storePath: String = ""
    private var checkTicketsPeriod: Int = 10000
    
    // MARK: - Initializers
    
    private init() {
    }
    
    // MARK: - Initialization
    
    /// Initializes de Iden3CoreSDK framework.
    ///
    /// - parameters:
    ///   - web3Url: String the Web3 url.
    ///   - storePath: String the absolute path where to store the identities.
    ///   - checkTicketsPeriod:Time in miliseconds of the period needed for checking the tickets.
    ///
    public func initialize(with web3Url: String, storePath: String, checkTicketsPeriod: Int) -> Bool {
        // Initialize member variables
        self.web3Url = web3Url
        self.storePath = storePath
        self.checkTicketsPeriod = checkTicketsPeriod
        return true
    }
    
    /// Checks if the Iden3IdentityFactory library has been initialized successfully
    ///
    /// - return:
    ///     - Bool if the initialization has been successful or not
    ///
    public func isInitialized() -> Bool {
        if (self.web3Url.count > 0 && self.storePath.count > 0 && self.checkTicketsPeriod > 0) {
            return true
        } else {
            return false
        }
    }
    
    /// Creates a new Iden3 identity.
    ///
    /// - parameters:
    ///   - alias: String the alias of the identity. Should be alphanumeric without spaces
    ///   - password: String the password to access the identity.
    ///   - eventDelegate: Delegate of events associated to the identity.
    ///
    ///   return The new Iden3 identity created.
    ///
    public func createIdentity(alias: String, password: String, eventDelegate: Iden3EventDelegate?) throws -> Iden3Identity?  {
        if (isInitialized()) {
            if (alias.count == 0 || !isAlphaNumeric(alias) || password.count == 0) {
                throw Iden3Error.IllegalArgumentError("Iden3 method called with not valid arguments")
            } else {
                let identityPath : String = self.storePath + "/identities/" + alias
                let identityPathURL = URL(string: identityPath)!
                let sharedPath : String = self.storePath + "/shared/"
                let sharedPathURL = URL(string: sharedPath)!
                
                if FileManager.default.fileExists(atPath: identityPathURL.absoluteString) {
                   // removing previously created identity with same alias
                   do {
                      let filePaths = try FileManager.default.contentsOfDirectory(atPath: identityPathURL.absoluteString)
                      for filePath in filePaths {
                          try FileManager.default.removeItem(atPath: identityPathURL.absoluteString + filePath)
                      }
                      try FileManager.default.removeItem(at: identityPathURL)
                   } catch {
                      print("Could not clear identity folder: \(error)")
                      throw Iden3Error.GenericError("There was an error while creating the identity")
                   }
                }
                
                // Creating new shared folder
                if !FileManager.default.fileExists(atPath: sharedPathURL.absoluteString) {
                    do {
                       try FileManager.default.createDirectory(atPath: sharedPathURL.absoluteString, withIntermediateDirectories: true, attributes: nil)
                    } catch {
                        print(error.localizedDescription);
                        throw Iden3Error.GenericError("There was an error while creating the identity")
                    }
                }
                
                // Creating new identity folder
                do {
                   try FileManager.default.createDirectory(atPath: identityPathURL.absoluteString, withIntermediateDirectories: true, attributes: nil)
                } catch {
                    print(error.localizedDescription);
                    throw Iden3Error.GenericError("There was an error while creating the identity")
                }
                
                // Creating new identity
                let identity = Iden3mobileIdentity(identityPathURL.absoluteString, sharedStorePath: sharedPath, pass: password, web3Url: self.web3Url, checkTicketsPeriodMilis: self.checkTicketsPeriod, extraGenesisClaims: nil , eventHandler: nil)
                if (identity != nil) {
                    return Iden3Identity(identity: identity!)
                } else {
                    throw Iden3Error.GenericError("There was an error while creating the identity")
                }
            }
        } else {
            throw Iden3Error.IllegalStateError("Iden3 API is not initialized. Please, call initializeAPI method before doing this call")
        }
    }
    
    
    /// Loads an Iden3 identity.
    ///
    /// - parameters:
    ///   - alias: String the alias of the identity. Should be alphanumeric without spaces
    ///   - password: String the password to access the identity.
    ///   - eventDelegate: Delegate of events associated to the identity.
    ///
    ///   return The new Iden3 identity created.
    ///
    public func loadIdentity(alias: String, password: String, eventDelegate: Iden3EventDelegate?) throws -> Iden3Identity?  {
        if (isInitialized()) {
            if (alias.count == 0 || !isAlphaNumeric(alias) || password.count == 0) {
                throw Iden3Error.IllegalArgumentError("Iden3 method called with not valid arguments")
            } else {
                let identityPath : String = self.storePath + "/identities/" + alias
                let identityPathURL = URL(string: identityPath)!
                let sharedPath : String = self.storePath + "/shared/"
                let sharedPathURL = URL(string: sharedPath)!
                
                // Creating new shared folder
                if !FileManager.default.fileExists(atPath: sharedPathURL.absoluteString) {
                    do {
                       try FileManager.default.createDirectory(atPath: sharedPathURL.absoluteString, withIntermediateDirectories: true, attributes: nil)
                    } catch {
                        print(error.localizedDescription);
                        throw Iden3Error.GenericError("There was an error while loading the identity")
                    }
                }
                
                if FileManager.default.fileExists(atPath: identityPathURL.absoluteString) {
                    // Loading identity
                    let identity = Iden3mobileIdentity.init(load: identityPathURL.absoluteString, sharedStorePath: sharedPath, pass: password, web3Url: self.web3Url, checkTicketsPeriodMilis: self.checkTicketsPeriod, eventHandler: nil)
                   if (identity != nil) {
                       return Iden3Identity(identity: identity!)
                   } else {
                       throw Iden3Error.GenericError("There was an error while loading the identity")
                   }
                } else {
                    throw Iden3Error.FileNotFoundError("Identity not found. Please be sure the identity is created calling createIdentity method before loading it")
                }
            }
        } else {
            throw Iden3Error.IllegalStateError("Iden3 API is not initialized. Please, call initializeAPI method before doing this call")
        }
    }
    
    private func isAlphaNumeric(_ text: String) -> Bool {
        let letters = CharacterSet.letters
        let digits = CharacterSet.decimalDigits

        var letterCount = 0
        var digitCount = 0

        for uni in text.unicodeScalars {
            if letters.contains(uni) {
                letterCount += 1
            } else if digits.contains(uni) {
                digitCount += 1
            }
        }
        if (text.count == letterCount + digitCount) {
            return true
        } else {
            return false
        }
    }
}
