import Foundation
import Combine

@MainActor
class HomeViewModel: ObservableObject {
    @Published var sections: [LibrarySection] = []
    @Published var sectionMedia: [String: [Media]] = [:]  // slug -> media
    @Published var continueWatching: [ContinueWatchingItem] = []
    @Published var isLoading = false
    @Published var error: Error?

    private let apiClient: APIClient

    init(apiClient: APIClient = .shared) {
        self.apiClient = apiClient
    }

    func loadData() async {
        isLoading = true
        defer { isLoading = false }

        do {
            // Load sections first
            sections = try await apiClient.getSections()

            // Load continue watching
            let continueWatchingResponse = try await apiClient.getContinueWatching()
            continueWatching = continueWatchingResponse.items

            // Load media for each visible section (first 20 items)
            for section in sections where section.isVisible {
                let media = try await apiClient.getSectionMedia(slug: section.slug, limit: 20)
                sectionMedia[section.slug] = media
            }
        } catch {
            self.error = error
            print("Error loading home data: \(error)")
        }
    }

    func loadMoreMedia(for section: LibrarySection) async {
        guard let existing = sectionMedia[section.slug] else { return }

        do {
            let newMedia = try await apiClient.getSectionMedia(
                slug: section.slug,
                limit: 20,
                offset: existing.count
            )

            sectionMedia[section.slug] = existing + newMedia
        } catch {
            print("Error loading more media for section \(section.name): \(error)")
        }
    }
}
