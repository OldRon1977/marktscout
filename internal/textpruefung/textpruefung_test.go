package textpruefung

import (
	"strings"
	"testing"
)

var rw Regelwerk

func TestEinstufungVergleich(t *testing.T) {
	r := Regeln{Suchbegriff: "RTX 2080"}
	for _, f := range []struct {
		titel, text string
		erwartet    Einstufung
	}{
		// Beispiel aus den Anforderungen
		{"Gaming PC", "Es ist eine RTX 2070 TI verbaut, fast wie eine RTX 2080.", Vergleich},
		{"Gaming PC mit RTX 2070", "Leistung wie eine Nvidia RTX 2080 zum halben Preis.", Vergleich},
		{"RTX 2070 Super", "Schneller als eine RTX 2080? Fast.", Vergleich},
		{"PC", "Das ist keine RTX 2080, sondern eine 2070.", Vergleich},
		{"Gaming PC", "Vergelijkbaar met een RTX 2080.", Vergleich},
		{"MSI RTX 2080 Ventus 8G", "Voll funktionsfähig, schneller als eine RTX 2070.", Echt},
		{"Grafikkarte", "Verkaufe meine MSI RTX2080 Ventus.", Echt},
		{"Grafikkarte rtx-2080", "", Echt},
		{"Zustand wie neu RTX 2080", "", Echt},
		{"Gaming PC", "Mit Grafikkarte, wie neu.", Zweifelhaft},
		{"PC mit RTX 2080", "Fast wie eine RTX 2080 Ti, aber eben eine RTX 2080.", Echt},
		{"Gaming PC", "RTX 2080 verbaut. Leistung wie eine RTX 2080 Super.", Zweifelhaft},
	} {
		e := rw.Bewerte(f.titel, f.text, r)
		if e.Einstufung != f.erwartet {
			t.Errorf("%q / %q: %s, erwartet %s", f.titel, f.text, e.Einstufung, f.erwartet)
		}
	}
}

func TestSuchbegriffGrenzen(t *testing.T) {
	r := Regeln{Suchbegriff: "2080"}
	for text, anzahl := range map[string]int{
		"RTX 2080":       1,
		"RTX 20800":      0,
		"RTX 2080Ti":     0,
		"RTX 2080 Ti":    1,
		"12080":          0,
		"2080, 2080!":    2,
		"Modell:2080/8G": 1,
		"RTX2080 OC":     1,
		"x2080":          1,
	} {
		n := 0
		for _, f := range rw.Bewerte(text, "", r).Fundstellen {
			if f.Art == ArtSuchbegriff {
				n++
			}
		}
		if n != anzahl {
			t.Errorf("%q: %d Fundstellen, erwartet %d", text, n, anzahl)
		}
	}
}

func TestAusschlussModi(t *testing.T) {
	titel := "RTX 2080 defekt"
	text := "Lüfter dreht nicht. Für Bastler."
	for _, f := range []struct {
		modus    Modus
		sichtbar bool
	}{{Markieren, true}, {Titel, false}, {Hart, false}} {
		e := rw.Bewerte(titel, text, Regeln{StandardAktiv: true, StandardModus: f.modus})
		if e.Sichtbar != f.sichtbar {
			t.Errorf("Modus %s: sichtbar=%v, erwartet %v (%v)", f.modus, e.Sichtbar, f.sichtbar, e.Gruende)
		}
	}
	// Nur im Text: Titel-Modus markiert, Hart blendet aus
	nurText := "Grafikkarte"
	if e := rw.Bewerte(nurText, text, Regeln{StandardAktiv: true, StandardModus: Titel}); !e.Sichtbar {
		t.Errorf("Titel-Modus blendet bei Treffer im Text aus: %v", e.Gruende)
	}
	if e := rw.Bewerte(nurText, text, Regeln{StandardAktiv: true, StandardModus: Hart}); e.Sichtbar {
		t.Error("Hart-Modus blendet bei Treffer im Text nicht aus")
	}
	// Standardliste abgeschaltet
	if e := rw.Bewerte(titel, text, Regeln{StandardAktiv: false, StandardModus: Hart}); !e.Sichtbar {
		t.Errorf("abgeschaltete Standardliste wirkt: %v", e.Gruende)
	}
	// Eigene Liste mit eigenem Modus
	e := rw.Bewerte("RTX 2080 mit Wasserkühlung", "", Regeln{Ausschluss: []string{"Wasserkühlung"}, AusschlussModus: Titel})
	if e.Sichtbar {
		t.Error("eigenes Ausschlusswort im Titel blendet nicht aus")
	}
}

func TestVerneinung(t *testing.T) {
	r := Regeln{StandardAktiv: true, StandardModus: Hart}
	for text, sichtbar := range map[string]bool{
		"Kein Tausch.":                       true,
		"Tausch nicht möglich.":              true,
		"Tausch ausgeschlossen":              true,
		"Nicht defekt, voll funktionsfähig.": true,
		"Ohne Defekte, keine Kratzer.":       true,
		"Geen ruilen.":                       true,
		"Tausch gegen RX 6800 möglich.":      false,
		"Tausch nicht ausgeschlossen.":       false,
		"Leider defekt.":                     false,
		"Defekte Lüfter, sonst ok.":          false,
		"Nicht getestet. Defekt.":            false,
	} {
		e := rw.Bewerte("Grafikkarte", text, r)
		if e.Sichtbar != sichtbar {
			t.Errorf("%q: sichtbar=%v, erwartet %v (%v)", text, e.Sichtbar, sichtbar, e.Gruende)
		}
	}
}

func TestPflichtwoerter(t *testing.T) {
	r := Regeln{Pflicht: []string{"2080", "8GB"}}
	if e := rw.Bewerte("RTX 2080 8 GB", "", r); !e.Sichtbar {
		t.Errorf("Pflichtwörter vorhanden, trotzdem ausgeblendet: %v", e.Gruende)
	}
	e := rw.Bewerte("RTX 2080", "", r)
	if e.Sichtbar || len(e.Gruende) != 1 || !strings.Contains(e.Gruende[0], "8GB") {
		t.Errorf("fehlendes Pflichtwort: sichtbar=%v %v", e.Sichtbar, e.Gruende)
	}
	if e := rw.Bewerte("Undefekte Ware", "", Regeln{StandardAktiv: true, StandardModus: Hart}); !e.Sichtbar {
		t.Error("„defekt“ mitten im Wort zählt")
	}
	if e := rw.Bewerte("PC", "keine 2080 8GB verbaut", r); e.Sichtbar {
		t.Error("verneintes Pflichtwort zählt")
	}
}

func TestFundstellenPositionen(t *testing.T) {
	titel := "Verkaufe Grafikkarte – defekt"
	e := rw.Bewerte(titel, "", Regeln{StandardAktiv: true, StandardModus: Markieren})
	if len(e.Fundstellen) != 1 {
		t.Fatalf("Fundstellen: %+v", e.Fundstellen)
	}
	f := e.Fundstellen[0]
	if titel[f.Start:f.Ende] != "defekt" || f.Feld != FeldTitel || f.Art != ArtAusschluss {
		t.Errorf("Fundstelle %+v → %q", f, titel[f.Start:f.Ende])
	}
}
