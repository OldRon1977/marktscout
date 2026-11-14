#!/bin/sh
# Gibt die Release-Notiz für einen Versions-Tag aus, zusammengesetzt aus
# CHANGELOG.de.md und CHANGELOG.md. Bricht ab, wenn ein Changelog keinen
# Abschnitt für die Version hat.
# Aufruf: scripts/release-notes.sh v1.2.3
set -eu

cd "$(dirname "$0")/.."
tag="${1:?Aufruf: scripts/release-notes.sh <tag>}"
version="${tag#v}"
repo="https://github.com/OldRon1977/marktscout/blob/$tag"

abschnitt() {
	# Zeilen nach "## <version> " bis zur nächsten "## "-Überschrift, ohne Leerzeilen am Rand
	awk -v v="$version" '
		/^## / { drin = (index($0, "## " v " ") == 1); next }
		drin { print }
	' "$1" | sed -e '/./,$!d' | sed -e ':a' -e '/^\n*$/{$d;N;ba' -e '}'
}

de="$(abschnitt CHANGELOG.de.md)"
en="$(abschnitt CHANGELOG.md)"
[ -n "$de" ] || { echo "CHANGELOG.de.md: kein Abschnitt für $version" >&2; exit 1; }
[ -n "$en" ] || { echo "CHANGELOG.md: kein Abschnitt für $version" >&2; exit 1; }

cat <<EOF
**Deutsch**

$de

Bedienung: [docs/de/BEDIENUNG.md]($repo/docs/de/BEDIENUNG.md)

**English**

$en

Usage: [docs/en/USAGE.md]($repo/docs/en/USAGE.md)

| Datei / File | System |
|---|---|
| \`marktscout-linux-amd64\` | Linux x86-64 (\`chmod +x\` einmal / once) |
| \`marktscout-windows-amd64.exe\` | Windows x86-64 |
| \`SHA256SUMS\` | Prüfsummen / checksums |
EOF
