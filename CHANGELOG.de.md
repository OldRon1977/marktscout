# Changelog

English: [CHANGELOG.md](CHANGELOG.md)

## 0.1.0 – 2026-11-14

- Anbindung kleinanzeigen.de: Ortsvorschläge, Suche, Detailseite, Status; Trefferzahl der Folgeseiten mit Toleranz geprüft (10 Treffer oder 5 %)
- Anbindung marktplaats.nl: Suche, Gesuch-Filter, Detailseite, Status; Blättern bis zu einer leeren Seite, der Gesamtzahl oder der Seitengrenze
- Mehrere Orte je Suchauftrag, Deutschland und Niederlande getrennt, je Ort eigener Umkreis
- Preisbewertung: Suchpreis, Toleranz, Mindestpreis, VB-Spielraum
- Textprüfung: Pflicht-/Ausschlusswörter mit drei Modi, Verneinung, Vergleichserkennung
- Ergebnisliste nach Ort und Preis, ohne Dubletten; Statusprüfung aktiv / reserviert / gelöscht; Löschen von Anzeigen
- SQLite-Speicher für Suchaufträge, Anzeigen, Treffer und Einstellungen
- Lokale Weboberfläche, bedienbar mit Maus und Tastatur
- Zeitsteuerung, Desktopmeldung, Ton, Tray-Symbol
- Logdatei mit Rotation und Export
- Programmdateien für Linux und Windows, portabel ohne Installation
- GitHub Actions: Tests unter Linux und Windows bei jedem Push und täglich, Oberflächentests mit Playwright, Build, Release bei Versions-Tags
- Pre-Commit- und Commit-Msg-Hook sowie CI-Prüfung gegen Daten, die einen Nutzer verraten (E-Mail-Adressen, Passwörter, Sitzungscookies, Schlüssel, Browserprofile, Autor-Adressen)
- Dokumentation in Deutsch und Englisch, Regeln für Mitwirkende, Issue-Formulare, Pull-Request-Vorlage
