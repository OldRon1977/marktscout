# Hürden

Was nachweislich nicht funktioniert, mit Grund.

English: [../en/OBSTACLES.md](../en/OBSTACLES.md)

## kleinanzeigen.de

- **Seitennummer hinter dem Suchwort** (`…/k0l945r10/seite:2`) wird nicht erkannt. Die Seite liefert dann ohne Fehlermeldung alle Anzeigen Deutschlands (Zähler im Millionenbereich). `seite:{n}` muss vor dem Suchwort stehen. Ein Zähler, der deutlich von der ersten Seite abweicht, zeigt einen falschen Pfad an. Exakte Gleichheit lässt sich nicht verlangen: Zwischen zwei Seitenabrufen ändert sich die Zahl durch neue oder gelöschte Anzeigen (in der CI gesehen: 811, dann 810); geduldet werden 10 Treffer oder 5 %.
- **Ortsname ohne Auswahl übernehmen:** Die Ortssuche ist unscharf. „Elmpt“ liefert Elmstein, Kempten, Belm. Orte, die Kleinanzeigen nicht als eigenen Ort führt, über die PLZ suchen (Elmpt: `41372` → Niederkrüchten). Den Ort vom Nutzer aus den Vorschlägen wählen lassen.
- **Status „reserviert“** lässt sich ohne Login nicht auslesen; es gibt kein Merkmal in Suche oder Detailseite.
- **`<article>` der ganzen Seite zählen** liefert zu viele Treffer: Unter „Weitere Ergebnisse in anderen Orten“ (`#srchrslt-adtable-altads`) stehen Anzeigen außerhalb des Umkreises.

## marktplaats.nl

- **Seitenende an einer kurzen Seite erkennen:** Eine Ergebnisseite kann weniger als 100 Anzeigen enthalten, obwohl `totalResultCount` weitere meldet (in der CI auf Windows gesehen: 84 statt 100). Geblättert wird bis zu einer leeren Seite, der Gesamtzahl oder `maxAllowedPageNumber`.
- **Ortsname oder deutsche PLZ als `postcode`** (`Venlo`, `41372`) wird ohne Fehlermeldung ignoriert; gesucht wird dann in den ganzen Niederlanden. Erkennbar an `distanceMeters: -1000` in jeder Anzeige. Nur NL-Postleitzahlen verwenden.
- **Detailseite nur über die ID** (`/v/a/b/{itemId}`, auch `/v/{kategorie}/{unterkategorie}/{itemId}` ohne Slug) liefert 404. Die `vipUrl` aus der Suche speichern.
- **Beschreibung aus der Suche** ist auf 200 Zeichen gekürzt und reicht für die Textprüfung nicht.
- **Formal gültige, nicht vergebene Postleitzahl** (z. B. `8888`) wird ebenso still ignoriert wie eine ungültige; `9999` und `9998` sind vergeben.
- **Deutsche Suchbegriffe** („Grafikkarte“) liefern bei marktplaats.nl kaum Treffer; die Suche dort braucht niederländische oder neutrale Begriffe („grafische kaart“, „rtx“).

## Oberfläche

- **`hidden`-Attribut mit eigener `display`-Regel:** Eine CSS-Regel wie `.beendet { display: flex }` überschreibt `hidden`; das Element bleibt sichtbar und verdeckt die Seite. Globale Regel `[hidden] { display: none !important; }` nötig.
