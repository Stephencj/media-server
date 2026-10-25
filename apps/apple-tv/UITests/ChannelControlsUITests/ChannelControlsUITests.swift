import XCTest

/// Pause, skip and Live on a channel, pressed on the remote the way a viewer
/// would. Needs the app installed on the simulator and the server reachable.
final class ChannelControlsUITests: XCTestCase {
    private let app = XCUIApplication(bundleIdentifier: "com.mediaserver.tv")
    private let remote = XCUIRemote.shared

    override func setUp() {
        continueAfterFailure = false
        app.launchArguments = ["-open-channel", "0"]
        app.launch()
    }

    override func tearDown() {
        app.terminate()
    }

    private func element(_ id: String) -> XCUIElement {
        app.descendants(matching: .any).matching(identifier: id).firstMatch
    }

    /// Seconds in a badge's "m:ss behind live".
    private func secondsBehind(_ id: String) -> Int {
        // The label may lead with its icon's name.
        let label = element(id).label
        guard let range = label.range(of: #"\d+(:\d\d)+"#, options: .regularExpression) else {
            XCTFail("no time in \(id)'s label \"\(label)\"")
            return -1
        }
        return label[range].split(separator: ":").reduce(0) { $0 * 60 + (Int($1) ?? 0) }
    }

    /// With SHOTS_DIR set (TEST_RUNNER_SHOTS_DIR to xcodebuild), saves a
    /// screenshot there, for looking over the bar by eye.
    private func shot(_ name: String) {
        guard let dir = ProcessInfo.processInfo.environment["SHOTS_DIR"] else { return }
        try? XCUIScreen.main.screenshot().pngRepresentation
            .write(to: URL(fileURLWithPath: dir).appendingPathComponent("\(name).png"))
    }

    private func showBar() {
        if !element("transportBar").exists { remote.press(.select) }
        XCTAssertTrue(element("transportBar").waitForExistence(timeout: 5), "the bar comes up")
    }

    func testPauseSkipAndJumpToLive() {
        // Tuned in: the bar is up and the channel is live.
        XCTAssertTrue(element("liveBadge").waitForExistence(timeout: 30), "a channel starts live")
        sleep(3)
        shot("1-live")

        // Paused, the bar stays up and the time behind grows.
        remote.press(.playPause)
        XCTAssertTrue(element("behindBadge").waitForExistence(timeout: 6), "pausing falls behind live")
        sleep(7)
        XCTAssertTrue(element("transportBar").exists, "the bar stays up while paused")
        shot("2-paused")
        let paused = secondsBehind("behindBadge")
        XCTAssertGreaterThanOrEqual(paused, 6, "behind by the time paused")

        // Playing again it stays that far behind, and the corner says so once
        // the bar has gone.
        remote.press(.playPause)
        sleep(7)
        XCTAssertFalse(element("transportBar").exists, "the bar goes after playing again")
        XCTAssertTrue(element("behindTag").exists, "the corner shows how far behind")
        shot("3-behind-corner")
        let resumed = secondsBehind("behindTag")
        XCTAssertGreaterThanOrEqual(resumed, paused, "at least as far behind as when it was read paused")
        sleep(6)
        XCTAssertLessThanOrEqual(abs(secondsBehind("behindTag") - resumed), 1, "and playing on, it stays that far behind")

        // Forward thirty catches up to live, no further.
        remote.press(.right)
        XCTAssertTrue(element("liveBadge").waitForExistence(timeout: 5), "skipping forward catches up to live")

        // Back twice is twenty behind.
        remote.press(.left)
        remote.press(.left)
        XCTAssertTrue(element("behindBadge").waitForExistence(timeout: 5), "skipping back falls behind")
        sleep(2)
        let back = secondsBehind("behindBadge")
        XCTAssertTrue((18...23).contains(back), "two skips back is twenty behind (was \(back))")

        // Holding the center button offers Live.
        remote.press(.select, forDuration: 1.2)
        let live = app.buttons["jumpToLive"]
        XCTAssertTrue(live.waitForExistence(timeout: 5), "Jump to Live is offered while behind")
        shot("4-options")
        remote.press(.select)
        sleep(3)
        showBar()
        XCTAssertTrue(element("liveBadge").waitForExistence(timeout: 5), "jumping to live is live")

        // Behind, then a channel up and back down: live again.
        remote.press(.left)
        XCTAssertTrue(element("behindBadge").waitForExistence(timeout: 5))
        remote.press(.up)
        sleep(3)
        remote.press(.down)
        sleep(4)
        showBar()
        XCTAssertTrue(element("liveBadge").waitForExistence(timeout: 10), "changing channel lands live")
    }

    /// A programme ending while behind: the next follows on, still behind.
    /// Needs a channel whose programme ends within a few minutes, given as
    /// ENDING_CHANNEL (its place in the list), ENDS_AT (Unix time) and
    /// NEXT_TITLE, each passed to xcodebuild with a TEST_RUNNER_ prefix.
    func testProgrammeEndingWhileBehind() throws {
        let env = ProcessInfo.processInfo.environment
        guard let channel = env["ENDING_CHANNEL"], let endsAt = env["ENDS_AT"].flatMap(TimeInterval.init),
              let next = env["NEXT_TITLE"] else {
            throw XCTSkip("set ENDING_CHANNEL, ENDS_AT and NEXT_TITLE to run")
        }
        app.terminate()
        app.launchArguments = ["-open-channel", channel]
        app.launch()
        XCTAssertTrue(element("liveBadge").waitForExistence(timeout: 30))

        remote.press(.playPause)
        sleep(20)
        remote.press(.playPause)
        sleep(3)
        let behind = secondsBehind("behindTag")
        XCTAssertGreaterThanOrEqual(behind, 19)

        // Past the end on the channel's clock, then past it for the viewer.
        let wait = endsAt + Double(behind) + 15 - Date().timeIntervalSince1970
        if wait > 0 { sleep(UInt32(wait)) }
        shot("5-next-programme")
        XCTAssertTrue(element("behindTag").exists, "still behind after the programme changed")
        XCTAssertLessThanOrEqual(abs(secondsBehind("behindTag") - behind), 3, "by the same amount")
        showBar()
        XCTAssertEqual(element("nowPlaying").label, "Now: \(next)", "the next programme, from its start")
        let elapsed = element("elapsedTime").label
        shot("6-next-programme-bar")
        XCTAssertTrue(elapsed.hasPrefix("0:"), "near its beginning (bar shows \(elapsed))")
    }

    func testForwardAtLiveGoesNowhere() {
        XCTAssertTrue(element("liveBadge").waitForExistence(timeout: 30))
        remote.press(.right)
        sleep(2)
        XCTAssertTrue(element("liveBadge").exists, "already live: forward stays live")
        XCTAssertFalse(element("behindBadge").exists)
    }
}
