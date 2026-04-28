import AVFoundation
import Foundation

/// Configures `AVAudioSession` for video playback and observes interruptions.
/// Required for Picture-in-Picture, AirPlay, and audio continuation when the
/// app is backgrounded (with `UIBackgroundModes: audio` in Info.plist).
final class AudioSessionManager {
    static let shared = AudioSessionManager()

    private var observersInstalled = false

    private init() {}

    /// Configures the shared session with `.playback` and starts it.
    /// Idempotent — safe to call from app init and again before each `play()`.
    func activatePlayback() {
        let session = AVAudioSession.sharedInstance()
        do {
            if session.category != .playback {
                try session.setCategory(.playback, mode: .moviePlayback, options: [.allowAirPlay])
            }
            try session.setActive(true, options: [])
        } catch {
            print("[AudioSession] activate failed: \(error.localizedDescription)")
        }
        installObserversIfNeeded()
    }

    private func installObserversIfNeeded() {
        guard !observersInstalled else { return }
        observersInstalled = true

        let center = NotificationCenter.default
        center.addObserver(self,
                           selector: #selector(handleInterruption(_:)),
                           name: AVAudioSession.interruptionNotification,
                           object: nil)
        center.addObserver(self,
                           selector: #selector(handleRouteChange(_:)),
                           name: AVAudioSession.routeChangeNotification,
                           object: nil)
    }

    @objc private func handleInterruption(_ note: Notification) {
        guard let typeValue = note.userInfo?[AVAudioSessionInterruptionTypeKey] as? UInt,
              let type = AVAudioSession.InterruptionType(rawValue: typeValue) else { return }
        if type == .ended {
            // Re-activate so a paused player can resume.
            try? AVAudioSession.sharedInstance().setActive(true, options: [])
        }
    }

    @objc private func handleRouteChange(_ note: Notification) {
        // System pauses the AVPlayer for us when headphones unplug; nothing to do.
    }
}
