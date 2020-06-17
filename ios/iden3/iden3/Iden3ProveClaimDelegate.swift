//
//  Iden3EventDelegate.swift
//  iden3
//
//  Created by Iden3 on 12/06/2020.
//

import Foundation

public protocol Iden3ProveClaimDelegate : NSObjectProtocol {
    
    func onVerifierResponse(verified: Bool)

    func onError(error: Error)
}
