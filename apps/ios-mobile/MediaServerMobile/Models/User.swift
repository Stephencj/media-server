import Foundation

struct User: Codable {
    let id: Int64
    let username: String
    let email: String
}

struct AuthResponse: Codable {
    let token: String
    let expiresAt: Int64
    let user: User

    enum CodingKeys: String, CodingKey {
        case token, user
        case expiresAt = "expires_at"
    }
}

struct AuthMessage: Codable {
    let token: String
    let expiresAt: Int64
    let user: User
}
