# Extras Feature - Apple TV Implementation

## Overview

The Extras feature allows users to browse and play bonus content (commentaries, deleted scenes, featurettes, interviews, etc.) associated with movies and TV shows.

---

## API Endpoints

| Method | Endpoint | Description |
|--------|----------|-------------|
| GET | `/api/extras` | Get all extras (paginated) |
| GET | `/api/extras?limit=50&offset=0` | Pagination parameters |
| GET | `/api/extras/categories` | Get categories with counts |
| GET | `/api/extras/category/:category` | Get extras by category |
| GET | `/api/extras/:id` | Get single extra details |
| GET | `/api/media/:id/extras` | Get movie extras |
| GET | `/api/shows/:showId/extras` | Get TV show extras |
| GET | `/api/episodes/:episodeId/extras` | Get episode extras |
| GET | `/api/stream/:id/manifest.m3u8?type=extra` | Stream extra (HLS) |
| GET | `/api/stream/:id/direct?type=extra` | Direct play extra |

---

## Models

### Extra
```swift
struct Extra: Codable, Identifiable {
    let id: Int64
    let title: String
    let category: ExtraCategory
    let movieId: Int64?       // Optional - set if linked to movie
    let tvShowId: Int64?      // Optional - set if linked to TV show
    let episodeId: Int64?     // Optional - set if linked to episode
    let seasonNumber: Int?
    let episodeNumber: Int?
    let duration: Int         // Seconds
    let resolution: String?
    let videoCodec: String?
    let audioCodec: String?
    let createdAt: Date
}
```

### ExtraCategory Enum
```swift
enum ExtraCategory: String, Codable, CaseIterable {
    case commentary = "commentary"
    case deletedScene = "deleted_scene"
    case featurette = "featurette"
    case interview = "interview"
    case gagReel = "gag_reel"
    case musicVideo = "music_video"
    case behindTheScenes = "behind_the_scenes"
    case other = "other"

    var displayName: String {
        switch self {
        case .commentary: return "Commentaries"
        case .deletedScene: return "Deleted Scenes"
        case .featurette: return "Featurettes"
        case .interview: return "Interviews"
        case .gagReel: return "Gag Reels"
        case .musicVideo: return "Music Videos"
        case .behindTheScenes: return "Behind the Scenes"
        case .other: return "Other"
        }
    }

    var icon: String {
        switch self {
        case .commentary: return "mic.fill"           // SF Symbol
        case .deletedScene: return "scissors"
        case .featurette: return "film"
        case .interview: return "person.2.fill"
        case .gagReel: return "face.smiling.fill"
        case .musicVideo: return "music.note"
        case .behindTheScenes: return "video.fill"
        case .other: return "doc.fill"
        }
    }

    var color: Color {
        switch self {
        case .commentary: return .orange
        case .deletedScene: return .red
        case .featurette: return .blue
        case .interview: return .green
        case .gagReel: return .yellow
        case .musicVideo: return .pink
        case .behindTheScenes: return .purple
        case .other: return .gray
        }
    }
}
```

### CategoryCount
```swift
struct CategoryCount: Codable {
    let category: ExtraCategory
    let count: Int
}
```

---

## UI Components

### 1. ExtrasGridView
Main browsable library with category filter chips.

```swift
struct ExtrasGridView: View {
    @StateObject var viewModel = ExtrasViewModel()
    @FocusState private var focusedCategory: String?

    var body: some View {
        VStack(alignment: .leading) {
            // Filter chips (horizontal scroll)
            ScrollView(.horizontal, showsIndicators: false) {
                HStack(spacing: 12) {
                    FilterChip(label: "All", isActive: viewModel.selectedCategory == nil) {
                        viewModel.selectCategory(nil)
                    }
                    ForEach(viewModel.categories, id: \.category) { cat in
                        FilterChip(
                            label: cat.category.displayName,
                            icon: cat.category.icon,
                            isActive: viewModel.selectedCategory == cat.category
                        ) {
                            viewModel.selectCategory(cat.category)
                        }
                    }
                }
                .padding(.horizontal)
            }

            // Extras grid
            LazyVGrid(columns: gridColumns, spacing: 20) {
                ForEach(viewModel.extras) { extra in
                    ExtraCard(extra: extra)
                }
            }
        }
    }
}
```

### 2. FilterChip
Category filter chip with focus support.

```swift
struct FilterChip: View {
    let label: String
    var icon: String? = nil
    let isActive: Bool
    let action: () -> Void

    @FocusState private var isFocused: Bool

    var body: some View {
        Button(action: action) {
            HStack(spacing: 6) {
                if let icon = icon {
                    Image(systemName: icon)
                }
                Text(label)
            }
            .padding(.horizontal, 16)
            .padding(.vertical, 10)
            .background(isActive ? Color.blue : Color.gray.opacity(0.3))
            .foregroundColor(isActive ? .white : .secondary)
            .cornerRadius(20)
        }
        .buttonStyle(.card)
        .focused($isFocused)
    }
}
```

### 3. ExtraCard
Card for displaying an extra in the grid.

