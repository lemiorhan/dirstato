#!/bin/sh
# Usage: ./start.sh [path] [--min SIZE]
set -e
repo="$(cd "$(dirname "$0")" && pwd)"
bin="${TMPDIR:-/tmp}/dirstato-build/dirstato"
# Built outside the repo and run from the caller's directory: with no path,
# dirstato scans where you are, not the repo.
go build -C "$repo" -o "$bin" .
exec "$bin" "$@"
