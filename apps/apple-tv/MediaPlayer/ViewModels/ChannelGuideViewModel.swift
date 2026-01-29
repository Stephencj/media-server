import Foundation

struct ChannelGuideEntry: Identifiable {
    let id: Int64
    let channel: Channel
    var nowPlaying: ChannelScheduleItem?
    var upNext: [ChannelScheduleItem]
    var elapsed: Int
}

@MainActor
class ChannelGuideViewModel: ObservableObject {
    @Published var entries: [ChannelGuideEntry] = []
    @Published var isLoading = false
    @Published var errorMessage: String?

    private let api = APIClient.shared
    private var refreshTask: Task<Void, Never>?

    func loadGuide() async {
        isLoading = true
        errorMessage = nil

        do {
            // First load all channels
            let channelsResponse = try await api.getChannels()
            let channels = channelsResponse.items

            // Then fetch now playing for each channel in parallel
            await withTaskGroup(of: ChannelGuideEntry?.self) { group in
                for channel in channels {
                    group.addTask {
                        do {
                            let nowResponse = try await self.api.getChannelNowPlaying(channelId: channel.id)
                            return ChannelGuideEntry(
                                id: channel.id,
                                channel: channel,
                                nowPlaying: nowResponse.nowPlaying,
                                upNext: nowResponse.upNext,
                                elapsed: nowResponse.elapsed
                            )
                        } catch {
                            // Return entry with no content on error
                            return ChannelGuideEntry(
                                id: channel.id,
                                channel: channel,
                                nowPlaying: nil,
                                upNext: [],
                                elapsed: 0
                            )
                        }
                    }
                }

                var loadedEntries: [ChannelGuideEntry] = []
                for await entry in group {
                    if let entry = entry {
                        loadedEntries.append(entry)
                    }
                }

                // Sort entries to maintain channel order
                entries = loadedEntries.sorted { $0.channel.id < $1.channel.id }
            }
        } catch {
            errorMessage = error.localizedDescription
        }

        isLoading = false
    }

    func startAutoRefresh() {
        refreshTask?.cancel()
        refreshTask = Task {
            while !Task.isCancelled {
                try? await Task.sleep(nanoseconds: 60_000_000_000) // 60 seconds
                if !Task.isCancelled {
                    await loadGuide()
                }
            }
        }
    }

    func stopAutoRefresh() {
        refreshTask?.cancel()
        refreshTask = nil
    }
}
