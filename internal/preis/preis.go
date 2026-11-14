// Package preis bewertet den Preis einer Anzeige gegen die Preisregeln eines Suchauftrags.
package preis

import (
	"math"

	"github.com/OldRon1977/marktscout/internal/anzeige"
)

// ToleranzArt: Toleranz in Prozent oder als fester Betrag.
type ToleranzArt string

const (
	Prozent ToleranzArt = "prozent"
	Euro    ToleranzArt = "euro"
)

// Regel enthält die Preisangaben eines Suchauftrags. Beträge in Cent, 0 = nicht gesetzt.
type Regel struct {
	SuchpreisCent    int64       `json:"suchpreisCent"`
	ToleranzArt      ToleranzArt `json:"toleranzArt"`
	ToleranzWert     float64     `json:"toleranzWert"` // Prozent oder Euro
	MindestpreisCent int64       `json:"mindestpreisCent"`
	VBSpielraum      bool        `json:"vbSpielraum"`
	VBProzent        float64     `json:"vbProzent"`
}

// Kennung der Preisbewertung.
type Kennung string

const (
	ImBudget    Kennung = "budget"       // bis Suchpreis
	InToleranz  Kennung = "toleranz"     // über Suchpreis, bis Obergrenze
	VBSpielraum Kennung = "vb_spielraum" // VB-Betrag abzüglich Spielraum bis Obergrenze
	OhneGrenze  Kennung = "ohne_grenze"  // kein Suchpreis gesetzt
	OhnePreis   Kennung = "ohne_preis"   // VB ohne Betrag, auf Anfrage
	Verschenken Kennung = "verschenken"
	Darueber    Kennung = "darueber" // über Obergrenze (ausgeblendet)
	Darunter    Kennung = "darunter" // unter Mindestpreis (ausgeblendet)
)

// Ergebnis der Preisbewertung.
type Ergebnis struct {
	Sichtbar   bool    `json:"sichtbar"`
	Kennung    Kennung `json:"kennung"`
	Abweichung float64 `json:"abweichung"` // Prozent über dem Suchpreis
	Grund      string  `json:"grund,omitempty"`
}

// ObergrenzeCent = Suchpreis + Toleranz; 0, wenn kein Suchpreis gesetzt ist.
func (r Regel) ObergrenzeCent() int64 {
	if r.SuchpreisCent <= 0 {
		return 0
	}
	switch r.ToleranzArt {
	case Prozent:
		return r.SuchpreisCent + int64(math.Round(float64(r.SuchpreisCent)*r.ToleranzWert/100))
	case Euro:
		return r.SuchpreisCent + int64(math.Round(r.ToleranzWert*100))
	}
	return r.SuchpreisCent
}

// SeitenGrenzenCent liefert Unter- und Obergrenze für die Suche auf der Seite.
// Mit VB-Spielraum wird die Obergrenze so angehoben, dass VB-Anzeigen im Spielraum mitkommen.
func (r Regel) SeitenGrenzenCent() (min, max int64) {
	min = r.MindestpreisCent
	max = r.ObergrenzeCent()
	if max > 0 && r.VBSpielraum && r.VBProzent > 0 && r.VBProzent < 100 {
		max = int64(math.Ceil(float64(max) / (1 - r.VBProzent/100)))
	}
	return min, max
}

// Bewerte ordnet einen Preis ein.
func (r Regel) Bewerte(typ anzeige.PreisTyp, cent int64) Ergebnis {
	switch {
	case typ == anzeige.PreisVerschenken:
		if r.MindestpreisCent > 0 {
			return Ergebnis{Kennung: Darunter, Grund: "zu verschenken, Mindestpreis gesetzt"}
		}
		return Ergebnis{Sichtbar: true, Kennung: Verschenken}
	case !typ.HatBetrag():
		return Ergebnis{Sichtbar: true, Kennung: OhnePreis}
	}
	if r.MindestpreisCent > 0 && cent < r.MindestpreisCent {
		return Ergebnis{Kennung: Darunter, Grund: "unter Mindestpreis"}
	}
	if r.SuchpreisCent <= 0 {
		return Ergebnis{Sichtbar: true, Kennung: OhneGrenze}
	}
	abw := math.Round((float64(cent)/float64(r.SuchpreisCent)-1)*1000) / 10
	ober := r.ObergrenzeCent()
	switch {
	case cent <= r.SuchpreisCent:
		return Ergebnis{Sichtbar: true, Kennung: ImBudget, Abweichung: abw}
	case cent <= ober:
		return Ergebnis{Sichtbar: true, Kennung: InToleranz, Abweichung: abw}
	case typ == anzeige.PreisVB && r.VBSpielraum && r.VBProzent > 0 &&
		float64(cent)*(1-r.VBProzent/100) <= float64(ober):
		return Ergebnis{Sichtbar: true, Kennung: VBSpielraum, Abweichung: abw}
	}
	return Ergebnis{Kennung: Darueber, Abweichung: abw, Grund: "über Obergrenze"}
}
