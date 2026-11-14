package marktplaats

import (
	"context"
	"errors"
	"regexp"
	"strings"
	"testing"
	"time"

	"github.com/OldRon1977/marktscout/internal/abruf"
	"github.com/OldRon1977/marktscout/internal/anzeige"
)

// Live-Tests gegen marktplaats.nl. Geprüft werden Struktur und Plausibilität.

func anbindung() *Anbindung { return Neu(abruf.Standard(nil)) }

func ctx(t *testing.T) context.Context {
	c, abbruch := context.WithTimeout(context.Background(), 3*time.Minute)
	t.Cleanup(abbruch)
	return c
}

// m… private, a… gewerbliche Anzeigen
var idMuster = regexp.MustCompile(`^[am]\d+$`)

func TestSucheLiefertPlausibleAnzeigen(t *testing.T) {
	a := anbindung()
	erg, err := a.Suche(ctx(t), Suche{Suchbegriff: "grafische kaart", Postleitzahl: "6041", RadiusKm: 50, MaxSeiten: 1})
	if err != nil {
		t.Fatal(err)
	}
	if erg.Gesamt < 10 || len(erg.Anzeigen) < 10 {
		t.Fatalf("zu wenige Treffer: gesamt %d, gelesen %d", erg.Gesamt, len(erg.Anzeigen))
	}
	mitBetrag := 0
	for _, an := range erg.Anzeigen {
		if !idMuster.MatchString(an.ID) {
			t.Errorf("ID %q hat nicht das Format m…/a…", an.ID)
		}
		if an.Titel == "" || an.Ort == "" {
			t.Errorf("%s: Titel oder Ort leer", an.ID)
		}
		if !strings.HasPrefix(an.URL, Basis+"/v/") {
			t.Errorf("%s: URL %q", an.ID, an.URL)
		}
		if an.EntfernungKm < 0 || an.EntfernungKm > 50 {
			t.Errorf("%s: Entfernung %.1f km außerhalb 0–50", an.ID, an.EntfernungKm)
		}
		if IstGesuch(an.Titel) {
			t.Errorf("%s: Gesuch nicht verworfen: %q", an.ID, an.Titel)
		}
		if an.PreisTyp.HatBetrag() {
			mitBetrag++
			if an.PreisCent <= 0 {
				t.Errorf("%s: Preistyp %s ohne Betrag", an.ID, an.PreisTyp)
			}
		}
	}
	if mitBetrag == 0 {
		t.Error("keine Anzeige mit Betrag")
	}
}

func TestSuchePreisgrenzen(t *testing.T) {
	a := anbindung()
	erg, err := a.Suche(ctx(t), Suche{Suchbegriff: "grafische kaart", Postleitzahl: "6041", RadiusKm: 50,
		PreisMinCent: 5000, PreisMaxCent: 20000, MaxSeiten: 1})
	if err != nil {
		t.Fatal(err)
	}
	if len(erg.Anzeigen) == 0 {
		t.Fatal("keine Treffer im Preisbereich")
	}
	for _, an := range erg.Anzeigen {
		if an.PreisTyp.HatBetrag() && (an.PreisCent < 5000 || an.PreisCent > 20000) {
			t.Errorf("%s: %d Cent außerhalb 5000–20000", an.ID, an.PreisCent)
		}
	}
}

func TestSucheMehrereSeitenOhneDoppelte(t *testing.T) {
	a := anbindung()
	erg, err := a.Suche(ctx(t), Suche{Suchbegriff: "fiets", Postleitzahl: "5911", RadiusKm: 25, MaxSeiten: 2})
	if err != nil {
		t.Fatal(err)
	}
	if erg.Gesamt <= seitenGroesse {
		t.Fatalf("Testvoraussetzung: mehr als %d Treffer nötig, Seite meldet %d", seitenGroesse, erg.Gesamt)
	}
	if len(erg.Anzeigen)+erg.Gesuche <= seitenGroesse {
		t.Fatalf("zweite Seite nicht gelesen: %d Anzeigen", len(erg.Anzeigen))
	}
	ids := map[string]bool{}
	for _, an := range erg.Anzeigen {
		if ids[an.ID] {
			t.Errorf("doppelte ID %s", an.ID)
		}
		ids[an.ID] = true
	}
}

