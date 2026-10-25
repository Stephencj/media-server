import Foundation

// Checks for ChannelTimeshift, the rules for pausing and skipping on a
// channel. Run with ./run-tests.sh.

var failures = 0
func check(_ ok: Bool, _ what: String, line: Int = #line) {
    if ok { print("  ok  \(what)") } else { failures += 1; print("FAIL  \(what) (line \(line))") }
}
func near(_ a: TimeInterval, _ b: TimeInterval) -> Bool { abs(a - b) < 0.001 }

let t0 = Date(timeIntervalSince1970: 1_800_000_000)
func at(_ s: TimeInterval) -> Date { t0.addingTimeInterval(s) }

print("A channel tunes in live")
do {
    let shift = ChannelTimeshift()
    check(shift.isLive(at: t0), "live at first")
    check(!shift.isPaused, "playing at first")
    check(near(shift.behind(at: at(500)), 0), "playing on, stays live")
}

print("Pausing falls behind by as long as it's paused, and stays there")
do {
    var shift = ChannelTimeshift()
    shift.pause(at: at(10))
    check(shift.isPaused, "paused")
    check(near(shift.behind(at: at(70)), 60), "a minute paused is a minute behind, while still paused")
    shift.resume(at: at(130))
    check(near(shift.behind(at: at(130)), 120), "two minutes behind on resuming")
    check(near(shift.behind(at: at(5000)), 120), "and still two minutes behind much later: it never catches up by itself")
    check(!shift.isLive(at: at(5000)), "not live")
    check(shift.watching(at: at(5000)) == at(4880), "watching the channel as it was two minutes ago")
    shift.pause(at: at(5000)); shift.pause(at: at(5010))
    shift.resume(at: at(5030))
    check(near(shift.behind(at: at(5030)), 150), "pausing twice counts from the first press")
}

print("Skipping back falls behind; skipping forward catches up, never past live")
do {
    var shift = ChannelTimeshift()
    var landed = shift.skip(from: 600, by: -10, duration: 3600, at: t0)
    check(near(landed, 590), "back ten seconds")
    check(near(shift.behind(at: t0), 10), "ten seconds behind")
    landed = shift.skip(from: 590, by: 30, duration: 3600, at: t0)
    check(near(landed, 600), "forward thirty stops at live, ten on")
    check(shift.isLive(at: t0), "back to live")
    landed = shift.skip(from: 600, by: 30, duration: 3600, at: t0)
    check(near(landed, 600), "at live, forward goes nowhere")
    landed = shift.skip(from: 5, by: -10, duration: 3600, at: t0)
    check(near(landed, 0), "back stops at the start of the programme")
    check(near(shift.behind(at: t0), 5) && !shift.isLive(at: t0), "five seconds behind is behind")
    var nudge = ChannelTimeshift()
    _ = nudge.skip(from: 100, by: -2, duration: 3600, at: t0)
    check(nudge.isLive(at: t0) && near(nudge.behind(at: t0), 0), "two seconds is within the live tolerance: still live")
}

print("Forward stops at the end of the programme when live is past it")
do {
    var shift = ChannelTimeshift(behind: 900)
    let landed = shift.skip(from: 3500, by: 300, duration: 3600, at: t0)
    check(near(landed, 3599), "a second before the end")
    check(near(shift.behind(at: t0), 801), "still behind by the rest")
}

print("Skipping while paused keeps the pause going")
do {
    var shift = ChannelTimeshift()
    shift.pause(at: t0)
    let landed = shift.skip(from: 100, by: -10, duration: 3600, at: at(20))
    check(near(landed, 90), "back ten")
    check(shift.isPaused, "still paused")
    check(near(shift.behind(at: at(20)), 30), "twenty paused plus ten back")
    check(near(shift.behind(at: at(50)), 60), "the pause keeps counting")
    let forward = shift.skip(from: 90, by: 30, duration: 3600, at: at(50))
    check(near(forward, 120), "forward thirty, still well behind live")
    check(near(shift.behind(at: at(50)), 30), "thirty behind")
}

print("Jumping to live")
do {
    var shift = ChannelTimeshift(behind: 400)
    shift.pause(at: t0)
    shift.goLive()
    check(shift.isLive(at: at(100)) && !shift.isPaused, "live and playing")
    check(ChannelTimeshift(behind: 2).isLive(at: t0), "a couple of seconds behind counts as live")
}

print("What follows, behind live, when a programme ends")
do {
    let ids: [Int64] = [7, 8, 9, 7]
    let starts = [at(0), at(1800), at(3600), at(5400)]
    check(ChannelTimeshift.nextProgramme(ids: ids, starts: starts, ending: 8, watching: at(3600)) == 2,
          "the one after it in the guide")
    check(ChannelTimeshift.nextProgramme(ids: ids, starts: starts, ending: 8, watching: at(3590)) == 2,
          "even when its file ran a few seconds short of its listing")
    check(ChannelTimeshift.nextProgramme(ids: ids, starts: starts, ending: 7, watching: at(5410)) == nil
          || ChannelTimeshift.nextProgramme(ids: ids, starts: starts, ending: 7, watching: at(5410)) == 3,
          "the last in the window: falls back to what is on")
    check(ChannelTimeshift.nextProgramme(ids: ids, starts: starts, ending: 7, watching: at(1795)) == 1,
          "a row aired twice: the airing being watched, not the later one")
    check(ChannelTimeshift.nextProgramme(ids: ids, starts: starts, ending: 99, watching: at(2000)) == 1,
          "not in the guide: whatever is on at the time being watched")
}

print("Clock")
check(ChannelTimeshift.clock(65) == "1:05", "1:05")
check(ChannelTimeshift.clock(3725) == "1:02:05", "1:02:05")
check(ChannelTimeshift.clock(-4) == "0:00", "never negative")

if failures > 0 { print("\(failures) failed"); exit(1) }
print("All passed")
