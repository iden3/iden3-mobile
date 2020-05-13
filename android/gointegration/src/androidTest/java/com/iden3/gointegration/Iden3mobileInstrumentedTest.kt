package com.iden3.gointegration

import android.content.Context
import androidx.test.ext.junit.runners.AndroidJUnit4
import androidx.test.platform.app.InstrumentationRegistry
import iden3mobile.Iden3mobile
import org.hamcrest.core.StringContains
import org.junit.Before
import org.junit.Rule
import org.junit.Test
import org.junit.rules.ExpectedException
import org.junit.runner.RunWith
import java.io.File


/**
 * Iden3mobile local unit test, which will execute on the development machine (host).
 *
 * See [testing documentation](http://d.android.com/tools/testing).
 */

@RunWith(AndroidJUnit4::class)
class Iden3mobileInstrumentedTest {

    private val web3Url = BuildConfig.INFURA_URL
    private val issuerUrl = "http://167.172.104.160:6100/api/unstable"
    private val verifierUrl = "http://167.172.104.160:6200/api/unstable"
    private lateinit var instrumentationCtx: Context
    private lateinit var storePath: String

    @Rule
    @JvmField
    val expectedException: ExpectedException = ExpectedException.none()

    @Before
    fun setup() {
        instrumentationCtx = InstrumentationRegistry.getInstrumentation().targetContext
        storePath = instrumentationCtx.filesDir.absolutePath
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
            "password",
            web3Url,
            1000,
            null
        ) { event -> print(event) }

        identity.stop()

        Iden3mobile.newIdentityLoad(
            "$storePath/alias",
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
            "password",
            web3Url,
            1000,
            null
        ) { event -> print(event) }

        Iden3mobile.newIdentityLoad(
            "$storePath/alias",
            "wrongPassword",
            web3Url,
            1000
        ) { event -> print(event) }
    }

    @Test
    fun testLoadIdentityErrorWrongPassword() {
        expectedException.expect(Exception::class.java)
        expectedException.expectMessage(StringContains("Error unlocking babyjub key from keystore: Invalid encrypted data"))
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

        identity.stop()

        Iden3mobile.newIdentityLoad(
            "$storePath/alias",
            "wrongPassword",
            web3Url,
            1000
        ) { event -> print(event) }
    }

}
