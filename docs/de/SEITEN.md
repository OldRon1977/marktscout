# Aufbau der Seiten

Wie kleinanzeigen.de und marktplaats.nl auf einfache HTTP-Abrufe antworten. Grundlage für die Seitenanbindung.

English: [../en/SITES.md](../en/SITES.md)

Beide Seiten liefern ihre Inhalte an einfache HTTP-Abrufe (Firefox-User-Agent, ohne Cookies, ohne Login), auch von GitHub-Actions-Servern. Ein mitgelieferter Browser ist nicht nötig.

## kleinanzeigen.de

### Ortssuche

`GET /s-ort-empfehlungen.json?query={text}`

Antwort: JSON-Objekt `{"_{ortId}": "{Anzeigename}", …}`; `_0` ist immer „Deutschland“.

- Beispiele: Köln `945`, Düsseldorf `2068`, Mönchengladbach `1957`, Heinsberg `1208`, PLZ `41372` → `1184` „41372 Niederkrüchten“.
- Die Suche ist unscharf: Für Orte, die Kleinanzeigen nicht kennt, kommen fremde Orte (siehe [HUERDEN.md](HUERDEN.md)).

### Suche

`GET /s-[{ortSlug}/]anzeige:angebote/[preis:{min}:{max}/][seite:{n}/]{suchwort}/k0l{ortId}r{radiusKm}`

- `{ortSlug}` (z. B. `koeln`) ist optional; maßgeblich ist `l{ortId}`.
- `anzeige:angebote`: nur Angebote, keine Gesuche.
- `preis:{min}:{max}`: ganze Euro, eine Seite darf leer sein (`preis::500`).
- `seite:{n}`: Seitennummer ab 2, vor dem Suchwort.
- `{suchwort}`: Leerzeichen als `-`.
- `l{ortId}r{radiusKm}`: Ort und Umkreis.
- `{radiusKm}` ist frei wählbar (z. B. 7, 15, 25).
- 25 Treffer pro Seite. Zähler im Text „1 - 25 von 1.616“.
- Ohne Treffer fehlt `#srchrslt-adtable`; die Seite enthält „Es wurden keine Ergebnisse … gefunden“.

Aufbau der Ergebnisseite (HTML):

| Inhalt | Fundstelle |
|---|---|
| Treffer | `ul#srchrslt-adtable > li > article[data-adid]` |
| Fremdtreffer außerhalb des Umkreises | `ul#srchrslt-adtable-altads` (nicht verwenden) |
| Anzeigen-ID | `article@data-adid` |
| Link | `article@data-href` |
| Titel | `article h3 a` |
| Ort | `span` „{PLZ} {Ort}“, die Entfernung „({n} km)“ bzw. „(ca. {n} km)“ steht im benachbarten `span` desselben Elternelements |
| Preis | `p` mit „{Betrag} €“, „{Betrag} € VB“, „VB“, „Zu verschenken“; ein zweites `p` mit `line-through` ist der alte Preis |
| Textanfang | `article p` nach dem Titel, gekürzt |
| Datum | „Heute, 12:24“, „Gestern, 23:28“ oder Datum; fehlt bei hervorgehobenen Anzeigen |
| Merkmale | `span[data-dhl-promotion]`: „Direkt kaufen“, „Versand möglich“ |

Die CSS-Klassen sind Utility-Klassen (Tailwind) und eignen sich nicht als Anker. Stabil sind `id`- und `data-`-Attribute.

### Detailseite

`GET /s-anzeige/{ID}` genügt, der Slug ist nicht nötig.

| Inhalt | Fundstelle |
|---|---|
| Titel | `#viewad-title` |
| Preis | `#viewad-price` |
| Ort | `#viewad-locality` („53844 Nordrhein-Westfalen - Troisdorf“) |
| Beschreibung, vollständig | `#viewad-description-text` |
| Einstelldatum | `#viewad-extra-info` |
| Merkmale | `#viewad-details` |
| Koordinaten | `meta[property="og:latitude"]`, `meta[property="og:longitude"]` |

### Status

| Antwort auf `GET /s-anzeige/{ID}` | Bedeutung |
|---|---|
| 200 | aktiv |
| 301 auf eine Kategorieseite (`/s-…/c{n}`) | gelöscht oder nie vorhanden |

Ein Merkmal „reserviert“ gibt es weder auf der Detailseite noch in der Suche (ohne Login). Verkäufer schreiben „reserviert“ teils in Titel oder Text.

## marktplaats.nl

### Suche

`GET /lrp/api/search` → JSON

| Parameter | Bedeutung |
|---|---|
| `query` | Suchwort |
| `postcode` | vierstellige NL-Postleitzahl (`6041`) oder vollständig (`6041CA`) |
| `distanceMeters` | Umkreis in Metern |
| `attributeRanges[]=PriceCents:{min}:{max}` | Preis in Cent; offene Seite als `null` (`PriceCents:null:30000`) |
| `limit`, `offset` | Seitengröße (100 möglich) und Versatz |

Antwortfelder: `totalResultCount`, `maxAllowedPageNumber`, `listings[]`.

Felder je Anzeige:

| Feld | Inhalt |
|---|---|
| `itemId` | `m{Zahl}` (privat) oder `a{Zahl}` (gewerblich, Admarkt) |
| `title` | Titel |
| `description` | nur die ersten 200 Zeichen |
| `priceInfo.priceCents`, `priceInfo.priceType` | siehe Preistypen |
| `location.cityName`, `.distanceMeters`, `.latitude`, `.longitude` | Ort; `distanceMeters` = `-1000`, wenn kein gültiger Mittelpunkt gesetzt ist |
| `reserved` | `true`/`false` |
| `vipUrl` | Pfad der Detailseite |
| `date` | „Vandaag“, „Gisteren“ oder Datum |
| `priorityProduct` | z. B. `DAGTOPPER` (hervorgehoben) |
| `sellerInformation` | Name, ID (persönliche Daten, nicht speichern) |

Preistypen:

| `priceType` | Bedeutung | Behandlung |
|---|---|---|
| `FIXED` | Festpreis | Betrag |
| `MIN_BID` | „Bieden vanaf“ (Gebote ab Betrag) | wie VB mit Betrag; mit `priceCents` = 0 wie VB ohne Betrag |
| `FAST_BID` | „Bieden“, `priceCents` = 0 | wie VB ohne Betrag |
| `SEE_DESCRIPTION` | „Zie omschrijving“ | ohne Preis |
| `FREE` | „Gratis“ | verschenken |

Gesuche: Es gibt keinen Filter. Gesuche tragen meist „Gezocht“, „Gevraagd“ oder „Zoek“ im Titel und haben überwiegend `SEE_DESCRIPTION`.

### Detailseite

`GET https://www.marktplaats.nl{vipUrl}`

- Eingebettetes JSON `window.__CONFIG__ = {…}` → `listing` mit `itemId`, `title`, `priceInfo`, `isReserved`, `adType`, `category`, `stats.since` (Einstelldatum).
- Vollständige Beschreibung: `[data-collapsable="description"]`.

### Status

| Antwort auf `GET {vipUrl}` | Bedeutung |
|---|---|
| 200 | aktiv; `listing.isReserved` zeigt „reserviert“ |
| 410 | gelöscht |
| 404 | Pfad ungültig |
| 301 | Slug abweichend, Ziel ist die gültige URL |

Die Detailseite braucht Kategoriepfad und ID (`/v/{kategorie}/{unterkategorie}/{itemId}-{slug}`). Der Slug darf falsch sein.
