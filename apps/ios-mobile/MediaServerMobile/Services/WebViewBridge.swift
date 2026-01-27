import Foundation
import WebKit
import UserNotifications

@MainActor
class WebViewBridge: NSObject {
    weak var authService: AuthService?
    weak var appState: AppState?

    init(authService: AuthService, appState: AppState) {
        self.authService = authService
        self.appState = appState
        super.init()
    }

    // MARK: - JavaScript Injection

    func getInjectionScript() -> String {
        return """
        (function() {
            // Create native bridge
            window.mediaServerNative = {
                // Auth handling
                setAuth: function(token, expiresAt, user) {
                    window.webkit.messageHandlers.auth.postMessage({
                        token: token,
                        expiresAt: expiresAt,
                        user: user
                    });
                },

                // Get stored auth token
                getAuthToken: function() {
                    return window.__authToken;
                },

                // Playback controls
                playMedia: function(mediaId, position) {
                    window.webkit.messageHandlers.playback.postMessage({
                        action: 'play',
                        mediaId: mediaId,
                        position: position
                    });
                },

                pauseMedia: function() {
                    window.webkit.messageHandlers.playback.postMessage({
                        action: 'pause'
                    });
                },

                // Share functionality
                share: function(title, url) {
                    window.webkit.messageHandlers.share.postMessage({
                        title: title,
                        url: url
                    });
                },

                // Notifications
                requestNotificationPermission: function() {
                    window.webkit.messageHandlers.notification.postMessage({
                        action: 'requestPermission'
                    });
                },

                // Deep linking
                openDeepLink: function(path) {
                    window.webkit.messageHandlers.deepLink.postMessage({
                        path: path
                    });
                },

                // Logging
                log: function(level, message) {
                    window.webkit.messageHandlers.log.postMessage({
                        level: level,
                        message: message
                    });
                },

                // Platform info
                platform: 'ios',
                version: '\(Configuration.appVersion)'
            };

            // Override console methods to forward to native
            const originalLog = console.log;
            const originalError = console.error;
            const originalWarn = console.warn;

            console.log = function(...args) {
                originalLog.apply(console, args);
                window.mediaServerNative.log('info', args.join(' '));
            };

            console.error = function(...args) {
                originalError.apply(console, args);
                window.mediaServerNative.log('error', args.join(' '));
            };

            console.warn = function(...args) {
                originalWarn.apply(console, args);
                window.mediaServerNative.log('warn', args.join(' '));
            };

            // Notify web app that native bridge is ready
            window.dispatchEvent(new Event('nativeBridgeReady'));

            console.log('Media Server Native Bridge initialized');
        })();
        """
    }

    func getAuthTokenInjectionScript(token: String?) -> String {
        if let token = token {
            return "window.__authToken = '\(token)';"
        } else {
            return "window.__authToken = null;"
        }
    }

    // MARK: - Message Handling

    func handleMessage(_ message: WKScriptMessage) {
        guard let body = message.body as? [String: Any] else {
            print("Invalid message body: \(message.body)")
            return
        }

        switch message.name {
        case Configuration.MessageHandler.auth.rawValue:
            handleAuthMessage(body)

        case Configuration.MessageHandler.playback.rawValue:
            handlePlaybackMessage(body)

        case Configuration.MessageHandler.share.rawValue:
            handleShareMessage(body)

        case Configuration.MessageHandler.notification.rawValue:
            handleNotificationMessage(body)

        case Configuration.MessageHandler.deepLink.rawValue:
            handleDeepLinkMessage(body)

        case Configuration.MessageHandler.log.rawValue:
            handleLogMessage(body)

        default:
            print("Unknown message handler: \(message.name)")
        }
    }

    private func handleAuthMessage(_ body: [String: Any]) {
        guard let token = body["token"] as? String,
              let expiresAt = body["expiresAt"] as? Int64,
              let userData = body["user"] as? [String: Any],
              let userId = userData["id"] as? Int64,
              let username = userData["username"] as? String,
              let email = userData["email"] as? String else {
            print("Invalid auth message format")
            return
        }

        let user = User(id: userId, username: username, email: email)
        let authMessage = AuthMessage(token: token, expiresAt: expiresAt, user: user)

        authService?.handleAuthFromWeb(authMessage)
        print("Auth received from web: user=\(username)")
    }

    private func handlePlaybackMessage(_ body: [String: Any]) {
        guard let action = body["action"] as? String else { return }

        switch action {
        case "play":
            if let mediaId = body["mediaId"] as? Int64 {
                print("Play media request: \(mediaId)")
                // In the future, could handle native playback here
            }
        case "pause":
            print("Pause media request")
        default:
            print("Unknown playback action: \(action)")
        }
    }

    private func handleShareMessage(_ body: [String: Any]) {
        guard let title = body["title"] as? String,
              let urlString = body["url"] as? String else {
            return
        }

        print("Share request: \(title) - \(urlString)")
        // Share functionality will be handled in WebView
        NotificationCenter.default.post(
            name: .init("shareRequest"),
            object: nil,
            userInfo: ["title": title, "url": urlString]
        )
    }

    private func handleNotificationMessage(_ body: [String: Any]) {
        guard let action = body["action"] as? String else { return }

        if action == "requestPermission" {
            print("Notification permission requested")
            // Request notification permission
            Task {
                await requestNotificationPermission()
            }
        }
    }

    private func handleDeepLinkMessage(_ body: [String: Any]) {
        guard let path = body["path"] as? String else { return }
        print("Deep link request: \(path)")
        // Handle deep linking navigation
    }

    private func handleLogMessage(_ body: [String: Any]) {
        guard let level = body["level"] as? String,
              let message = body["message"] as? String else { return }

        let prefix = "WebView[\(level)]"
        print("\(prefix): \(message)")
    }

    // MARK: - Helper Methods

    private func requestNotificationPermission() async {
        do {
            let granted = try await UNUserNotificationCenter.current().requestAuthorization(options: [.alert, .sound, .badge])
            print("Notification permission granted: \(granted)")
        } catch {
            print("Error requesting notification permission: \(error)")
        }
    }
}
