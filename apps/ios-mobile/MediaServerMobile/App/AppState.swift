import SwiftUI
import Combine

struct AppError: Identifiable {
    let id = UUID()
    let title: String
    let message: String
}

@MainActor
class AppState: ObservableObject {
    static let shared = AppState()

    @Published var serverURL: String {
        didSet {
            UserDefaults.standard.set(serverURL, forKey: "serverURL")
            // Notify WebView to reload
            NotificationCenter.default.post(name: .serverURLChanged, object: nil)
        }
    }

    @Published var isLoading = false
    @Published var errorMessage: String?
    @Published var errors: [AppError] = []

    // WebView state
    @Published var canGoBack = false
    @Published var canGoForward = false
    @Published var isWebViewLoading = false

    private init() {
        self.serverURL = UserDefaults.standard.string(forKey: "serverURL") ?? ""
    }

    func showError(_ message: String) {
        errorMessage = message

        // Auto-dismiss after 3 seconds
        Task {
            try? await Task.sleep(nanoseconds: 3_000_000_000)
            errorMessage = nil
        }
    }

    func showError(_ title: String, message: String) {
        let error = AppError(title: title, message: message)
        errors.append(error)

        // Auto-dismiss after 5 seconds
        Task {
            try? await Task.sleep(nanoseconds: 5_000_000_000)
            await MainActor.run {
                errors.removeAll { $0.id == error.id }
            }
        }
    }
}

extension Notification.Name {
    static let serverURLChanged = Notification.Name("serverURLChanged")
    static let authTokenChanged = Notification.Name("authTokenChanged")
}
