# Changelog

Deutsch: [CHANGELOG.de.md](CHANGELOG.de.md)

## 0.1.0 – 2026-11-14

- kleinanzeigen.de connection: place suggestions, search, detail page, status; result count of following pages checked with tolerance (10 results or 5 %)
- marktplaats.nl connection: search, wanted-listing filter, detail page, status; paging until an empty page, the total count or the page limit
- Several places per search profile, Germany and Netherlands separately, own radius per place
- Price evaluation: target price, tolerance, minimum price, negotiation margin
- Text check: required/excluded words with three modes, negation, comparison detection
- Result list by place and price, without duplicates; status check active / reserved / deleted; deleting listings
- SQLite storage for search profiles, listings, matches and settings
- Local web UI, usable with mouse and keyboard
- Scheduling, desktop notification, sound, tray icon
- Log file with rotation and export
- Executables for Linux and Windows, portable without installation
- GitHub Actions: tests on Linux and Windows on every push and daily, UI tests with Playwright, build, release for version tags
- Pre-commit and commit-msg hooks and CI check against data identifying a user (email addresses, passwords, session cookies, keys, browser profiles, author addresses)
- Documentation in German and English, contribution rules, issue forms, pull request template
