// Package kleinanzeigen bindet kleinanzeigen.de über die HTML-Seiten an.
package kleinanzeigen

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
	"github.com/OldRon1977/marktscout/internal/marktplaats"
	"github.com/PuerkitoBio/goquery"
)

// Basis ist die Adresse der Seite.
const Basis = "https://www.kleinanzeigen.de"

// SeitenGroesse: Treffer je Ergebnisseite.
const SeitenGroesse = 25

// ErrAufbau: Die Seite entspricht nicht dem erwarteten Aufbau.
var ErrAufbau = errors.New("kleinanzeigen: Seitenaufbau unbekannt")

// Anbindung an kleinanzeigen.de.
type Anbindung struct {
	c *abruf.Client
}

// Neu erzeugt eine Anbindung.
func Neu(c *abruf.Client) *Anbindung { return &Anbindung{c: c} }

// Ort ist ein Ortsvorschlag der Seite.
type Ort struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

// Orte liefert die Ortsvorschläge der Seite zu einem Ortsnamen oder einer Postleitzahl,
// in der Reihenfolge der Seite. „Deutschland“ (ID 0) wird ausgelassen.
func (a *Anbindung) Orte(ctx context.Context, text string) ([]Ort, error) {
	text = strings.TrimSpace(text)
	if text == "" {
		return nil, nil
	}
	r, err := a.c.Hole(ctx, Basis+"/s-ort-empfehlungen.json?query="+url.QueryEscape(text), true)
	if err != nil {
		return nil, err
	}
	if r.Status != http.StatusOK {
		return nil, fmt.Errorf("kleinanzeigen: Ortssuche liefert HTTP %d", r.Status)
	}
	dec := json.NewDecoder(bytes.NewReader(r.Body))
	if t, err := dec.Token(); err != nil || t != json.Delim('{') {
		return nil, fmt.Errorf("%w: Ortssuche ist kein JSON-Objekt", ErrAufbau)
	}
	var orte []Ort
	for dec.More() {
		t, err := dec.Token()
		if err != nil {
			return nil, err
		}
		schluessel, _ := t.(string)
		var name string
		if err := dec.Decode(&name); err != nil {
			return nil, err
		}
		id := strings.TrimPrefix(schluessel, "_")
		if id == "0" || id == "" {
			continue
		}
		orte = append(orte, Ort{ID: id, Name: name})
	}
	return orte, nil
}

// Suche beschreibt eine Suchanfrage.
type Suche struct {
	Suchbegriff string
	OrtID       string
	RadiusKm    int
	PreisMin    int64 // ganze Euro, 0 = ohne Untergrenze
	PreisMax    int64 // ganze Euro, 0 = ohne Obergrenze
	MaxSeiten   int   // 0 = alle Seiten
}

// Ergebnis einer Suche.
type Ergebnis struct {
	Anzeigen []anzeige.Anzeige
	Gesamt   int
}

var ortIDMuster = regexp.MustCompile(`^\d+$`)

// SuchURL baut die Adresse einer Ergebnisseite (seite ab 1).
func SuchURL(s Suche, seite int) string {
	var b strings.Builder
	b.WriteString(Basis + "/s-anzeige:angebote/")
	if s.PreisMin > 0 || s.PreisMax > 0 {
		grenze := func(v int64) string {
			if v <= 0 {
				return ""
			}
			return strconv.FormatInt(v, 10)
		}
		b.WriteString("preis:" + grenze(s.PreisMin) + ":" + grenze(s.PreisMax) + "/")
	}
	if seite > 1 {
		b.WriteString("seite:" + strconv.Itoa(seite) + "/")
	}
	b.WriteString(url.PathEscape(strings.Join(strings.Fields(strings.ToLower(s.Suchbegriff)), "-")))
	b.WriteString("/k0l" + s.OrtID + "r" + strconv.Itoa(s.RadiusKm))
	return b.String()
}

var zaehlerMuster = regexp.MustCompile(`([\d.]+)\s*-\s*([\d.]+)\s*von\s*([\d.]+)\s*Ergebnis`)

