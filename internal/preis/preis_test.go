package preis

import (
	"testing"

	"github.com/OldRon1977/marktscout/internal/anzeige"
)

func TestObergrenze(t *testing.T) {
	for _, f := range []struct {
		r    Regel
		cent int64
	}{
		{Regel{SuchpreisCent: 50000, ToleranzArt: Prozent, ToleranzWert: 10}, 55000},
		{Regel{SuchpreisCent: 50000, ToleranzArt: Euro, ToleranzWert: 50}, 55000},
		{Regel{SuchpreisCent: 50000}, 50000},
		{Regel{}, 0},
	} {
		if got := f.r.ObergrenzeCent(); got != f.cent {
			t.Errorf("%+v: Obergrenze %d, erwartet %d", f.r, got, f.cent)
		}
	}
}

func TestBewerte(t *testing.T) {
	// Beispiel aus den Anforderungen: 500 € + 10 %, VB-Spielraum 10 %
	r := Regel{SuchpreisCent: 50000, ToleranzArt: Prozent, ToleranzWert: 10, MindestpreisCent: 10000, VBSpielraum: true, VBProzent: 10}
	for _, f := range []struct {
		typ      anzeige.PreisTyp
		cent     int64
		kennung  Kennung
		sichtbar bool
	}{
		{anzeige.PreisFest, 45000, ImBudget, true},
		{anzeige.PreisFest, 50000, ImBudget, true},
		{anzeige.PreisFest, 54000, InToleranz, true},
		{anzeige.PreisFest, 55000, InToleranz, true},
		{anzeige.PreisFest, 55100, Darueber, false},
		{anzeige.PreisVB, 60000, VBSpielraum, true}, // 600 € VB − 10 % = 540 € ≤ 550 €
		{anzeige.PreisVB, 62000, Darueber, false},   // 620 € − 10 % = 558 € > 550 €
		{anzeige.PreisFest, 60000, Darueber, false}, // Festpreis bekommt keinen Spielraum
		{anzeige.PreisFest, 9900, Darunter, false},
		{anzeige.PreisVBOhne, 0, OhnePreis, true},
		{anzeige.PreisOhne, 0, OhnePreis, true},
		{anzeige.PreisVerschenken, 0, Darunter, false},
	} {
		e := r.Bewerte(f.typ, f.cent)
		if e.Kennung != f.kennung || e.Sichtbar != f.sichtbar {
			t.Errorf("%s %d: %s/%v, erwartet %s/%v", f.typ, f.cent, e.Kennung, e.Sichtbar, f.kennung, f.sichtbar)
		}
	}
	if e := r.Bewerte(anzeige.PreisFest, 54000); e.Abweichung != 8 {
		t.Errorf("Abweichung 540 € zu 500 €: %.1f %%, erwartet 8", e.Abweichung)
	}
	ohne := Regel{}
	if e := ohne.Bewerte(anzeige.PreisVerschenken, 0); !e.Sichtbar {
		t.Error("verschenken ohne Mindestpreis ausgeblendet")
	}
	if e := ohne.Bewerte(anzeige.PreisFest, 99999900); e.Kennung != OhneGrenze || !e.Sichtbar {
		t.Errorf("ohne Suchpreis: %+v", e)
	}
}

func TestSeitenGrenzen(t *testing.T) {
	r := Regel{SuchpreisCent: 50000, ToleranzArt: Prozent, ToleranzWert: 10, MindestpreisCent: 10000, VBSpielraum: true, VBProzent: 10}
	min, max := r.SeitenGrenzenCent()
	// 550 € / 0,9 = 611,12 €: VB-Anzeige zu 611 € liegt nach Abzug bei 549,90 €
	if min != 10000 || max != 61112 {
		t.Fatalf("Seitengrenzen %d–%d, erwartet 10000–61112", min, max)
	}
	if e := r.Bewerte(anzeige.PreisVB, 61100); e.Kennung != VBSpielraum {
		t.Errorf("VB an der Seitengrenze: %s", e.Kennung)
	}
	r.VBSpielraum = false
	if _, max := r.SeitenGrenzenCent(); max != 55000 {
		t.Errorf("ohne VB-Spielraum: %d, erwartet 55000", max)
	}
}
