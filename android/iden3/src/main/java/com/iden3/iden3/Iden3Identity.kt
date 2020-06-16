package com.iden3.iden3

import iden3mobile.*
import java.util.ArrayList
import java.util.HashMap

class Iden3Identity(private var identity: Identity) {

    /**
     * Stop an Iden3 identity.
     *
     */
    fun stopIdentity() {
        identity.stop()
    }

    /**
     * Request a claim from an issuer.
     *
     * @param issuerUrl: Url of the issuer.
     * @param data: data needed by the issuer for requesting a claim.
     * @param ticketCallback: Callback of tickets received after requesting the claim.
     *
     */
    @Throws(Exception::class)
    fun requestClaim(issuerUrl: String, data: String, ticketCallback: Iden3TicketCallback?) {
        identity.requestClaimWithCb(issuerUrl, data) {
                ticket, exception ->
            if (ticketCallback != null) {
                if (exception != null) {
                    ticketCallback.onError(exception)
                } else if (ticket != null) {
                    ticketCallback.onTicketReceived(Iden3Ticket(ticket))
                }
            }
        }
    }

    /**
     * Send a proof of a claim to specified verifier.
     * If withZKProof is set to true, the generated proof will hide part of the claim information to the verifier
     *
     * @param verifierUrl: Url of the verifier.
     * @param credentialId: Id of the credential that will be used to generate the proof.
     * @param withZKProof: Boolean that indicates if zero-knowledge proof shoud be used or not.
     * @param proveClaimCallback: Callback of the response of the verifier proving a claim.
     *
     */
    @Throws(Exception::class)
    fun proveClaim(verifierUrl: String, credentialId: String, withZKProof: Boolean, proveClaimCallback: Iden3ProveClaimCallback?) {
        if (withZKProof) {
            identity.proveClaimWithCb(verifierUrl, credentialId) {
                    isProven, exception ->
                if (proveClaimCallback != null) {
                    if (exception != null) {
                        proveClaimCallback.onError(exception)
                    } else {
                        proveClaimCallback.onVerifierResponse(isProven)
                    }
                }
            }
        } else {
            identity.proveClaimZKWithCb(verifierUrl, credentialId) {
                    isProven, exception ->
                if (proveClaimCallback != null) {
                    if (exception != null) {
                        proveClaimCallback.onError(exception)
                    } else {
                        proveClaimCallback.onVerifierResponse(isProven)
                    }
                }
            }
        }
    }

    /**
     * List of claims of the identity.
     *
     * @return ArrayList with the claims of the identity.
     */
    @Throws(Exception::class)
    fun listClaims() : ArrayList<Iden3Claim> {
        val claims = ArrayList<Iden3Claim>()
        try {
            val cdb = identity.claimDB
            cdb.iterateClaimsJSON { key, claim ->
                val iden3Claim = Iden3Claim(key, claim)
                claims.add(iden3Claim)
            }
        } catch (e: java.lang.Exception) {

        }
        return claims
    }

    /**
     * List of credentials of the identity.
     *
     * @return ArrayList with the credentials of the identity.
     */
    @Throws(Exception::class)
    fun listCredentials() : ArrayList<Iden3Credential> {
        val credentials = ArrayList<Iden3Credential>()
        try {
            val cdb = identity.claimDB
            cdb.iterateCredExistJSON { key, credential ->
                val iden3Credential = Iden3Credential(key, credential)
                credentials.add(iden3Credential)
            }
        } catch (e: Exception) {

        }
        return credentials
    }

    /**
     * List of events of the identity.
     *
     * @return ArrayList with the events of the identity.
     */
    fun listEvents() : ArrayList<Iden3Ticket> {
        val events = ArrayList<Iden3Ticket>()
        identity.tickets.iterate {
            events.add(Iden3Ticket(it))
        }
        return events
    }

    /**
     * Cancel an event of the identity.
     *
     * @return Boolean that indicates if the event has been cancelled or not.
     */
    fun cancelEvent(eventId: String) : Boolean {
        return try {
            identity.tickets.cancelTicket(eventId)
            true
        } catch (e: Exception) {
            false
        }
    }
}