func zahl(s string) int {
	n, _ := strconv.Atoi(strings.ReplaceAll(s, ".", ""))
	return n
}

// Suche führt eine Suche aus und liest die Seiten bis MaxSeiten.
func (a *Anbindung) Suche(ctx context.Context, s Suche) (*Ergebnis, error) {
	if !ortIDMuster.MatchString(s.OrtID) || s.OrtID == "0" {
		return nil, fmt.Errorf("kleinanzeigen: Ort-ID %q ungültig", s.OrtID)
	}
	if s.RadiusKm <= 0 {
		return nil, fmt.Errorf("kleinanzeigen: Umkreis fehlt")
	}
	if strings.TrimSpace(s.Suchbegriff) == "" {
		return nil, fmt.Errorf("kleinanzeigen: Suchbegriff fehlt")
	}
	erg := &Ergebnis{}
	gesehen := map[string]bool{}
	for seite := 1; ; seite++ {
		r, err := a.c.Hole(ctx, SuchURL(s, seite), true)
		if err != nil {
			return nil, err
		}
		if r.Status != http.StatusOK {
			return nil, fmt.Errorf("kleinanzeigen: Suche liefert HTTP %d", r.Status)
		}
		anzeigen, von, gesamt, err := LeseErgebnisseite(r.Body)
		if err != nil {
			return nil, err
		}
		if seite == 1 {
			erg.Gesamt = gesamt
		} else if !gesamtPasst(gesamt, erg.Gesamt) || von != (seite-1)*SeitenGroesse+1 {
			// Falscher Pfad: Seite fällt still auf eine andere Suche zurück.
			return nil, fmt.Errorf("%w: Seite %d meldet Treffer ab %d von %d, erwartet ab %d von %d",
				ErrAufbau, seite, von, gesamt, (seite-1)*SeitenGroesse+1, erg.Gesamt)
		}
		for _, an := range anzeigen {
			if !gesehen[an.ID] {
				gesehen[an.ID] = true
				erg.Anzeigen = append(erg.Anzeigen, an)
			}
		}
		if seite*SeitenGroesse >= erg.Gesamt || len(anzeigen) == 0 || (s.MaxSeiten > 0 && seite >= s.MaxSeiten) {
			break
		}
	}
	return erg, nil
}

var (
	plzAnfang   = regexp.MustCompile(`^\d{5}\s`)
	ortMuster   = regexp.MustCompile(`^(\d{5})\s+(.+?)(?:\s*\((?:ca\.\s*)?(\d+)\s*km\))?$`)
	preisMuster = regexp.MustCompile(`^(?:(\d[\d.]*(?:,\d{1,2})?)\s*€)?\s*(VB)?$`)
	datumMuster = regexp.MustCompile(`(Heute|Gestern),\s*\d{1,2}:\d{2}|\d{2}\.\d{2}\.\d{4}`)
)

type ldBild struct {
	Title       string `json:"title"`
	Description string `json:"description"`
	ContentURL  string `json:"contentUrl"`
}

// LeseErgebnisseite liest die Treffer einer Ergebnisseite (nur Treffer im Umkreis).
// gesamtPasst: Zwischen zwei Seitenabrufen ändert sich die Trefferzahl durch neue oder
// gelöschte Anzeigen leicht. Ein falscher Pfad liefert dagegen alle Anzeigen Deutschlands.
func gesamtPasst(seite, erste int) bool {
	d := seite - erste
	if d < 0 {
		d = -d
	}
	return d <= 10 || d*20 <= erste // höchstens 10 Treffer oder 5 % Abweichung
}

