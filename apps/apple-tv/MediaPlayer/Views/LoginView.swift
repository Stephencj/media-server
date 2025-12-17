import SwiftUI

struct LoginView: View {
    @EnvironmentObject var authService: AuthService
    @EnvironmentObject var appState: AppState

    @State private var serverURL = ""
    @State private var username = ""
    @State private var password = ""
    @State private var email = ""
    @State private var isRegistering = false
    @State private var errorMessage: String?
    @FocusState private var focusedField: Field?

    enum Field {
        case server, username, email, password
    }

    var body: some View {
        VStack(spacing: 60) {
            // Logo/Title
            VStack(spacing: 20) {
                Image(systemName: "play.rectangle.fill")
                    .font(.system(size: 100))
                    .foregroundColor(.blue)

                Text("Media Server")
                    .font(.largeTitle)
                    .fontWeight(.bold)
            }

            // Form
            VStack(spacing: 30) {
                // Server URL
                VStack(alignment: .leading, spacing: 10) {
                    Text("Server Address")
                        .font(.headline)
                        .foregroundColor(.secondary)

                    TextField("http://192.168.1.100:8080", text: $serverURL)
                        .textFieldStyle(.plain)
                        .padding()
                        .background(Color.white.opacity(0.1))
                        .cornerRadius(10)
                        .focused($focusedField, equals: .server)
                        .autocorrectionDisabled()
                        .textInputAutocapitalization(.never)
                }

                // Username
                VStack(alignment: .leading, spacing: 10) {
                    Text("Username")
                        .font(.headline)
                        .foregroundColor(.secondary)

                    TextField("Username", text: $username)
                        .textFieldStyle(.plain)
                        .padding()
                        .background(Color.white.opacity(0.1))
                        .cornerRadius(10)
                        .focused($focusedField, equals: .username)
                        .autocorrectionDisabled()
                        .textInputAutocapitalization(.never)
                }

                // Email (registration only)
                if isRegistering {
                    VStack(alignment: .leading, spacing: 10) {
                        Text("Email")
                            .font(.headline)
                            .foregroundColor(.secondary)

                        TextField("email@example.com", text: $email)
                            .textFieldStyle(.plain)
                            .padding()
                            .background(Color.white.opacity(0.1))
                            .cornerRadius(10)
                            .focused($focusedField, equals: .email)
                            .autocorrectionDisabled()
                            .textInputAutocapitalization(.never)
                    }
                }

                // Password
                VStack(alignment: .leading, spacing: 10) {
                    Text("Password")
                        .font(.headline)
                        .foregroundColor(.secondary)

                    SecureField("Password", text: $password)
                        .textFieldStyle(.plain)
                        .padding()
                        .background(Color.white.opacity(0.1))
                        .cornerRadius(10)
                        .focused($focusedField, equals: .password)
                }
            }
            .frame(maxWidth: 500)

            // Error message
            if let error = errorMessage {
                Text(error)
                    .foregroundColor(.red)
                    .multilineTextAlignment(.center)
            }

            // Buttons
            VStack(spacing: 20) {
                Button(action: submit) {
                    HStack {
                        if authService.isLoading {
                            ProgressView()
                                .progressViewStyle(CircularProgressViewStyle())
                        }
                        Text(isRegistering ? "Create Account" : "Sign In")
                            .fontWeight(.semibold)
                    }
                    .frame(minWidth: 200)
                }
                .disabled(authService.isLoading || !isFormValid)

                Button(action: { isRegistering.toggle() }) {
                    Text(isRegistering ? "Already have an account? Sign In" : "Don't have an account? Register")
                        .foregroundColor(.secondary)
                }
                .buttonStyle(.plain)
            }
        }
        .padding(80)
        .onAppear {
            serverURL = appState.serverURL
        }
    }

    private var isFormValid: Bool {
        !serverURL.isEmpty && !username.isEmpty && !password.isEmpty &&
        (!isRegistering || !email.isEmpty)
    }

    private func submit() {
        errorMessage = nil
        appState.serverURL = serverURL

        Task {
            do {
                if isRegistering {
                    try await authService.register(username: username, email: email, password: password)
                } else {
                    try await authService.login(username: username, password: password)
                }
            } catch {
                errorMessage = error.localizedDescription
            }
        }
    }
}

#Preview {
    LoginView()
        .environmentObject(AuthService.shared)
        .environmentObject(AppState.shared)
}
