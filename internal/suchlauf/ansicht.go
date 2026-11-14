package suchlauf

import (
	"sort"
	"strings"

	"github.com/OldRon1977/marktscout/internal/auftrag"
	"github.com/OldRon1977/marktscout/internal/preis"
	"github.com/OldRon1977/marktscout/internal/speicher"
	"github.com/OldRon1977/marktscout/internal/textpruefung"
)

// Teil eines Textes für die Anzeige; Art leer = normaler Text.
type Teil struct {
	T   string           `json:"t"`
	Art textpruefung.Art `json:"art,omitempty"`
}

// Eintrag ist ein bewerteter Treffer.
type Eintrag struct {
	speicher.Treffer
	Preis      preis.Ergebnis          `json:"preis"`
	Einstufung textpruefung.Einstufung `json:"einstufung,omitempty"`
	Sichtbar   bool                    `json:"sichtbar"`
	Gruende    []string                `json:"gruende,omitempty"`
	TitelTeile []Teil                  `json:"titelTeile"`
	TextTeile  []Teil                  `json:"textTeile"`
	Markiert   bool                    `json:"markiert"` // enthält Ausschlusswort (nicht ausgeblendet)
}

// Gruppe fasst die Einträge eines Orts zusammen.
type Gruppe struct {
	Land      string    `json:"land"`
	Ort       string    `json:"ort"`
	Eintraege []Eintrag `json:"eintraege"`
}

// Liste ist die Ergebnisliste eines Auftrags.
type Liste struct {
	Gruppen      []Gruppe `json:"gruppen"`
	Sichtbar     int      `json:"sichtbar"`
	Ausgeblendet int      `json:"ausgeblendet"`
	Neu          int      `json:"neu"`
}

// Bewerte bewertet alle Treffer eines Auftrags.
func Bewerte(a auftrag.Auftrag, tr []speicher.Treffer, b textpruefung.Bewerter) []Eintrag {
	regeln := a.Regeln()
	aus := make([]Eintrag, 0, len(tr))
	for _, t := range tr {
		p := a.Preis.Bewerte(t.PreisTyp, t.PreisCent)
		tx := b.Bewerte(t.Titel, t.Beschreibung, regeln)
		e := Eintrag{Treffer: t, Preis: p, Einstufung: tx.Einstufung, Sichtbar: p.Sichtbar && tx.Sichtbar}
		if !p.Sichtbar {
			e.Gruende = append(e.Gruende, "Preis: "+p.Grund)
		}
		e.Gruende = append(e.Gruende, tx.Gruende...)
		var ft, fx []textpruefung.Fundstelle
		for _, f := range tx.Fundstellen {
			if f.Art == textpruefung.ArtAusschluss {
				e.Markiert = true
			}
			if f.Feld == textpruefung.FeldTitel {
				ft = append(ft, f)
			} else {
				fx = append(fx, f)
			}
		}
		e.TitelTeile, e.TextTeile = zerteile(t.Titel, ft), zerteile(t.Beschreibung, fx)
		aus = append(aus, e)
	}
	return aus
}

// zerteile schneidet s an den Fundstellen (sortiert, überlappende werden ausgelassen).
func zerteile(s string, fs []textpruefung.Fundstelle) []Teil {
	var aus []Teil
	pos := 0
	for _, f := range fs {
		if f.Start < pos || f.Ende > len(s) {
			continue
		}
		if f.Start > pos {
			aus = append(aus, Teil{T: s[pos:f.Start]})
		}
		aus = append(aus, Teil{T: s[f.Start:f.Ende], Art: f.Art})
		pos = f.Ende
	}
	if pos < len(s) {
		aus = append(aus, Teil{T: s[pos:]})
	}
	return aus
}

var landFolge = map[string]int{"DE": 0, "NL": 1, "BE": 2}

// Ergebnisse baut die nach Ort gruppierte, nach Preis sortierte Liste.
// Anzeigen ohne Betrag stehen am Ende ihrer Ortsgruppe.
func Ergebnisse(a auftrag.Auftrag, tr []speicher.Treffer, b textpruefung.Bewerter, mitAusgeblendeten bool) Liste {
	var l Liste
	gruppen := map[string]*Gruppe{}
	for _, e := range Bewerte(a, tr, b) {
		if e.Sichtbar {
			l.Sichtbar++
			if e.Neu {
				l.Neu++
			}
		} else {
			l.Ausgeblendet++
			if !mitAusgeblendeten {
				continue
			}
		}
		ort := strings.TrimSpace(e.Ort)
		if ort == "" {
			ort = "ohne Ort"
		}
		k := e.Land + "\x00" + strings.ToLower(ort)
		g := gruppen[k]
		if g == nil {
			g = &Gruppe{Land: e.Land, Ort: ort}
			gruppen[k] = g
		}
		g.Eintraege = append(g.Eintraege, e)
	}
	for _, g := range gruppen {
		sort.SliceStable(g.Eintraege, func(i, j int) bool {
			x, y := g.Eintraege[i], g.Eintraege[j]
			bx, by := x.PreisTyp.HatBetrag(), y.PreisTyp.HatBetrag()
			if bx != by {
				return bx
			}
			if x.PreisCent != y.PreisCent {
				return x.PreisCent < y.PreisCent
			}
			return x.Erstmals.After(y.Erstmals)
		})
		l.Gruppen = append(l.Gruppen, *g)
	}
	sort.Slice(l.Gruppen, func(i, j int) bool {
		x, y := l.Gruppen[i], l.Gruppen[j]
		if landFolge[x.Land] != landFolge[y.Land] {
			return landFolge[x.Land] < landFolge[y.Land]
		}
		return strings.ToLower(x.Ort) < strings.ToLower(y.Ort)
	})
	return l
}