// von und gesamt stammen aus dem Zähler „1 - 25 von 1.616“.
func LeseErgebnisseite(body []byte) (anzeigen []anzeige.Anzeige, von, gesamt int, err error) {
	doc, err := goquery.NewDocumentFromReader(bytes.NewReader(body))
	if err != nil {
		return nil, 0, 0, err
	}
	tabelle := doc.Find("#srchrslt-adtable")
	if tabelle.Length() == 0 {
		if strings.Contains(doc.Text(), "keine Ergebnisse") {
			return nil, 0, 0, nil
		}
		return nil, 0, 0, fmt.Errorf("%w: Treffertabelle fehlt", ErrAufbau)
	}
	m := zaehlerMuster.FindStringSubmatch(doc.Find("#srchrslt-results").Text())
	if m == nil {
		m = zaehlerMuster.FindStringSubmatch(doc.Text())
	}
	if m == nil {
		return nil, 0, 0, fmt.Errorf("%w: Trefferzähler fehlt", ErrAufbau)
	}
	von, gesamt = zahl(m[1]), zahl(m[3])

	var fehler error
	tabelle.Find("article[data-adid]").Each(func(_ int, e *goquery.Selection) {
		an, err := leseArtikel(e)
		if err != nil {
			fehler = err
			return
		}
		anzeigen = append(anzeigen, an)
	})
	if fehler != nil {
		return nil, 0, 0, fehler
	}
	return anzeigen, von, gesamt, nil
}

func leseArtikel(e *goquery.Selection) (anzeige.Anzeige, error) {
	id, _ := e.Attr("data-adid")
	href, _ := e.Attr("data-href")
	an := anzeige.Anzeige{
		Quelle:       anzeige.Kleinanzeigen,
		ID:           id,
		URL:          Basis + href,
		Land:         "DE",
		EntfernungKm: -1,
		Status:       anzeige.StatusAktiv,
		PreisTyp:     anzeige.PreisOhne,
	}
	var ld ldBild
	if j := e.Find(`script[type="application/ld+json"]`).First(); j.Length() > 0 {
		_ = json.Unmarshal([]byte(j.Text()), &ld)
	}
	an.Titel = strings.TrimSpace(ld.Title)
	if an.Titel == "" {
		an.Titel = strings.TrimSpace(e.Find("h3").First().Text())
	}
	an.Beschreibung = strings.TrimSpace(ld.Description)
	an.BildURL = ld.ContentURL

	e.Find("span").EachWithBreak(func(_ int, s *goquery.Selection) bool {
		t := strings.Join(strings.Fields(s.Text()), " ")
		if !plzAnfang.MatchString(t) {
			return true
		}
		// Entfernung steht in einem benachbarten span: Text des umgebenden Elements lesen.
		t = strings.Join(strings.Fields(s.Parent().Text()), " ")
		if m := ortMuster.FindStringSubmatch(t); m != nil {
			an.PLZ, an.Ort = m[1], m[2]
			if m[3] != "" {
				km, _ := strconv.Atoi(m[3])
				an.EntfernungKm = float64(km)
			}
			return false
		}
		return true
	})
	e.Find("p").EachWithBreak(func(_ int, p *goquery.Selection) bool {
		if klasse, _ := p.Attr("class"); strings.Contains(klasse, "line-through") {
			return true
		}
		t := strings.Join(strings.Fields(p.Text()), " ")
		if typ, cent, ok := LesePreis(t); ok {
			an.PreisTyp, an.PreisCent = typ, cent
			return false
		}
		return true
	})
	tx := strings.Join(strings.Fields(e.Text()), " ")
	if d := datumMuster.FindString(tx); d != "" {
		an.Eingestellt = d
	} else {
		an.Hervorgehoben = true
	}
	if an.ID == "" || href == "" || an.Titel == "" {
		return an, fmt.Errorf("%w: Treffer ohne ID, Link oder Titel", ErrAufbau)
	}
	return an, nil
}

// LesePreis wertet einen Preistext wie „650 € VB“, „249 €“, „VB“ oder „Zu verschenken“ aus.
func LesePreis(t string) (anzeige.PreisTyp, int64, bool) {
	t = strings.TrimSpace(t)
	switch {
	case t == "":
		return "", 0, false
	case strings.EqualFold(t, "Zu verschenken"):
		return anzeige.PreisVerschenken, 0, true
	}
	m := preisMuster.FindStringSubmatch(t)
	if m == nil || (m[1] == "" && m[2] == "") {
		return "", 0, false
	}
	if m[1] == "" {
		return anzeige.PreisVBOhne, 0, true
	}
	betrag := strings.ReplaceAll(m[1], ".", "")
	euro, cent := betrag, "0"
	if i := strings.Index(betrag, ","); i >= 0 {
		euro, cent = betrag[:i], betrag[i+1:]
		if len(cent) == 1 {
			cent += "0"
		}
	}
	e, _ := strconv.ParseInt(euro, 10, 64)
	c, _ := strconv.ParseInt(cent, 10, 64)
	wert := e*100 + c
	if m[2] != "" {
		if wert == 0 {
			return anzeige.PreisVBOhne, 0, true
		}
		return anzeige.PreisVB, wert, true
	}
	return anzeige.PreisFest, wert, true
}

