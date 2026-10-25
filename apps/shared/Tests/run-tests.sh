#!/usr/bin/env bash
# Checks for the code the iPhone and Apple TV apps share: compiled for the Mac and run.
set -euo pipefail
here="$(cd "$(dirname "$0")" && pwd)"
out="$(mktemp -d)"
trap 'rm -rf "$out"' EXIT
swiftc "$here/../ShowExtras.swift" "$here/main.swift" -o "$out/shared-tests"
"$out/shared-tests"
