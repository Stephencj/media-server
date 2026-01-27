import Foundation
import Security

@MainActor
class AuthService: ObservableObject {
    static let shared = AuthService()

    @Published var isAuthenticated = false
    @Published var currentUser: User?
    @Published var isLoading = false

    private(set) var token: String?
    private var tokenExpiration: Date?

    private let keychainService = "com.mediaserver.mobile"
    private let tokenKey = "authToken"
    private let userKey = "currentUser"

    private init() {
        loadStoredAuth()
    }

    // MARK: - Authentication Methods

    func handleAuthFromWeb(_ authMessage: AuthMessage) {
        token = authMessage.token
        tokenExpiration = Date(timeIntervalSince1970: TimeInterval(authMessage.expiresAt))
        currentUser = authMessage.user
        isAuthenticated = true

        saveToKeychain(token: authMessage.token)
        saveUser(authMessage.user)

        // Notify WebView to set token
        NotificationCenter.default.post(
            name: .authTokenChanged,
            object: nil,
            userInfo: ["token": authMessage.token]
        )
    }

    func logout() {
        token = nil
        tokenExpiration = nil
        currentUser = nil
        isAuthenticated = false

        deleteFromKeychain(key: tokenKey)
        UserDefaults.standard.removeObject(forKey: userKey)

        // Notify WebView
        NotificationCenter.default.post(name: .authTokenChanged, object: nil)
    }

    func checkAuthState() async {
        // Token already loaded in init
        if let expiration = tokenExpiration {
            if Date() > expiration {
                // Token expired
                logout()
            }
        }
    }

    func getAuthToken() -> String? {
        return token
    }

    // MARK: - Private Methods

    private func loadStoredAuth() {
        if let storedToken = loadFromKeychain(key: tokenKey) {
            token = storedToken
            isAuthenticated = true
        }

        if let userData = UserDefaults.standard.data(forKey: userKey),
           let user = try? JSONDecoder().decode(User.self, from: userData) {
            currentUser = user
        }
    }

    private func saveUser(_ user: User) {
        if let data = try? JSONEncoder().encode(user) {
            UserDefaults.standard.set(data, forKey: userKey)
        }
    }

    // MARK: - Keychain Methods

    private func saveToKeychain(token: String) {
        guard let data = token.data(using: .utf8) else { return }

        let query: [String: Any] = [
            kSecClass as String: kSecClassGenericPassword,
            kSecAttrService as String: keychainService,
            kSecAttrAccount as String: tokenKey,
            kSecValueData as String: data
        ]

        SecItemDelete(query as CFDictionary)
        SecItemAdd(query as CFDictionary, nil)
    }

    private func loadFromKeychain(key: String) -> String? {
        let query: [String: Any] = [
            kSecClass as String: kSecClassGenericPassword,
            kSecAttrService as String: keychainService,
            kSecAttrAccount as String: key,
            kSecReturnData as String: true,
            kSecMatchLimit as String: kSecMatchLimitOne
        ]

        var result: AnyObject?
        let status = SecItemCopyMatching(query as CFDictionary, &result)

        if status == errSecSuccess, let data = result as? Data {
            return String(data: data, encoding: .utf8)
        }
        return nil
    }

    private func deleteFromKeychain(key: String) {
        let query: [String: Any] = [
            kSecClass as String: kSecClassGenericPassword,
            kSecAttrService as String: keychainService,
            kSecAttrAccount as String: key
        ]
        SecItemDelete(query as CFDictionary)
    }
}
