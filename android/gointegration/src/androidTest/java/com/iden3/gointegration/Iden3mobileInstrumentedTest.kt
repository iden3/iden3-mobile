package com.iden3.gointegration

import android.content.Context
import android.util.Log
import androidx.test.ext.junit.runners.AndroidJUnit4
import androidx.test.platform.app.InstrumentationRegistry
import iden3mobile.Iden3mobile
import iden3mobile.Identity
import junit.framework.TestCase.assertEquals
import org.hamcrest.core.StringContains
import org.junit.Assert
import org.junit.Assert.assertNotEquals
import org.junit.Before
import org.junit.Rule
import org.junit.Test
import org.junit.rules.ExpectedException
import org.junit.runner.RunWith
import java.io.File
import java.time.Instant
import java.util.*


/**
 * Iden3mobile local unit test, which will execute on the development machine (host).
 *
 * See [testing documentation](http://d.android.com/tools/testing).
 */

@RunWith(AndroidJUnit4::class)
class Iden3mobileInstrumentedTest {

    private val web3Url = BuildConfig.INFURA_URL
    private val issuerUrl = "http://192.168.200.181:6100/api/unstable/"
    private val verifierUrl = "http://192.168.200.181:6200/api/unstable/"
    private lateinit var instrumentationCtx: Context
    private lateinit var storePath: String
    private lateinit var sharedStorePath: String

    @Rule
    @JvmField
    val expectedException: ExpectedException = ExpectedException.none()

    @Before
    fun setup() {
        instrumentationCtx = InstrumentationRegistry.getInstrumentation().targetContext
        storePath = instrumentationCtx.filesDir.absolutePath
        sharedStorePath = instrumentationCtx.filesDir.absolutePath + "/shared"
        // getSystemService(instrumentationCtx.DOWNLOAD_SERVICE)
    }

    @Test
    fun testCreateIdentitySuccess() {
        val file = File("$storePath/alias")
        if (file.exists()) {
            file.deleteRecursively()
        }
        file.mkdirs()
        Iden3mobile.newIdentity(
            "$storePath/alias",
            sharedStorePath,
            "password",
            web3Url,
            1000,
            null
        ) { event -> print(event) }
        assert(true)
    }

    @Test
    fun testCreateIdentityErrorPathNotExist() {
        expectedException.expect(Exception::class.java)
        expectedException.expectMessage(StringContains("no such file or directory"))
        val file = File("$storePath/alias")
        if (file.exists()) {
            file.deleteRecursively()
        }
        Iden3mobile.newIdentity(
            "$storePath/alias",
            sharedStorePath,
            "password",
            web3Url,
            1000,
            null
        ) { event -> print(event) }
    }

    @Test
    fun testCreateIdentityErrorNullStorePath() {
        expectedException.expect(Exception::class.java)
        expectedException.expectMessage(StringContains("no such file or directory"))
        Iden3mobile.newIdentity(
            null,
            sharedStorePath,
            "password",
            web3Url,
            1000,
            null
        ) { event -> print(event) }
    }

    @Test
    fun testCreateIdentityErrorNullPassword() {
        //expectedException.expect(Exception::class.java)
        //expectedException.expectMessage(StringContains("password cannot be null"))
        val file = File("$storePath/alias")
        if (file.exists()) {
            file.deleteRecursively()
        }
        file.mkdirs()
        Iden3mobile.newIdentity(
            "$storePath/alias",
            sharedStorePath,
            null,
            web3Url,
            1000,
            null
        ) { event -> print(event) }
    }

    @Test
    fun testCreateIdentityErrorNullWeb3Url() {
        expectedException.expect(Exception::class.java)
        expectedException.expectMessage(StringContains("Error dialing with ethclient: dial unix: missing address"))
        val file = File("$storePath/alias")
        if (file.exists()) {
            file.deleteRecursively()
        }
        file.mkdirs()
        Iden3mobile.newIdentity(
            "$storePath/alias",
            sharedStorePath,
            "password",
            null,
            1000,
            null
        ) { event -> print(event) }
    }

