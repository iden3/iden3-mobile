package com.iden3.iden3coreapi

class Iden3CoreAPI {
    companion object {
        @JvmStatic lateinit var instance: Iden3CoreAPI
    }

    init {
        instance = this
    }
}