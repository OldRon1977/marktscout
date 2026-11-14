# Obstacles

What has been shown not to work, with the reason.

Deutsch: [../de/HUERDEN.md](../de/HUERDEN.md)

## kleinanzeigen.de

- **Page number after the keyword** (`…/k0l945r10/seite:2`) is not recognised. The site then returns all listings in Germany without an error (counter in the millions). `seite:{n}` must come before the keyword. A counter that differs clearly from the first page indicates a wrong path. Exact equality cannot be required: between two page requests the count changes through new or deleted listings (seen in CI: 811, then 810); 10 results or 5 % are tolerated.
- **Taking a place name without selection:** the place lookup is fuzzy. "Elmpt" returns Elmstein, Kempten, Belm. Places that Kleinanzeigen does not list separately are found via postcode (Elmpt: `41372` → Niederkrüchten). Let the user choose the place from the suggestions.
- **Status "reserved"** cannot be read without login; there is no marker in search or on the detail page.
- **Counting all `<article>` elements on the page** yields too many results: "Weitere Ergebnisse in anderen Orten" (`#srchrslt-adtable-altads`) contains listings outside the radius.

## marktplaats.nl

- **Detecting the end at a short page:** a result page can contain fewer than 100 listings although `totalResultCount` reports more (seen in CI on Windows: 84 instead of 100). Paging continues until an empty page, the total count or `maxAllowedPageNumber`.
- **Place name or German postcode as `postcode`** (`Venlo`, `41372`) is ignored without an error; the search then covers all of the Netherlands. Recognisable by `distanceMeters: -1000` on every listing. Use Dutch postcodes only.
- **Detail page by ID only** (`/v/a/b/{itemId}`, also `/v/{category}/{subcategory}/{itemId}` without slug) returns 404. Store the `vipUrl` from the search.
- **Description from search** is truncated to 200 characters and is insufficient for the text check.
- **Formally valid but unassigned postcode** (e.g. `8888`) is silently ignored just like an invalid one; `9999` and `9998` are assigned.
- **German search terms** ("Grafikkarte") return hardly any results on marktplaats.nl; searches there need Dutch or neutral terms ("grafische kaart", "rtx").

## UI

- **`hidden` attribute with an own `display` rule:** a CSS rule such as `.beendet { display: flex }` overrides `hidden`; the element stays visible and covers the page. A global rule `[hidden] { display: none !important; }` is needed.
