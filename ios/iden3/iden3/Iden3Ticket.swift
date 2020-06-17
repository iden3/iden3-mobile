//
//  Iden3Identity.swift
//  iden3
//
//  Created by Iden3 on 12/06/2020.
//

import Foundation
import Iden3mobile

public final class Iden3Ticket {
    
    // MARK: - Properties
    
    private var ticket: Iden3mobileTicket?
    
    // MARK: - Initializers
       
    init(_ ticket: Iden3mobileTicket?) {
        self.ticket = ticket
    }
    
    public func getId() -> String? {
        return ticket?.id_
    }

    public func getLastChecked() -> Int64? {
        return ticket?.lastChecked
    }

    public func getType() -> String? {
        return ticket?.type
    }

    public func getStatus() -> String? {
        return ticket?.status
    }
}
