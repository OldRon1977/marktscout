# Marktscout – Arbeitsgrundlage

## Stand

Wird bei jedem Arbeitsschritt überschrieben, nicht ergänzt.

- Funktionsumfang laut Anforderungen umgesetzt, alle Go- und Playwright-Tests grün, Gegenprobe (gezielte Verfälschung) rot.
- Version 0.1.0 ist das aktuelle Release (Tag `v0.1.0`, vom Release-Job angelegt): beide Programmdateien, `SHA256SUMS`.
- `main` geschützt: Änderungen nur per Branch und PR, Prüfungen test (ubuntu/windows), ui, build müssen grün sein.
- Release-Job in der CI: Tag `v*` → Tests, Build, GitHub-Release mit Notiz aus beiden Changelogs (`scripts/release-notes.sh`). Ablauf in `CONTRIBUTING.de.md`.
- Keine Sprachmodell-Anbindung; nur auf ausdrücklichen Auftrag des Nutzers einbauen.
- Nicht von Hand geprüft: Tray/Meldung/Ton unter Windows, Start von FAT32/exFAT-Stick.
- Aufbau: `docs/de/AUFBAU.md`; Seiten: `docs/de/SEITEN.md`; Hürden: `docs/de/HUERDEN.md`.

## Dokumentation

- Jede Doku gibt es in Deutsch und Englisch, in getrennten Dateien mit gegenseitigem Link:
  - `README.md` (en) / `README.de.md` (de)
  - `CHANGELOG.md` (en) / `CHANGELOG.de.md` (de)
  - `docs/en/…` / `docs/de/…`
- Beide Sprachfassungen werden im selben Commit geändert und haben denselben Inhalt.
- Dokumentiert wird nur der aktuelle Stand. Weicht die Doku vom Code ab, wird die Doku korrigiert.
- Keine Historie: nichts darüber, was es früher gab, was verworfen oder entfernt wurde und warum.
- Ausnahme: Was nachweislich nicht funktioniert (live gesehen oder per Test belegt), wird mit Grund dokumentiert, damit derselbe Weg nicht erneut versucht wird. Ablage: `docs/de/HUERDEN.md` / `docs/en/OBSTACLES.md`.
- Versionsgeschichte nur im Changelog: je Version ein Stichpunkt pro Änderung, ohne Begründung.
- Anforderungen: `docs/de/ANFORDERUNGEN.md` / `docs/en/REQUIREMENTS.md`. Änderungen am Funktionsumfang dort nachziehen.

## Tests

- Keine Mocks: keine nachgebauten Server, vorgetäuschten Antworten oder ersetzten Funktionen. Echter Code, echte Eingaben, Seitenanbindung gegen die echten Seiten.
- Echte, abgespeicherte Antworten der Seiten sind erlaubt (Quelle, Abrufdatum, ohne Namen/Kontaktdaten). Ein Live-Test prüft, dass sie noch zur Seite passen.
- Jeder Test wird einmal gegen absichtlich kaputten Code laufen gelassen und muss fehlschlagen.
- Regeln für Mitwirkende: `CONTRIBUTING.md` / `CONTRIBUTING.de.md`.

## Persönliche Daten

- Öffentliches Repository: nichts committen oder im Chat ausgeben, was einen Nutzer verrät (E-Mail, Passwort, Cookies, Browserprofile, Schlüssel); Plattformen sperren identifizierte Konten.
- `scripts/datenschutz.sh`: Hooks (`git config core.hooksPath .githooks`) und CI-Schritt über Dateien und alle Commits. Commits nur mit der GitHub-noreply-Adresse (`git config user.email`).
