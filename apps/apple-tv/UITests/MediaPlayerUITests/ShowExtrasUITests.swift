import XCTest

/// A show's extras on its page: a Commentary card beside an episode that has
/// one, and the season's and whole show's extras on shelves by kind. Runs
/// against the real library: needs the app installed on the simulator and
/// the server reachable.
final class ShowExtrasUITests: XCTestCase {
    private let app = XCUIApplication(bundleIdentifier: "com.mediaserver.tv")
    private let remote = XCUIRemote.shared

    override func setUp() {
        continueAfterFailure = false
    }

    override func tearDown() {
        app.terminate()
    }

    private func open(_ title: String) {
        app.launchArguments = ["-tab", "2", "-open-title", title]
        app.launch()
    }

    private func element(_ id: String) -> XCUIElement {
        app.descendants(matching: .any).matching(identifier: id).firstMatch
    }

    private func shot(_ name: String) {
        guard let dir = ProcessInfo.processInfo.environment["SHOTS_DIR"] else { return }
        try? XCUIScreen.main.screenshot().pngRepresentation
            .write(to: URL(fileURLWithPath: dir).appendingPathComponent("\(name).png"))
    }

    /// Presses down until `target` has focus, or gives up.
    private func moveDown(to target: XCUIElement, max: Int = 40) -> Bool {
        for _ in 0..<max {
            if target.exists && target.hasFocus { return true }
            remote.press(.down)
        }
        return target.exists && target.hasFocus
    }

    func testCommentaryBesideItsEpisodeAndExtrasBySeason() {
        open("Futurama")
        let commentary = element("commentary-S01E01-1")
        XCTAssertTrue(commentary.waitForExistence(timeout: 30), "S01E01's commentary is beside it")
        XCTAssertFalse(element("commentary-S01E01-2").exists, "and only the one")

        // Down to the first episode, then one press right is its commentary.
        let firstEpisode = app.buttons.containing(.staticText, identifier: "Space Pilot 3000").firstMatch
        XCTAssertTrue(moveDown(to: firstEpisode, max: 12), "the first episode takes focus")
        remote.press(.right)
        XCTAssertTrue(commentary.hasFocus, "one press right of the episode is its commentary")
        shot("tv-1-episode-commentary")

        // Further down: the season's extras, a shelf per kind.
        let shelf = element("extrasShelf-Commentaries")
        for _ in 0..<40 where !shelf.exists || !shelf.isHittable { remote.press(.down) }
        XCTAssertTrue(app.staticTexts["Season 1 Extras"].exists, "the season's extras")
        XCTAssertTrue(shelf.exists, "commentaries on a shelf")
        shot("tv-2-season-extras")
        for _ in 0..<12 { remote.press(.down) }
        XCTAssertTrue(app.staticTexts["Extras"].exists, "then the whole show's")
        shot("tv-3-whole-show-extras")

        // Back up and play the commentary.
        // Rows off screen are unloaded, so look each time before asking.
        for _ in 0..<60 where !(commentary.exists && commentary.hasFocus) {
            remote.press(.up)
            if firstEpisode.exists && firstEpisode.hasFocus { remote.press(.right) }
        }
        XCTAssertTrue(commentary.exists && commentary.hasFocus, "back on the commentary")
        remote.press(.select)
        sleep(8)
        shot("tv-4-playing-commentary")
        XCTAssertFalse(commentary.isHittable, "the player covers the page")
    }

    func testAShowWithNoExtrasShowsNone() {
        open("Taskmaster")
        XCTAssertTrue(app.staticTexts["Episodes"].waitForExistence(timeout: 30))
        sleep(5)
        XCTAssertFalse(app.staticTexts["Extras"].exists)
        XCTAssertFalse(element("extrasShelf-Commentaries").exists)
    }
}
