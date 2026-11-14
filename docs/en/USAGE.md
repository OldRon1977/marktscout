# Usage

Deutsch: [../de/BEDIENUNG.md](../de/BEDIENUNG.md)

## Start

| System | File | Start |
|---|---|---|
| Linux | `marktscout-linux-amd64` | double-click or `./marktscout-linux-amd64` (run `chmod +x` once first) |
| Windows | `marktscout-windows-amd64.exe` | double-click |

- The program opens the UI in the default browser at `http://127.0.0.1:47811`.
- A magnifier icon with "Öffnen" (open) and "Beenden" (quit) appears in the system tray.
- Starting a second time only opens the UI of the running instance.
- Data is stored in the folder `daten` next to the executable: `marktscout.db` (search profiles, listings, settings) and `logs/`. On a USB stick it stays with the program.
- Quit: tray → "Beenden" or "Programm beenden" in the UI. Closing the browser window does not stop the program.

Command line options:

| Option | Effect |
|---|---|
| `-daten <folder>` | different data folder |
| `-port <n>` | different port (default 47811) |
| `-kein-browser` | do not open the browser |
| `-kein-tray` | no tray icon |
| `-version` | print version |

## Search profile

"+ Neuer Suchauftrag" opens the form.

- **Search term:** passed to both sites and searched in the listing text for the classification.
- **Places Germany:** enter place name or postcode, choose a suggestion with the mouse or arrow keys + Enter. Radius editable per place.
- **Places Netherlands:** four-digit postcode (e.g. 5911 Venlo, 6041 Roermond), "Hinzufügen" (add) or Enter.
- **Price:**
  - Target price = upper limit for "in budget".
  - Tolerance in % or €: listings up to target price + tolerance appear as "in tolerance".
  - Minimum price: hides cheaper listings, including "free".
  - Negotiation margin: negotiable ("VB") listings whose amount minus X % is below the upper limit appear as "via negotiation margin".
  - The computed upper limit is shown below the fields.
- **Required words:** must appear non-negated in title or text, otherwise the listing is hidden.
- **Excluded words** (custom and default list), effect selectable for each:
  - *mark:* highlight only
  - *filter title, mark text:* match in title hides the listing
  - *hard filter:* match in title or text hides the listing
  - Negated mentions ("kein Tausch", "nicht defekt", "Tausch nicht möglich") do not count.
- **Interval:** minutes between automatic searches (0 = off, otherwise at least 10).
- **Pages per place:** maximum number of result pages read per place (kleinanzeigen.de 25, marktplaats.nl 100 results per page).

## Result list

- "Jetzt suchen" (search now) starts the search; progress is shown in the header. Afterwards the listing texts of results passing the price and title check are loaded.
- Grouped by the listing's location (Germany, then Netherlands, alphabetically), sorted by price within each group; listings without an amount at the end.
- Labels: source, new, reserved (marktplaats.nl only), deleted, price classification, "nur Vergleich" (search term only used as comparison, e.g. "fast wie eine RTX 2080"), "zweifelhaft" (doubtful), "Ausschlusswort" (excluded word).
- Highlights in the text: red excluded word, grey negated excluded word, blue search term/required word, yellow search term as comparison.
- "Ausgeblendete zeigen" (show hidden): shows hidden listings with the reason.
- "Status prüfen" (check status; at the top for all, on a card for one): fetches the listing again and records active / reserved / deleted.
- "Löschen" (delete) on a card: removes the listing from the search profile; it does not reappear in later searches.

## Settings

- Scheduling, desktop notification, sound: master switches. Notification and sound follow every search run with new visible listings.
- "Meldung testen" (test notification) outputs notification and sound once.
- "Logdatei speichern" (save log file) downloads all log files as one text file (for attaching to an issue). The log contains request URLs, result counts and errors, no seller data.

## Sound on Linux

The sound is played with `pw-play`, `paplay` or `aplay` (the first one available).
