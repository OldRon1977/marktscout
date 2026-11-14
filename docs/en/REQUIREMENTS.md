# Requirements

Feature set. Fully implemented.

Deutsch: [../de/ANFORDERUNGEN.md](../de/ANFORDERUNGEN.md)

## Platform and operation

- Runs on Linux and Windows directly from a USB stick, without installation.
- One executable per operating system, both buildable on Linux.
- Graphical user interface (mouse and keyboard), served locally in a browser window.
- Data (search profiles, listings, cache) is stored in a file next to the program.
- Both sites are fetched via plain HTTP requests, without a bundled browser (see [SITES.md](SITES.md)).

## Technology

- Programming language Go, one executable each for Linux and Windows, built on Linux.
- UI in HTML/CSS/JavaScript, embedded in the program.
- SQLite as database via a pure Go library.

## Tests

- Automatically via GitHub Actions on every push and once daily: Go tests on Linux and Windows, UI tests on Linux.
- UI tests with Playwright in a real browser against the running program.
- Live tests also run on GitHub servers; both sites respond there.

## Sources

- kleinanzeigen.de: search within a radius around German places.
- marktplaats.nl: search within a radius around Dutch postcodes.
- kleinanzeigen.de: input of place name or postcode; the place is selected from the site's suggestions.
- marktplaats.nl: input of a four-digit postcode. If the search returns no distances (invalid postcode), the tool reports an error.
- Wanted listings on marktplaats.nl are recognised by the title ("Gezocht", "Gevraagd", "Zoek") and hidden.
- Offers only, no "wanted" listings.
- The search itself runs without login. Login is used only for the site analysis.

## Search profile

- Can be saved and edited later.
- Search term(s) used for the site's own search.
- Place list DE and place list NL, each place with its own radius (default 10 km). Both lists run in one search run.
- Required words and excluded words.
- Default exclusion list, can be disabled and extended per profile:
  - DE: defekt, Bastler, für Bastler, Tausch, tausche, nur OVP, leere Verpackung, Ersatzteile, kaputt
  - NL: defect, kapot, ruilen, onderdelen, voor onderdelen, alleen doos
- Price:
  - Target price as upper limit.
  - Tolerance either in % or €; effective upper limit = target price + tolerance.
  - Optional minimum price (default empty).
  - "VB" (negotiable) with an amount counts as that amount and is labelled as negotiable.
  - Option "negotiation margin" (checkbox, with X %): for negotiable listings, the amount minus X % is additionally checked against the upper limit. Example: upper limit 550 €, margin 10 %, listing "600 € VB" → 540 € → shown, labelled "via negotiation margin". Sorting uses the stated amount.
- Interval for scheduled runs.

## Text check

- The listing text is fetched and checked for required and excluded words.
- Effect of excluded words, configurable per profile, separately for the default exclusion list and custom excluded words:
  - **mark**: listing stays visible, matches highlighted in title and text.
  - **filter title** (default): match in title hides the listing, match in text marks it.
  - **hard filter**: match in title or text hides the listing.
- Matches recognised by the rule set as negated ("kein Tausch", "nicht defekt") never count as a match in any mode, but are marked subtly.
- Hidden listings remain viewable via a "show hidden" switch, together with the reason.
- Evaluation by a rule set (exchangeable component, interface `textpruefung.Bewerter`). Recognises comparison patterns ("wie eine", "fast wie", "vergleichbar mit", "besser als", "statt") and negations ("kein Tausch", "nicht defekt"). Classification: genuine / doubtful / comparison.
- Requests are throttled; already checked listings are cached.

## Result list

- Duplicates from overlapping radii are merged.
- Sorting: location stated in the listing, then price within each location.
- Listings without a fixed price (negotiable without amount, on request, free) at the end of each location group.
- Price labels: "in budget" up to the target price, "in tolerance" above it, with the deviation (e.g. "+8 %").
- Label "new since last search".

## Saved listings

- Re-check: status active / reserved / deleted. "Reserved" exists only on marktplaats.nl.
- Reserved listings appear normally in the result list, labelled "reserved".
- Manual deletion. Deleted listings do not reappear in later search runs of this profile.

## Scheduling and notification

- Runs while the program is open; can be minimised to the system tray.
- Interval per search profile.
- New visible results after a search run (manual or scheduled): sound and desktop notification. Master switches for scheduling, desktop notification and sound.

## Log file

- The program writes a log file next to its data (requests, errors, result per search run).
- Button "Save log file": saves a copy to a location of choice, for attaching to an issue.
- The log file contains no contact data from listings.
- Older log files are rotated by size.
