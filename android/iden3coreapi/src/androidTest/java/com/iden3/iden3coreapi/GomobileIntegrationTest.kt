package com.iden3.iden3coreapi

import android.util.Log
import androidx.test.ext.junit.runners.AndroidJUnit4
import androidx.test.platform.app.InstrumentationRegistry
import iden3mobile.*
import org.junit.Assert.assertEquals
import org.junit.Assert.assertNotEquals
import org.junit.Test
import org.junit.runner.RunWith
import java.io.File
import java.lang.Exception
import java.time.Instant

/**
 * Instrumented test, which will execute on an Android device.
 *
 * See [testing documentation](http://d.android.com/tools/testing).
 */
@RunWith(AndroidJUnit4::class)
class GomobileIntegrationTest {
    @Test
    fun fullFlow() {
        // Test config
        val nIdentities = 2
        val nClaimsPerId = 5
        val web3Url = BuildConfig.INFURA_URL
        val issuerUrl = "http://167.172.104.160:6100/api/unstable"
        val verifierUrl = "http://167.172.104.160:6200/api/unstable"

        // Create a new directory for each identity
        val appContext = InstrumentationRegistry.getInstrumentation().targetContext
        val storePath = appContext.filesDir.absolutePath
        for (i in 0 until nIdentities){
            // Remove directory in case last test did't finish
            File("$storePath/$i").deleteRecursively()
            // Create directory
            File("$storePath/$i/").mkdirs()
        }

        // 1. Create nIdentities
        var eventCounter = 0
        Log.i("fullFlow","CREATING $nIdentities IDENTITIES")
        var ids = List(nIdentities) { i ->
            try {
                Iden3mobile.newIdentity(
                    "$storePath/$i",
                    "password",
                    web3Url,
                    1000,
                    null,
                    object : Sender {
                        override fun send(event: Event) {
                            eventCounter++
                            Log.i("fullFlow","EVENT RECEIVED: ${event.ticketId}. $eventCounter EVENTS RECEIVED SO FAR")
                            assertEquals(null, event.err)
                        }
                    }
                )
            } catch (e: Exception) {
                assertEquals(null, e)
                null
            }
        }

        // 2. Request claims
        Log.i("fullFlow", "REQUESTING $nClaimsPerId CLAIMS FOR EACH IDENTITY")
        var ticketCounter = 0
        for (i in 0 until nClaimsPerId){
            var idCount = 0
            for (id in ids){
                id?.requestClaimWithCb(issuerUrl, "$idCount/$i/${Instant.now()}", object: CallbackRequestClaim{
                    override fun fn(ticket: Ticket?, e: Exception?) {
                        assertNotEquals(null, ticket)
                        assertEquals(null, e)
                        ticketCounter++
                        Log.i("fullFlow","REQUEST CLAIM TICKET RECEIVED: ${ticket?.id}. $ticketCounter TICKETS RECEIVED SO FAR}")
                    }
                })
                idCount++
            }
        }
        // Wait for callbacks
        while (ticketCounter < nIdentities*nClaimsPerId){
            Log.i("fullFlow","WAITING FOR REQUEST CLAIM TICKETS")
            Thread.sleep(100)
        }

        // Restart identities
        ids = restartIdentities(ids, storePath, web3Url, fun (event: Event) {
            eventCounter++
            Log.i("fullFlow","EVENT RECEIVED: ${event.ticketId}. $eventCounter EVENTS RECEIVED SO FAR")
            assertEquals(null, event.err)
        })

        // Wait to receive claims
        Log.i("fullFlow","WAITING TO RECEIVE CLAIMS")
        while (eventCounter < nIdentities*nClaimsPerId){
            Log.i("fullFlow","WAITING FOR REQUEST CLAIM EVENTS.")
            Thread.sleep(1000)
        }
        // Check claims on DB
        assertEquals(nIdentities*nClaimsPerId, countClaims(ids))

        // 3. Prove claims
        // Since the claims have been issued instants ago, they may not be on chain yet
        // so it's normal to receive error in the upcoming seconds
        var provedClaims = 0
        var attempts = 10
        while (provedClaims < nIdentities*nClaimsPerId && attempts >= 0){
            provedClaims = 0
            for (id in ids){
                id?.getClaimDB()?.iterateClaimsJSON(object: ClaimDBIterFner{
                    override fun fn(key: String, claim: String): Boolean{
                        id.proveClaimWithCb(verifierUrl, key, object: CallbackProveClaim {
                            override fun fn(success: Boolean, e: Exception?) {
                                Log.i("fullFlow", "Verify claim: $key. Success? $success. Error? $e")
                                if(e == null){
                                    assertEquals(true, success)
                                    provedClaims++
                                }
                            }
                        })
                        return true
                    }
                })
            }
            attempts--
            Log.i("fullFlow","WAITING FOR CLAIMS TO BE PROVED.")
            Thread.sleep(2_000)
        }
        assertEquals(nClaimsPerId*nIdentities, provedClaims)

        // Restart identities
        ids = restartIdentities(ids, storePath, web3Url, fun (event: Event) {
            eventCounter++
            Log.i("fullFlow","UNEXPECTED EVENT RECEIVED: ${event.ticketId}.")
            assertEquals(null, event)
        })

        // Check claims on DB
        assertEquals(nIdentities*nClaimsPerId, countClaims(ids))

        // Test cancel ticket
        // Request a claim to generate a ticket
        ticketCounter = 0
        for (id in ids){
            id?.requestClaimWithCb(issuerUrl, "${Instant.now()}", object: CallbackRequestClaim{
                override fun fn(ticket: Ticket?, e: Exception?) {
                    assertNotEquals(null, ticket)
                    assertEquals(null, e)
                    Log.i("fullFlow","REQUEST CLAIM TICKET RECEIVED.")
                    // Cancel ticket
                    id?.tickets.cancelTicket(ticket?.id)
                    ticketCounter++
                }
            })
        }
        while (ticketCounter < nIdentities){
            Log.i("fullFlow","WAITING FOR TICKETS TO BE GENERATED.")
            Thread.sleep(1000)
        }
        // Check tickets
        Log.i("fullFlow","CHECK TICKETS AFTER RESTART")
        // Give time for cancellation to take effect
        Thread.sleep(1000)
        ticketCounter = 0
        var cancelledTicketCounter = 0
        for (id in ids){
            id?.tickets?.iterate(object: TicketOperator{
                override fun iterate(ticket: Ticket?): Boolean {
                    ticketCounter++
                    if(ticket?.status == Iden3mobile.TicketStatusCancel){
                        cancelledTicketCounter++
                    }
                    return true
                }
            })
        }
        assertEquals(nIdentities*nClaimsPerId + nIdentities, ticketCounter)
        assertEquals(nIdentities, cancelledTicketCounter)

        // Stop identities
        for(id in ids){
            id?.stop()
        }

        // Remove identity directories
        for (i in 0 until nIdentities){
            File("$storePath/$i").deleteRecursively()
        }
    }

    fun restartIdentities(ids: List<Identity?>, storePath: String, web3Url: String, fn: (e:Event)->Unit): List<Identity?>{
        Log.i("fullFlow","RESTARTING IDENTITIES")
        for (id in ids){
            id?.stop()
        }
        return List(ids.size) { i ->
            try {
                Iden3mobile.newIdentityLoad(
                        "$storePath/$i",
                        "password",
                        web3Url,
                        1000,
                        object : Sender {
                            override fun send(event: Event) {
                                fn(event)
                            }
                        }
                )
            } catch (e: Exception) {
                assertEquals(null, e)
                null
            }
        }
    }

    fun countClaims(ids: List<Identity?>): Int {
        var claimCounter = 0
        for (id in ids){
            id?.getClaimDB()?.iterateClaimsJSON(object: ClaimDBIterFner{
                override fun fn(claim: String, key: String): Boolean{
                    claimCounter++
                    return true
                }
            })
        }
        return claimCounter
    }
}
