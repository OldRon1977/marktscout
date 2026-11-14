#!/bin/sh
# Baut Marktscout für Linux und Windows (amd64) nach dist/.
# Aufruf: scripts/build.sh [version]   (Standard: git describe)
set -eu

cd "$(dirname "$0")/.."
version="${1:-$(git describe --tags --always --dirty 2>/dev/null || echo dev)}"
ldflags="-s -w -X main.version=$version"

mkdir -p dist
CGO_ENABLED=0 GOOS=linux   GOARCH=amd64 go build -trimpath -ldflags "$ldflags" -o dist/marktscout-linux-amd64       ./cmd/marktscout
# -H=windowsgui: kein Konsolenfenster unter Windows
CGO_ENABLED=0 GOOS=windows GOARCH=amd64 go build -trimpath -ldflags "$ldflags -H=windowsgui" -o dist/marktscout-windows-amd64.exe ./cmd/marktscout

ls -l dist/