    @Test
    fun testCreateIdentityErrorCheckTicketsZero() {
        //expectedException.expect(Exception::class.java)
        //expectedException.expectMessage(StringContains("checkTicketsPeriodMilis should be bigger than zero"))
        val file = File("$storePath/alias")
        if (file.exists()) {
            file.deleteRecursively()
        }
        file.mkdirs()
        Iden3mobile.newIdentity(
            "$storePath/alias",
            sharedStorePath,
            "password",
            web3Url,
            0,
            null
        ) { event -> print(event) }
    }

    @Test
    fun testCreateIdentityErrorCheckTicketsNegative() {
        //expectedException.expect(Exception::class.java)
        //expectedException.expectMessage(StringContains("checkTicketsPeriodMilis should be bigger than zero"))
        val file = File("$storePath/alias")
        if (file.exists()) {
            file.deleteRecursively()
        }
        file.mkdirs()
        Iden3mobile.newIdentity(
            "$storePath/alias",
            sharedStorePath,
            "password",
            web3Url,
            -1000,
            null
        ) { event -> print(event) }
    }

    @Test
    fun testLoadIdentitySuccess() {
        val file = File("$storePath/alias")
        if (file.exists()) {
            file.deleteRecursively()
        }
        file.mkdirs()
        val identity = Iden3mobile.newIdentity(
            "$storePath/alias",
            sharedStorePath,
            "password",
            web3Url,
            1000,
            null
        ) { event -> print(event) }

        identity.stop()

        Iden3mobile.newIdentityLoad(
            "$storePath/alias",
            sharedStorePath,
            "password",
            web3Url,
            1000
        ) { event -> print(event) }
    }

    @Test
    fun testLoadIdentityErrorNotCreatedYet() {
        expectedException.expect(Exception::class.java)
        expectedException.expectMessage(StringContains("no such file or directory"))
        val file = File("$storePath/alias")
        if (file.exists()) {
            file.deleteRecursively()
        }
        Iden3mobile.newIdentityLoad(
            "$storePath/alias",
            sharedStorePath,
            "password",
            web3Url,
            1000
        ) { event -> print(event) }
    }

    @Test
    fun testLoadIdentityErrorNotStopped() {
        expectedException.expect(Exception::class.java)
        expectedException.expectMessage(StringContains("Error opening leveldb storage: resource temporarily unavailable"))
        val file = File("$storePath/alias")
        if (file.exists()) {
            file.deleteRecursively()
        }
        file.mkdirs()
        Iden3mobile.newIdentity(
            "$storePath/alias",
            sharedStorePath,
            "password",
            web3Url,
            1000,
            null
        ) { event -> print(event) }

        Iden3mobile.newIdentityLoad(
            "$storePath/alias",
            sharedStorePath,
            "wrongPassword",
            web3Url,
            1000
        ) { event -> print(event) }
    }

    @Test
    fun testLoadIdentityErrorWrongPassword() {
        expectedException.expect(Exception::class.java)
        expectedException.expectMessage(StringContains("Error unlocking babyjub key from keystore: invalid encrypted data"))
        val file = File("$storePath/alias")
        if (file.exists()) {
            file.deleteRecursively()
        }
        file.mkdirs()
        val identity = Iden3mobile.newIdentity(
            "$storePath/alias",
            sharedStorePath,
            "password",
            web3Url,
            1000,
            null
        ) { event -> print(event) }

        identity.stop()

        Iden3mobile.newIdentityLoad(
            "$storePath/alias",
            sharedStorePath,
            "wrongPassword",
            web3Url,
            1000
        ) { event -> print(event) }
    }

