# Contributing

Deutsch: [CONTRIBUTING.de.md](CONTRIBUTING.de.md)

Contributions are welcome. The following rules apply.

## Setup

```sh
git clone https://github.com/OldRon1977/marktscout.git
cd marktscout
git config core.hooksPath .githooks
git config user.email <id>+<name>@users.noreply.github.com
```

## Personal data

- Never commit anything that identifies a user: email addresses, platform user names and passwords, session cookies, browser profiles, API keys, tokens.
- Author and committer address of every commit is the GitHub noreply address: `git config user.email <id>+<name>@users.noreply.github.com` (found under GitHub → Settings → Emails).
- [scripts/datenschutz.sh](scripts/datenschutz.sh) checks this: as pre-commit hook the staged changes and author address, as commit-msg hook the message, in CI all files and all commits. Only example addresses (`@example.com`, `*.invalid`) and GitHub noreply addresses are allowed. It reports location and kind, never the value.
- Test output must not contain personal data from listings (names, phone numbers).

## Tests

- **No mocks.** No fake servers, no simulated responses, no replaced functions.
- Tests call the real code with real input. Tests of the site connectors run against the real sites kleinanzeigen.de and marktplaats.nl.
- Real data is allowed: responses fetched from the real sites and saved. Each saved response records its source and fetch date. Names and contact data are removed before saving.
- A live test checks that the saved responses still match the current structure of the site. If the site differs, it fails and the data is fetched again.
- Every test must be meaningful: it is run once against deliberately broken code and must fail. A test that can never fail is not accepted.
- Live tests check structure and plausibility (e.g. "results have title, price and location"), not specific listings that may be gone tomorrow.
- Live tests pause between requests and only fetch as much as needed.
- Run: `go test ./...` and `npm ci && npx playwright test`.
- Every new feature comes with tests. A pull request is merged only when all tests pass.
- `main` is protected: changes only via pull request; the checks `test (ubuntu-latest)`, `test (windows-latest)`, `ui` and `build` must pass. No force push. Workflows of first-time contributors start after approval by the maintainer.
- Security vulnerabilities: report privately via "Security → Report a vulnerability", not as a public issue.

## Issues

- Bugs, feature requests and questions via the forms under "New issue".
- For bugs, attach the log file (UI: "Save log file").

## Documentation

- All documentation exists in German and English, in separate files linking to each other (`README.md`/`README.de.md`, `CHANGELOG.md`/`CHANGELOG.de.md`, `docs/en/…`/`docs/de/…`). Both versions are changed in the same commit.
- Only the current state is documented. If the documentation differs from the code, the documentation is corrected.
- No history: nothing about what existed before or what was discarded.
- Anything proven not to work (observed live or shown by a test) is recorded with the reason in `docs/en/OBSTACLES.md` / `docs/de/HUERDEN.md`, so nobody tries the same approach again.
- Every change gets one bullet point in the changelog (both languages), without justification.
- Factual tone, no exclamation marks for emphasis.

## Release

1. Rename the changelog section in both languages to `## X.Y.Z – YYYY-MM-DD`.
2. Commit, then `git tag -a vX.Y.Z -m "Version X.Y.Z"` and `git push origin main vX.Y.Z`.
3. The CI runs tests and build and then creates the GitHub release with both executables, `SHA256SUMS` and both changelog sections as notes (`scripts/release-notes.sh`). The release job aborts if a changelog has no section for the version.

## Handling the sites

- Requests are throttled, no parallel crawling.
- No bypassing of captchas or access blocks.

## License

By contributing, you agree that your contribution is published under the [GPL-3.0](LICENSE).
