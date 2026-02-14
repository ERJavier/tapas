#!/usr/bin/env sh
# Install tapas so you can run "tapas" from anywhere.
# Run from the repo root, or use: make install

set -e

if [ -n "$1" ] && [ "$1" = "--remote" ]; then
  echo "Installing tapas from github.com/javiercepeda/tapas@latest..."
  go install github.com/javiercepeda/tapas@latest
else
  if [ ! -f "go.mod" ]; then
    echo "Run this from the tapas repo root (where go.mod is)."
    exit 1
  fi
  echo "Installing tapas from current directory..."
  go install .
fi

BIN="$(go env GOBIN)"
[ -z "$BIN" ] && BIN="$(go env GOPATH)/bin"
echo "Installed. Run: tapas"
echo "(binary: $BIN/tapas)"
