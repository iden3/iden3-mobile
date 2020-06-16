package com.iden3.iden3

import iden3mobile.*

class Iden3Ticket(private var ticket: Ticket) {

    fun getId() : String {
        return ticket.id
    }

    fun getLastChecked() : Long {
        return ticket.lastChecked
    }

    fun getType() : String {
        return ticket.type
    }

    fun getStatus() : String {
        return ticket.status
    }

}