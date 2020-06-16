package com.iden3.iden3

import iden3mobile.*
import java.lang.Exception

class Iden3Event(private var event: Event) {

    fun getTicketId() : String {
        return event.ticketId
    }

    fun getData() : String {
        return event.data
    }

    fun getType() : String {
        return event.type
    }

    fun getError() : Exception {
        return event.err
    }

}