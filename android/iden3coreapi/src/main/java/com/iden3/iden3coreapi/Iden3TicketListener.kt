package com.iden3.iden3coreapi

import java.lang.Exception

interface Iden3TicketListener {

    fun onTicketReceived(ticket: Iden3Ticket)

    fun onError(error: Exception)

}