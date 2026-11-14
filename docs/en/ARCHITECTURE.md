# Architecture

Deutsch: [../de/AUFBAU.md](../de/AUFBAU.md)

## Packages

| Package | Purpose |
|---|---|
| `cmd/marktscout` | entry point: options, data folder, server, tray, open browser, detect second instance |
| `internal/abruf` | HTTP client with Firefox user agent, 2–4 s interval per host, retry on network error/429/5xx |
| `internal/anzeige` | shared types (listing, price type, status) |
| `internal/kleinanzeigen` | place suggestions, search (HTML), detail page, status |
| `internal/marktplaats` | search (JSON), wanted-listing detection, detail page, status |
| `internal/preis` | upper limit, tolerance, minimum price, negotiation margin, site limits |
| `internal/textpruefung` | interface `Bewerter`; `Regelwerk`: required/excluded words, negation, comparison detection |
| `internal/auftrag` | search profile and input validation |
| `internal/speicher` | SQLite file: profiles, listings, matches, settings |
| `internal/suchlauf` | search run, status check, result list (evaluation, grouping, sorting) |
| `internal/web` | JSON API, embedded UI (`ui/`), job queue, scheduling, notifications |
| `internal/protokoll` | log file with rotation (2 MB, 3 older files), export |
| `internal/melden` | desktop notification (beeep), sound (WAV, system player) |
| `internal/symbol` | program icon as PNG and ICO |

## Search run

1. Reset the "new" flag of the profile's matches.
2. Per German place: search on kleinanzeigen.de with site price limits (minimum price to upper limit, raised by the negotiation margin).
3. Per Dutch place: search on marktplaats.nl; wanted listings are discarded.
4. Every listing is stored and linked to the profile. The same listing found from several places is kept once, with the shortest distance. Matches deleted by the user stay deleted.
5. Detail pages only for listings without full description that pass the price and title check.
6. Store time and errors of the run; count new visible matches; notification and sound if enabled.

The evaluation (price, text) is not stored but computed for every listing in the list from profile and listing. Changes to the profile take effect on existing matches immediately.

## Queue and scheduling

- One worker runs search runs and status checks one after another. The same job is not queued twice.
- Scheduling checks every 30 s whether a profile with an interval is due (last run + interval) and queues it. A failed run also sets the time.

## API

- Host must be `127.0.0.1`/`localhost` (against DNS rebinding).
- Modifying requests require the header `X-Marktscout: 1` (against requests from other websites).
- Content Security Policy: own scripts and styles only, images from `https:`.

## Build and release

- `scripts/build.sh [version]`: builds `dist/marktscout-linux-amd64` and `dist/marktscout-windows-amd64.exe` (CGO off, Windows without console window).
- CI job `build`: after the tests, artifact `marktscout` with the contents of `dist/`.
- CI job `release`: only for tags `v*`; creates the GitHub release with the `dist/` files and `SHA256SUMS`, notes from `scripts/release-notes.sh` (the version's sections from both changelogs, aborts if one is missing).

## Tests

| Area | Kind |
|---|---|
| `kleinanzeigen`, `marktplaats`, `suchlauf` (live test), `web` (place suggestions) | live against the real sites |
| `preis`, `textpruefung`, `suchlauf` (list), `auftrag` | real functions with real input |
| `speicher`, `protokoll` | real files in a temp folder |
| `web` | real server on 127.0.0.1 |
| `cmd/marktscout` | builds and runs the program |
| `scripts` | `datenschutz.sh` as real pre-commit and commit-msg hooks and over the history, in a temporary git repository (Linux only) |
| `tests/ui` (Playwright) | real program, real Chromium, live search |

Run:

```sh
go test ./...
npm ci && npx playwright test
```

Counter-check: every test must fail when the rule it checks is deliberately broken in the code (see [CONTRIBUTING.md](../../CONTRIBUTING.md)).
