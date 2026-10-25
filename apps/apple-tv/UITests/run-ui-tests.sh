#!/usr/bin/env bash
# Builds the Apple TV app, installs it on a tvOS simulator, and runs the UI tests
# against it. SIM= picks the simulator (default: the first booted Apple TV).
set -euo pipefail
here="$(cd "$(dirname "$0")" && pwd)"
SIM="${SIM:-$(xcrun simctl list devices booted | awk -F'[()]' '/Apple TV/ {print $2; exit}')}"
[ -n "$SIM" ] || { echo "Boot an Apple TV simulator first, or set SIM=<udid>"; exit 1; }
DD="${DD:-$here/.build}"
xcodebuild -quiet -project "$here/../MediaPlayer.xcodeproj" -scheme MediaPlayer -configuration Debug \
	-destination "platform=tvOS Simulator,id=$SIM" -derivedDataPath "$DD/app" build
xcrun simctl install "$SIM" "$DD/app/Build/Products/Debug-appletvsimulator/MediaPlayer.app"
(cd "$here" && xcodegen -q)
xcodebuild -project "$here/MediaPlayerUITests.xcodeproj" -scheme ChannelControlsUITests \
	-destination "platform=tvOS Simulator,id=$SIM" -derivedDataPath "$DD/tests" test "$@" 2>&1 \
	| grep -E "Test Case|error|failed|passed|XCTAssert|TEST (SUCCEEDED|FAILED)"
