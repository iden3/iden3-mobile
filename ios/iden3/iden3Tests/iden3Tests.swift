//
//  iden3Tests.swift
//  iden3Tests
//
//  Created by Iden3 on 02/06/2020.
//

import XCTest
@testable import iden3

class iden3Tests: XCTestCase {
    
    var iden3IdentityFactory : Iden3IdentityFactory!
    var iden3Identity : Iden3Identity?
    var web3Url : String?
    let issuerUrl = "http://167.172.104.160:6100/api/unstable"
    let verifierUrl = "http://167.172.104.160:6200/api/unstable"
    var storePath : String?
    
    override func setUp() {
        // Put setup code here. This method is called before the invocation of each test method in the class.
        super.setUp()
        loadPreferences()
        storePath = FileManager.default.urls(for: .documentDirectory, in: .userDomainMask).first!.absoluteString
        iden3IdentityFactory = Iden3IdentityFactory.sharedInstance
        if (!isInitialized()) {
            XCTFail("Iden3 framework is not initialized")
        }
    }
    
    struct Preferences: Codable {
        var web3Url:String
    }
    
    private func loadPreferences() {
        if  let path        = Bundle(for: type(of: self)).path(forResource: "Preferences", ofType:"plist"),
            let xml         = FileManager.default.contents(atPath: path),
            let preferences = try? PropertyListDecoder().decode(Preferences.self, from: xml)
        {
            self.web3Url = preferences.web3Url
        }
    }

    override func tearDown() {
        // Put teardown code here. This method is called after the invocation of each test method in the class.
        if (iden3Identity != nil) {
            iden3Identity!.stopIdentity()
        }
        iden3Identity = nil
        iden3IdentityFactory = nil
        super.tearDown()
    }
    
    func testInitializeSuccess() {
        XCTAssert(isInitialized())
    }
    
    func testCreateIdentitySuccess() {
        do {
            iden3Identity = try iden3IdentityFactory.createIdentity(alias: "alias", password:"password", eventDelegate: nil)
            XCTAssertNotNil(iden3Identity)
        } catch {
            XCTFail()
        }
    }
    
    func testRequestClaimSuccess() {
        do {
            iden3Identity = try iden3IdentityFactory.createIdentity(alias: "alias", password:"password", eventDelegate: nil)
            if (iden3Identity != nil) {
                iden3Identity?.requestClaim(issuerUrl: self.issuerUrl, data: "\(currentTimeInMiliseconds())", ticketDelegate: nil)
            } else {
                XCTFail()
            }
        } catch let error {
            XCTFail(error.localizedDescription)
        }
    }
    
    func testProveClaimSuccess() {
        do {
            iden3Identity = try iden3IdentityFactory.createIdentity(alias: "alias", password:"password", eventDelegate: nil)
            if (iden3Identity != nil) {
                iden3Identity?.proveClaim(verifierUrl: self.verifierUrl, credentialId: "\(currentTimeInMiliseconds())", withZKProof: false, proveClaimDelegate: nil)
            } else {
                XCTFail()
            }
        } catch let error {
            XCTFail(error.localizedDescription)
        }
    }
    
    func testZKProveClaimSuccess() {
        do {
            iden3Identity = try iden3IdentityFactory.createIdentity(alias: "alias", password:"password", eventDelegate: nil)
            if (iden3Identity != nil) {
                iden3Identity?.proveClaim(verifierUrl: self.verifierUrl, credentialId:"\(currentTimeInMiliseconds())", withZKProof: true, proveClaimDelegate: nil)
            } else {
                XCTFail()
            }
        } catch let error {
            XCTFail(error.localizedDescription)
        }
    }
    
    private func isInitialized() -> Bool {
        return iden3IdentityFactory.initialize(with: web3Url!, storePath: storePath!, checkTicketsPeriod: 10000)
    }
    
    private func currentTimeInMiliseconds() -> Int {
        let currentDate = Date()
        let since1970 = currentDate.timeIntervalSince1970
        return Int(since1970 * 1000)
    }
    
    
}
