# Mitmachen

English: [CONTRIBUTING.md](CONTRIBUTING.md)

Beiträge sind willkommen. Es gelten die folgenden Regeln.

## Einrichtung

```sh
git clone https://github.com/OldRon1977/marktscout.git
cd marktscout
git config core.hooksPath .githooks
git config user.email <id>+<name>@users.noreply.github.com
```

## Persönliche Daten

- Nichts committen, was einen Nutzer verrät: E-Mail-Adressen, Benutzernamen und Passwörter von Plattformen, Sitzungscookies, Browserprofile, API-Schlüssel, Tokens.
- Autor- und Committer-Adresse jedes Commits ist die GitHub-noreply-Adresse: `git config user.email <id>+<name>@users.noreply.github.com` (zu finden unter GitHub → Settings → Emails).
- [scripts/datenschutz.sh](scripts/datenschutz.sh) prüft das: als Pre-Commit-Hook gestagte Änderungen und Autor-Adresse, als Commit-Msg-Hook die Nachricht, in der CI alle Dateien und alle Commits. Erlaubt sind nur Beispieladressen (`@example.com`, `*.invalid`) und GitHub-noreply-Adressen. Gemeldet werden Ort und Art, nie der Wert.
- Testausgaben dürfen keine persönlichen Daten aus Anzeigen enthalten (Namen, Telefonnummern).

## Tests

- **Keine Mocks.** Keine nachgebauten Server, keine vorgetäuschten Antworten, keine ersetzten Funktionen.
- Tests rufen den echten Code mit echten Eingaben auf. Tests der Seitenanbindung laufen gegen die echten Seiten kleinanzeigen.de und marktplaats.nl.
- Erlaubt sind echte Daten: von den echten Seiten abgerufene und abgespeicherte Antworten. Jede gespeicherte Antwort trägt Quelle und Abrufdatum. Namen und Kontaktdaten werden vor dem Speichern entfernt.
- Ein Live-Test prüft, ob die gespeicherten Antworten noch dem aktuellen Aufbau der Seite entsprechen. Weicht die Seite ab, schlägt er fehl, und die Daten werden neu abgerufen.
- Jeder Test muss eine Aussage treffen: Er wird einmal mit absichtlich kaputtem Code ausgeführt und muss dabei fehlschlagen. Ein Test, der nie fehlschlagen kann, wird nicht übernommen.
- Live-Tests prüfen Struktur und Plausibilität (z. B. „Treffer haben Titel, Preis und Ort“), nicht konkrete Anzeigen, die morgen verschwunden sein können.
- Live-Tests halten Pausen zwischen Abrufen ein und rufen nur so viel ab, wie für die Aussage nötig ist.
- Ausführen: `go test ./...` und `npm ci && npx playwright test`.
- Jede neue Funktion bringt Tests mit. Ein Pull Request wird nur übernommen, wenn alle Tests grün sind.
- `main` ist geschützt: Änderungen nur per Pull Request; die Prüfungen `test (ubuntu-latest)`, `test (windows-latest)`, `ui` und `build` müssen grün sein. Kein Force-Push. Workflows von Erstbeitragenden starten erst nach Freigabe durch den Maintainer.
- Sicherheitslücken vertraulich melden über „Security → Report a vulnerability“, nicht als öffentliches Issue.

## Issues

- Fehler, Wünsche und Fragen über die Formulare unter „New issue“.
- Bei Fehlern die Logdatei anhängen (Oberfläche: „Logdatei speichern“).

## Dokumentation

- Jede Doku gibt es in Deutsch und Englisch, in getrennten Dateien mit gegenseitigem Link (`README.md`/`README.de.md`, `CHANGELOG.md`/`CHANGELOG.de.md`, `docs/en/…`/`docs/de/…`). Beide Fassungen werden im selben Commit geändert.
- Dokumentiert wird nur der aktuelle Stand. Weicht die Doku vom Code ab, wird die Doku korrigiert.
- Keine Historie: nichts darüber, was es früher gab oder was verworfen wurde.
- Was nachweislich nicht funktioniert (live gesehen oder per Test belegt), wird mit Grund in `docs/de/HUERDEN.md` / `docs/en/OBSTACLES.md` festgehalten, damit niemand denselben Weg erneut versucht.
- Jede Änderung bekommt einen Stichpunkt im Changelog (beide Sprachen), ohne Begründung.
- Sachlicher Ton, keine Ausrufezeichen zur Betonung.

## Veröffentlichen

1. Den Changelog-Abschnitt in beiden Sprachen in `## X.Y.Z – JJJJ-MM-TT` umbenennen.
2. Committen, dann `git tag -a vX.Y.Z -m "Version X.Y.Z"` und `git push origin main vX.Y.Z`.
3. Die CI führt Tests und Build aus und legt danach das GitHub-Release mit beiden Programmdateien, `SHA256SUMS` und den beiden Changelog-Abschnitten als Notiz an (`scripts/release-notes.sh`). Der Release-Job bricht ab, wenn ein Changelog keinen Abschnitt für die Version hat.

## Umgang mit den Seiten

- Abrufe gedrosselt, kein paralleles Abgrasen.
- Keine Umgehung von Captchas oder Zugangssperren.

## Lizenz

Mit einem Beitrag stimmst du zu, dass er unter der [GPL-3.0](LICENSE) veröffentlicht wird.
