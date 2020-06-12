//
//  Iden3EventDelegate.swift
//  iden3CoreSDK
//
//  Created by Iden3 on 12/06/2020.
//  Copyright © 2020 Iden3. All rights reserved.
//

import Foundation

public protocol Iden3TicketDelegate : NSObjectProtocol {
    
    func onTicketReceived(ticket: Iden3Ticket)

    func onError(error: Error)
}
