import Foundation

struct Configuration {
    static let appName = "Media Server Mobile"
    static let appVersion = "1.0.0"
    static let bundleIdentifier = "com.mediaserver.mobile"

    // WebView configuration
    static let userAgent = "MediaServer-iOS/\(appVersion)"
    static let allowsInlineMediaPlayback = true
    static let allowsPictureInPictureMediaPlayback = true

    // Deep linking
    static let deepLinkScheme = "mediaserver"
    static let deepLinkHost = "app"

    // JavaScript bridge message handlers
    enum MessageHandler: String {
        case auth = "auth"
        case playback = "playback"
        case share = "share"
        case notification = "notification"
        case deepLink = "deepLink"
        case log = "log"
    }
}

// Deep link paths
enum DeepLinkPath: String {
    case media = "/media"
    case play = "/play"
    case library = "/library"
    case search = "/search"
}
