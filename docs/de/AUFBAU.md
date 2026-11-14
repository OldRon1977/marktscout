# Aufbau

English: [../en/ARCHITECTURE.md](../en/ARCHITECTURE.md)

## Pakete

| Paket | Aufgabe |
|---|---|
| `cmd/marktscout` | Programmeinstieg: Startoptionen, Datenverzeichnis, Server, Tray, Browser öffnen, zweite Instanz erkennen |
| `internal/abruf` | HTTP-Client mit Firefox-User-Agent, 2–4 s Abstand je Host, Wiederholung bei Netzfehler/429/5xx |
| `internal/anzeige` | gemeinsame Datentypen (Anzeige, Preistyp, Status) |
| `internal/kleinanzeigen` | Ortsvorschläge, Suche (HTML), Detailseite, Status |
| `internal/marktplaats` | Suche (JSON), Gesuch-Erkennung, Detailseite, Status |
| `internal/preis` | Obergrenze, Toleranz, Mindestpreis, VB-Spielraum, Seitengrenzen |
| `internal/textpruefung` | Schnittstelle `Bewerter`; `Regelwerk`: Pflicht-/Ausschlusswörter, Verneinung, Vergleichserkennung |
| `internal/auftrag` | Suchauftrag und Eingabeprüfung |
| `internal/speicher` | SQLite-Datei: Aufträge, Anzeigen, Treffer, Einstellungen |
| `internal/suchlauf` | Suchlauf, Statusprüfung, Ergebnisliste (Bewertung, Gruppierung, Sortierung) |
| `internal/web` | JSON-Schnittstelle, eingebettete Oberfläche (`ui/`), Warteschlange, Zeitsteuerung, Meldungen |
| `internal/protokoll` | Logdatei mit Rotation (2 MB, 3 ältere Dateien), Export |
| `internal/melden` | Desktopmeldung (beeep), Hinweiston (WAV, Systemabspieler) |
| `internal/symbol` | Programmsymbol als PNG und ICO |

## Suchlauf

1. Treffer-Kennung „neu“ des Auftrags zurücksetzen.
2. Je Ort DE: Suche bei kleinanzeigen.de mit Preisgrenzen der Seite (Mindestpreis bis Obergrenze, mit VB-Spielraum angehoben).
3. Je Ort NL: Suche bei marktplaats.nl; Gesuche werden verworfen.
4. Jede Anzeige wird gespeichert und dem Auftrag zugeordnet. Dieselbe Anzeige aus mehreren Orten wird einmal geführt, mit der kürzesten Entfernung. Vom Nutzer gelöschte Treffer bleiben gelöscht.
5. Detailseiten nur für Anzeigen ohne vollständige Beschreibung, die Preis- und Titelprüfung bestehen.
6. Zeitpunkt und Fehler des Laufs speichern; neue sichtbare Treffer zählen; ggf. Meldung und Ton.

Die Bewertung (Preis, Text) wird nicht gespeichert, sondern bei jeder Anzeige der Liste aus Auftrag und Anzeige berechnet. Änderungen am Auftrag wirken sofort auf vorhandene Treffer.

## Warteschlange und Zeitsteuerung

- Ein Arbeiter führt Suchläufe und Statusprüfungen nacheinander aus. Derselbe Job wird nicht doppelt eingereiht.
- Die Zeitsteuerung prüft alle 30 s, ob ein Auftrag mit Intervall fällig ist (letzter Lauf + Intervall), und reiht ihn ein. Auch ein gescheiterter Lauf setzt den Zeitpunkt.

## Schnittstelle

- Nur `127.0.0.1`/`localhost` als Host (gegen DNS-Rebinding).
- Ändernde Anfragen brauchen den Kopf `X-Marktscout: 1` (gegen Anfragen fremder Webseiten).
- Content-Security-Policy: nur eigene Skripte und Stile, Bilder von `https:`.

## Build und Release

- `scripts/build.sh [version]`: baut `dist/marktscout-linux-amd64` und `dist/marktscout-windows-amd64.exe` (CGO aus, Windows ohne Konsolenfenster).
- CI-Job `build`: nach den Tests, Artefakt `marktscout` mit dem Inhalt von `dist/`.
- CI-Job `release`: nur bei Tags `v*`; legt das GitHub-Release mit `dist/`-Dateien und `SHA256SUMS` an, Notiz aus `scripts/release-notes.sh` (Abschnitte der Version aus beiden Changelogs, Abbruch, wenn einer fehlt).

## Tests

| Bereich | Art |
|---|---|
| `kleinanzeigen`, `marktplaats`, `suchlauf` (Live-Test), `web` (Ortsvorschläge) | live gegen die echten Seiten |
| `preis`, `textpruefung`, `suchlauf` (Liste), `auftrag` | echte Funktionen mit echten Eingaben |
| `speicher`, `protokoll` | echte Dateien im Temp-Verzeichnis |
| `web` | echter Server auf 127.0.0.1 |
| `cmd/marktscout` | baut und startet das Programm |
| `scripts` | `datenschutz.sh` als echte Pre-Commit- und Commit-Msg-Hooks und über die Historie, in einem temporären git-Repository (nur Linux) |
| `tests/ui` (Playwright) | echtes Programm, echter Chromium, Suche live |

Ausführen:

```sh
go test ./...
npm ci && npx playwright test
```

Gegenprobe: Jeder Test muss fehlschlagen, wenn die geprüfte Regel im Code gezielt verfälscht wird (siehe [CONTRIBUTING.de.md](../../CONTRIBUTING.de.md)).
