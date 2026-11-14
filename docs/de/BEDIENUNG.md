# Bedienung

English: [../en/USAGE.md](../en/USAGE.md)

## Start

| System | Datei | Start |
|---|---|---|
| Linux | `marktscout-linux-amd64` | Doppelklick oder `./marktscout-linux-amd64` (vorher einmal `chmod +x`) |
| Windows | `marktscout-windows-amd64.exe` | Doppelklick |

- Das Programm öffnet die Oberfläche im Standardbrowser unter `http://127.0.0.1:47811`.
- Im Infobereich (Tray) erscheint eine Lupe mit „Öffnen“ und „Beenden“.
- Ein zweiter Start öffnet nur die Oberfläche der laufenden Instanz.
- Daten liegen im Ordner `daten` neben der Programmdatei: `marktscout.db` (Suchaufträge, Anzeigen, Einstellungen) und `logs/`. Auf dem USB-Stick bleiben sie beim Programm.
- Beenden: Tray → „Beenden“ oder in der Oberfläche „Programm beenden“. Das Schließen des Browserfensters beendet das Programm nicht.

Startoptionen:

| Option | Wirkung |
|---|---|
| `-daten <ordner>` | anderes Datenverzeichnis |
| `-port <n>` | anderer Port (Standard 47811) |
| `-kein-browser` | Browser nicht öffnen |
| `-kein-tray` | kein Tray-Symbol |
| `-version` | Version ausgeben |

## Suchauftrag

„+ Neuer Suchauftrag“ öffnet das Formular.

- **Suchbegriff:** wird an beide Seiten übergeben und für die Einstufung im Anzeigentext gesucht.
- **Orte Deutschland:** Ortsname oder Postleitzahl eingeben, Vorschlag mit Maus oder Pfeiltasten + Enter wählen. Umkreis je Ort änderbar.
- **Orte Niederlande:** vierstellige Postleitzahl (z. B. 5911 Venlo, 6041 Roermond), „Hinzufügen“ oder Enter.
- **Preis:**
  - Suchpreis = Obergrenze für „im Budget“.
  - Toleranz in % oder €: Anzeigen bis Suchpreis + Toleranz erscheinen als „in Toleranz“.
  - Mindestpreis: blendet billigere Anzeigen aus, auch „zu verschenken“.
  - VB-Spielraum: VB-Anzeigen, deren Betrag abzüglich X % unter der Obergrenze liegt, erscheinen als „über VB-Spielraum“.
  - Unter den Feldern steht die berechnete Obergrenze.
- **Pflichtwörter:** müssen unverneint in Titel oder Text stehen, sonst wird die Anzeige ausgeblendet.
- **Ausschlusswörter** (eigene und Standardliste), Wirkung jeweils wählbar:
  - *markieren:* nur hervorheben
  - *Titel filtern, Text markieren:* Treffer im Titel blendet aus
  - *hart filtern:* Treffer in Titel oder Text blendet aus
  - Verneinte Stellen („kein Tausch“, „nicht defekt“, „Tausch nicht möglich“) zählen nicht.
- **Intervall:** Minuten zwischen zwei automatischen Suchen (0 = aus, sonst ab 10).
- **Seiten je Ort:** Höchstzahl gelesener Ergebnisseiten je Ort (kleinanzeigen.de 25, marktplaats.nl 100 Treffer je Seite).

## Ergebnisliste

- „Jetzt suchen“ startet die Suche, Fortschritt oben in der Kopfzeile. Danach werden die Anzeigentexte der Treffer geladen, die Preis- und Titelprüfung bestehen.
- Gruppiert nach Ort der Anzeige (Deutschland, dann Niederlande, alphabetisch), innerhalb nach Preis; Anzeigen ohne Betrag am Ende.
- Kennzeichnungen: Quelle, neu, reserviert (nur marktplaats.nl), gelöscht, Preiseinordnung, „nur Vergleich“ (Suchbegriff nur verglichen, z. B. „fast wie eine RTX 2080“), „zweifelhaft“, „Ausschlusswort“.
- Hervorhebungen im Text: rot Ausschlusswort, grau verneintes Ausschlusswort, blau Suchbegriff/Pflichtwort, gelb Suchbegriff als Vergleich.
- „Ausgeblendete zeigen“: zeigt ausgeblendete Anzeigen mit Grund.
- „Status prüfen“ (oben für alle, an der Karte für eine): ruft die Anzeige erneut ab und hält aktiv / reserviert / gelöscht fest.
- „Löschen“ an der Karte: entfernt die Anzeige aus dem Suchauftrag; sie erscheint bei späteren Suchen nicht wieder.

## Einstellungen

- Zeitsteuerung, Desktopmeldung, Ton: Hauptschalter. Meldung und Ton kommen nach jedem Suchlauf mit neuen sichtbaren Anzeigen.
- „Meldung testen“ gibt Desktopmeldung und Ton einmal aus.
- „Logdatei speichern“ lädt alle Logdateien als eine Textdatei herunter (zum Anhängen an ein Issue). Die Logdatei enthält Abruf-Adressen, Trefferzahlen und Fehler, keine Verkäuferdaten.

## Ton unter Linux

Der Ton wird mit `pw-play`, `paplay` oder `aplay` abgespielt (der erste vorhandene).
