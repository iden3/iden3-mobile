//
//  Iden3EventDelegate.swift
//  iden3
//
//  Created by Iden3 on 12/06/2020.
//

import Foundation

public protocol Iden3EventDelegate : NSObjectProtocol {

    func onEventReceived(event: Iden3Event)
}