    @Test
    fun testRequestClaimSuccess() {
        val file = File("$storePath/alias")
        if (file.exists()) {
            file.deleteRecursively()
        }
        file.mkdirs()
        val identity = Iden3mobile.newIdentity(
            "$storePath/alias",
            sharedStorePath,
            "password",
            web3Url,
            1000,
            null
        ) { event -> print(event) }
        val data = random()
        val ticket = identity.requestClaim(issuerUrl, data)
        Log.i("testRequestClaimSuccess","Ticket: $ticket")
        Assert.assertNotEquals(null, ticket)
        identity.stop()
    }

    /**
     * This test is broken, it will pass even when ticket is null or e is not null
    @Test
    fun testRequestClaimWithCallbackSuccess() {
        val file = File("$storePath/alias")
        if (file.exists()) {
            file.deleteRecursively()
        }
        file.mkdirs()
        val identity = Iden3mobile.newIdentity(
            "$storePath/alias",
            "password",
            web3Url,
            1000,
            null
        ) { event -> print(event) }

        var isFinished = false
        val data = random()
        identity.requestClaimWithCb(issuerUrl, data) { ticket, e ->
            isFinished = true
            Log.i("testRequestClaimWithCallbackSuccess","Ticket: $ticket")
            assertNotEquals(null, ticket)
            assertEquals(null, e)
            identity.stop()
        }

        // Wait for callback
        while (!isFinished){
            Log.i("testRequestClaimWithCallbackSuccess","Waiting for request claim ticket")
            Thread.sleep(100)
        }
    }
**/

    /**
     * This test is broken:
        * if requestClaimWithCb has error the test will run forever
        * proveClaimWithCb is called with ticket.id instead of claim.id
    @Test
    fun testProveClaimSuccess() {
        val file = File("$storePath/alias")
        if (file.exists()) {
            file.deleteRecursively()
        }
        file.mkdirs()
        val identity = Iden3mobile.newIdentity(
                "$storePath/alias",
                "password",
                web3Url,
                1000,
                null
        ) { event -> print(event) }

        var isFinished = false
        identity.requestClaimWithCb(issuerUrl, "${Instant.now()}") { ticket, e ->
            Log.i("testProveClaimWithCallbackSuccess","Ticket: $ticket")
            Assert.assertNotEquals(null, ticket)
            Assert.assertEquals(null, e)
            identity.proveClaimWithCb(verifierUrl, ticket.id) { success, exception ->
                Log.i("testProveClaimWithCallbackSuccess", "Proving Clam success: $success exception: $exception")
                isFinished = true
                Assert.assertEquals(true, success)
                Assert.assertEquals(null, exception)
                identity.stop()
            }
        }

        // Wait for callback
        while (!isFinished){
            Log.i("testProveClaimWithCallbackSuccess","Waiting for request claim ticket")
            Thread.sleep(100)
        }
    }
**/

    @Test
    fun testProveClaimWithCallbackSuccess() {
        var eventReceived = false
        val file = File("$storePath/alias")
        if (file.exists()) {
            file.deleteRecursively()
        }
        file.mkdirs()
        val identity = Iden3mobile.newIdentity(
            "$storePath/alias",
            sharedStorePath,
            "password",
            web3Url,
            1000,
            null
        ) { event ->
            eventReceived = true
            Log.i("fullFlow","Event Received: ${event.ticketId}.")
            assertEquals(null, event.err)
            print(event)
        }

        var isFinished = false
        var errMsg = ""
        val data = random()
        identity.requestClaimWithCb(issuerUrl, data) { ticket, e ->
            Log.i("testProveClaimWithCallbackSuccess","Ticket: $ticket")
            if(ticket == null || e != null){
                errMsg ="Wrong request response: ${e.message}"
                isFinished = true
                return@requestClaimWithCb
            }
            while (countClaims(identity) == 0 && !eventReceived) {
                Log.i("testProveClaimWithCallbackSuccess","Waiting for claim to be available in the database")
                Thread.sleep(1000)
            }
            identity.claimDB?.iterateClaimsJSON { key, _ ->
                identity.proveClaimWithCb(verifierUrl, key) { success, exception ->
                    if(!success || exception != null){
                        errMsg ="Error proving: ${exception.message}"
                    }
                    isFinished = true
                }
                true
            }
        }

        // Wait for callback
        while (!isFinished){
            Log.i("testProveClaimWithCallbackSuccess","Waiting for request claim ticket")
            Thread.sleep(1000)
        }
        identity.stop()
        Assert.assertEquals("", errMsg)
    }

