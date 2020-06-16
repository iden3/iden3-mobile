package com.iden3.iden3

class Iden3Credential(private var key: String, private var credential: String) {

    fun getKey() : String {
        return key
    }

    fun getCredential() : String {
        return credential
    }
}