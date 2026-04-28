import SwiftUI
import AVFoundation
import UIKit

struct QRLoginScannerView: View {
    @Environment(\.dismiss) private var dismiss
    @StateObject private var vm = QRLoginScannerViewModel()

    var body: some View {
        ZStack {
            if vm.permissionDenied {
                PermissionDeniedView(dismiss: { dismiss() })
            } else {
                ScannerCameraView(vm: vm)
                    .ignoresSafeArea()

                ScannerOverlay(status: vm.status, onClose: { dismiss() })
            }
        }
        .onChange(of: vm.status) { newValue in
            if case .success = newValue {
                DispatchQueue.main.asyncAfter(deadline: .now() + 0.8) {
                    dismiss()
                }
            }
        }
        .preferredColorScheme(.dark)
    }
}

@MainActor
final class QRLoginScannerViewModel: ObservableObject {
    enum ScanStatus: Equatable {
        case scanning
        case claiming
        case success
        case error(String)
    }

    @Published var status: ScanStatus = .scanning
    @Published var permissionDenied = false

    weak var scannerVC: QRScannerVC?

    func handleDetection(_ raw: String) {
        guard status == .scanning else { return }
        guard let code = parseLoginCode(raw) else {
            status = .error("QR not recognized — try again")
            UINotificationFeedbackGenerator().notificationOccurred(.warning)
            DispatchQueue.main.asyncAfter(deadline: .now() + 1.5) { [weak self] in
                guard let self else { return }
                self.status = .scanning
                self.scannerVC?.resumeScanning()
            }
            return
        }
        status = .claiming
        Task {
            do {
                try await AuthService.shared.claimWebLoginCode(code)
                UINotificationFeedbackGenerator().notificationOccurred(.success)
                status = .success
            } catch APIError.serverError(let s, _) where s == 410 {
                showError("Code expired — refresh the page on your computer.")
            } catch APIError.unauthorized {
                showError("Sign-in expired. Reopen the app and try again.")
            } catch {
                showError(error.localizedDescription)
            }
        }
    }

    private func showError(_ message: String) {
        status = .error(message)
        UINotificationFeedbackGenerator().notificationOccurred(.error)
        DispatchQueue.main.asyncAfter(deadline: .now() + 2.0) { [weak self] in
            guard let self, case .error = self.status else { return }
            self.status = .scanning
            self.scannerVC?.resumeScanning()
        }
    }

    func parseLoginCode(_ raw: String) -> String? {
        if let comps = URLComponents(string: raw),
           let code = comps.queryItems?.first(where: { $0.name == "code" })?.value,
           !code.isEmpty {
            return code
        }
        if raw.hasPrefix("code:") { return String(raw.dropFirst(5)) }
        if raw.range(of: "^[a-fA-F0-9]{32,128}$", options: .regularExpression) != nil { return raw }
        return nil
    }
}

// MARK: - Camera UIViewControllerRepresentable

struct ScannerCameraView: UIViewControllerRepresentable {
    @ObservedObject var vm: QRLoginScannerViewModel

    func makeUIViewController(context: Context) -> QRScannerVC {
        let vc = QRScannerVC()
        vc.onDetect = { [weak vm] code in vm?.handleDetection(code) }
        vc.onPermissionDenied = { [weak vm] in vm?.permissionDenied = true }
        vm.scannerVC = vc
        return vc
    }

    func updateUIViewController(_ uiViewController: QRScannerVC, context: Context) {}
}

final class QRScannerVC: UIViewController, AVCaptureMetadataOutputObjectsDelegate {
    private let session = AVCaptureSession()
    private var previewLayer: AVCaptureVideoPreviewLayer?
    var onDetect: ((String) -> Void)?
    var onPermissionDenied: (() -> Void)?

    override func viewDidLoad() {
        super.viewDidLoad()
        view.backgroundColor = .black
    }

    override func viewWillAppear(_ animated: Bool) {
        super.viewWillAppear(animated)
        switch AVCaptureDevice.authorizationStatus(for: .video) {
        case .authorized:
            configureAndStart()
        case .notDetermined:
            AVCaptureDevice.requestAccess(for: .video) { [weak self] granted in
                DispatchQueue.main.async {
                    if granted {
                        self?.configureAndStart()
                    } else {
                        self?.onPermissionDenied?()
                    }
                }
            }
        default:
            onPermissionDenied?()
        }
    }

    override func viewWillDisappear(_ animated: Bool) {
        super.viewWillDisappear(animated)
        DispatchQueue.global(qos: .userInitiated).async { [weak self] in
            self?.session.stopRunning()
        }
    }

    override func viewDidLayoutSubviews() {
        super.viewDidLayoutSubviews()
        previewLayer?.frame = view.bounds
    }

