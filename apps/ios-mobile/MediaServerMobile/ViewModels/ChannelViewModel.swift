import Foundation

@MainActor
class ChannelViewModel: ObservableObject {
    @Published var channels: [Channel] = []
    @Published var isLoading = false
    @Published var errorMessage: String?

    private let api = APIClient.shared

    func loadChannels() async {
        isLoading = true
        errorMessage = nil

        do {
            let response = try await api.getChannels()
            channels = response.items
        } catch {
            errorMessage = error.localizedDescription
        }

        isLoading = false
    }
}