var (
	idMuster        = regexp.MustCompile(`^\d+$`)
	ortDetailMuster = regexp.MustCompile(`^(\d{5})\s+(?:.+?\s+-\s+)?(.+)$`)
)

// Detail ruft die Detailseite einer Anzeige ab. Bei gelöschter Anzeige ist nur
// Status gesetzt (StatusGeloescht).
func (a *Anbindung) Detail(ctx context.Context, id string) (*anzeige.Anzeige, error) {
	if !idMuster.MatchString(id) {
		return nil, fmt.Errorf("kleinanzeigen: Anzeigen-ID %q ungültig", id)
	}
	adresse := Basis + "/s-anzeige/" + id
	r, err := a.c.Hole(ctx, adresse, false)
	if err != nil {
		return nil, err
	}
	switch {
	case r.Status == http.StatusOK:
	case r.Status >= 300 && r.Status < 400:
		return &anzeige.Anzeige{Quelle: anzeige.Kleinanzeigen, ID: id, URL: adresse, Status: anzeige.StatusGeloescht}, nil
	case r.Status == http.StatusNotFound || r.Status == http.StatusGone:
		return &anzeige.Anzeige{Quelle: anzeige.Kleinanzeigen, ID: id, URL: adresse, Status: anzeige.StatusGeloescht}, nil
	default:
		return nil, fmt.Errorf("kleinanzeigen: Detailseite liefert HTTP %d", r.Status)
	}
	doc, err := goquery.NewDocumentFromReader(bytes.NewReader(r.Body))
	if err != nil {
		return nil, err
	}
	titel := strings.Join(strings.Fields(doc.Find("#viewad-title").First().Text()), " ")
	beschr := doc.Find("#viewad-description-text").First()
	if titel == "" || beschr.Length() == 0 {
		return nil, fmt.Errorf("%w: Detailseite ohne Titel oder Beschreibung", ErrAufbau)
	}
	an := &anzeige.Anzeige{
		Quelle:                   anzeige.Kleinanzeigen,
		ID:                       id,
		URL:                      adresse,
		Titel:                    titel,
		Beschreibung:             marktplaats.TextMitUmbruechen(beschr),
		BeschreibungVollstaendig: true,
		Land:                     "DE",
		EntfernungKm:             -1,
		Status:                   anzeige.StatusAktiv,
		PreisTyp:                 anzeige.PreisOhne,
		Eingestellt:              strings.TrimSpace(doc.Find("#viewad-extra-info span").First().Text()),
	}
	if u, ok := doc.Find(`meta[property="og:url"]`).Attr("content"); ok && strings.HasPrefix(u, Basis) {
		an.URL = u
	}
	if typ, cent, ok := LesePreis(strings.Join(strings.Fields(doc.Find("#viewad-price").First().Text()), " ")); ok {
		an.PreisTyp, an.PreisCent = typ, cent
	}
	ort := strings.Join(strings.Fields(doc.Find("#viewad-locality").First().Text()), " ")
	if m := ortDetailMuster.FindStringSubmatch(ort); m != nil {
		an.PLZ, an.Ort = m[1], m[2]
	}
	lies := func(eig string) float64 {
		v, _ := doc.Find(`meta[property="` + eig + `"]`).Attr("content")
		f, _ := strconv.ParseFloat(v, 64)
		return f
	}
	an.Lat, an.Lon = lies("og:latitude"), lies("og:longitude")
	return an, nil
}
