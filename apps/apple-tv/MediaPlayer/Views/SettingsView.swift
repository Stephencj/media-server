import SwiftUI

struct SettingsView: View {
    @EnvironmentObject var authService: AuthService
    @EnvironmentObject var appState: AppState

    @State private var sources: [MediaSource] = []
    @State private var isLoading = false
    @State private var showAddSource = false
    @State private var scanStatus: String?

    var body: some View {
        NavigationStack {
            List {
                // User section
                Section("Account") {
                    if let user = authService.currentUser {
                        HStack {
                            Image(systemName: "person.circle.fill")
                                .font(.largeTitle)
                                .foregroundColor(.blue)

                            VStack(alignment: .leading) {
                                Text(user.username)
                                    .font(.headline)
                                Text(user.email)
                                    .font(.subheadline)
                                    .foregroundColor(.secondary)
                            }
                        }
                        .padding(.vertical, 10)
                    }

                    Button(role: .destructive) {
                        authService.logout()
                    } label: {
                        Label("Sign Out", systemImage: "rectangle.portrait.and.arrow.right")
                    }
                }

                // Server section
                Section("Server") {
                    HStack {
                        Text("Server URL")
                        Spacer()
                        Text(appState.serverURL)
                            .foregroundColor(.secondary)
                    }

                    Button {
                        triggerScan()
                    } label: {
                        HStack {
                            Label("Scan Library", systemImage: "arrow.clockwise")
                            Spacer()
                            if let status = scanStatus {
                                Text(status)
                                    .foregroundColor(.secondary)
                            }
                        }
                    }
                }

                // Media Sources section
                Section {
                    if sources.isEmpty && !isLoading {
                        Text("No media sources configured")
                            .foregroundColor(.secondary)
                    } else {
                        ForEach(sources) { source in
                            VStack(alignment: .leading, spacing: 5) {
                                HStack {
                                    Image(systemName: sourceIcon(for: source.type))
                                    Text(source.name)
                                        .font(.headline)
                                }

                                Text(source.path)
                                    .font(.caption)
                                    .foregroundColor(.secondary)

                                if let lastScan = source.lastScan {
                                    Text("Last scan: \(lastScan)")
                                        .font(.caption2)
                                        .foregroundColor(.secondary)
                                }
                            }
                            .padding(.vertical, 5)
                        }
                        .onDelete(perform: deleteSource)
                    }
                } header: {
                    HStack {
                        Text("Media Sources")
                        Spacer()
                        Button {
                            showAddSource = true
                        } label: {
                            Image(systemName: "plus")
                        }
                    }
                }

                // About section
                Section("About") {
                    HStack {
                        Text("Version")
                        Spacer()
                        Text("1.0.0")
                            .foregroundColor(.secondary)
                    }
                }
            }
            .navigationTitle("Settings")
            .sheet(isPresented: $showAddSource) {
                AddSourceView { source in
                    sources.append(source)
                }
            }
            .task {
                await loadSources()
            }
            .refreshable {
                await loadSources()
            }
        }
    }

    private func sourceIcon(for type: String) -> String {
        switch type {
        case "smb", "nfs":
            return "externaldrive.connected.to.line.below"
        case "local":
            return "folder"
        default:
            return "internaldrive"
        }
    }

    private func loadSources() async {
        isLoading = true
        do {
            let response = try await APIClient.shared.getSources()
            sources = response.sources
        } catch {
            // Handle error
        }
        isLoading = false
    }

    private func deleteSource(at offsets: IndexSet) {
        for index in offsets {
            let source = sources[index]
            Task {
                do {
                    try await APIClient.shared.deleteSource(id: source.id)
                    await MainActor.run {
                        sources.remove(at: index)
                    }
                } catch {
                    // Handle error
                }
            }
        }
    }

    private func triggerScan() {
        scanStatus = "Scanning..."
        Task {
            do {
                let response = try await APIClient.shared.triggerScan()
                await MainActor.run {
                    scanStatus = response.status
                }
                // Clear status after delay
                try? await Task.sleep(nanoseconds: 3_000_000_000)
                await MainActor.run {
                    scanStatus = nil
                }
            } catch {
                await MainActor.run {
                    scanStatus = "Failed"
                }
            }
        }
    }
}

struct AddSourceView: View {
    @Environment(\.dismiss) private var dismiss

    @State private var name = ""
    @State private var path = ""
    @State private var type = "local"
    @State private var isLoading = false
    @State private var errorMessage: String?

    let onAdd: (MediaSource) -> Void

    var body: some View {
        NavigationStack {
            Form {
                Section("Source Details") {
                    TextField("Name", text: $name)
                    TextField("Path", text: $path)

                    Picker("Type", selection: $type) {
                        Text("Local").tag("local")
                        Text("SMB/CIFS").tag("smb")
                        Text("NFS").tag("nfs")
                    }
                }

                if let error = errorMessage {
                    Section {
                        Text(error)
                            .foregroundColor(.red)
                    }
                }
            }
            .navigationTitle("Add Source")
            .toolbar {
                ToolbarItem(placement: .cancellationAction) {
                    Button("Cancel") {
                        dismiss()
                    }
                }

                ToolbarItem(placement: .confirmationAction) {
                    Button("Add") {
                        addSource()
                    }
                    .disabled(name.isEmpty || path.isEmpty || isLoading)
                }
            }
        }
    }

    private func addSource() {
        isLoading = true
        errorMessage = nil

        Task {
            do {
                let source = try await APIClient.shared.createSource(
                    name: name,
                    path: path,
                    type: type
                )
                await MainActor.run {
                    onAdd(source)
                    dismiss()
                }
            } catch {
                await MainActor.run {
                    errorMessage = error.localizedDescription
                    isLoading = false
                }
            }
        }
    }
}

#Preview {
    SettingsView()
        .environmentObject(AuthService.shared)
        .environmentObject(AppState.shared)
}
