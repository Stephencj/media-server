#!/bin/bash
# Build script for Media Server Mobile iOS app

set -e

echo "Building Media Server Mobile for iOS..."
echo ""

# Configuration
PROJECT="MediaServerMobile.xcodeproj"
SCHEME="MediaServerMobile"
CONFIGURATION="Debug"
SDK="iphonesimulator"
DESTINATION="platform=iOS Simulator,name=iPhone 17"

# Build
xcodebuild \
  -project "$PROJECT" \
  -scheme "$SCHEME" \
  -configuration "$CONFIGURATION" \
  -sdk "$SDK" \
  -destination "$DESTINATION" \
  clean build

echo ""
echo "Build completed successfully!"
echo ""
echo "To run in Xcode:"
echo "  1. open MediaServerMobile.xcodeproj"
echo "  2. Select target device/simulator"
echo "  3. Press Cmd+R"
echo ""
