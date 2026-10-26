import SwiftUI

struct AppError: Identifiable {
    let id = UUID()
    let title: String
    let message: String
}

@MainActor
class AppState: ObservableObject {
    static let shared = AppState()

    @Published var serverURL: String = Secrets.serverURL

    @Published var isLoading = false
    @Published var errorMessage: String?
    @Published var errors: [AppError] = []

    @Published var preferredAudioLanguage: String {
        didSet { UserDefaults.standard.set(preferredAudioLanguage, forKey: "preferredAudioLanguage") }
    }

    private init() {
        self.preferredAudioLanguage = UserDefaults.standard.string(forKey: "preferredAudioLanguage") ?? "en"
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
