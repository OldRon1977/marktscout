# Anforderungen

Funktionsumfang. Vollständig umgesetzt.

English: [../en/REQUIREMENTS.md](../en/REQUIREMENTS.md)

## Plattform und Betrieb

- Läuft unter Linux und Windows direkt vom USB-Stick, ohne Installation.
- Je Betriebssystem eine ausführbare Datei, beide aus Linux heraus baubar.
- Bedienung über eine grafische Oberfläche (Maus und Tastatur), lokal im Browserfenster ausgeliefert.
- Daten (Suchaufträge, Anzeigen, Cache) liegen in einer Datei neben dem Programm.
- Abruf beider Seiten über einfache HTTP-Abrufe, ohne mitgelieferten Browser (siehe [SEITEN.md](SEITEN.md)).

## Technik

- Programmiersprache Go, je eine ausführbare Datei für Linux und Windows, gebaut unter Linux.
- Oberfläche in HTML/CSS/JavaScript, ins Programm eingebettet.
- SQLite als Datenbank über eine reine Go-Bibliothek.

## Tests

- Automatisch über GitHub Actions bei jedem Push und einmal täglich: Go-Tests unter Linux und Windows, Oberflächentests unter Linux.
- Oberflächentests mit Playwright im echten Browser gegen das laufende Programm.
- Live-Tests laufen auch auf den GitHub-Servern; beide Seiten antworten dort.

## Quellen

- kleinanzeigen.de: Suche im Umkreis deutscher Orte.
- marktplaats.nl: Suche im Umkreis niederländischer Postleitzahlen.
- kleinanzeigen.de: Eingabe von Ortsname oder Postleitzahl; der Ort wird aus den Vorschlägen der Seite ausgewählt.
- marktplaats.nl: Eingabe einer vierstelligen Postleitzahl. Liefert die Suche keine Entfernungsangaben (ungültige Postleitzahl), meldet das Tool einen Fehler.
- Gesuche bei marktplaats.nl werden über den Titel erkannt („Gezocht“, „Gevraagd“, „Zoek“) und ausgeblendet.
- Nur Angebote, keine Gesuche.
- Die Suche selbst läuft ohne Login. Der Login wird nur für die Seitenanalyse genutzt.

## Suchauftrag

- Speicherbar, nachträglich änderbar.
- Suchbegriff(e) für die Suche auf der Seite.
- Ortsliste DE und Ortsliste NL, jeder Ort mit eigenem Umkreis (Standard 10 km). Beide Listen laufen in einem Suchlauf.
- Pflichtwörter und Ausschlusswörter.
- Standard-Ausschlussliste, pro Auftrag abschaltbar und ergänzbar:
  - DE: defekt, Bastler, für Bastler, Tausch, tausche, nur OVP, leere Verpackung, Ersatzteile, kaputt
  - NL: defect, kapot, ruilen, onderdelen, voor onderdelen, alleen doos
- Preis:
  - Suchpreis als Obergrenze.
  - Toleranz wahlweise in % oder €; effektive Obergrenze = Suchpreis + Toleranz.
  - Optionaler Mindestpreis (Standard leer).
  - „VB“ mit Betrag zählt als Betrag, wird als VB gekennzeichnet.
  - Option „VB-Spielraum“ (ankreuzbar, mit X %): Bei VB-Anzeigen wird zusätzlich der Betrag abzüglich X % gegen die Obergrenze geprüft. Beispiel: Obergrenze 550 €, VB-Spielraum 10 %, Anzeige „600 € VB“ → 540 € → wird angezeigt, gekennzeichnet als „über VB-Spielraum“. Sortiert wird nach dem angegebenen Betrag.
- Intervall für zeitgesteuerte Ausführung.

## Textprüfung

- Der Anzeigentext wird abgerufen und auf Pflicht- und Ausschlusswörter geprüft.
- Wirkung der Ausschlusswörter, pro Suchauftrag einstellbar, getrennt für Standard-Ausschlussliste und eigene Ausschlusswörter:
  - **markieren**: Anzeige bleibt sichtbar, Fundstelle in Titel und Text hervorgehoben.
  - **Titel filtern** (Standard): Treffer im Titel blendet aus, Treffer im Text markiert.
  - **hart filtern**: Treffer in Titel oder Text blendet aus.
- Vom Regelwerk als verneint erkannte Fundstellen („kein Tausch“, „nicht defekt“) zählen in keinem Modus als Treffer, werden aber dezent markiert.
- Ausgeblendete Anzeigen bleiben über einen Schalter „Ausgeblendete zeigen“ einsehbar, mit Grund.
- Bewertung durch ein Regelwerk (austauschbarer Baustein, Schnittstelle `textpruefung.Bewerter`). Erkennt Vergleichsmuster („wie eine“, „fast wie“, „vergleichbar mit“, „besser als“, „statt“) und Verneinungen („kein Tausch“, „nicht defekt“). Einstufung: echt / zweifelhaft / Vergleich.
- Abrufe gedrosselt; bereits geprüfte Anzeigen werden zwischengespeichert.

## Ergebnisliste

- Dubletten aus sich überschneidenden Umkreisen werden zusammengeführt.
- Sortierung: Standort aus der Anzeige, innerhalb des Orts nach Preis.
- Anzeigen ohne festen Preis (VB ohne Betrag, auf Anfrage, verschenken) am Ende der Ortsgruppe.
- Preiskennzeichnung: „im Budget“ bis Suchpreis, „in Toleranz“ darüber mit Abweichung (z. B. „+8 %“).
- Kennzeichnung „neu seit letzter Suche“.

## Gespeicherte Anzeigen

- Neu prüfen: Status aktiv / reserviert / gelöscht. „Reserviert“ gibt es nur bei marktplaats.nl.
- Reservierte Anzeigen erscheinen normal in der Ergebnisliste, gekennzeichnet als „reserviert“.
- Manuell löschen. Gelöschte Anzeigen erscheinen bei späteren Suchläufen dieses Auftrags nicht wieder.

## Zeitsteuerung und Benachrichtigung

- Läuft, solange das Programm geöffnet ist; minimierbar in den Infobereich (Tray).
- Intervall pro Suchauftrag.
- Neue sichtbare Treffer nach einem Suchlauf (manuell oder zeitgesteuert): Ton und Desktopmeldung. Hauptschalter für Zeitsteuerung, Desktopmeldung und Ton.

## Logdatei

- Das Programm schreibt eine Logdatei neben die Daten (Abrufe, Fehler, Ergebnis je Suchlauf).
- Schaltfläche „Logdatei speichern“: speichert eine Kopie an einen frei wählbaren Ort, zum Anhängen an ein Issue.
- Die Logdatei enthält keine Kontaktdaten aus Anzeigen.
- Ältere Logdateien werden nach Größe rotiert.
