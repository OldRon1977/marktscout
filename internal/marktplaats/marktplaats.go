// Package marktplaats bindet marktplaats.nl über die interne Such-Schnittstelle an.
package marktplaats

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"regexp"
	"strconv"
	"strings"

	"github.com/OldRon1977/marktscout/internal/abruf"
	"github.com/OldRon1977/marktscout/internal/anzeige"
	"github.com/PuerkitoBio/goquery"
)

// Basis ist die Adresse der Seite.
const Basis = "https://www.marktplaats.nl"

const seitenGroesse = 100

// ErrPostleitzahl: Die Seite hat die Postleitzahl nicht als Mittelpunkt angenommen.
var ErrPostleitzahl = errors.New("marktplaats: Postleitzahl ungültig, Seite sucht ohne Umkreis")

var plzMuster = regexp.MustCompile(`^[1-9][0-9]{3}([A-Z]{2})?$`)

// gesuchMuster erkennt Gesuche am Titel.
var gesuchMuster = regexp.MustCompile(`(?i)(^|[^\pL])(gezocht|gevraagd)([^\pL]|$)|^[^\pL]*zoek([^\pL]|$)`)

// IstGesuch meldet, ob ein Titel ein Gesuch kennzeichnet.
func IstGesuch(titel string) bool { return gesuchMuster.MatchString(titel) }

// Anbindung an marktplaats.nl.
type Anbindung struct {
	c *abruf.Client
}

// Neu erzeugt eine Anbindung.
func Neu(c *abruf.Client) *Anbindung { return &Anbindung{c: c} }

// Suche beschreibt eine Suchanfrage.
type Suche struct {
	Suchbegriff  string
	Postleitzahl string // vierstellig oder mit Buchstaben, z. B. 5911 oder 5911AB
	RadiusKm     int
	PreisMinCent int64 // 0 = ohne Untergrenze
	PreisMaxCent int64 // 0 = ohne Obergrenze
	MaxSeiten    int   // 0 = alle erlaubten Seiten
}

// Ergebnis einer Suche.
type Ergebnis struct {
	Anzeigen []anzeige.Anzeige
	Gesamt   int // Trefferzahl laut Seite
	Gesuche  int // als Gesuch erkannte und verworfene Anzeigen
}

type antwort struct {
	TotalResultCount     int       `json:"totalResultCount"`
	MaxAllowedPageNumber int       `json:"maxAllowedPageNumber"`
	Listings             []listing `json:"listings"`
}

type listing struct {
	ItemID      string `json:"itemId"`
	Title       string `json:"title"`
	Description string `json:"description"`
	PriceInfo   struct {
		PriceCents int64  `json:"priceCents"`
		PriceType  string `json:"priceType"`
	} `json:"priceInfo"`
	Location struct {
		CityName       string  `json:"cityName"`
		CountryName    string  `json:"countryName"`
		DistanceMeters int     `json:"distanceMeters"`
		Latitude       float64 `json:"latitude"`
		Longitude      float64 `json:"longitude"`
	} `json:"location"`
	Date            string `json:"date"`
	PriorityProduct string `json:"priorityProduct"`
	Reserved        bool   `json:"reserved"`
	VipURL          string `json:"vipUrl"`
	Pictures        []struct {
		MediumURL string `json:"mediumUrl"`
	} `json:"pictures"`
}

// NormalisierePostleitzahl entfernt Leerzeichen und prüft das Format.
func NormalisierePostleitzahl(p string) (string, error) {
	p = strings.ToUpper(strings.ReplaceAll(strings.TrimSpace(p), " ", ""))
	if !plzMuster.MatchString(p) {
		return "", fmt.Errorf("marktplaats: %q ist keine niederländische Postleitzahl", p)
	}
	return p, nil
}

// SuchURL baut die Adresse einer Ergebnisseite.
func SuchURL(s Suche, offset int) string {
	q := url.Values{}
	q.Set("query", s.Suchbegriff)
	q.Set("postcode", s.Postleitzahl)
	q.Set("distanceMeters", strconv.Itoa(s.RadiusKm*1000))
	q.Set("limit", strconv.Itoa(seitenGroesse))
	q.Set("offset", strconv.Itoa(offset))
	if s.PreisMinCent > 0 || s.PreisMaxCent > 0 {
		grenze := func(v int64) string {
			if v <= 0 {
				return "null"
			}
			return strconv.FormatInt(v, 10)
		}
		q.Set("attributeRanges[]", "PriceCents:"+grenze(s.PreisMinCent)+":"+grenze(s.PreisMaxCent))
	}
	return Basis + "/lrp/api/search?" + q.Encode()
}

