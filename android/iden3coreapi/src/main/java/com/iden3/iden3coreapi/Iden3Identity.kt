package com.iden3.iden3coreapi

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
     * @param ticketListener: Listener of tickets received after requesting the claim.
     *
     * @return An Iden3Ticket.
     */
    @Throws(Exception::class)
    fun requestClaim(issuerUrl: String, data: String, ticketListener: Iden3TicketListener?) : Iden3Ticket? {
        if (ticketListener == null) {
            return Iden3Ticket(identity.requestClaim(issuerUrl, data))
        } else {
            identity.requestClaimWithCb(issuerUrl, data) {
                ticket, exception ->
                if (exception != null) {
                    ticketListener.onError(exception)
                } else if (ticket != null) {
                    ticketListener.onTicketReceived(Iden3Ticket(ticket))
                }
            }
        }
        return null
    }

    /**
     * Prove a claim from an verifier.
     *
     * @param verifierUrl: url of the verifier.
     * @param credentialId: credential id needed by the verifier for proving a claim.
     * @param withZKProof: Boolean that indicates if zero-knowledge proof shoud be used or not.
     * @param proveClaimListener: Listener of proofs received after proving a claim.
     *
     * @return Boolean that indicates if the claim has been proven or not.
     */
    @Throws(Exception::class)
    fun proveClaim(verifierUrl: String, credentialId: String, withZKProof: Boolean, proveClaimListener: Iden3ProveClaimListener?) : Boolean?  {
        if (withZKProof) {
            if (proveClaimListener == null) {
                return identity.proveClaim(verifierUrl, credentialId)
            } else {
                identity.proveClaimWithCb(verifierUrl, credentialId) {
                    isProven, exception ->
                    if (exception != null) {
                        proveClaimListener.onError(exception)
                    } else {
                        proveClaimListener.onClaimProofReceived(isProven)
                    }
                }
            }
        } else {
            if (proveClaimListener == null) {
                return identity.proveClaimZK(verifierUrl, credentialId)
            } else {
                identity.proveClaimZKWithCb(verifierUrl, credentialId) {
                        isProven, exception ->
                        if (exception != null) {
                            proveClaimListener.onError(exception)
                        } else {
                            proveClaimListener.onClaimProofReceived(isProven)
                        }
                }
            }
        }
        return null
    }

    /**
     * List of claims of the identity.
     *
     * @return ArrayList with the claims of the identity.
     */
    @Throws(Exception::class)
    fun listClaims() : ArrayList<HashMap<*,*>> {
        val claims = ArrayList<HashMap<*, *>>()
        try {
            val cdb = identity.claimDB
            cdb.iterateClaimsJSON { key, claim ->
                val map = HashMap<String, Any?>()
                map["key"] = key
                map["claim"] = claim
                claims.add(map)
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
    fun listCredentials() : ArrayList<HashMap<*,*>> {
        val credentials = ArrayList<HashMap<*,*>>()
        try {
            val cdb = identity.claimDB
            cdb.iterateCredExistJSON { key, credential ->
                val map = HashMap<String, Any?>()
                map["key"] = key
                map["credential"] = credential
                credentials.add(map)
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