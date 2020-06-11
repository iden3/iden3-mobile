package com.iden3.iden3coreapi

import iden3mobile.*
import java.io.File
import java.io.FileNotFoundException
import java.lang.IllegalArgumentException

open class Iden3IdentityFactory {

    companion object {
        @JvmStatic lateinit var instance: Iden3IdentityFactory
    }

    init {
        instance = this
    }

    private lateinit var web3Url: String
    private lateinit var storePath: String
    private var checkTicketsPeriod : Long = 10000

    /**
     * Initialize the Iden3IdentityFactory library with the parameters need to setup.
     *
     * @param web3Url: String the Web3 url.
     * @param storePath: String the absolute path where to store the identities.
     * @param checkTicketsPeriod: Long time in miliseconds of the period needed for checking the tickets.
     *
     * @return Boolean if the initialization has been successful.
     */
    fun initializeAPI(web3Url: String, storePath: String, checkTicketsPeriod: Long) : Boolean {
        this.web3Url = web3Url
        this.storePath = storePath
        this.checkTicketsPeriod = checkTicketsPeriod
        return isInitialized()
    }

    /**
     * Checks if the Iden3IdentityFactory library has been initialized successfully.
     *
     * @return Boolean if the initialization has been successful.
     */
    fun isInitialized() : Boolean {
        return web3Url.isNotEmpty() && storePath.isNotEmpty() && checkTicketsPeriod > 0
    }

    /**
     * Creates a new Iden3 identity.
     *
     * @param alias: String the alias of the identity.
     * @param password: String the password to access the identity.
     * @param eventListener: Listener of events associated to the identity.
     *
     * @return The new Iden3 identity created.
     */
    @Throws(Exception::class)
    fun createIdentity(alias: String, password: String, eventListener: Iden3EventListener?) : Iden3Identity? {
        if (isInitialized()) {
            if (alias.isEmpty() && isAlphaNumeric(alias) || password.isEmpty()) {
                throw IllegalArgumentException("Iden3 method called with not valid arguments")
            } else {
                try {
                    val file = File("$storePath/identities/$alias")
                    if (!file.exists()) {
                        file.deleteRecursively()
                        file.mkdirs()
                    }
                    return Iden3Identity(Iden3mobile.newIdentity(
                        "$storePath/identities/$alias",
                        "$storePath/shared",
                        password,
                        web3Url,
                        checkTicketsPeriod,
                        null
                    ) { event -> eventListener?.onEventReceived(Iden3Event(event)) })
                } catch (e:Exception) {
                    throw e
                }
            }
        } else {
            throw IllegalStateException("Iden3 API is not initialized. Please, call initializeAPI method before doing this call")
        }
    }

    /**
     * Loads a previously created Iden3 identity.
     *
     * @param alias: String the alias of the identity.
     * @param password: String the password to access the identity.
     * @param eventListener: Listener of events associated to the identity.
     *
     * @return The Iden3 identity loaded.
     */
    @Throws(Exception::class)
    fun loadIdentity(alias: String, password: String, eventListener: Iden3EventListener?) : Iden3Identity? {
        if (isInitialized()) {
            if (alias.isEmpty() || password.isEmpty()) {
                throw IllegalArgumentException("Iden3 method called with not valid arguments")
            } else {
                try {
                    val file = File("$storePath/identities/$alias")
                    if (file.exists()) {
                        return Iden3Identity(Iden3mobile.newIdentityLoad("$storePath/identities/$alias",
                            "$storePath/shared",
                            password,
                            web3Url,
                            checkTicketsPeriod
                        ) { event -> eventListener?.onEventReceived(Iden3Event(event)) })
                    } else {
                        throw FileNotFoundException("Identity not found. Please be sure the identity" +
                                " is created calling createIdentity method before loading it")
                    }
                } catch (e:Exception) {
                    throw e
                }
            }
        } else {
            throw IllegalStateException("Iden3 API is not initialized. Please, call initializeAPI method before doing this call")
        }
    }

    private fun isAlphaNumeric(chars: String): Boolean {
        return chars.matches("^[a-zA-Z0-9]*$".toRegex())
    }
}