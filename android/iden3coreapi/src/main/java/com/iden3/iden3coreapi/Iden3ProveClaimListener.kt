package com.iden3.iden3coreapi

import java.lang.Exception

interface Iden3ProveClaimListener {

    fun onClaimProofReceived(proved: Boolean)

    fun onError(error: Exception)

}