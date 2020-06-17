package com.iden3.iden3integration

import android.content.Context
import android.util.Log
import androidx.test.ext.junit.runners.AndroidJUnit4
import androidx.test.platform.app.InstrumentationRegistry
import com.iden3.iden3.Iden3IdentityFactory
import com.iden3.iden3.Iden3Ticket
import com.iden3.iden3.Iden3TicketCallback
import org.junit.Assert.*
import org.junit.Before
import org.junit.Rule
import org.junit.Test
import org.junit.rules.ExpectedException
import org.junit.runner.RunWith
import java.io.File
import java.lang.Exception
import java.time.Instant

/**
 * Iden3 local unit test, which will execute on the development machine (host).
 *
 * See [testing documentation](http://d.android.com/tools/testing).
 */

@RunWith(AndroidJUnit4::class)
class Iden3InstrumentedTest {

    private val web3Url = BuildConfig.INFURA_URL
    private val issuerUrl = "http://167.172.104.160:6100/api/unstable"
    private val verifierUrl = "http://167.172.104.160:6200/api/unstable"
    private lateinit var instrumentationCtx: Context
    private lateinit var storePath: String
    private lateinit var iden3IdentityFactory: Iden3IdentityFactory

    @Rule
    @JvmField
    val expectedException: ExpectedException = ExpectedException.none()

    @Before
    fun setup() {
        instrumentationCtx = InstrumentationRegistry.getInstrumentation().targetContext
        storePath = instrumentationCtx.filesDir.absolutePath
        iden3IdentityFactory = Iden3IdentityFactory.instance
    }

    @Test
    fun testInitializeSuccess() {
        assertEquals(true, initialize())
    }

    @Test
    fun testCreateIdentitySuccess() {
        val isInitialized = initialize()
        if (isInitialized) {
            val identity = iden3IdentityFactory.createIdentity("alias", "password", null)
            assertNotEquals(identity, null)
            identity?.stopIdentity()
        } else {
            assert(false)
        }
    }

    @Test
    fun testLoadIdentitySuccess() {
        val isInitialized = initialize()
        if (isInitialized) {
            if (File("$storePath/alias").exists()) {
                val identityLoaded = iden3IdentityFactory.loadIdentity("alias", "password", null)
                assertNotEquals(identityLoaded, null)
                identityLoaded?.stopIdentity()
            } else {
                val identityCreated = iden3IdentityFactory.createIdentity("alias", "password", null)
                identityCreated?.stopIdentity()
                if (File("$storePath/alias").exists()) {
                    val identityLoaded = iden3IdentityFactory.loadIdentity("alias", "password", null)
                    assertNotEquals(identityLoaded, null)
                    identityLoaded?.stopIdentity()
                } else {
                    assert(false)
                }
            }
        } else {
            assert(false)
        }
    }

    @Test
    fun testRequestClaimSuccess() {
        val isInitialized = initialize()
        if (isInitialized) {
            val identity = iden3IdentityFactory.createIdentity("alias", "password", null)
            if (null != identity) {
                val ticket = identity.requestClaim(issuerUrl,"${Instant.now()}", null)
                assertNotEquals(ticket, null)
                identity.stopIdentity()
            } else {
                assert(false)
            }
        } else {
            assert(false)
        }
    }

    @Test
    fun testRequestClaimWithCallbackSuccess() {
        val isInitialized = initialize()
        if (isInitialized) {
            val identity = iden3IdentityFactory.createIdentity("alias", "password", null)
            if (null != identity) {
                var isFinished = false
                identity.requestClaim(issuerUrl,"${Instant.now()}", object : Iden3TicketCallback {
                    override fun onTicketReceived(ticket: Iden3Ticket) {
                        isFinished = true
                        Log.i("testRequestClaimWithCallbackSuccess", "Ticket: $ticket")
                        identity.stopIdentity()
                        assertNotEquals(null, ticket)
                    }

                    override fun onError(error: Exception) {
                        isFinished = true
                        Log.i("testRequestClaimWithCallbackSuccess", "Error: $error")
                        identity.stopIdentity()
                        assertEquals(null, error)
                    }
                }
                )
                // Wait for callback
                while (!isFinished){
                    Log.i("testRequestClaimWithCallbackSuccess","Waiting for request claim ticket")
                    Thread.sleep(1000)
                }
            } else {
                assert(false)
            }
        } else {
            assert(false)
        }
    }

    @Test
    fun testProveClaimSuccess() {
        val isInitialized = initialize()
        if (isInitialized) {
            val identity = iden3IdentityFactory.createIdentity("alias", "password", null)
            if (null != identity) {
                val ticket = identity.proveClaim(verifierUrl,"${Instant.now()}", false,null)
                assertNotEquals(ticket, null)
                identity.stopIdentity()
            } else {
                assert(false)
            }
        } else {
            assert(false)
        }
    }

    @Test
    fun testZKProveClaimSuccess() {
        val isInitialized = initialize()
        if (isInitialized) {
            val identity = iden3IdentityFactory.createIdentity("alias", "password", null)
            if (null != identity) {
                val ticket = identity.proveClaim(verifierUrl,"${Instant.now()}", true,null)
                assertNotEquals(ticket, null)
                identity.stopIdentity()
            } else {
                assert(false)
            }
        } else {
            assert(false)
        }
    }

    private fun initialize() : Boolean {
       return iden3IdentityFactory.initialize(web3Url, storePath, 10000)
    }
}
