import AVFoundation
import Combine

// Harness for SubtitleChoice; run by run.sh, which makes the video.
let url = URL(fileURLWithPath: CommandLine.arguments[1])
var failures = 0
func check(_ ok: Bool, _ what: String) {
    print(ok ? "ok   \(what)" : "FAIL \(what)")
    if !ok { failures += 1 }
}

@MainActor
func selected(_ item: AVPlayerItem) async -> AVMediaSelectionOption? {
    guard let group = try? await item.asset.loadMediaSelectionGroup(for: .legible) else { return nil }
    return item.currentMediaSelection.selectedMediaOption(in: group)
}

/// A new video, as each programme or playlist item is: in a player (the item only
/// reports selection changes there), followed, and given time to settle.
var players: [AVPlayer] = []
@MainActor
func newVideo() async -> (AVPlayerItem, AnyCancellable) {
    let item = AVPlayerItem(asset: AVURLAsset(url: url))
    players.append(AVPlayer(playerItem: item))
    let follow = SubtitleChoice.follow(item)
    while item.status == .unknown { try? await Task.sleep(nanoseconds: 100_000_000) }
    try? await Task.sleep(nanoseconds: 500_000_000)
    return (item, follow)
}

@MainActor
func pick(_ language: String, forced: Bool, in item: AVPlayerItem) async {
    let group = try! await item.asset.loadMediaSelectionGroup(for: .legible)!
    let option = group.options.first {
        SubtitleChoice.language(of: $0) == language && $0.hasMediaCharacteristic(.containsOnlyForcedSubtitles) == forced
    }!
    item.select(option, in: group)   // what the player's menu does
    try? await Task.sleep(nanoseconds: 300_000_000)
}

@MainActor
func main() async {
    UserDefaults.standard.removeObject(forKey: "subtitleChoice")
    defer { UserDefaults.standard.removeObject(forKey: "subtitleChoice") }

    let group = try! await AVURLAsset(url: url).loadMediaSelectionGroup(for: .legible)!
    check(Set(group.options.compactMap(SubtitleChoice.language(of:))) == ["en", "es"], "reads both tracks' languages (eng -> en, spa -> es)")
    let full = SubtitleChoice.option(for: "es", in: group.options)
    check(full.map { !$0.hasMediaCharacteristic(.containsOnlyForcedSubtitles) && SubtitleChoice.language(of: $0) == "es" } == true,
          "picks the full Spanish track, not its forced-only one")

    // Never chosen: AVFoundation's own pick (the system caption setting) stands, nothing is saved.
    let (a, followA) = await newVideo()
    check(SubtitleChoice.remembered == nil, "nothing remembered before the viewer chooses")

    // The viewer turns on Spanish in the menu; the next video starts with it.
    await pick("es", forced: false, in: a)
    check(SubtitleChoice.remembered == "es", "remembers Spanish picked in the menu")
    followA.cancel()
    let (b, followB) = await newVideo()
    let onB = await selected(b)
    check(onB.map { SubtitleChoice.language(of: $0) == "es" && !$0.hasMediaCharacteristic(.containsOnlyForcedSubtitles) } == true,
          "the next video starts with Spanish subtitles on")

    // Switching language in the next video carries on too.
    await pick("en", forced: false, in: b)
    followB.cancel()
    let (c, followC) = await newVideo()
    check((await selected(c)).flatMap(SubtitleChoice.language(of:)) == "en", "a later change (English) carries to the video after")

    // Off: the forced-only lines the player keeps showing count as off.
    await pick("en", forced: true, in: c)
    check(SubtitleChoice.remembered == "", "forced-only lines count as subtitles off")
    followC.cancel()
    let (d, followD) = await newVideo()
    check(await selected(d) == nil, "the next video starts with subtitles off")
    followD.cancel()
    print(failures == 0 ? "PASS" : "\(failures) FAILED")
    exit(failures == 0 ? 0 : 1)
}

Task { await main() }
RunLoop.main.run()
