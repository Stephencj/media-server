# Media Server Mobile - iOS

A native iOS wrapper app for the Media Server web interface, built with SwiftUI and WKWebView.

## Overview

This app provides a native iOS experience for the Media Server web interface with seamless authentication, deep linking, and native features like sharing and background audio.

## Features

- **Native WebView Integration**: Full-featured WKWebView with the media server web interface
- **Authentication Bridge**: Seamless authentication between native and web contexts
- **JavaScript Bridge**: Two-way communication between native iOS and web JavaScript
- **Deep Linking**: Custom URL scheme support (`mediaserver://`)
- **Safe Area Handling**: Proper support for notched devices (iPhone X and newer)
- **Background Audio**: Supports background audio playback
- **Share Sheet**: Native iOS share functionality
- **Settings Management**: Configure server URL and manage authentication
- **Picture-in-Picture**: Video playback with PiP support
- **Inline Media Playback**: Seamless video playback without full-screen mode

## Requirements

- iOS 15.0+
- Xcode 15.0+
- Swift 5.0+
- A running Media Server instance

## Project Structure

```
ios-mobile/
├── MediaServerMobile/
│   ├── App/
│   │   ├── MediaServerMobileApp.swift   # Main app entry point
│   │   └── AppState.swift                # Global app state management
│   ├── Views/
│   │   ├── WebView.swift                 # WKWebView wrapper with bridge
│   │   └── SettingsView.swift            # Settings configuration screen
│   ├── Services/
│   │   ├── AuthService.swift             # Authentication & keychain management
│   │   └── WebViewBridge.swift           # JavaScript ↔ Native bridge
│   ├── Models/
│   │   ├── User.swift                    # User and auth models
│   │   └── Configuration.swift           # App configuration constants
│   ├── Assets.xcassets/                  # App assets and icons
│   └── Info.plist                        # App configuration
└── MediaServerMobile.xcodeproj/          # Xcode project file
```

## Setup

1. Open `MediaServerMobile.xcodeproj` in Xcode
2. Update the development team in the project settings
3. Build and run on your device or simulator
4. Configure your server URL in Settings
5. Login through the web interface

## JavaScript Bridge

The app provides a native bridge accessible from JavaScript as `window.mediaServerNative`:

### Available Methods

```javascript
// Authentication
window.mediaServerNative.setAuth(token, expiresAt, user);
window.mediaServerNative.getAuthToken();

// Playback
window.mediaServerNative.playMedia(mediaId, position);
window.mediaServerNative.pauseMedia();

// Share
window.mediaServerNative.share(title, url);

// Notifications
window.mediaServerNative.requestNotificationPermission();

// Deep Linking
window.mediaServerNative.openDeepLink(path);

// Logging
window.mediaServerNative.log(level, message);

// Platform Info
window.mediaServerNative.platform; // 'ios'
window.mediaServerNative.version;  // '1.0.0'
```

### Bridge Events

```javascript
// Listen for native bridge ready
window.addEventListener('nativeBridgeReady', () => {
  console.log('Native bridge is ready');
});
```

## Authentication Flow

1. User opens app and configures server URL in Settings
2. App loads web interface in WKWebView
3. User logs in through web interface
4. Web interface calls `window.mediaServerNative.setAuth()` with token
5. Native app stores token in Keychain
6. Token is automatically injected on subsequent page loads

## Deep Linking

The app supports the `mediaserver://` URL scheme:

```
mediaserver://app/media?id=123
mediaserver://app/play?id=456
mediaserver://app/library
mediaserver://app/search
```

## Configuration

### Server URL

Configure your media server URL in Settings. The URL should be in the format:
- `http://192.168.1.100:3000`
- `https://media.example.com`

### Network Security

The app allows arbitrary loads for local network connections (configured in Info.plist). This is required for development servers and local network access.

## Building for Production

1. Update the bundle identifier in `project.pbxproj`
2. Configure your development team
3. Update app icons in `Assets.xcassets/AppIcon.appiconset`
4. Build and archive for App Store distribution

## Architecture

### AppState
Manages global application state including:
- Server URL configuration
- Loading states
- Error handling
- WebView navigation state

### AuthService
Handles authentication with:
- Secure keychain storage for tokens
- Token expiration management
- User session management

### WebViewBridge
Facilitates JavaScript ↔ Native communication:
- Message handlers for web-to-native calls
- JavaScript injection for native-to-web calls
- Console logging forwarding

### WebView
WKWebView wrapper providing:
- Navigation controls
- Loading indicators
- Share sheet integration
- Settings access

## Development

### Running the App

1. Open project in Xcode
2. Select target device/simulator
3. Press Cmd+R to build and run

### Debugging

- Web console logs are forwarded to Xcode console
- Use Safari Web Inspector for web debugging
- Enable "Connect via network" in Settings for wireless debugging

### Common Issues

**App won't load server**
- Check server URL is correct in Settings
- Ensure server is accessible from device
- Check network security settings in Info.plist

**Authentication not working**
- Clear app data in Settings (Debug section)
- Verify web interface calls `setAuth()` correctly
- Check Keychain access

## Integration with Web Interface

The web interface should detect the native environment and use the bridge:

```javascript
// Check if running in native app
if (window.mediaServerNative) {
  // After successful login
  window.mediaServerNative.setAuth(token, expiresAt, user);

  // Use native share
  window.mediaServerNative.share('Movie Title', 'https://...');
}
```

## Future Enhancements

- [ ] Native video player integration
- [ ] Download for offline viewing
- [ ] Push notifications
- [ ] Handoff support
- [ ] Siri shortcuts
- [ ] Widget support
- [ ] CarPlay integration

## License

See main repository license.
