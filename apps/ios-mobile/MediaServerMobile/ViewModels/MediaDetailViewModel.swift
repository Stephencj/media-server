import Foundation

@MainActor
class MediaDetailViewModel: ObservableObject {
    @Published var progress: WatchProgress?
    @Published var isLoading = false
    @Published var errorMessage: String?

    private let media: Media
    private let api = APIClient.shared

    init(media: Media) {
        self.media = media
    }

    func loadProgress() async {
        isLoading = true

        do {
            let fetchedProgress = try await api.getProgress(
                mediaId: media.id,
                type: media.type.rawValue
            )
            // Only show progress if there's actual progress
            if fetchedProgress.position > 0 {
                progress = fetchedProgress
            }
        } catch {
            // It's okay if there's no progress yet
        }

        isLoading = false
    }
}
