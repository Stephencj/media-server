import SwiftUI

enum ChannelViewMode: String, CaseIterable {
    case channels = "Channels"
    case guide = "Guide"
}

struct ChannelListView: View {
    @StateObject private var viewModel = ChannelViewModel()
    @State private var viewMode: ChannelViewMode = .guide
    @State private var showPlayer = false
    @State private var selectedIndex = 0

    private let columns = [
        GridItem(.adaptive(minimum: 160), spacing: 16)
    ]

    var body: some View {
        VStack(spacing: 0) {
            // Segmented control
            Picker("View Mode", selection: $viewMode) {
                ForEach(ChannelViewMode.allCases, id: \.self) { mode in
                    Text(mode.rawValue).tag(mode)
                }
            }
            .pickerStyle(.segmented)
            .padding(.horizontal, 16)
            .padding(.top, 12)

            if viewModel.isLoading && viewModel.channels.isEmpty {
                Spacer()
                ProgressView()
                Spacer()
            } else if viewModel.channels.isEmpty {
                Spacer()
                VStack(spacing: 12) {
                    Image(systemName: "tv.inset.filled")
                        .font(.system(size: 40))
                        .foregroundColor(.secondary)
                    Text("No Channels")
                        .font(.headline)
                    Text("Create channels in the web interface")
                        .font(.subheadline)
                        .foregroundColor(.secondary)
                }
                Spacer()
            } else {
                switch viewMode {
                case .channels:
                    ScrollView {
                        LazyVGrid(columns: columns, spacing: 16) {
                            ForEach(Array(viewModel.channels.enumerated()), id: \.element.id) { index, channel in
                                Button {
                                    selectedIndex = index
                                    showPlayer = true
                                } label: {
                                    VStack(spacing: 8) {
                                        Text(channel.icon)
                                            .font(.system(size: 32))
                                        Text(channel.name)
                                            .font(.subheadline)
                                            .fontWeight(.medium)
                                            .lineLimit(1)
                                    }
                                    .frame(maxWidth: .infinity)
                                    .padding(.vertical, 20)
                                    .background(Color(.systemGray6))
                                    .cornerRadius(12)
                                }
                                .buttonStyle(.plain)
                            }
                        }
                        .padding(16)
                    }

                case .guide:
                    ChannelGuideView(channels: viewModel.channels) { _, index in
                        selectedIndex = index
                        showPlayer = true
                    }
                }
            }
        }
        .navigationTitle("Channels")
        .fullScreenCover(isPresented: $showPlayer) {
            ChannelPlayerView(
                channels: viewModel.channels,
                startIndex: selectedIndex
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

#Preview {
    NavigationView {
        ChannelListView()
    }
    .environmentObject(AuthService.shared)
    .environmentObject(AppState.shared)
}
