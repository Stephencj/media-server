import Foundation

// Checks for ShowExtras, shared by the iPhone and Apple TV apps.
// Run with ./run-tests.sh.

var failures = 0
func check(_ ok: Bool, _ what: String, line: Int = #line) {
    if ok { print("  ok  \(what)") } else { failures += 1; print("FAIL  \(what) (line \(line))") }
}

func extra(_ id: Int64, _ title: String, _ category: String, season: Int? = nil, episode: Int? = nil, seconds: Int? = 300) -> ShowExtra {
    ShowExtra(id: id, title: title, category: category, duration: seconds, resolution: nil, fileSize: nil,
              seasonNumber: season, episodeNumber: episode)
}

print("Codes in file names")
check(ShowExtras.code(in: "Com1S01E01 - Pilot").map { [$0.season, $0.episode ?? -1] } == [1, 1], "Com1S01E01")
check(ShowExtras.code(in: "DSS01E12 Cloudy Chance of Murder").map { [$0.season, $0.episode ?? -1] } == [1, 12], "DSS01E12")
check(ShowExtras.code(in: "S01 - Deleted Scenes").map { [$0.season, $0.episode ?? -1] } == [1, -1], "a season alone")
check(ShowExtras.code(in: "S02BTS01").map { [$0.season, $0.episode ?? -1] } == [2, -1], "S02BTS01 is season 2")
check(ShowExtras.code(in: "Inside Breaking Bad Episode 501") == nil, "no code in an ordinary name")
check(ShowExtras.code(in: "Plot Holes 2") == nil, "a word ending in s before a number is no code")
check(ShowExtras.code(in: "LE - Fabulouse Making of Chappelle's Show Season 3ish") == nil, "\"Season 3\" spelled out is not a code")

print("Names without their codes")
check(ShowExtras.cleanTitle("Com1S01E01 - Pilot") == "Pilot", "Pilot")
check(ShowExtras.cleanTitle("DSS01E12 Cloudy Chance of Murder") == "Cloudy Chance of Murder", "Cloudy Chance of Murder")
check(ShowExtras.cleanTitle("S01 - Behind the Scenes") == "Behind the Scenes", "Behind the Scenes")
check(ShowExtras.cleanTitle("S01E01") == "S01E01", "a name that is only a code keeps it")
check(ShowExtras.cleanTitle("Inside Breaking Bad Episode 501") == "Inside Breaking Bad Episode 501", "an ordinary name is left alone")

print("Seasons and the whole show")
let all = [
    extra(1, "Com1S01E02 - Two", "commentary", season: 1, episode: 2),
    extra(2, "Com1S01E01 - Pilot", "commentary", season: 1, episode: 1),
    extra(3, "Com2S01E01 - Pilot", "commentary", season: 1, episode: 1),
    extra(4, "DSS01E12 Cloudy", "deleted_scene"),                     // season from its name
    extra(5, "S02 - Deleted Scenes", "deleted_scene"),
    extra(6, "Gag Reel", "gag_reel"),
    extra(7, "Mystery", "trailer"),                                    // a kind the apps don't know
    extra(8, "Com1S02E03 - Three", "commentary", season: 2, episode: 3),
]
let one = ShowExtras.season(1, of: all)
check(one.map(\.title) == ["Commentaries", "Deleted Scenes"], "season 1: commentaries first, then deleted scenes")
check(one.first?.extras.map(\.id) == [2, 3, 1], "commentaries in episode order, then by name")
check(one.last?.extras.map(\.id) == [4], "a deleted scene placed by its name")
check(ShowExtras.season(2, of: all).flatMap(\.extras).map(\.id).sorted() == [5, 8], "season 2")
let whole = ShowExtras.wholeShow(all)
check(whole.map(\.title) == ["Gag Reels", "More"], "the whole show: gag reel, and the unknown kind under More")
check(whole.flatMap(\.extras).map(\.id) == [6, 7], "nothing with a season")
check(ShowExtras.commentaries(season: 1, episode: 1, of: all).map(\.id).sorted() == [2, 3], "an episode's two commentaries")
check(ShowExtras.commentaries(season: 1, episode: 12, of: all).isEmpty, "a deleted scene is not a commentary")

print("The line under a name")
check(all[1].detail == "S01E01 \u{00B7} 5m 0s", "episode and length")
check(extra(9, "x", "other", seconds: 4000).detail == "1h 6m", "length alone")
check(extra(9, "x", "other", seconds: nil).detail == "", "nothing known")

if failures > 0 { print("\(failures) failed"); exit(1) }
print("All passed")
