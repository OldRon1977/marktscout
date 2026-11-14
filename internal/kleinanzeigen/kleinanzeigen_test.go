package kleinanzeigen

import (
	"context"
	"regexp"
	"strings"
	"testing"
	"time"

	"github.com/OldRon1977/marktscout/internal/abruf"
	"github.com/OldRon1977/marktscout/internal/anzeige"
)

// Live-Tests gegen kleinanzeigen.de. Geprüft werden Struktur und Plausibilität.

func anbindung() *Anbindung { return Neu(abruf.Standard(nil)) }

func ctx(t *testing.T) context.Context {
	c, abbruch := context.WithTimeout(context.Background(), 4*time.Minute)
	t.Cleanup(abbruch)
	return c
}

const koeln = "945"

func TestOrte(t *testing.T) {
	a := anbindung()
	c := ctx(t)
	orte, err := a.Orte(c, "Köln")
	if err != nil {
		t.Fatal(err)
	}
	if len(orte) == 0 || orte[0].ID != koeln || !strings.HasPrefix(orte[0].Name, "Köln") {
		t.Fatalf("erster Vorschlag für Köln: %+v", orte)
	}
	for _, o := range orte {
		if o.ID == "0" {
			t.Error("„Deutschland“ nicht ausgelassen")
		}
	}
	orte, err = a.Orte(c, "41372")
	if err != nil {
		t.Fatal(err)
	}
	if len(orte) == 0 || !strings.Contains(orte[0].Name, "Niederkrüchten") {
		t.Fatalf("Vorschläge für 41372: %+v", orte)
	}
}

var plzMuster = regexp.MustCompile(`^\d{5}$`)

func TestSucheLiefertPlausibleAnzeigen(t *testing.T) {
	a := anbindung()
	erg, err := a.Suche(ctx(t), Suche{Suchbegriff: "grafikkarte", OrtID: koeln, RadiusKm: 20, MaxSeiten: 1})
	if err != nil {
		t.Fatal(err)
	}
	if erg.Gesamt < SeitenGroesse || len(erg.Anzeigen) < SeitenGroesse {
		t.Fatalf("gesamt %d, gelesen %d", erg.Gesamt, len(erg.Anzeigen))
	}
	mitPreis, mitEntfernung := 0, 0
	for _, an := range erg.Anzeigen {
		if !idMuster.MatchString(an.ID) || an.Titel == "" || !strings.HasPrefix(an.URL, Basis+"/s-anzeige/") {
			t.Errorf("Treffer unvollständig: %+v", an)
		}
		if !plzMuster.MatchString(an.PLZ) || an.Ort == "" {
			t.Errorf("%s: PLZ %q / Ort %q", an.ID, an.PLZ, an.Ort)
		}
		if an.PreisTyp.HatBetrag() {
			mitPreis++
		}
		if an.EntfernungKm >= 0 {
			mitEntfernung++
			if an.EntfernungKm > 20 {
				t.Errorf("%s: %.0f km außerhalb des Umkreises", an.ID, an.EntfernungKm)
			}
		}
	}
	if mitPreis < len(erg.Anzeigen)/2 {
		t.Errorf("nur %d von %d Treffern mit Preis", mitPreis, len(erg.Anzeigen))
	}
	if mitEntfernung == 0 {
		t.Error("kein Treffer mit Entfernungsangabe")
	}
}

func TestSucheZweiSeiten(t *testing.T) {
	a := anbindung()
	erg, err := a.Suche(ctx(t), Suche{Suchbegriff: "grafikkarte", OrtID: koeln, RadiusKm: 30, MaxSeiten: 2})
	if err != nil {
		t.Fatal(err)
	}
	if erg.Gesamt <= SeitenGroesse {
		t.Fatalf("Testvoraussetzung: mehr als %d Treffer nötig, Seite meldet %d", SeitenGroesse, erg.Gesamt)
	}
	if len(erg.Anzeigen) <= SeitenGroesse {
		t.Fatalf("zweite Seite nicht gelesen: %d Anzeigen", len(erg.Anzeigen))
	}
}

func TestSuchePreisgrenzen(t *testing.T) {
	a := anbindung()
	erg, err := a.Suche(ctx(t), Suche{Suchbegriff: "grafikkarte", OrtID: koeln, RadiusKm: 30, PreisMin: 50, PreisMax: 200, MaxSeiten: 1})
	if err != nil {
		t.Fatal(err)
	}
	if len(erg.Anzeigen) == 0 {
		t.Fatal("keine Treffer im Preisbereich")
	}
	for _, an := range erg.Anzeigen {
		if an.PreisTyp.HatBetrag() && (an.PreisCent < 5000 || an.PreisCent > 20000) {
			t.Errorf("%s: %d Cent außerhalb 50–200 €", an.ID, an.PreisCent)
		}
	}
}

func TestSucheOhneTreffer(t *testing.T) {
	erg, err := anbindung().Suche(ctx(t), Suche{Suchbegriff: "xqzvwkjhgfd", OrtID: koeln, RadiusKm: 10})
	if err != nil {
		t.Fatal(err)
	}
	if len(erg.Anzeigen) != 0 || erg.Gesamt != 0 {
		t.Fatalf("Treffer für Unsinnswort: %d / %d", len(erg.Anzeigen), erg.Gesamt)
	}
}

