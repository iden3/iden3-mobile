package com.iden3.iden3

import java.lang.Exception

interface Iden3ProveClaimCallback {

    fun onVerifierResponse(verified: Boolean)

    fun onError(error: Exception)

}