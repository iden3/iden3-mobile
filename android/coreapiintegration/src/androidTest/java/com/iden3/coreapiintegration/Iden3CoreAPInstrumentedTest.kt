package com.iden3.coreapiintegration

import android.content.Context
import android.util.Log
import androidx.test.ext.junit.runners.AndroidJUnit4
import androidx.test.platform.app.InstrumentationRegistry
import com.iden3.coreapiintegration.test.BuildConfig
import com.iden3.iden3coreapi.Iden3CoreAPI
import iden3mobile.CallbackRequestClaim
import org.junit.Assert.*
import org.junit.Before
import org.junit.Rule
import org.junit.Test
import org.junit.rules.ExpectedException
import org.junit.runner.RunWith
import java.io.File
import java.time.Instant

/**
 * Iden3CoreAPI local unit test, which will execute on the development machine (host).
 *
 * See [testing documentation](http://d.android.com/tools/testing).
 */

@RunWith(AndroidJUnit4::class)
class Iden3CoreAPInstrumentedTest {

    private val web3Url = BuildConfig.INFURA_URL
    private val issuerUrl = "http://167.172.104.160:6100/api/unstable"
    private val verifierUrl = "http://167.172.104.160:6200/api/unstable"
    private lateinit var instrumentationCtx: Context
    private lateinit var storePath: String
    private lateinit var iden3CoreAPI: Iden3CoreAPI

    @Rule
    @JvmField
    val expectedException: ExpectedException = ExpectedException.none()

    @Before
    fun setup() {
        instrumentationCtx = InstrumentationRegistry.getInstrumentation().targetContext
        storePath = instrumentationCtx.filesDir.absolutePath
        iden3CoreAPI = Iden3CoreAPI()
    }

    @Test
    fun testInitializeAPISuccess() {
        assertEquals(true, initializeAPI())
    }

    @Test
    fun testCreateIdentitySuccess() {
        val isInitialized = initializeAPI()
        if (isInitialized) {
            val identity = iden3CoreAPI.createIdentity("alias", "password")
            assertNotEquals(identity, null)
            iden3CoreAPI.stopIdentity(identity!!)
        } else {
            assert(false)
        }
    }

    @Test
    fun testLoadIdentitySuccess() {
        val isInitialized = initializeAPI()
        if (isInitialized) {
            if (File("$storePath/alias").exists()) {
                val identityLoaded = iden3CoreAPI.loadIdentity("alias", "password")
                assertNotEquals(identityLoaded, null)
                iden3CoreAPI.stopIdentity(identityLoaded!!)
            } else {
                val identityCreated = iden3CoreAPI.createIdentity("alias", "password")
                iden3CoreAPI.stopIdentity(identityCreated!!)
                if (File("$storePath/alias").exists()) {
                    val identityLoaded = iden3CoreAPI.loadIdentity("alias", "password")
                    assertNotEquals(identityLoaded, null)
                    iden3CoreAPI.stopIdentity(identityLoaded!!)
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
        val isInitialized = initializeAPI()
        if (isInitialized) {
            val identity = iden3CoreAPI.createIdentity("alias", "password")
            if (null != identity) {
                val ticket = iden3CoreAPI.requestClaim(identity,"${Instant.now()}", null)
                assertNotEquals(ticket, null)
                iden3CoreAPI.stopIdentity(identity)
            } else {
                assert(false)
            }
        } else {
            assert(false)
        }
    }

    @Test
    fun testRequestClaimWithCallbackSuccess() {
        val isInitialized = initializeAPI()
        if (isInitialized) {
            val identity = iden3CoreAPI.createIdentity("alias", "password")
            if (null != identity) {
                var isFinished = false
                iden3CoreAPI.requestClaim(identity,"${Instant.now()}",
                    CallbackRequestClaim { ticket, exception ->
                        isFinished = true
                        Log.i("testRequestClaimWithCallbackSuccess", "Ticket: $ticket")
                        iden3CoreAPI.stopIdentity(identity)
                        assertNotEquals(null, ticket)
                        assertEquals(null, exception)
                    })
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

    /*@Test
    fun testProveClaimSuccess() {
        val isInitialized = initializeAPI()
        if (isInitialized) {
            val identity = iden3CoreAPI.createIdentity("alias", "password")
            if (null != identity) {
                val ticket = iden3CoreAPI.proveClaim(identity,"${Instant.now()}", null)
                assertNotEquals(ticket, null)
                iden3CoreAPI.stopIdentity(identity)
            } else {
                assert(false)
            }
        } else {
            assert(false)
        }
    }*/

    private fun initializeAPI() : Boolean {
       return iden3CoreAPI.initializeAPI(web3Url, issuerUrl, verifierUrl, storePath, 10000)
    }
}