// Suche führt eine Suche aus und liest alle Seiten bis MaxSeiten.
// Gesuche (am Titel erkannt) werden verworfen.
func (a *Anbindung) Suche(ctx context.Context, s Suche) (*Ergebnis, error) {
	plz, err := NormalisierePostleitzahl(s.Postleitzahl)
	if err != nil {
		return nil, err
	}
	s.Postleitzahl = plz
	if s.RadiusKm <= 0 {
		return nil, fmt.Errorf("marktplaats: Umkreis fehlt")
	}

	erg := &Ergebnis{}
	gesehen := map[string]bool{}
	for seite := 0; ; seite++ {
		r, err := a.c.Hole(ctx, SuchURL(s, seite*seitenGroesse), true)
		if err != nil {
			return nil, err
		}
		if r.Status != http.StatusOK {
			return nil, fmt.Errorf("marktplaats: Suche liefert HTTP %d", r.Status)
		}
		var ant antwort
		if err := json.Unmarshal(r.Body, &ant); err != nil {
			return nil, fmt.Errorf("marktplaats: Antwort nicht lesbar: %w", err)
		}
		if seite == 0 {
			erg.Gesamt = ant.TotalResultCount
			if len(ant.Listings) > 0 && ohneUmkreis(ant.Listings) {
				return nil, ErrPostleitzahl
			}
		}
		for _, l := range ant.Listings {
			if gesehen[l.ItemID] {
				continue
			}
			gesehen[l.ItemID] = true
			if IstGesuch(l.Title) {
				erg.Gesuche++
				continue
			}
			erg.Anzeigen = append(erg.Anzeigen, wandle(l))
		}
		naechste := seite + 1
		// Eine Seite kann weniger als seitenGroesse Anzeigen liefern, obwohl weitere folgen;
		// Ende erst bei leerer Seite, Gesamtzahl, Seitengrenze der Seite oder MaxSeiten.
		if len(ant.Listings) == 0 ||
			naechste*seitenGroesse >= ant.TotalResultCount ||
			(ant.MaxAllowedPageNumber > 0 && naechste >= ant.MaxAllowedPageNumber) ||
			(s.MaxSeiten > 0 && naechste >= s.MaxSeiten) {
			break
		}
	}
	return erg, nil
}

// ohneUmkreis: Alle Anzeigen tragen distanceMeters -1000, die Seite hat keinen Mittelpunkt.
func ohneUmkreis(ls []listing) bool {
	for _, l := range ls {
		if l.Location.DistanceMeters >= 0 {
			return false
		}
	}
	return true
}

func preisTyp(t string) anzeige.PreisTyp {
	switch t {
	case "FIXED":
		return anzeige.PreisFest
	case "MIN_BID":
		return anzeige.PreisVB
	case "FAST_BID":
		return anzeige.PreisVBOhne
	case "FREE":
		return anzeige.PreisVerschenken
	default:
		return anzeige.PreisOhne
	}
}

// bereinigePreis: Gebot ohne Betrag (MIN_BID mit 0) zählt als VB ohne Betrag.
func bereinigePreis(t anzeige.PreisTyp, cent int64) (anzeige.PreisTyp, int64) {
	if t == anzeige.PreisVB && cent <= 0 {
		return anzeige.PreisVBOhne, 0
	}
	if !t.HatBetrag() {
		return t, 0
	}
	return t, cent
}

func land(name string) string {
	switch name {
	case "Nederland":
		return "NL"
	case "België", "Belgie", "Belgium":
		return "BE"
	case "Duitsland", "Deutschland":
		return "DE"
	}
	return "NL"
}

func wandle(l listing) anzeige.Anzeige {
	a := anzeige.Anzeige{
		Quelle:        anzeige.Marktplaats,
		ID:            l.ItemID,
		URL:           Basis + l.VipURL,
		Titel:         strings.TrimSpace(l.Title),
		Beschreibung:  strings.TrimSpace(l.Description),
		PreisCent:     l.PriceInfo.PriceCents,
		PreisTyp:      preisTyp(l.PriceInfo.PriceType),
		Ort:           l.Location.CityName,
		Land:          land(l.Location.CountryName),
		Lat:           l.Location.Latitude,
		Lon:           l.Location.Longitude,
		EntfernungKm:  -1,
		Eingestellt:   l.Date,
		Status:        anzeige.StatusAktiv,
		Hervorgehoben: l.PriorityProduct != "" && l.PriorityProduct != "NONE",
	}
	a.PreisTyp, a.PreisCent = bereinigePreis(a.PreisTyp, a.PreisCent)
	if l.Location.DistanceMeters >= 0 {
		a.EntfernungKm = float64(l.Location.DistanceMeters) / 1000
	}
	if l.Reserved {
		a.Status = anzeige.StatusReserviert
	}
	if len(l.Pictures) > 0 {
		a.BildURL = l.Pictures[0].MediumURL
	}
	return a
}

