import SwiftUI

struct SettingsView: View {
    @State private var showScanner = false

    var body: some View {
        Form {
            Section("Account") {
                Button {
                    showScanner = true
                } label: {
                    Label("Scan to log in browser", systemImage: "qrcode.viewfinder")
                }
            }
        }
        .navigationTitle("Settings")
        .fullScreenCover(isPresented: $showScanner) {
            QRLoginScannerView()
        }
    }
}
