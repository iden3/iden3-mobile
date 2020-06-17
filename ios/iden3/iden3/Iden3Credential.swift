//
//  Iden3Credential.swift
//  iden3
//
//  Created by Iden3 on 16/06/2020.
//

import Foundation

public final class Iden3Credential {
    
    // MARK: - Properties
    
    private var key: String?
    private var credential: String?
    
    // MARK: - Initializers
       
    init(key: String?, credential: String?) {
        self.key = key
        self.credential = credential
    }
    
    public func getKey() -> String? {
        return self.key
    }

    public func getCredential() -> String? {
        return self.credential
    }
}
