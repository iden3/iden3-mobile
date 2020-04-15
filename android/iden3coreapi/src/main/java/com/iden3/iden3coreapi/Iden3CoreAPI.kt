package com.iden3.iden3coreapi

import iden3mobile.*
import java.lang.IllegalArgumentException
import java.util.ArrayList
import java.util.HashMap

class Iden3CoreAPI {

    companion object {
        @JvmStatic lateinit var instance: Iden3CoreAPI
    }

    init {
        instance = this
    }

    private lateinit var web3Url: String
    private lateinit var issuerUrl: String
    private lateinit var verifierUrl: String
    private lateinit var storePath: String
    private var checkTicketsPeriod : Long = 10000

    fun initializeAPI(web3Url: String, issuerUrl: String, verifierUrl: String, storePath: String, checkTicketsPeriod: Long) {
        this.web3Url = web3Url
        this.issuerUrl = issuerUrl
        this.verifierUrl = verifierUrl
        this.storePath = storePath
        this.checkTicketsPeriod = checkTicketsPeriod
    }

    fun isInitialized() : Boolean {
        return web3Url.isNotEmpty() && issuerUrl.isNotEmpty() && verifierUrl.isNotEmpty() && storePath.isNotEmpty() && checkTicketsPeriod > 0
    }

    @Throws(Exception::class)
    fun createIdentity(alias: String, password: String) : Identity? {
        if (isInitialized()) {
            if (alias.isEmpty() || password.isEmpty()) {
                throw IllegalArgumentException("Iden3 method called with not valid arguments")
            } else {
                return Iden3mobile.newIdentity(
                    "$storePath/$alias",
                    password,
                    web3Url,
                    checkTicketsPeriod,
                    null
                ) { event -> print(event) }
            }
        } else {
            throw IllegalStateException("Iden3 API is not initialized. Please, call initializeAPI method before doing this call")
        }
    }

    @Throws(Exception::class)
    fun requestClaim(identity: Identity, data: String, callback: CallbackRequestClaim?) : Ticket? {
        if (isInitialized()) {
                if (callback == null) {
                    return identity.requestClaim(issuerUrl, data)
                } else {
                    identity.requestClaimWithCb(issuerUrl, data, callback)
                }
        } else {
            throw IllegalStateException("Iden3 API is not initialized. Please, call initializeAPI method before doing this call")
        }
        return null
    }

    @Throws(Exception::class)
    fun proveClaim(identity: Identity, credentialId: String, callback: CallbackProveClaim?) : Boolean?  {
        if (isInitialized()) {
            if (callback == null) {
                return identity.proveClaim(verifierUrl, credentialId)
            } else {
                identity.proveClaimWithCb(verifierUrl, credentialId, callback)
            }
        } else {
            throw IllegalStateException("Iden3 API is not initialized. Please, call initializeAPI method before doing this call")
        }
        return null
    }

    @Throws(Exception::class)
    fun listClaims(identity: Identity) : ArrayList<HashMap<*,*>> {
        if (isInitialized()) {
            val claims = ArrayList<HashMap<*, *>>()
            try {
                val cdb = identity.claimDB
                cdb.iterateClaimsJSON { key, claim ->
                    val cMap = HashMap<String, Any?>()
                    cMap["DBKey"] = key
                    cMap["claim"] = claim
                    claims.add(cMap)
                }
            } catch (e: java.lang.Exception) {

            }
            return claims
        } else {
            throw IllegalStateException("Iden3 API is not initialized. Please, call initializeAPI method before doing this call")
        }
    }

    @Throws(Exception::class)
    fun listCredentials(identity: Identity) : ArrayList<HashMap<*,*>> {
        if (isInitialized()) {
            val credentials = ArrayList<HashMap<*,*>>()
            try {
                val cdb = identity.claimDB
                cdb.iterateCredExistJSON { key, claim ->
                    val cMap = HashMap<String, Any?>()
                    cMap["DBKey"] = key
                    cMap["claim"] = claim
                    credentials.add(cMap)
                }
            } catch (e: java.lang.Exception) {

            }
            return credentials
        } else {
            throw IllegalStateException("Iden3 API is not initialized. Please, call initializeAPI method before doing this call")
        }
    }
}