func TestDetailUndStatus(t *testing.T) {
	a := anbindung()
	c := ctx(t)
	erg, err := a.Suche(c, Suche{Suchbegriff: "grafikkarte", OrtID: koeln, RadiusKm: 20, MaxSeiten: 1})
	if err != nil {
		t.Fatal(err)
	}
	if len(erg.Anzeigen) == 0 {
		t.Fatal("keine Treffer")
	}
	q := erg.Anzeigen[0]
	d, err := a.Detail(c, q.ID)
	if err != nil {
		t.Fatal(err)
	}
	if d.Status != anzeige.StatusAktiv {
		t.Fatalf("Status %q für aktive Anzeige", d.Status)
	}
	if d.Titel == "" || !d.BeschreibungVollstaendig || d.Beschreibung == "" {
		t.Errorf("Detail unvollständig: Titel %q, Beschreibung %d Zeichen", d.Titel, len(d.Beschreibung))
	}
	if d.PLZ != q.PLZ {
		t.Errorf("PLZ Detail %q, Suche %q", d.PLZ, q.PLZ)
	}
	if d.Lat < 47 || d.Lat > 55.5 || d.Lon < 5.5 || d.Lon > 15.5 {
		t.Errorf("Koordinaten %.3f/%.3f außerhalb Deutschlands", d.Lat, d.Lon)
	}

	g, err := a.Detail(c, "2000000000")
	if err != nil {
		t.Fatal(err)
	}
	if g.Status != anzeige.StatusGeloescht {
		t.Fatalf("gelöschte Anzeige: Status %q", g.Status)
	}
}

func TestLesePreis(t *testing.T) {
	for text, w := range map[string]struct {
		typ  anzeige.PreisTyp
		cent int64
	}{
		"650 € VB":       {anzeige.PreisVB, 65000},
		"249 €":          {anzeige.PreisFest, 24900},
		"1.200 €":        {anzeige.PreisFest, 120000},
		"12,5 €":         {anzeige.PreisFest, 1250},
		"VB":             {anzeige.PreisVBOhne, 0},
		"Zu verschenken": {anzeige.PreisVerschenken, 0},
	} {
		typ, cent, ok := LesePreis(text)
		if !ok || typ != w.typ || cent != w.cent {
			t.Errorf("LesePreis(%q) = %s %d %v, erwartet %s %d", text, typ, cent, ok, w.typ, w.cent)
		}
	}
	for _, text := range []string{"", "Versand möglich", "Direkt kaufen", "Heute, 12:24"} {
		if _, _, ok := LesePreis(text); ok {
			t.Errorf("LesePreis(%q) als Preis erkannt", text)
		}
	}
}

func TestSuchURL(t *testing.T) {
	for _, f := range []struct {
		s     Suche
		seite int
		url   string
	}{
		{Suche{Suchbegriff: "RTX 2080", OrtID: "945", RadiusKm: 10}, 1, Basis + "/s-anzeige:angebote/rtx-2080/k0l945r10"},
		{Suche{Suchbegriff: "rtx 2080", OrtID: "945", RadiusKm: 10, PreisMax: 550}, 3, Basis + "/s-anzeige:angebote/preis::550/seite:3/rtx-2080/k0l945r10"},
		{Suche{Suchbegriff: "rtx", OrtID: "1", RadiusKm: 5, PreisMin: 100}, 1, Basis + "/s-anzeige:angebote/preis:100:/rtx/k0l1r5"},
	} {
		if got := SuchURL(f.s, f.seite); got != f.url {
			t.Errorf("SuchURL = %s, erwartet %s", got, f.url)
		}
	}
}

// Bei weniger als einer Seite Treffern hängt die Seite „Weitere Ergebnisse in anderen
// Orten“ an. Diese dürfen nicht mitgezählt werden: gelesen == Zähler der Seite.
func TestSucheOhneFremdtreffer(t *testing.T) {
	a := anbindung()
	c := ctx(t)
	for _, begriff := range []string{"rtx 2080", "rtx 2080 ti", "gtx 1080 ti", "rx 6800"} {
		erg, err := a.Suche(c, Suche{Suchbegriff: begriff, OrtID: koeln, RadiusKm: 5})
		if err != nil {
			t.Fatal(err)
		}
		if erg.Gesamt < 1 || erg.Gesamt >= SeitenGroesse {
			continue // Testvoraussetzung: 1–24 Treffer
		}
		if len(erg.Anzeigen) != erg.Gesamt {
			t.Fatalf("%q: %d gelesen, Seite meldet %d", begriff, len(erg.Anzeigen), erg.Gesamt)
		}
		return
	}
	t.Fatal("Testvoraussetzung: kein Suchbegriff mit 1–24 Treffern gefunden")
}

func TestGesamtPasst(t *testing.T) {
	for _, f := range []struct {
		seite, erste int
		ok           bool
	}{
		{811, 811, true}, {810, 811, true}, {821, 811, true}, {822, 811, true}, {852, 811, false},
		{8, 0, true}, {12, 0, false},
		{1_200_000, 811, false}, {1_200_000, 40, false},
	} {
		if gesamtPasst(f.seite, f.erste) != f.ok {
			t.Errorf("gesamtPasst(%d, %d) = %v", f.seite, f.erste, !f.ok)
		}
	}
}
