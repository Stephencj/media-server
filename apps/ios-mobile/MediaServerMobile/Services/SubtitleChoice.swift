import AVFoundation
import Combine

/// The viewer's subtitle choice, carried from one video to the next.
///
/// Subtitles are turned on in the player's own menu, and that menu belongs to
/// one video. Every new video started with them off again: a channel's next
/// programme, a playlist's next item. In Picture in Picture, which has no menu,
/// there was then no way to get them back. So the last choice made in the menu
/// is remembered and put on every new video. Until the viewer has chosen,
/// AVFoundation's own pick stands, which follows the iPhone's Subtitles &
/// Captioning setting.
@MainActor
enum SubtitleChoice {
    /// What was last picked: nil when the viewer has never chosen, "" for off,
    /// otherwise the language ("en").
    static var remembered: String? {
        get { UserDefaults.standard.string(forKey: key) }
        set { UserDefaults.standard.set(newValue, forKey: key) }
    }
    private static let key = "subtitleChoice"

    /// Puts the remembered choice on a new video, then remembers whatever the
    /// viewer picks in it. Keep the returned subscription for as long as the
    /// item plays.
    static func follow(_ item: AVPlayerItem) -> AnyCancellable {
        // The selection once the video is ready and the remembered choice is on.
        // The player picks on its own until then; only changes from it are the viewer's.
        var group: AVMediaSelectionGroup?
        var settled: String?
        let task = Task { @MainActor in
            guard let legible = try? await item.asset.loadMediaSelectionGroup(for: .legible) else { return }
            while item.status == .unknown {
                guard (try? await Task.sleep(nanoseconds: 100_000_000)) != nil else { return }   // cancelled
            }
            apply(remembered, to: item, in: legible)
            group = legible
            settled = choice(of: item.currentMediaSelection.selectedMediaOption(in: legible))
        }
        let picks = NotificationCenter.default
            .publisher(for: AVPlayerItem.mediaSelectionDidChangeNotification, object: item)
            .receive(on: DispatchQueue.main)
            .sink { _ in
                guard let group, settled != nil else { return }
                let now = choice(of: item.currentMediaSelection.selectedMediaOption(in: group))
                if now != settled {
                    remembered = now
                    settled = now
                }
            }
        return AnyCancellable { task.cancel(); picks.cancel() }
    }

    static func apply(_ choice: String?, to item: AVPlayerItem, in group: AVMediaSelectionGroup) {
        guard let choice else { return }   // never chosen: leave AVFoundation's pick
        if choice.isEmpty {
            item.select(nil, in: group)
        } else if let option = option(for: choice, in: group.options) {
            item.select(option, in: group)
        }
    }

    /// The option to show for a language: the full track rather than the one
    /// that only carries forced lines (foreign dialogue, signs).
    static func option(for language: String, in options: [AVMediaSelectionOption]) -> AVMediaSelectionOption? {
        let matching = options.filter { self.language(of: $0) == language }
        return matching.first { !$0.hasMediaCharacteristic(.containsOnlyForcedSubtitles) } ?? matching.first
    }

    /// What a selection means as a choice to remember: forced-only lines are
    /// what the player shows with subtitles off, so they count as off.
    static func choice(of option: AVMediaSelectionOption?) -> String {
        guard let option, !option.hasMediaCharacteristic(.containsOnlyForcedSubtitles) else { return "" }
        return language(of: option) ?? ""
    }

    static func language(of option: AVMediaSelectionOption) -> String? {
        guard let tag = option.extendedLanguageTag ?? option.locale?.identifier else { return nil }
        return Locale(identifier: Locale.canonicalLanguageIdentifier(from: tag)).languageCode
    }
}
