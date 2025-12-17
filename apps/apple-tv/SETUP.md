# Apple TV App

## Opening the Project

1. Open `MediaPlayer.xcodeproj` in Xcode
2. Select your Development Team in Signing & Capabilities
3. Select an Apple TV simulator or device
4. Press Cmd+R to build and run

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
├── Components/
│   └── MediaCardView.swift     # Reusable media card
└── Assets.xcassets/            # App icons and colors
```

## Configuration

On first launch:
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