var configMuster = regexp.MustCompile(`(?s)window\.__CONFIG__\s*=\s*(\{.*?\});\s*</script>`)

type detailConfig struct {
	Listing struct {
		ItemID    string `json:"itemId"`
		Title     string `json:"title"`
		PriceInfo struct {
			PriceCents int64  `json:"priceCents"`
			PriceType  string `json:"priceType"`
		} `json:"priceInfo"`
		IsReserved bool `json:"isReserved"`
		Stats      struct {
			Since string `json:"since"`
		} `json:"stats"`
	} `json:"listing"`
}

// Detail ruft die Detailseite ab. Liefert die Anzeige mit vollständiger Beschreibung
// und Status. Bei gelöschter Anzeige ist nur Status gesetzt (StatusGeloescht).
func (a *Anbindung) Detail(ctx context.Context, adresse string) (*anzeige.Anzeige, error) {
	if !strings.HasPrefix(adresse, Basis+"/v/") {
		return nil, fmt.Errorf("marktplaats: keine Detailadresse: %s", adresse)
	}
	r, err := a.c.Hole(ctx, adresse, true)
	if err != nil {
		return nil, err
	}
	switch r.Status {
	case http.StatusOK:
	case http.StatusGone, http.StatusNotFound:
		return &anzeige.Anzeige{Quelle: anzeige.Marktplaats, URL: adresse, Status: anzeige.StatusGeloescht}, nil
	default:
		return nil, fmt.Errorf("marktplaats: Detailseite liefert HTTP %d", r.Status)
	}

	m := configMuster.FindSubmatch(r.Body)
	if m == nil {
		return nil, fmt.Errorf("marktplaats: Detailseite ohne __CONFIG__")
	}
	var cfg detailConfig
	if err := json.Unmarshal(m[1], &cfg); err != nil {
		return nil, fmt.Errorf("marktplaats: __CONFIG__ nicht lesbar: %w", err)
	}
	doc, err := goquery.NewDocumentFromReader(bytes.NewReader(r.Body))
	if err != nil {
		return nil, err
	}
	beschr := doc.Find(`[data-collapsable="description"]`).First()
	if beschr.Length() == 0 {
		return nil, fmt.Errorf("marktplaats: Detailseite ohne Beschreibung")
	}

	l := cfg.Listing
	erg := &anzeige.Anzeige{
		Quelle:                   anzeige.Marktplaats,
		ID:                       l.ItemID,
		URL:                      r.URL,
		Titel:                    strings.TrimSpace(l.Title),
		Beschreibung:             TextMitUmbruechen(beschr),
		BeschreibungVollstaendig: true,
		PreisCent:                l.PriceInfo.PriceCents,
		PreisTyp:                 preisTyp(l.PriceInfo.PriceType),
		Eingestellt:              l.Stats.Since,
		Status:                   anzeige.StatusAktiv,
		EntfernungKm:             -1,
	}
	erg.PreisTyp, erg.PreisCent = bereinigePreis(erg.PreisTyp, erg.PreisCent)
	if l.IsReserved {
		erg.Status = anzeige.StatusReserviert
	}
	return erg, nil
}

// TextMitUmbruechen liefert den Text eines Elements, <br> und Blockgrenzen als Zeilenumbruch.
func TextMitUmbruechen(s *goquery.Selection) string {
	k := s.Clone()
	k.Find("br").ReplaceWithHtml("\n")
	k.Find("p, div, li").Each(func(_ int, e *goquery.Selection) { e.AppendHtml("\n") })
	zeilen := strings.Split(k.Text(), "\n")
	aus := make([]string, 0, len(zeilen))
	leer := 0
	for _, z := range zeilen {
		z = strings.Join(strings.Fields(z), " ")
		if z == "" {
			leer++
			if leer > 1 || len(aus) == 0 {
				continue
			}
		} else {
			leer = 0
		}
		aus = append(aus, z)
	}
	return strings.TrimSpace(strings.Join(aus, "\n"))
}
