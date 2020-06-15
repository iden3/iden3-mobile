//
//  Iden3Error.swift
//  iden3CoreSDK
//
//  Created by Iden3 on 12/06/2020.
//  Copyright © 2020 Iden3. All rights reserved.
//

import Foundation

enum Iden3Error: Error {
    case GenericError(String)
    case IllegalArgumentError(String)
    case IllegalStateError(String)
    case FileNotFoundError(String)
}
