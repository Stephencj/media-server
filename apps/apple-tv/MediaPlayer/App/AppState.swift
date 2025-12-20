import SwiftUI

@MainActor
class AppState: ObservableObject {
    static let shared = AppState()

    @Published var serverURL: String {
        didSet {
            UserDefaults.standard.set(serverURL, forKey: "serverURL")
        }
    }

    @Published var isLoading = false
    @Published var errorMessage: String?

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
}
