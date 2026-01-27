import SwiftUI
import WebKit

struct WebView: View {
    @EnvironmentObject var authService: AuthService
    @EnvironmentObject var appState: AppState
    @StateObject private var webViewManager = WebViewManager()

    @State private var showSettings = false
    @State private var showShareSheet = false
    @State private var shareItems: [Any] = []

    var body: some View {
        NavigationView {
            ZStack {
                WebViewRepresentable(
                    url: URL(string: appState.serverURL)!,
                    authService: authService,
                    appState: appState,
                    webViewManager: webViewManager
                )
                .edgesIgnoringSafeArea(.all)

                // Loading indicator
                if appState.isWebViewLoading {
                    ProgressView()
                        .scaleEffect(1.5)
                        .frame(maxWidth: .infinity, maxHeight: .infinity)
                        .background(Color.black.opacity(0.2))
                }
            }
            .navigationBarTitleDisplayMode(.inline)
            .toolbar {
                ToolbarItem(placement: .navigationBarLeading) {
                    HStack(spacing: 16) {
                        Button(action: { webViewManager.goBack() }) {
                            Image(systemName: "chevron.left")
                        }
                        .disabled(!appState.canGoBack)

                        Button(action: { webViewManager.goForward() }) {
                            Image(systemName: "chevron.right")
                        }
                        .disabled(!appState.canGoForward)
                    }
                }

                ToolbarItem(placement: .navigationBarTrailing) {
                    HStack(spacing: 16) {
                        Button(action: { webViewManager.reload() }) {
                            Image(systemName: "arrow.clockwise")
                        }

                        Button(action: { showSettings = true }) {
                            Image(systemName: "gear")
                        }
                    }
                }
            }
            .sheet(isPresented: $showSettings) {
                SettingsView()
            }
            .sheet(isPresented: $showShareSheet) {
                ShareSheet(items: shareItems)
            }
            .onReceive(NotificationCenter.default.publisher(for: .init("shareRequest"))) { notification in
                if let userInfo = notification.userInfo,
                   let title = userInfo["title"] as? String,
                   let urlString = userInfo["url"] as? String,
                   let url = URL(string: urlString) {
                    shareItems = [title, url]
                    showShareSheet = true
                }
            }
        }
        .navigationViewStyle(StackNavigationViewStyle())
    }
}

@MainActor
class WebViewManager: ObservableObject {
    var webView: WKWebView?

    func goBack() {
        webView?.goBack()
    }

    func goForward() {
        webView?.goForward()
    }

    func reload() {
        webView?.reload()
    }
}

struct WebViewRepresentable: UIViewRepresentable {
    let url: URL
    let authService: AuthService
    let appState: AppState
    @ObservedObject var webViewManager: WebViewManager

    func makeCoordinator() -> Coordinator {
        Coordinator(self, authService: authService, appState: appState)
    }

    func makeUIView(context: Context) -> WKWebView {
        // Configure WKWebView
        let config = WKWebViewConfiguration()

        // Enable media playback
        config.allowsInlineMediaPlayback = Configuration.allowsInlineMediaPlayback
        config.allowsPictureInPictureMediaPlayback = Configuration.allowsPictureInPictureMediaPlayback
        config.mediaTypesRequiringUserActionForPlayback = []

        // Set custom user agent
        config.applicationNameForUserAgent = Configuration.userAgent

        // Configure content controller for JavaScript bridge
        let contentController = WKUserContentController()

        // Add message handlers
        for handler in [Configuration.MessageHandler.auth,
                       Configuration.MessageHandler.playback,
                       Configuration.MessageHandler.share,
                       Configuration.MessageHandler.notification,
                       Configuration.MessageHandler.deepLink,
                       Configuration.MessageHandler.log] {
            contentController.add(context.coordinator, name: handler.rawValue)
        }

        // Inject JavaScript bridge
        let bridge = WebViewBridge(authService: authService, appState: appState)
        let bridgeScript = WKUserScript(
            source: bridge.getInjectionScript(),
            injectionTime: .atDocumentStart,
            forMainFrameOnly: false
        )
        contentController.addUserScript(bridgeScript)

        // Inject auth token if available
        if let token = authService.getAuthToken() {
            let tokenScript = WKUserScript(
                source: bridge.getAuthTokenInjectionScript(token: token),
                injectionTime: .atDocumentStart,
                forMainFrameOnly: false
            )
            contentController.addUserScript(tokenScript)
        }

        config.userContentController = contentController

        // Create WebView
        let webView = WKWebView(frame: .zero, configuration: config)
        webView.navigationDelegate = context.coordinator
        webView.allowsBackForwardNavigationGestures = true

        // Store reference
        webViewManager.webView = webView

        // Load URL
        let request = URLRequest(url: url)
        webView.load(request)

        // Listen for auth token changes
        NotificationCenter.default.addObserver(
            forName: .authTokenChanged,
            object: nil,
            queue: .main
        ) { _ in
            let token = authService.getAuthToken()
            let script = bridge.getAuthTokenInjectionScript(token: token)
            webView.evaluateJavaScript(script) { _, error in
                if let error = error {
                    print("Error injecting auth token: \(error)")
                }
            }
        }

        // Listen for server URL changes
        NotificationCenter.default.addObserver(
            forName: .serverURLChanged,
            object: nil,
            queue: .main
        ) { _ in
            if let newURL = URL(string: appState.serverURL) {
                let request = URLRequest(url: newURL)
                webView.load(request)
            }
        }

        return webView
    }

