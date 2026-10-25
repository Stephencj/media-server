import XCTest

/// A show's extras on its page: commentaries on their episodes, the season's
/// extras and the whole show's, each kind opening to play. Runs against the
/// real library, so needs the app installed on the simulator and the server
/// reachable.
final class ShowExtrasUITests: XCTestCase {
    private let app = XCUIApplication(bundleIdentifier: "com.mediaserver.mobile")

    override func setUp() {
        continueAfterFailure = false
        app.launch()
    }

    override func tearDown() {
        app.terminate()
    }

    /// With SHOTS_DIR set (TEST_RUNNER_SHOTS_DIR to xcodebuild), saves a
    /// screenshot there, for looking over by eye.
    private func shot(_ name: String) {
        guard let dir = ProcessInfo.processInfo.environment["SHOTS_DIR"] else { return }
        try? XCUIScreen.main.screenshot().pngRepresentation
            .write(to: URL(fileURLWithPath: dir).appendingPathComponent("\(name).png"))
    }

    private func element(_ id: String) -> XCUIElement {
        app.descendants(matching: .any).matching(identifier: id).firstMatch
    }

    /// Scrolls until `target` is on screen, or gives up.
    private func scroll(to target: XCUIElement, max: Int = 25) {
        for _ in 0..<max where !(target.exists && target.isHittable) {
            app.swipeUp()
        }
    }

    private func openShow(_ title: String) {
        app.tabBars.buttons["TV Shows"].tap()
        let show = app.staticTexts[title].firstMatch
        scroll(to: show)
        XCTAssertTrue(show.waitForExistence(timeout: 20), "\(title) is in the library")
        show.tap()
        XCTAssertTrue(app.staticTexts["Episodes"].waitForExistence(timeout: 20), "\(title)'s page opened")
    }

    func testCommentaryOnItsEpisodeAndExtrasBySeason() {
        openShow("Futurama")

        // Season 1, episode 1 has a commentary, on its row.
        let commentary = element("commentary-S01E01-1")
        XCTAssertFalse(element("commentary-S01E01-2").exists, "and only the one")
        XCTAssertTrue(commentary.waitForExistence(timeout: 20), "S01E01's commentary is on its row")
        shot("1-episode-commentary")

        // Below the episodes: this season's extras, by kind, folded.
        let seasonHeading = app.staticTexts["Season 1 Extras"]
        scroll(to: seasonHeading)
        XCTAssertTrue(seasonHeading.exists, "the season's extras")
        let commentaries = element("extrasGroup-Commentaries")
        scroll(to: commentaries)
        XCTAssertTrue(commentaries.exists)
        commentaries.tap()
        let pilot = app.staticTexts["Space Pilot 3000"].firstMatch
        scroll(to: pilot, max: 5)
        shot("2-season-extras-open")
        XCTAssertTrue(pilot.exists, "a commentary listed by its episode's name, not its file's")

        // The whole show's extras after that.
        let wholeShow = app.staticTexts["Extras"]
        scroll(to: wholeShow)
        XCTAssertTrue(wholeShow.exists, "the whole show's extras")
        shot("3-whole-show-extras")

        // Playing one opens the player.
        let back = element("commentary-S01E01-1")
        for _ in 0..<25 where !(back.exists && back.isHittable) {
            app.swipeDown()
        }
        back.tap()
        XCTAssertTrue(app.otherElements["Video"].waitForExistence(timeout: 15)
                      || app.buttons["Close"].waitForExistence(timeout: 5)
                      || app.buttons["Done"].waitForExistence(timeout: 5),
                      "the commentary plays")
        sleep(3)
        shot("4-playing-commentary")
    }

    func testAShowWithNoExtrasShowsNone() {
        openShow("Taskmaster")
        sleep(5)
        XCTAssertFalse(app.staticTexts["Extras"].exists)
        XCTAssertFalse(element("extrasGroup-Commentaries").exists)
    }
}
