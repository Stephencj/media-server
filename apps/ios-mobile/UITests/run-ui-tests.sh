#!/usr/bin/env bash
# Builds the iPhone app, installs it on an iPhone simulator, and runs the UI
# tests against it. SIM= picks the simulator (default: the first booted iPhone).
set -euo pipefail
here="$(cd "$(dirname "$0")" && pwd)"
SIM="${SIM:-$(xcrun simctl list devices booted | awk -F'[()]' '/iPhone/ {print $2; exit}')}"
[ -n "$SIM" ] || { echo "Boot an iPhone simulator first, or set SIM=<udid>"; exit 1; }
DD="${DD:-$here/.build}"
xcodebuild -quiet -project "$here/../MediaServerMobile.xcodeproj" -scheme MediaServerMobile -configuration Debug \
	-destination "platform=iOS Simulator,id=$SIM" -derivedDataPath "$DD/app" build
xcrun simctl install "$SIM" "$DD/app/Build/Products/Debug-iphonesimulator/MediaServerMobile.app"
(cd "$here" && xcodegen -q)
xcodebuild -project "$here/MediaServerMobileUITests.xcodeproj" -scheme ShowExtrasUITests \
	-destination "platform=iOS Simulator,id=$SIM" -derivedDataPath "$DD/tests" test "$@" 2>&1 \
	| grep -E "Test Case|error|failed|passed|XCTAssert|TEST (SUCCEEDED|FAILED)"
