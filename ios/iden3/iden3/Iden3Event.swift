//
//  Iden3Identity.swift
//  iden3
//
//  Created by Iden3 on 12/06/2020.
//

import Foundation
import Iden3mobile

public final class Iden3Event {
    
    // MARK: - Properties
    
    private var event: Iden3mobileEvent
    
    // MARK: - Initializers
       
    init(event: Iden3mobileEvent) {
        self.event = event
    }
    
    public func getTicketId() -> String {
        return event.ticketId
    }

    public func getData() -> String {
        return event.data
    }

    public func getType() -> String {
        return event.type
    }

    public func getError() -> Error? {
        return event.err
    }
}
