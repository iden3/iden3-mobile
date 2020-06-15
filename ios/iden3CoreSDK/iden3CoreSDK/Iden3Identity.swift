//
//  Iden3Identity.swift
//  iden3CoreSDK
//
//  Created by Iden3 on 12/06/2020.
//  Copyright © 2020 Iden3. All rights reserved.
//

import Foundation
import Iden3mobile

public final class Iden3Identity {
    
    // MARK: - Properties
    
    private var identity: Iden3mobileIdentity
    
    // MARK: - Initializers
       
    init(identity: Iden3mobileIdentity) {
        self.identity = identity
    }
    
    /// Stops the Iden3 identity.
    ///
    public func stopIdentity() {
        identity.stop()
    }
    
    /// Request a claim from an issuer.
    ///
    /// - parameters:
    ///   - issuerUrl: Url of the issuer
    ///   - data: data needed by the issuer for requesting a claim.
    ///   - ticketDelegate: Delegate of tickets received after requesting the claim.
    ///
    ///   return Iden3Ticket.
    ///
    public func requestClaim(issuerUrl: String, data: String, ticketDelegate: Iden3TicketDelegate?) throws -> Iden3Ticket? {
        if (ticketDelegate == nil) {
            do {
                return Iden3Ticket(try identity.requestClaim(issuerUrl, data: data))
            } catch {
                return nil
            }
        } else {
            identity.requestClaim(withCb: issuerUrl, data: data, c: nil)
        }
        return nil
    }
    
    /// Sends a credentialValidity build from the given credentialExistance to a verifier.
    ///
    /// - parameters:
    ///   - verifierUrl: url of the verifier
    ///   - credentialId: credential id needed by the verifier for proving a claim.
    ///   - withZKProof: Boolean that indicates if zero-knowledge proof shoud be used or not.
    ///   - proveClaimDelegate: Delegate of proofs received after proving a claim.
    ///
    ///   return Bool should be true if the verifier accepted the prove as valid
    ///
    public func proveClaim(verifierUrl: String, credentialId: String, withZKProof: Bool, proveClaimDelegate: Iden3ProveClaimDelegate?) throws -> Bool {
        if (withZKProof) {
            if (proveClaimDelegate == nil) {
                do {
                    var result: ObjCBool = false
                    try identity.proveClaim(verifierUrl, credID: credentialId, ret0_:&result)
                    if (result.boolValue == true) {
                        return true
                    } else {
                        return false
                    }
                } catch {
                    return false
                }
            } else {
                identity.proveClaim(withCb: verifierUrl, credID: credentialId, c: nil)
            }
        } else {
            if (proveClaimDelegate == nil) {
                do {
                    var result: ObjCBool = false
                    try identity.proveClaimZK(verifierUrl, credID: credentialId, ret0_:&result)
                    if (result.boolValue == true) {
                        return true
                    } else {
                        return false
                    }
                } catch {
                    return false
                }
            } else {
                identity.proveClaimZK(withCb: verifierUrl, credID: credentialId, c: nil)
            }
        }
        return false
    }
    
    /// List of claims of the identity.
    ///
    /// return ArrayList with the claims of the identity.
    ///
    public func listClaims() -> Array<Dictionary<String, String>> {
        let claims = [Dictionary<String,String>]()
        let cdb = identity.claimDB
        do {
            let callback = ClaimsIterator<Iden3mobileClaimDBIterFnerProtocol>(claims)
            try cdb?.iterateClaimsJSON(callback as? Iden3mobileClaimDBIterFnerProtocol)
        } catch {

        }
        return claims
    }
    
    /// List of credentials of the identity.
    ///
    /// return ArrayList with the claims of the identity.
    ///
    public func listCredentials() -> Array<Dictionary<String, String>> {
        let credentials = [Dictionary<String,String>]()
        let cdb = identity.claimDB
        do {
            let callback = ClaimsIterator<Iden3mobileClaimDBIterFnerProtocol>(credentials)
            try cdb?.iterateCredExistJSON(callback as? Iden3mobileClaimDBIterFnerProtocol)
        } catch {

        }
        return credentials
    }
    
    private class ClaimsIterator<Iden3mobileClaimDBIterFnerProtocol> {
           
        private var claims : Array<Dictionary<String, String>>
           
        init(_ claims: Array<Dictionary<String, String>>) {
            self.claims = claims
        }
           
        func fn(_ key: String?, claim: String?, ret0_: UnsafeMutablePointer<ObjCBool>?) throws {
            var map = Dictionary<String, String>()
            map["key"] = key
            map["claim"] = claim
            self.claims.append(map)
        }
    }

    /**
     * List of events of the identity.
     *
     * @return ArrayList with the events of the identity.
     */
    public func listEvents() -> Array<Iden3Ticket> {
        let events = [Iden3Ticket]()
        do {
        try identity.tickets?.iterate(TicketIterator<Iden3mobileTicketOperatorProtocol>(events) as? Iden3mobileTicketOperatorProtocol)
        } catch {
            
        }
        return events
    }
    
    private class TicketIterator<Iden3mobileTicketOperatorProtocol> {
           
        private var tickets : Array<Iden3Ticket>
           
        init(_ tickets: Array<Iden3Ticket>) {
            self.tickets = tickets
        }
        
        func iterate(_ ticket: Iden3mobileTicket?, ret0_: UnsafeMutablePointer<ObjCBool>?) throws {
            if (ticket != nil) {
                self.tickets.append(Iden3Ticket(ticket!))
            }
        }
    }

    /**
     * Cancel an event of the identity.
     *
     * @return Boolean that indicates if the event has been cancelled or not.
     */
    public func cancelEvent(eventId: String) -> Bool {
        do {
            try identity.tickets?.cancelTicket(eventId)
            return true
        } catch  {
            return false
        }
    }
}