    @Test
    fun testProveClaimZKWithCallbackSuccess() {
        var eventReceived = false
        val file = File("$storePath/alias")
        if (file.exists()) {
            file.deleteRecursively()
        }
        file.mkdirs()
        val identity = Iden3mobile.newIdentity(
                "$storePath/alias",
                sharedStorePath,
                "password",
                web3Url,
                1000,
                null
        ) { event ->
            eventReceived = true
            Log.i("fullFlow","Event Received: ${event.ticketId}.")
            assertEquals(null, event.err)
            print(event)
        }

        var isFinished = false
        var errMsg = ""
        val data = random()
        identity.requestClaimWithCb(issuerUrl, data) { ticket, e ->
            Log.i("testProveClaimZKWithCallbackSuccess","Ticket: $ticket")
            if(ticket == null || e != null){
                errMsg ="Wrong request response: ${e.message}"
                isFinished = true
                return@requestClaimWithCb
            }
            while (countClaims(identity) == 0 && !eventReceived) {
                Log.i("testProveClaimZKWithCallbackSuccess","Waiting for claim to be available in the database")
                Thread.sleep(1000)
            }
            identity.claimDB?.iterateClaimsJSON { key, _ ->
                identity.proveClaimZKWithCb(verifierUrl, key) { success, exception ->
                    if(!success || exception != null){
                        errMsg ="Error proving: ${exception.message}"
                    }
                    isFinished = true
                }
                true
            }
        }

        // Wait for callback
        while (!isFinished){
            Log.i("testProveClaimZKWithCallbackSuccess","Waiting for request claim ticket")
            Thread.sleep(1000)
        }
        identity.stop()
        Assert.assertEquals("", errMsg)
    }

    @Test
    fun testProveClaimWithCallbackErrorKeyNotFound() {
        expectedException.expect(Exception::class.java)
        expectedException.expectMessage(StringContains("key not found"))
        val file = File("$storePath/alias")
        if (file.exists()) {
            file.deleteRecursively()
        }
        file.mkdirs()
        val identity = Iden3mobile.newIdentity(
            "$storePath/alias",
            sharedStorePath,
            "password",
            web3Url,
            1000,
            null
        ) { event -> print(event) }

        var exp : Exception? = null
        var isFinished = false
        identity.proveClaimWithCb(verifierUrl, "wrong_key") { success, exception ->
            Log.i("testProveClaimWithCallbackErrorKeyNotFound", "Proving Clam success: $success exception: $exception")
            assertEquals(false, success)
            assertNotEquals(null, exception)
            identity.stop()
            exp = exception
            isFinished = true
        }

        // Wait for callback
        while (!isFinished){
            Log.i("testProveClaimWithCallbackErrorKeyNotFound","Waiting for callbacks")
            Thread.sleep(100)
        }
        throw exp!!
    }

    private fun countClaims(identity: Identity): Int {
        var claimCounter = 0
        identity.claimDB?.iterateClaimsJSON { _, _ ->
            claimCounter++
            true
        }
        return claimCounter
    }

    fun random(): String? {
        val generator = Random()
        val randomStringBuilder = StringBuilder()
        var tempChar: Char
        for (i in 0 until 16) {
            tempChar = ((generator.nextInt(10) + 48).toChar())
            randomStringBuilder.append(tempChar)
        }
        return randomStringBuilder.toString()
    }
}
