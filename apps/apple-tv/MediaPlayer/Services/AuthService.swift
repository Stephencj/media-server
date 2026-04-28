import Foundation
import Security

@MainActor
class AuthService: ObservableObject {
    static let shared = AuthService()

    @Published var isAuthenticated = false
    @Published var currentUser: User?
    @Published var isLoading = false
    @Published var lastAuthError: String?

    private(set) var token: String?
    private var tokenExpiration: Date?

    private let keychainService = "com.mediaserver.app"
    private let tokenKey = "authToken"
    private let userKey = "currentUser"
    private let tokenExpirationKey = "tokenExpiration"

    private init() {
        loadStoredAuth()
    }

    func logout() {
        token = nil
        tokenExpiration = nil
        currentUser = nil
        isAuthenticated = false

        deleteFromKeychain(key: tokenKey)
        UserDefaults.standard.removeObject(forKey: userKey)
        UserDefaults.standard.removeObject(forKey: tokenExpirationKey)
    }

    /// Silently exchanges the embedded device key for a JWT, retrying on
    /// transient network errors. Surfaces the failure cause via
    /// `lastAuthError` so the connect-failure UI can show what happened.
    func ensureAuthenticated() async {
        if isAuthenticated, token != nil, let expiration = tokenExpiration, Date() < expiration {
            return
        }

        isLoading = true
        lastAuthError = nil
        defer { isLoading = false }

        let backoffs: [UInt64] = [500_000_000, 2_000_000_000, 5_000_000_000]
        for (attempt, delay) in backoffs.enumerated() {
            do {
                try await exchangeDeviceKey()
                return
            } catch APIError.serverError(let status, let message) where status == 401 {
                lastAuthError = "Invalid device key — rebuild the app with a valid Secrets.swift."
                print("[Auth] device-key rejected: \(message)")
                return
            } catch {
                let isLast = attempt == backoffs.count - 1
                if isLast {
                    lastAuthError = "Cannot reach \(Secrets.serverURL): \(error.localizedDescription)"
                    print("[Auth] exchange failed after \(backoffs.count) attempts: \(error)")
                    return
                }
                try? await Task.sleep(nanoseconds: delay)
            }
        }
    }

    private func exchangeDeviceKey() async throws {
        guard let url = URL(string: Secrets.serverURL + "/api/auth/exchange") else {
            throw APIError.invalidURL
        }

        var urlRequest = URLRequest(url: url)
        urlRequest.httpMethod = "POST"
        urlRequest.setValue("application/json", forHTTPHeaderField: "Content-Type")
        let body: [String: String] = ["key": Secrets.apiKey]
        urlRequest.httpBody = try JSONEncoder().encode(body)

        let (data, response) = try await URLSession.shared.data(for: urlRequest)

        guard let httpResponse = response as? HTTPURLResponse else {
            throw APIError.networkError(URLError(.badServerResponse))
        }

        if httpResponse.statusCode == 401 {
            throw APIError.serverError(401, "Invalid device key")
        }

        if httpResponse.statusCode >= 400 {
            let errorMessage = String(data: data, encoding: .utf8) ?? "Exchange failed"
            throw APIError.serverError(httpResponse.statusCode, errorMessage)
        }

        let authResponse = try JSONDecoder().decode(AuthResponse.self, from: data)
        handleAuthResponse(authResponse)
    }

    private func handleAuthResponse(_ response: AuthResponse) {
        token = response.token
        tokenExpiration = Date(timeIntervalSince1970: TimeInterval(response.expiresAt))
        currentUser = response.user
        isAuthenticated = true

        saveToKeychain(token: response.token)
        saveUser(response.user)
        UserDefaults.standard.set(response.expiresAt, forKey: tokenExpirationKey)
    }

    private func loadStoredAuth() {
        let storedToken = loadFromKeychain(key: tokenKey)
        let storedExpiration = UserDefaults.standard.integer(forKey: tokenExpirationKey)
        let expiration = storedExpiration > 0 ? Date(timeIntervalSince1970: TimeInterval(storedExpiration)) : nil

        if let storedToken, let expiration, Date() < expiration {
            token = storedToken
            tokenExpiration = expiration
            isAuthenticated = true
        } else {
            token = storedToken
            tokenExpiration = expiration
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
