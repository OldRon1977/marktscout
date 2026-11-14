# Marktscout

Searches **kleinanzeigen.de** and **marktplaats.nl** in several places at once, filters results by keywords and price, and reports new listings. Runs portably from a USB stick on Linux and Windows, without installation.

Deutsch: [README.de.md](README.de.md)

## Features

- **Several places in one search run:** separate place lists for Germany (place name or postcode, chosen from the site's suggestions) and the Netherlands (postcode), each place with its own radius.
- **Keywords:** required and excluded words, checked in title and full listing text. Excluded words can mark, filter by title or hard-filter. Negations ("kein Tausch", "nicht defekt") are recognised. A rule set recognises comparisons such as "fast wie eine RTX 2080".
- **Price:** target price with tolerance in % or €, minimum price, negotiation margin for negotiable listings.
- **Offers only:** "wanted" listings are excluded.
- **Result list** grouped by the listing's location, sorted by price, without duplicates; new, reserved and deleted listings labelled.
- **Saved listings** can be re-checked (active / reserved / deleted) and deleted.
- **Saved search profiles**, editable at any time.
- **Scheduled search** with sound and desktop notification, can be switched off; keeps running in the system tray.
- **Log file** export for bug reports.

Usage: [docs/en/USAGE.md](docs/en/USAGE.md)

## Download and start

Executables for each version: [Releases](https://github.com/OldRon1977/marktscout/releases). Builds of the current state: **Actions → CI → latest run → Artifacts → `marktscout`**. Alternatively build them yourself (see below).

1. Copy `marktscout-linux-amd64` and/or `marktscout-windows-amd64.exe` to the USB stick.
2. Start by double-click (Linux: run `chmod +x marktscout-linux-amd64` once).
3. The UI opens in the browser. Data is stored in the folder `daten` next to the executable.

The release also contains `SHA256SUMS` (checksums).

## Developer setup

```sh
git clone https://github.com/OldRon1977/marktscout.git
cd marktscout
git config core.hooksPath .githooks
git config user.email <id>+<name>@users.noreply.github.com
```

The hooks abort a commit containing data that identifies a user (email addresses, passwords, session cookies, keys, browser profiles), also in the commit message and as author address. The same check runs in CI over all files and all commits.

### Build and test

Requires Go (version in [go.mod](go.mod)); UI tests additionally require Node.js.

```sh
go test ./...                       # all Go tests, including live tests against the real sites
npm ci && npx playwright test       # UI tests: real program in a real browser
sh scripts/build.sh                 # dist/marktscout-linux-amd64, dist/marktscout-windows-amd64.exe
go run ./cmd/marktscout -daten /tmp/ms-daten   # start from source
```

GitHub Actions runs the Go tests on Linux and Windows and the UI tests on Linux on every push and once daily, then builds both executables ([.github/workflows/ci.yml](.github/workflows/ci.yml)). A version tag `v*` additionally creates the GitHub release.

Structure of the code: [docs/en/ARCHITECTURE.md](docs/en/ARCHITECTURE.md). How the sites respond: [docs/en/SITES.md](docs/en/SITES.md). Known obstacles: [docs/en/OBSTACLES.md](docs/en/OBSTACLES.md).

## Usage notice

kleinanzeigen.de and marktplaats.nl offer no public API. The tool requests the pages the way a browser does and pauses 2–4 seconds between requests to the same site. The sites' terms of use restrict automated access. Use at your own risk; intended for private use.

This project is not affiliated with kleinanzeigen.de or marktplaats.nl.

## Contributing

See [CONTRIBUTING.md](CONTRIBUTING.md).

## License

[GPL-3.0](LICENSE)
