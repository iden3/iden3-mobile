package com.iden3.gointegration

import android.content.Context
import androidx.test.core.app.ApplicationProvider
import androidx.test.ext.junit.runners.AndroidJUnit4
import androidx.test.platform.app.InstrumentationRegistry
import com.iden3.iden3coreapi.Iden3CoreAPI
import iden3mobile.Iden3mobile
import org.hamcrest.core.StringContains
import org.junit.Assert.assertEquals
import org.junit.Assert.assertTrue
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
            assertTrue(identity != null)
        } else {
            assert(false)
        }
    }

    @Test
    fun testLoadIdentitySuccess() {
        val isInitialized = initializeAPI()
        if (isInitialized) {
            if (File("$storePath/alias").exists()) {
                val identityLoaded = iden3CoreAPI.createIdentity("alias", "password")
                assertTrue(identityLoaded != null)
            } else {
                val identityCreated = iden3CoreAPI.createIdentity("alias", "password")
                if (File("$storePath/alias").exists()) {
                    val identityLoaded = iden3CoreAPI.createIdentity("alias", "password")
                    assertTrue(identityLoaded != null)
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
            if (identity != null) {
                val ticket = iden3CoreAPI.requestClaim(identity,"${Instant.now()}", null)
                assertTrue(ticket != null)
            } else {
                assert(false)
            }
        } else {
            assert(false)
        }
    }

    @Test
    fun testRequestClaimErrorPermissionDenied() {
        expectedException.expect(Exception::class.java)
        expectedException.expectMessage(StringContains("permission denied"))
        val isInitialized = initializeAPI()
        if (isInitialized) {
            val identity = iden3CoreAPI.createIdentity("alias", "password")
            if (identity != null) {
                val ticket = iden3CoreAPI.requestClaim(identity,"${Instant.now()}", null)
                assertTrue(ticket != null)
            } else {
                assert(false)
            }
        } else {
            assert(false)
        }
    }

    private fun initializeAPI() : Boolean {
       return iden3CoreAPI.initializeAPI(web3Url, issuerUrl, verifierUrl, storePath, 10000)
    }
}
