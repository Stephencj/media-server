#!/bin/bash
# Tests SubtitleChoice (MediaServerMobile/Services/SubtitleChoice.swift) against real
# AVFoundation on this Mac: a generated MP4 with English and Spanish subtitle tracks,
# packaged as the server's remux packages them (mov_text, not default). Exit 0: pass.
#
#   apps/ios-mobile/tests/subtitle-choice/run.sh
set -euo pipefail
HERE="$(cd "$(dirname "$0")" && pwd)"
WORK="$(mktemp -d)"
trap 'rm -rf "$WORK"' EXIT

for lang in eng spa; do
	python3 -I -c "
import sys
print('\n'.join(f'{i+1}\n00:00:{i*2:02d},000 --> 00:00:{i*2+2:02d},000\n{sys.argv[1]} line {i+1}\n' for i in range(5)))" "$lang" > "$WORK/$lang.srt"
done
ffmpeg -nostdin -v error -y -f lavfi -i "testsrc2=s=320x180:r=25:d=10" -f lavfi -i "sine=d=10" \
	-i "$WORK/eng.srt" -i "$WORK/spa.srt" -map 0:v -map 1:a -map 2:0 -map 3:0 \
	-c:v libx264 -preset ultrafast -pix_fmt yuv420p -c:a aac -c:s mov_text \
	-metadata:s:s:0 language=eng -metadata:s:s:1 language=spa -disposition:s:0 0 -disposition:s:1 0 \
	-movflags +faststart "$WORK/subs.mp4"

xcrun swiftc -swift-version 5 -o "$WORK/test" "$HERE/../../MediaServerMobile/Services/SubtitleChoice.swift" "$HERE/main.swift"
"$WORK/test" "$WORK/subs.mp4"
