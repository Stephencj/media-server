import SwiftUI

enum ChannelViewMode: String, CaseIterable {
    case channels = "Channels"
    case guide = "Guide"
}

struct ChannelListView: View {
    @StateObject private var viewModel = ChannelViewModel()
    @State private var viewMode: ChannelViewMode = .channels
    @State private var selectedChannelIndex: Int?

    private let columns = [
        GridItem(.adaptive(minimum: 300), spacing: 40)
    ]

    var body: some View {
        NavigationStack {
            VStack(spacing: 0) {
                // Segmented control
                Picker("View Mode", selection: $viewMode) {
                    ForEach(ChannelViewMode.allCases, id: \.self) { mode in
                        Text(mode.rawValue).tag(mode)
                    }
                }
                .pickerStyle(.segmented)
                .padding(.horizontal, 40)
                .padding(.top, 20)

                if viewModel.isLoading && viewModel.channels.isEmpty {
                    Spacer()
                    ProgressView()
                    Spacer()
                } else if viewModel.channels.isEmpty {
                    Spacer()
                    ContentUnavailableView(
                        "No Channels",
                        systemImage: "tv.inset.filled",
                        description: Text("Create channels in the web interface to watch here")
                    )
                    Spacer()
                } else {
                    switch viewMode {
                    case .channels:
                        ScrollView {
                            LazyVGrid(columns: columns, spacing: 40) {
                                ForEach(Array(viewModel.channels.enumerated()), id: \.element.id) { index, channel in
                                    Button {
                                        selectedChannelIndex = index
                                    } label: {
                                        ChannelCardView(channel: channel)
                                    }
                                    .buttonStyle(.card)
                                }
                            }
                            .padding(40)
                        }

                    case .guide:
                        ChannelGuideView(channels: viewModel.channels) { _, index in
                            selectedChannelIndex = index
                        }
                    }
                }
            }
            .navigationTitle("Channels")
            .fullScreenCover(item: $selectedChannelIndex) { index in
                ChannelPlayerView(
                    channels: viewModel.channels,
                    startIndex: index
                )
            }
            .task {
                await viewModel.loadChannels()
            }
            .refreshable {
                await viewModel.loadChannels()
            }
        }
    }
}

extension Int: @retroactive Identifiable {
    public var id: Int { self }
}

#Preview {
    ChannelListView()
        .environmentObject(AuthService.shared)
        .environmentObject(AppState.shared)
}
