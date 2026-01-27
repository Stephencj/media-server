import SwiftUI

struct SettingsView: View {
    @EnvironmentObject var authService: AuthService
    @EnvironmentObject var appState: AppState
    @Environment(\.dismiss) private var dismiss

    @State private var serverURL: String = ""
    @State private var showingLogoutAlert = false

    var body: some View {
        NavigationView {
            Form {
                // Server Configuration
                Section {
                    TextField("Server URL", text: $serverURL)
                        .keyboardType(.URL)
                        .autocapitalization(.none)
                        .disableAutocorrection(true)
                        .textContentType(.URL)

                    Button("Save Server URL") {
                        saveServerURL()
                    }
                    .disabled(serverURL.isEmpty || serverURL == appState.serverURL)
                } header: {
                    Text("Server Configuration")
                } footer: {
                    Text("Enter the full URL of your media server (e.g., http://192.168.1.100:3000)")
                }

                // Account Section
                if authService.isAuthenticated {
                    Section("Account") {
                        if let user = authService.currentUser {
                            HStack {
                                Image(systemName: "person.circle.fill")
                                    .font(.largeTitle)
                                    .foregroundColor(.blue)

                                VStack(alignment: .leading, spacing: 4) {
                                    Text(user.username)
                                        .font(.headline)
                                    Text(user.email)
                                        .font(.subheadline)
                                        .foregroundColor(.secondary)
                                }
                            }
                            .padding(.vertical, 8)
                        }

                        Button(role: .destructive) {
                            showingLogoutAlert = true
                        } label: {
                            Label("Sign Out", systemImage: "rectangle.portrait.and.arrow.right")
                        }
                    }
                }

                // App Info
                Section("About") {
                    HStack {
                        Text("App Version")
                        Spacer()
                        Text(Configuration.appVersion)
                            .foregroundColor(.secondary)
                    }

                    HStack {
                        Text("Bundle ID")
                        Spacer()
                        Text(Configuration.bundleIdentifier)
                            .font(.caption)
                            .foregroundColor(.secondary)
                    }

                    if !appState.serverURL.isEmpty {
                        HStack {
                            Text("Current Server")
                            Spacer()
                            Text(appState.serverURL)
                                .font(.caption)
                                .foregroundColor(.secondary)
                                .lineLimit(1)
                        }
                    }
                }

                // Debug Section (only in debug builds)
                #if DEBUG
                Section("Debug") {
                    Button("Clear All Data") {
                        clearAllData()
                    }
                    .foregroundColor(.red)

                    Button("Reload WebView") {
                        NotificationCenter.default.post(name: .serverURLChanged, object: nil)
                    }
                }
                #endif
            }
            .navigationTitle("Settings")
            .navigationBarTitleDisplayMode(.inline)
            .toolbar {
                ToolbarItem(placement: .navigationBarTrailing) {
                    Button("Done") {
                        dismiss()
                    }
                }
            }
            .alert("Sign Out", isPresented: $showingLogoutAlert) {
                Button("Cancel", role: .cancel) {}
                Button("Sign Out", role: .destructive) {
                    authService.logout()
                    dismiss()
                }
            } message: {
                Text("Are you sure you want to sign out?")
            }
            .onAppear {
                serverURL = appState.serverURL
            }
        }
    }

    private func saveServerURL() {
        var url = serverURL.trimmingCharacters(in: .whitespacesAndNewlines)

        // Ensure URL has scheme
        if !url.hasPrefix("http://") && !url.hasPrefix("https://") {
            url = "http://" + url
        }

        // Remove trailing slash
        if url.hasSuffix("/") {
            url = String(url.dropLast())
        }

        appState.serverURL = url
        appState.showError("Server URL saved. Please login through the web interface.")

        // If settings was presented as a sheet, dismiss it
        if authService.isAuthenticated {
            dismiss()
        }
    }

    private func clearAllData() {
        authService.logout()
        appState.serverURL = ""
        serverURL = ""
        appState.showError("All data cleared")
    }
}

#Preview {
    SettingsView()
        .environmentObject(AuthService.shared)
        .environmentObject(AppState.shared)
}
