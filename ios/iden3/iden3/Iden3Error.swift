//
//  Iden3Error.swift
//  iden3
//
//  Created by Iden3 on 12/06/2020.
//

import Foundation

enum Iden3Error: Error {
    case GenericError(String)
    case IllegalArgumentError(String)
    case IllegalStateError(String)
    case FileNotFoundError(String)
}
