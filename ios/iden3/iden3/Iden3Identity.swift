//
//  Iden3Identity.swift
//  iden3
//
//  Created by Iden3 on 12/06/2020.
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
    ///
    public func requestClaim(issuerUrl: String, data: String, ticketDelegate: Iden3TicketDelegate?) {
        let callback = RequestClaimCallback<Iden3mobileCallbackRequestClaimProtocol>(ticketDelegate)
            identity.requestClaim(withCb: issuerUrl, data: data, c: callback as? Iden3mobileCallbackRequestClaimProtocol)
    }
    
    /// Send a proof of a claim to specified verifier.
    /// If withZKProof is set to true, the generated proof will hide part of the claim information to the verifier
    ///
    /// - parameters:
    ///   - verifierUrl: url of the verifier
    ///   - credentialId: credential id needed by the verifier for proving a claim.
    ///   - withZKProof: Boolean that indicates if zero-knowledge proof shoud be used or not.
    ///   - proveClaimDelegate: Delegate of the response of the verifier proving a claim.
    ///
    ///
    public func proveClaim(verifierUrl: String, credentialId: String, withZKProof: Bool, proveClaimDelegate: Iden3ProveClaimDelegate?) {
        let callback = ProveClaimCallback<Iden3mobileCallbackProveClaimProtocol>(proveClaimDelegate)
        if (!withZKProof) {
            identity.proveClaim(withCb: verifierUrl, credID: credentialId, c: callback as? Iden3mobileCallbackProveClaimProtocol)
        } else {
            identity.proveClaimZK(withCb: verifierUrl, credID: credentialId, c: callback as? Iden3mobileCallbackProveClaimProtocol)
        }
    }
    
    /// List of claims of the identity.
    ///
    /// return ArrayList with the claims of the identity.
    ///
    public func listClaims() -> Array<Iden3Claim> {
        let claims = [Iden3Claim]()
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
    /// return ArrayList with the credentials of the identity.
    ///
    public func listCredentials() -> Array<Iden3Credential> {
        let credentials = [Iden3Credential]()
        let cdb = identity.claimDB
        do {
            let callback = CredentialsIterator<Iden3mobileClaimDBIterFnerProtocol>(credentials)
            try cdb?.iterateCredExistJSON(callback as? Iden3mobileClaimDBIterFnerProtocol)
        } catch {

        }
        return credentials
    }
    
    private class RequestClaimCallback<Iden3mobileCallbackRequestClaimProtocol> {
           
        private var delegate : Iden3TicketDelegate?
           
        init(_ delegate: Iden3TicketDelegate?) {
            self.delegate = delegate
        }
           
        func fn(_ ticket: Iden3mobileTicket?, error: Error?) throws {
            if (delegate != nil) {
                if (error == nil) {
                    delegate?.onTicketReceived(ticket: Iden3Ticket(ticket))
                } else {
                    delegate?.onError(error: error!)
                }
            }
        }
    }
    
    private class ProveClaimCallback<Iden3mobileCallbackProveClaimProtocol> {
           
        private var delegate : Iden3ProveClaimDelegate?
           
        init(_ delegate: Iden3ProveClaimDelegate?) {
            self.delegate = delegate
        }
           
        func fn(_ verified: Bool, error: Error?) throws {
            if (delegate != nil) {
                if (error == nil) {
                    delegate?.onVerifierResponse(verified: verified)
                } else {
                    delegate?.onError(error: error!)
                }
            }
        }
    }
    
    private class ClaimsIterator<Iden3mobileClaimDBIterFnerProtocol> {
           
        private var claims : Array<Iden3Claim>
           
        init(_ claims: Array<Iden3Claim>) {
            self.claims = claims
        }
           
        func fn(_ key: String?, claim: String?, ret0_: UnsafeMutablePointer<ObjCBool>?) throws {
            let iden3Claim = Iden3Claim.init(key: key, claim: claim)
            self.claims.append(iden3Claim)
        }
    }
    
    private class CredentialsIterator<Iden3mobileClaimDBIterFnerProtocol> {
           
        private var credentials : Array<Iden3Credential>
           
        init(_ credentials: Array<Iden3Credential>) {
            self.credentials = credentials
        }
           
        func fn(_ key: String?, claim: String?, ret0_: UnsafeMutablePointer<ObjCBool>?) throws {
            let iden3Credential = Iden3Credential.init(key: key, credential: claim)
            self.credentials.append(iden3Credential)
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
