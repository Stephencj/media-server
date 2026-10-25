#!/usr/bin/env bash
# The Apple TV app's checks that need no simulator: compiled for the Mac and run.
set -euo pipefail
here="$(cd "$(dirname "$0")" && pwd)"
out="$(mktemp -d)"
trap 'rm -rf "$out"' EXIT
swiftc -O "$here/../MediaPlayer/ViewModels/ChannelTimeshift.swift" "$here/main.swift" -o "$out/timeshift-tests"
"$out/timeshift-tests"
