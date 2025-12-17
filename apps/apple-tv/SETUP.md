# Apple TV App Setup

## Creating the Xcode Project

Since I can't create Xcode projects directly, follow these steps:

### 1. Create New Project in Xcode

1. Open Xcode
2. File > New > Project
3. Select **tvOS** tab
4. Choose **App**
5. Configure:
   - Product Name: `MediaPlayer`
   - Team: Your development team
   - Organization Identifier: `com.yourname` (or your bundle ID prefix)
   - Interface: **SwiftUI**
   - Language: **Swift**
6. Save in: `apps/apple-tv/` (replace the MediaPlayer folder)

### 2. Add Source Files

After creating the project:

1. Delete the default `ContentView.swift` file
2. In Xcode's Project Navigator, right-click on the `MediaPlayer` folder
3. Select **Add Files to "MediaPlayer"...**
4. Navigate to the `MediaPlayer` folder with all the Swift files
5. Select all folders: `App`, `Views`, `ViewModels`, `Models`, `Services`, `Components`, `Extensions`
6. Check **"Copy items if needed"** and **"Create groups"**
7. Click Add

### 3. Project Settings

1. Select the project in the navigator
2. Select the `MediaPlayer` target
3. Go to **Signing & Capabilities**:
   - Add your Team
   - Check "Automatically manage signing"

4. Go to **Info** tab and add:
   - `App Transport Security Settings` > `Allow Arbitrary Loads` = `YES`
   (Required for local network HTTP during development)

### 4. Build and Run

1. Select an Apple TV simulator or your device
2. Press Cmd+R to build and run

## Project Structure

```
MediaPlayer/
├── App/
│   ├── MediaPlayerApp.swift    # App entry point
│   └── AppState.swift          # Global app state
├── Views/
│   ├── LoginView.swift         # Authentication screen
│   ├── MainTabView.swift       # Tab navigation
│   ├── HomeView.swift          # Home with continue watching
│   ├── LibraryView.swift       # Movies/Shows grid
│   ├── MediaDetailView.swift   # Media details + play
│   ├── PlayerView.swift        # Video player
│   ├── SearchView.swift        # Search functionality
│   └── SettingsView.swift      # Settings + sources
├── ViewModels/
│   ├── HomeViewModel.swift
│   ├── LibraryViewModel.swift
│   ├── MediaDetailViewModel.swift
│   └── PlayerViewModel.swift
├── Models/
│   ├── Media.swift             # Media data models
│   ├── User.swift              # Auth models
│   └── Progress.swift          # Watch progress
├── Services/
│   ├── APIClient.swift         # HTTP client
│   └── AuthService.swift       # Authentication
└── Components/
    └── MediaCardView.swift     # Reusable media card
```

## Configuration

The app connects to your media server. On first launch:

1. Enter your server URL (e.g., `http://192.168.1.100:8080`)
2. Create an account or log in
3. The app will load your media library

## Features

- **Home**: Continue watching, recently added, quick access to library
- **Movies/TV Shows**: Full library grid with posters
- **Search**: Search your entire library
- **Player**: HLS streaming with progress tracking
- **Settings**: Manage media sources, trigger scans

## Requirements

- Xcode 15+
- tvOS 17+
- Apple Developer account (for device testing)
