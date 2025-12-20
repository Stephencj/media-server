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

    private let keychainService = "com.mediaserver.app"
    private let tokenKey = "authToken"
    private let userKey = "currentUser"

    private init() {
        loadStoredAuth()
    }

    // MARK: - Authentication Methods

    func login(username: String, password: String) async throws {
        isLoading = true
        defer { isLoading = false }

        let request = LoginRequest(username: username, password: password)

        guard let url = URL(string: AppState.shared.serverURL + "/api/auth/login") else {
            throw APIError.invalidURL
        }

        var urlRequest = URLRequest(url: url)
        urlRequest.httpMethod = "POST"
        urlRequest.setValue("application/json", forHTTPHeaderField: "Content-Type")
        urlRequest.httpBody = try JSONEncoder().encode(request)

        let (data, response) = try await URLSession.shared.data(for: urlRequest)

        guard let httpResponse = response as? HTTPURLResponse else {
            throw APIError.networkError(URLError(.badServerResponse))
        }

        if httpResponse.statusCode == 401 {
            throw APIError.serverError(401, "Invalid username or password")
        }

        if httpResponse.statusCode >= 400 {
            let errorMessage = String(data: data, encoding: .utf8) ?? "Login failed"
            throw APIError.serverError(httpResponse.statusCode, errorMessage)
        }

        let authResponse = try JSONDecoder().decode(AuthResponse.self, from: data)
        handleAuthResponse(authResponse)
    }

    func register(username: String, email: String, password: String) async throws {
        isLoading = true
        defer { isLoading = false }

        let request = RegisterRequest(username: username, email: email, password: password)

        guard let url = URL(string: AppState.shared.serverURL + "/api/auth/register") else {
            throw APIError.invalidURL
        }

        var urlRequest = URLRequest(url: url)
        urlRequest.httpMethod = "POST"
        urlRequest.setValue("application/json", forHTTPHeaderField: "Content-Type")
        urlRequest.httpBody = try JSONEncoder().encode(request)

        let (data, response) = try await URLSession.shared.data(for: urlRequest)

        guard let httpResponse = response as? HTTPURLResponse else {
            throw APIError.networkError(URLError(.badServerResponse))
        }

        if httpResponse.statusCode >= 400 {
            let errorMessage = String(data: data, encoding: .utf8) ?? "Registration failed"
            throw APIError.serverError(httpResponse.statusCode, errorMessage)
        }

        let authResponse = try JSONDecoder().decode(AuthResponse.self, from: data)
        handleAuthResponse(authResponse)
    }

    func logout() {
        token = nil
        tokenExpiration = nil
        currentUser = nil
        isAuthenticated = false

        deleteFromKeychain(key: tokenKey)
        UserDefaults.standard.removeObject(forKey: userKey)
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

    // MARK: - Private Methods

    private func handleAuthResponse(_ response: AuthResponse) {
        token = response.token
        tokenExpiration = Date(timeIntervalSince1970: TimeInterval(response.expiresAt))
        currentUser = response.user
        isAuthenticated = true

        saveToKeychain(token: response.token)
        saveUser(response.user)
    }

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