    func updateUIView(_ webView: WKWebView, context: Context) {
        // Update navigation state
        DispatchQueue.main.async {
            appState.canGoBack = webView.canGoBack
            appState.canGoForward = webView.canGoForward
        }
    }

    class Coordinator: NSObject, WKNavigationDelegate, WKScriptMessageHandler {
        var parent: WebViewRepresentable
        var bridge: WebViewBridge

        init(_ parent: WebViewRepresentable, authService: AuthService, appState: AppState) {
            self.parent = parent
            self.bridge = WebViewBridge(authService: authService, appState: appState)
            super.init()
        }

        // MARK: - WKScriptMessageHandler

        func userContentController(_ userContentController: WKUserContentController, didReceive message: WKScriptMessage) {
            bridge.handleMessage(message)
        }

        // MARK: - WKNavigationDelegate

        func webView(_ webView: WKWebView, didStartProvisionalNavigation navigation: WKNavigation!) {
            DispatchQueue.main.async {
                self.parent.appState.isWebViewLoading = true
            }
        }

        func webView(_ webView: WKWebView, didFinish navigation: WKNavigation!) {
            DispatchQueue.main.async {
                self.parent.appState.isWebViewLoading = false
                self.parent.appState.canGoBack = webView.canGoBack
                self.parent.appState.canGoForward = webView.canGoForward
            }
        }

        func webView(_ webView: WKWebView, didFail navigation: WKNavigation!, withError error: Error) {
            DispatchQueue.main.async {
                self.parent.appState.isWebViewLoading = false
                self.parent.appState.showError("Navigation Error", message: error.localizedDescription)
            }
        }

        func webView(_ webView: WKWebView, didFailProvisionalNavigation navigation: WKNavigation!, withError error: Error) {
            DispatchQueue.main.async {
                self.parent.appState.isWebViewLoading = false
                self.parent.appState.showError("Failed to Load", message: error.localizedDescription)
            }
        }

        func webView(_ webView: WKWebView, decidePolicyFor navigationAction: WKNavigationAction, decisionHandler: @escaping (WKNavigationActionPolicy) -> Void) {
            // Handle deep links
            if let url = navigationAction.request.url,
               url.scheme == Configuration.deepLinkScheme {
                // Handle deep link
                print("Deep link: \(url)")
                decisionHandler(.cancel)
                return
            }

            decisionHandler(.allow)
        }
    }
}

// Share Sheet
struct ShareSheet: UIViewControllerRepresentable {
    let items: [Any]

    func makeUIViewController(context: Context) -> UIActivityViewController {
        let controller = UIActivityViewController(activityItems: items, applicationActivities: nil)
        return controller
    }

    func updateUIViewController(_ uiViewController: UIActivityViewController, context: Context) {}
}

#Preview {
    WebView()
        .environmentObject(AuthService.shared)
        .environmentObject(AppState.shared)
}
