# Fire TV App Setup

## Building the Project

### Prerequisites

- Android Studio Arctic Fox or newer
- JDK 17
- Android SDK with API 34

### Steps

1. Open Android Studio
2. Select **Open an existing project**
3. Navigate to `apps/fire-tv` and open it
4. Let Gradle sync complete
5. Connect Fire TV device or use emulator
6. Click **Run** (or Shift+F10)

### Fire TV Device Setup

To install on a physical Fire TV:

1. Enable Developer Options on Fire TV:
   - Settings > My Fire TV > About > Click "About" 7 times
   - Settings > My Fire TV > Developer Options > ADB Debugging = ON

2. Connect via ADB:
   ```bash
   adb connect <fire-tv-ip>:5555
   ```

3. Run from Android Studio or:
   ```bash
   ./gradlew installDebug
   ```

## Project Structure

```
app/src/main/java/com/mediaserver/tv/
├── MediaServerApp.kt              # Hilt Application
├── di/
│   └── AppModule.kt               # Dependency injection
├── data/
│   ├── api/
│   │   └── MediaServerApi.kt      # Retrofit API interface
│   ├── models/                    # Data classes
│   │   ├── Media.kt
│   │   ├── User.kt
│   │   ├── Progress.kt
│   │   └── Responses.kt
│   └── repository/
│       ├── AuthRepository.kt      # Auth + token storage
│       └── MediaRepository.kt     # Media data access
├── ui/
│   ├── browse/
│   │   ├── MainActivity.kt        # Entry point
│   │   ├── MainFragment.kt        # Leanback browse
│   │   ├── MainViewModel.kt
│   │   └── *Presenter.kt          # Card presenters
│   ├── detail/
│   │   ├── DetailActivity.kt
│   │   ├── DetailFragment.kt      # Media details
│   │   └── DetailViewModel.kt
│   ├── player/
│   │   ├── PlayerActivity.kt      # ExoPlayer
│   │   └── PlayerViewModel.kt
│   ├── login/
│   │   ├── LoginActivity.kt
│   │   └── LoginViewModel.kt
│   └── settings/
│       └── SettingsActivity.kt
└── util/
```

## Features

- **Leanback UI**: TV-optimized interface with D-pad navigation
- **ExoPlayer**: Hardware-accelerated video playback
- **HLS Streaming**: Adaptive bitrate support
- **Progress Sync**: Saves every 10 seconds
- **Continue Watching**: Resume from where you left off
- **Encrypted Storage**: Secure token storage

## Remote Control Mapping

| Button | Action |
|--------|--------|
| D-pad | Navigation |
| Select | Play/Pause, Select item |
| Left/Right (in player) | Seek -/+ 10 seconds |
| Back | Exit player, Go back |
| Menu | Options (future) |

## Configuration

On first launch:
1. Enter your media server URL
2. Create account or sign in
3. Browse your library

## Requirements

- Fire TV (2nd gen or newer) or Fire TV Stick 4K
- Android TV / Fire OS 5.0+
- Network access to media server
