package com.iden3.iden3

class Iden3Claim(private var key: String, private var claim: String) {

    fun getKey() : String {
        return key
    }

    fun getClaim() : String {
        return claim
    }
}