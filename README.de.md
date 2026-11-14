# Marktscout

Durchsucht **kleinanzeigen.de** und **marktplaats.nl** in mehreren Orten gleichzeitig, filtert die Treffer nach Stichworten und Preis und meldet neue Anzeigen. Läuft portabel vom USB-Stick unter Linux und Windows, ohne Installation.

English: [README.md](README.md)

## Funktionsumfang

- **Mehrere Orte in einem Suchlauf:** getrennte Ortslisten für Deutschland (Ortsname oder Postleitzahl, aus den Vorschlägen der Seite gewählt) und die Niederlande (Postleitzahl), jeder Ort mit eigenem Umkreis.
- **Stichworte:** Pflicht- und Ausschlusswörter, geprüft in Titel und vollständigem Anzeigentext. Ausschlusswörter markieren, filtern im Titel oder filtern hart. Verneinungen („kein Tausch“, „nicht defekt“) werden erkannt. Ein Regelwerk erkennt Vergleiche wie „fast wie eine RTX 2080“.
- **Preis:** Suchpreis mit Toleranz in % oder €, Mindestpreis, VB-Spielraum.
- **Nur Angebote:** Gesuche werden ausgeschlossen.
- **Ergebnisliste** nach Standort der Anzeige gruppiert, nach Preis sortiert, ohne Dubletten; neue, reservierte und gelöschte Anzeigen gekennzeichnet.
- **Gespeicherte Anzeigen** lassen sich neu prüfen (aktiv / reserviert / gelöscht) und löschen.
- **Gespeicherte Suchaufträge**, jederzeit änderbar.
- **Zeitgesteuerte Suche** mit Ton und Desktopmeldung, abschaltbar; läuft im Infobereich weiter.
- **Logdatei** zum Speichern und Anhängen an Fehlermeldungen.

Bedienung: [docs/de/BEDIENUNG.md](docs/de/BEDIENUNG.md)

## Herunterladen und starten

Programmdateien je Version: [Releases](https://github.com/OldRon1977/marktscout/releases). Stand des aktuellen Codes: **Actions → CI → letzter Lauf → Artifacts → `marktscout`**. Alternativ selbst bauen (siehe unten).

1. `marktscout-linux-amd64` und/oder `marktscout-windows-amd64.exe` auf den USB-Stick kopieren.
2. Per Doppelklick starten (Linux: einmal `chmod +x marktscout-linux-amd64`).
3. Die Oberfläche öffnet sich im Browser. Die Daten liegen im Ordner `daten` neben der Programmdatei.

Das Release enthält außerdem `SHA256SUMS` (Prüfsummen).

## Einrichtung für Entwickler

```sh
git clone https://github.com/OldRon1977/marktscout.git
cd marktscout
git config core.hooksPath .githooks
git config user.email <id>+<name>@users.noreply.github.com
```

Die Hooks brechen einen Commit ab, der Daten enthält, die einen Nutzer verraten (E-Mail-Adressen, Passwörter, Sitzungscookies, Schlüssel, Browserprofile), auch in der Commit-Nachricht und als Autor-Adresse. Dieselbe Prüfung läuft in der CI über alle Dateien und alle Commits.

### Bauen und testen

Benötigt Go (Version in [go.mod](go.mod)); die Oberflächentests zusätzlich Node.js.

```sh
go test ./...                       # alle Go-Tests, auch Live-Tests gegen die echten Seiten
npm ci && npx playwright test       # Oberflächentests: echtes Programm im echten Browser
sh scripts/build.sh                 # dist/marktscout-linux-amd64, dist/marktscout-windows-amd64.exe
go run ./cmd/marktscout -daten /tmp/ms-daten   # aus dem Quelltext starten
```

GitHub Actions führt bei jedem Push und einmal täglich die Go-Tests unter Linux und Windows und die Oberflächentests unter Linux aus und baut danach beide Programmdateien ([.github/workflows/ci.yml](.github/workflows/ci.yml)). Ein Versions-Tag `v*` legt zusätzlich das GitHub-Release an.

Aufbau des Codes: [docs/de/AUFBAU.md](docs/de/AUFBAU.md). Antworten der Seiten: [docs/de/SEITEN.md](docs/de/SEITEN.md). Bekannte Hürden: [docs/de/HUERDEN.md](docs/de/HUERDEN.md).

## Hinweis zur Nutzung

kleinanzeigen.de und marktplaats.nl bieten keine öffentliche Schnittstelle. Das Tool ruft die Seiten so ab, wie ein Browser es tut, und hält 2–4 Sekunden Abstand zwischen zwei Abrufen derselben Seite. Die Nutzungsbedingungen der Seiten schränken automatisierte Abrufe ein. Die Nutzung erfolgt auf eigene Verantwortung und ist für den privaten Gebrauch gedacht.

Dieses Projekt steht in keiner Verbindung zu kleinanzeigen.de oder marktplaats.nl.

## Mitmachen

Siehe [CONTRIBUTING.de.md](CONTRIBUTING.de.md).

## Lizenz

[GPL-3.0](LICENSE)