```swift
struct ExtraCard: View {
    let extra: Extra

    var body: some View {
        Button(action: { playExtra(extra) }) {
            VStack {
                // Icon with category color
                ZStack {
                    LinearGradient(
                        colors: [Color.black.opacity(0.8), Color.gray.opacity(0.3)],
                        startPoint: .topLeading,
                        endPoint: .bottomTrailing
                    )

                    Image(systemName: extra.category.icon)
                        .font(.system(size: 40))
                        .foregroundColor(extra.category.color)
                }
                .aspectRatio(2/3, contentMode: .fit)
                .cornerRadius(8)

                // Title and metadata
                VStack(alignment: .leading, spacing: 4) {
                    Text(extra.title)
                        .font(.caption)
                        .lineLimit(2)

                    HStack {
                        Text(extra.category.displayName)
                            .font(.caption2)
                            .foregroundColor(.secondary)

                        if extra.duration > 0 {
                            Text("•")
                            Text(formatDuration(extra.duration))
                                .font(.caption2)
                                .foregroundColor(.secondary)
                        }
                    }
                }
            }
        }
        .buttonStyle(.card)
    }
}
```

### 4. ExtrasListView
Horizontal scrolling list for detail views (movie/show detail).

```swift
struct ExtrasListView: View {
    let extras: [Extra]

    // Group by category
    var groupedExtras: [(ExtraCategory, [Extra])] {
        Dictionary(grouping: extras, by: \.category)
            .sorted { $0.key.rawValue < $1.key.rawValue }
            .map { ($0.key, $0.value) }
    }

    var body: some View {
        VStack(alignment: .leading, spacing: 20) {
            ForEach(groupedExtras, id: \.0) { category, items in
                VStack(alignment: .leading, spacing: 12) {
                    // Category header
                    HStack(spacing: 8) {
                        Image(systemName: category.icon)
                            .foregroundColor(category.color)
                        Text(category.displayName)
                            .font(.headline)
                    }

                    // Horizontal scroll of extras
                    ScrollView(.horizontal, showsIndicators: false) {
                        HStack(spacing: 16) {
                            ForEach(items) { extra in
                                ExtraListItem(extra: extra)
                            }
                        }
                    }
                }
            }
        }
    }
}
```

---

## Siri Remote Gestures

| Gesture | Action |
|---------|--------|
| Swipe left/right | Navigate between filter chips |
| Click | Select category filter / Play extra |
| Menu button | Return to previous screen |
| Play/Pause | Play focused extra |

---

## Focus Handling

Use SwiftUI's `@FocusState` for tvOS focus management:

```swift
@FocusState private var focusedExtraId: Int64?

// In grid:
ForEach(viewModel.extras) { extra in
    ExtraCard(extra: extra)
        .focused($focusedExtraId, equals: extra.id)
}
```

---

## ViewModel

```swift
@MainActor
class ExtrasViewModel: ObservableObject {
    @Published var extras: [Extra] = []
    @Published var categories: [CategoryCount] = []
    @Published var selectedCategory: ExtraCategory?
    @Published var isLoading = false

    private var offset = 0
    private let limit = 50

    func loadExtras() async {
        isLoading = true
        defer { isLoading = false }

        do {
            if let category = selectedCategory {
                extras = try await APIClient.shared.getExtrasByCategory(category, limit: limit, offset: offset)
            } else {
                extras = try await APIClient.shared.getExtras(limit: limit, offset: offset)
            }
        } catch {
            print("Failed to load extras: \(error)")
        }
    }

    func loadCategories() async {
        do {
            categories = try await APIClient.shared.getExtrasCategories()
        } catch {
            print("Failed to load categories: \(error)")
        }
    }

    func selectCategory(_ category: ExtraCategory?) {
        selectedCategory = category
        offset = 0
        Task { await loadExtras() }
    }
}
```

---

## Playback

Use the `type=extra` query parameter when streaming:

```swift
func playExtra(_ extra: Extra) {
    let streamURL = "\(baseURL)/api/stream/\(extra.id)/manifest.m3u8?type=extra"
    let player = AVPlayer(url: URL(string: streamURL)!)
    // Present player...
}
```

---

## Integration in Detail Views

### Movie Detail
Add extras section after overview:

```swift
// In MovieDetailView
if !viewModel.extras.isEmpty {
    Section("Extras") {
        ExtrasListView(extras: viewModel.extras)
    }
}
```

### TV Show Detail
Add extras tab or section:

```swift
// In TVShowDetailView
TabView {
    EpisodesView(show: show)
        .tabItem { Label("Episodes", systemImage: "list.number") }

    if !viewModel.extras.isEmpty {
        ExtrasListView(extras: viewModel.extras)
            .tabItem { Label("Extras", systemImage: "star.fill") }
    }
}
```

---

## Testing Checklist

- [ ] Browse all extras in grid view
- [ ] Filter by category using chips
- [ ] Search extras by title
- [ ] Sorting: Recently Added, Title, Duration, Category
- [ ] Play extras from grid view
- [ ] Play extras from movie detail
- [ ] Play extras from TV show detail
- [ ] Verify category icons display correctly
- [ ] Verify category colors are visible
- [ ] Test focus navigation with Siri Remote
- [ ] Test pagination (Load More)

---

## Known Limitations

1. Extras don't have poster images - use category icon with color
2. Some extras may not have duration metadata
3. Parent title may not be set for all extras

---

## Future Enhancements

1. Add thumbnail/still images for extras when available
2. Add "Recently Watched Extras" section
3. Add extras count badge on movie/show cards
4. Support extras search in global search