    func resumeScanning() {
        DispatchQueue.global(qos: .userInitiated).async { [weak self] in
            guard let self, !self.session.isRunning else { return }
            self.session.startRunning()
        }
    }

    private func configureAndStart() {
        DispatchQueue.global(qos: .userInitiated).async { [weak self] in
            guard let self else { return }
            if !self.session.inputs.isEmpty {
                self.session.startRunning()
                return
            }
            self.session.beginConfiguration()
            guard let device = AVCaptureDevice.default(for: .video),
                  let input = try? AVCaptureDeviceInput(device: device),
                  self.session.canAddInput(input) else {
                self.session.commitConfiguration()
                DispatchQueue.main.async { self.onPermissionDenied?() }
                return
            }
            self.session.addInput(input)

            let metadataOutput = AVCaptureMetadataOutput()
            guard self.session.canAddOutput(metadataOutput) else {
                self.session.commitConfiguration()
                return
            }
            self.session.addOutput(metadataOutput)
            metadataOutput.setMetadataObjectsDelegate(self, queue: .main)
            metadataOutput.metadataObjectTypes = [.qr]
            self.session.commitConfiguration()

            DispatchQueue.main.async {
                let layer = AVCaptureVideoPreviewLayer(session: self.session)
                layer.videoGravity = .resizeAspectFill
                layer.frame = self.view.bounds
                self.view.layer.addSublayer(layer)
                self.previewLayer = layer
            }
            self.session.startRunning()
        }
    }

    func metadataOutput(_ output: AVCaptureMetadataOutput,
                        didOutput metadataObjects: [AVMetadataObject],
                        from connection: AVCaptureConnection) {
        guard let first = metadataObjects.first as? AVMetadataMachineReadableCodeObject,
              let payload = first.stringValue else { return }
        DispatchQueue.global(qos: .userInitiated).async { [weak self] in
            self?.session.stopRunning()
        }
        onDetect?(payload)
    }
}

// MARK: - Overlay UI

private struct ScannerOverlay: View {
    let status: QRLoginScannerViewModel.ScanStatus
    let onClose: () -> Void

    private var reticleColor: Color {
        switch status {
        case .scanning: return .white
        case .claiming, .success: return .green
        case .error: return .red
        }
    }

    var body: some View {
        VStack {
            HStack {
                Button(action: onClose) {
                    Image(systemName: "xmark.circle.fill")
                        .font(.title)
                        .foregroundStyle(.white, .black.opacity(0.4))
                }
                .padding(16)
                Spacer()
            }

            Spacer()

            ZStack {
                RoundedRectangle(cornerRadius: 24)
                    .stroke(reticleColor, lineWidth: 3)
                    .frame(width: 260, height: 260)
                    .animation(.easeInOut(duration: 0.2), value: reticleColor)

                Group {
                    switch status {
                    case .scanning:
                        EmptyView()
                    case .claiming:
                        ProgressView()
                            .progressViewStyle(.circular)
                            .tint(.white)
                            .scaleEffect(1.5)
                    case .success:
                        Image(systemName: "checkmark.circle.fill")
                            .font(.system(size: 80))
                            .foregroundColor(.green)
                            .transition(.scale.combined(with: .opacity))
                    case .error:
                        EmptyView()
                    }
                }
            }

            Spacer()

            VStack(spacing: 8) {
                if case .error(let message) = status {
                    Text(message)
                        .font(.callout)
                        .foregroundColor(.red)
                        .multilineTextAlignment(.center)
                        .padding(.horizontal, 24)
                }
                Text("Open the login page on your computer and point your camera at the QR code.")
                    .font(.callout)
                    .foregroundColor(.white)
                    .multilineTextAlignment(.center)
                    .padding(.horizontal, 24)
            }
            .padding(.bottom, 32)
        }
    }
}

private struct PermissionDeniedView: View {
    let dismiss: () -> Void

    var body: some View {
        VStack(spacing: 20) {
            Image(systemName: "video.slash")
                .font(.system(size: 50))
                .foregroundColor(.white.opacity(0.6))
            Text("Camera access denied")
                .font(.title3)
                .foregroundColor(.white)
            Text("Open Settings to allow camera access for Media Server.")
                .font(.callout)
                .foregroundColor(.white.opacity(0.7))
                .multilineTextAlignment(.center)
                .padding(.horizontal, 24)
            Button("Open Settings") {
                if let url = URL(string: UIApplication.openSettingsURLString) {
                    UIApplication.shared.open(url)
                }
            }
            .buttonStyle(.borderedProminent)
            Button("Cancel", action: dismiss)
                .foregroundColor(.white.opacity(0.7))
        }
        .frame(maxWidth: .infinity, maxHeight: .infinity)
        .background(Color.black)
    }
}
