# Site structure

How kleinanzeigen.de and marktplaats.nl respond to plain HTTP requests. Basis for the site connectors.

Deutsch: [../de/SEITEN.md](../de/SEITEN.md)

Both sites serve their content to plain HTTP requests (Firefox user agent, no cookies, no login), also from GitHub Actions servers. A bundled browser is not needed.

## kleinanzeigen.de

### Place lookup

`GET /s-ort-empfehlungen.json?query={text}`

Response: JSON object `{"_{placeId}": "{display name}", …}`; `_0` is always "Deutschland".

- Examples: Köln `945`, Düsseldorf `2068`, Mönchengladbach `1957`, Heinsberg `1208`, postcode `41372` → `1184` "41372 Niederkrüchten".
- The lookup is fuzzy: places unknown to Kleinanzeigen return unrelated places (see [OBSTACLES.md](OBSTACLES.md)).

### Search

`GET /s-[{placeSlug}/]anzeige:angebote/[preis:{min}:{max}/][seite:{n}/]{keyword}/k0l{placeId}r{radiusKm}`

- `{placeSlug}` (e.g. `koeln`) is optional; `l{placeId}` is what counts.
- `anzeige:angebote`: offers only, no "wanted" listings.
- `preis:{min}:{max}`: whole euros, one side may be empty (`preis::500`).
- `seite:{n}`: page number from 2, before the keyword.
- `{keyword}`: spaces as `-`.
- `l{placeId}r{radiusKm}`: place and radius.
- `{radiusKm}` can be any value (e.g. 7, 15, 25).
- 25 results per page. Counter in the text "1 - 25 von 1.616".
- Without results `#srchrslt-adtable` is missing; the page contains "Es wurden keine Ergebnisse … gefunden".

Result page structure (HTML):

| Content | Location |
|---|---|
| Results | `ul#srchrslt-adtable > li > article[data-adid]` |
| Results outside the radius | `ul#srchrslt-adtable-altads` (do not use) |
| Listing ID | `article@data-adid` |
| Link | `article@data-href` |
| Title | `article h3 a` |
| Location | `span` "{postcode} {place}"; the distance "({n} km)" or "(ca. {n} km)" is in the adjacent `span` of the same parent |
| Price | `p` with "{amount} €", "{amount} € VB", "VB", "Zu verschenken"; a second `p` with `line-through` is the old price |
| Text excerpt | `article p` after the title, truncated |
| Date | "Heute, 12:24", "Gestern, 23:28" or a date; missing on highlighted listings |
| Features | `span[data-dhl-promotion]`: "Direkt kaufen", "Versand möglich" |

CSS classes are utility classes (Tailwind) and are unsuitable as anchors. `id` and `data-` attributes are stable.

### Detail page

`GET /s-anzeige/{ID}` is sufficient; the slug is not needed.

| Content | Location |
|---|---|
| Title | `#viewad-title` |
| Price | `#viewad-price` |
| Location | `#viewad-locality` ("53844 Nordrhein-Westfalen - Troisdorf") |
| Full description | `#viewad-description-text` |
| Posting date | `#viewad-extra-info` |
| Features | `#viewad-details` |
| Coordinates | `meta[property="og:latitude"]`, `meta[property="og:longitude"]` |

### Status

| Response to `GET /s-anzeige/{ID}` | Meaning |
|---|---|
| 200 | active |
| 301 to a category page (`/s-…/c{n}`) | deleted or never existed |

There is no "reserved" marker on the detail page or in search (without login). Sellers sometimes write "reserviert" in title or text.

## marktplaats.nl

### Search

`GET /lrp/api/search` → JSON

| Parameter | Meaning |
|---|---|
| `query` | keyword |
| `postcode` | four-digit Dutch postcode (`6041`) or full (`6041CA`) |
| `distanceMeters` | radius in metres |
| `attributeRanges[]=PriceCents:{min}:{max}` | price in cents; open side as `null` (`PriceCents:null:30000`) |
| `limit`, `offset` | page size (100 possible) and offset |

Response fields: `totalResultCount`, `maxAllowedPageNumber`, `listings[]`.

Fields per listing:

| Field | Content |
|---|---|
| `itemId` | `m{number}` (private) or `a{number}` (commercial, Admarkt) |
| `title` | title |
| `description` | first 200 characters only |
| `priceInfo.priceCents`, `priceInfo.priceType` | see price types |
| `location.cityName`, `.distanceMeters`, `.latitude`, `.longitude` | location; `distanceMeters` = `-1000` if no valid centre is set |
| `reserved` | `true`/`false` |
| `vipUrl` | path of the detail page |
| `date` | "Vandaag", "Gisteren" or a date |
| `priorityProduct` | e.g. `DAGTOPPER` (highlighted) |
| `sellerInformation` | name, ID (personal data, do not store) |

Price types:

| `priceType` | Meaning | Handling |
|---|---|---|
| `FIXED` | fixed price | amount |
| `MIN_BID` | "Bieden vanaf" (bids from amount) | like negotiable with amount; with `priceCents` = 0 like negotiable without amount |
| `FAST_BID` | "Bieden", `priceCents` = 0 | like negotiable without amount |
| `SEE_DESCRIPTION` | "Zie omschrijving" | no price |
| `FREE` | "Gratis" | free |

Wanted listings: there is no filter. They usually have "Gezocht", "Gevraagd" or "Zoek" in the title and mostly `SEE_DESCRIPTION`.

### Detail page

`GET https://www.marktplaats.nl{vipUrl}`

- Embedded JSON `window.__CONFIG__ = {…}` → `listing` with `itemId`, `title`, `priceInfo`, `isReserved`, `adType`, `category`, `stats.since` (posting date).
- Full description: `[data-collapsable="description"]`.

### Status

| Response to `GET {vipUrl}` | Meaning |
|---|---|
| 200 | active; `listing.isReserved` indicates "reserved" |
| 410 | deleted |
| 404 | invalid path |
| 301 | slug differs, target is the valid URL |

The detail page needs category path and ID (`/v/{category}/{subcategory}/{itemId}-{slug}`). The slug may be wrong.
