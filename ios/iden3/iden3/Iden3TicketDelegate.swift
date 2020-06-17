//
//  Iden3EventDelegate.swift
//  iden3
//
//  Created by Iden3 on 12/06/2020.
//

import Foundation

public protocol Iden3TicketDelegate : NSObjectProtocol {
    
    func onTicketReceived(ticket: Iden3Ticket)

    func onError(error: Error)
}
