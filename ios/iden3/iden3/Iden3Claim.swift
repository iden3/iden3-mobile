//
//  Iden3Claim.swift
//  iden3
//
//  Created by iden3 on 16/06/2020.
//

import Foundation

public final class Iden3Claim {
    
    // MARK: - Properties
    
    private var key: String?
    private var claim: String?
    
    // MARK: - Initializers
       
    init(key: String?, claim: String?) {
        self.key = key
        self.claim = claim
    }
    
    public func getKey() -> String? {
        return self.key
    }

    public func getClaim() -> String? {
        return self.claim
    }
}
