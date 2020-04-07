package com.iden3.iden3coreapi

import android.content.Context
import org.junit.Test
import org.junit.Assert.*
import androidx.test.core.app.ApplicationProvider

/**
 * Example local unit test, which will execute on the development machine (host).
 *
 * See [testing documentation](http://d.android.com/tools/testing).
 */
class ExampleUnitTest {
    //val context = ApplicationProvider.getApplicationContext<Context>()

    @Test
    fun addition_isCorrect() {
        assertEquals(4, 2 + 2)
    }

    @Test
    fun testIden3CoreAPI() {
        val iden3CoreAPI = Iden3CoreAPI()
        assertTrue(iden3CoreAPI != null)
    }
}