func TestSucheUngueltigePostleitzahl(t *testing.T) {
	a := anbindung()
	_, err := a.Suche(ctx(t), Suche{Suchbegriff: "fiets", Postleitzahl: "0123", RadiusKm: 10})
	if err == nil {
		t.Fatal("0123 als Postleitzahl angenommen")
	}
	// Formal gültig, aber keine vergebene Postleitzahl: Seite ignoriert sie.
	_, err = a.Suche(ctx(t), Suche{Suchbegriff: "fiets", Postleitzahl: "8888", RadiusKm: 10})
	if !errors.Is(err, ErrPostleitzahl) {
		t.Fatalf("8888: Fehler %v, erwartet ErrPostleitzahl", err)
	}
}

func TestDetailUndStatus(t *testing.T) {
	a := anbindung()
	c := ctx(t)
	erg, err := a.Suche(c, Suche{Suchbegriff: "grafische kaart", Postleitzahl: "6041", RadiusKm: 50, MaxSeiten: 1})
	if err != nil {
		t.Fatal(err)
	}
	if len(erg.Anzeigen) == 0 {
		t.Fatal("keine Treffer")
	}
	// Anzeige mit gekürzter Beschreibung suchen (200 Zeichen aus der Suche)
	quelle := erg.Anzeigen[0]
	for _, an := range erg.Anzeigen {
		if len([]rune(an.Beschreibung)) >= 190 {
			quelle = an
			break
		}
	}
	d, err := a.Detail(c, quelle.URL)
	if err != nil {
		t.Fatal(err)
	}
	if d.Status != anzeige.StatusAktiv && d.Status != anzeige.StatusReserviert {
		t.Fatalf("Status %q für aktive Anzeige", d.Status)
	}
	if d.ID != quelle.ID || d.Titel == "" {
		t.Errorf("ID %q / Titel %q, erwartet ID %q", d.ID, d.Titel, quelle.ID)
	}
	if !d.BeschreibungVollstaendig || len(d.Beschreibung) < len(quelle.Beschreibung)-5 {
		t.Errorf("Beschreibung %d Zeichen, aus der Suche %d", len(d.Beschreibung), len(quelle.Beschreibung))
	}

	// Gelöschte Anzeige: alte ID mit gültigem Kategoriepfad
	g, err := a.Detail(c, Basis+"/v/computers-en-software/videokaarten/m1900000000-test")
	if err != nil {
		t.Fatal(err)
	}
	if g.Status != anzeige.StatusGeloescht {
		t.Fatalf("gelöschte Anzeige: Status %q", g.Status)
	}
}

func TestIstGesuch(t *testing.T) {
	for titel, erwartet := range map[string]bool{
		"Gezocht: RX 9070 (XT)":                true,
		"GEZOCHT : NVIDIA GeForce RTX":         true,
		"RTX 3090 Gezocht!":                    true,
		"Zoek RTX 2080":                        true,
		"gevraagd cmp 170hx":                   true,
		"MSI RTX 2080 Ventus":                  false,
		"Zoeklicht voor fiets":                 false,
		"Opgezocht en gevonden: GTX 1080 doos": false,
	} {
		if IstGesuch(titel) != erwartet {
			t.Errorf("IstGesuch(%q) = %v", titel, !erwartet)
		}
	}
}

func TestSucheVerwirftGesuche(t *testing.T) {
	erg, err := anbindung().Suche(ctx(t), Suche{Suchbegriff: "gezocht", Postleitzahl: "3511", RadiusKm: 50, MaxSeiten: 1})
	if err != nil {
		t.Fatal(err)
	}
	if erg.Gesuche == 0 {
		t.Fatal("Testvoraussetzung: Suche nach „gezocht“ liefert keine Gesuche")
	}
	for _, an := range erg.Anzeigen {
		if IstGesuch(an.Titel) {
			t.Errorf("Gesuch nicht verworfen: %q", an.Titel)
		}
	}
}
