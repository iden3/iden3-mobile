package com.iden3.iden3

import java.lang.Exception

interface Iden3TicketCallback {

    fun onTicketReceived(ticket: Iden3Ticket)

    fun onError(error: Exception)

}