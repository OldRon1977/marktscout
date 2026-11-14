package speicher

import (
	"errors"
	"path/filepath"
	"testing"
	"time"

	"github.com/OldRon1977/marktscout/internal/anzeige"
	"github.com/OldRon1977/marktscout/internal/auftrag"
)

// Tests mit echter SQLite-Datei im Temp-Verzeichnis.

func oeffne(t *testing.T) (*DB, string) {
	pfad := filepath.Join(t.TempDir(), "test.db")
	db, err := Oeffne(pfad)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { db.Schliesse() })
	return db, pfad
}

func beispielAuftrag() auftrag.Auftrag {
	a := auftrag.Neu()
	a.Name, a.Suchbegriff = "RTX 2080", "rtx 2080"
	a.OrteDE = []auftrag.OrtDE{{ID: "945", Name: "Köln", RadiusKm: 10}}
	a.OrteNL = []auftrag.OrtNL{{Postleitzahl: "5911", RadiusKm: 10}}
	a.Preis.SuchpreisCent = 50000
	return a
}

func beispielAnzeige(id string, km float64) anzeige.Anzeige {
	return anzeige.Anzeige{Quelle: anzeige.Kleinanzeigen, ID: id, URL: "https://www.kleinanzeigen.de/s-anzeige/" + id,
		Titel: "RTX 2080", Beschreibung: "kurz…", PreisCent: 25000, PreisTyp: anzeige.PreisVB, Ort: "Troisdorf", PLZ: "53844",
		Land: "DE", EntfernungKm: km, Status: anzeige.StatusAktiv}
}

func TestAuftragSpeichernLadenAendernLoeschen(t *testing.T) {
	db, pfad := oeffne(t)
	a := beispielAuftrag()
	if err := db.SpeichereAuftrag(&a); err != nil {
		t.Fatal(err)
	}
	if a.ID == 0 {
		t.Fatal("keine ID vergeben")
	}
	a.Pflicht = []string{"2080"}
	if err := db.SpeichereAuftrag(&a); err != nil {
		t.Fatal(err)
	}
	// Datei schließen und neu öffnen: Daten müssen auf der Platte liegen.
	db.Schliesse()
	db2, err := Oeffne(pfad)
	if err != nil {
		t.Fatal(err)
	}
	defer db2.Schliesse()
	g, err := db2.Auftrag(a.ID)
	if err != nil {
		t.Fatal(err)
	}
	if g.Suchbegriff != "rtx 2080" || len(g.Pflicht) != 1 || g.OrteNL[0].Postleitzahl != "5911" || g.Preis.SuchpreisCent != 50000 {
		t.Fatalf("geladen: %+v", g)
	}
	if err := db2.LoescheAuftrag(a.ID); err != nil {
		t.Fatal(err)
	}
	if _, err := db2.Auftrag(a.ID); !errors.Is(err, ErrNichtGefunden) {
		t.Fatalf("nach Löschen: %v", err)
	}
}

func TestTrefferNeuDoppeltVerworfen(t *testing.T) {
	db, _ := oeffne(t)
	a := beispielAuftrag()
	if err := db.SpeichereAuftrag(&a); err != nil {
		t.Fatal(err)
	}
	t0 := time.Unix(1_800_000_000, 0)

	// Lauf 1: zwei Anzeigen, eine davon von zwei Suchorten
	db.BeginneLauf(a.ID)
	for _, f := range []struct {
		an  anzeige.Anzeige
		ort string
		neu bool
	}{
		{beispielAnzeige("1", 20), "Köln", true},
		{beispielAnzeige("1", 5), "Bonn", false},
		{beispielAnzeige("2", 8), "Köln", true},
	} {
		neu, err := db.MerkeTreffer(a.ID, f.an, f.ort, t0)
		if err != nil {
			t.Fatal(err)
		}
		if neu != f.neu {
			t.Errorf("%s/%s: neu=%v, erwartet %v", f.an.ID, f.ort, neu, f.neu)
		}
	}
	tr, err := db.Treffer(a.ID, TrefferFilter{})
	if err != nil {
		t.Fatal(err)
	}
	if len(tr) != 2 {
		t.Fatalf("%d Treffer, erwartet 2 (Dublette zusammengeführt)", len(tr))
	}
	if tr[0].ID != "1" || tr[0].EntfernungKm != 5 || tr[0].Suchort != "Bonn" || !tr[0].Neu {
		t.Errorf("Treffer 1: %+v", tr[0])
	}

	// Detail laden: vollständige Beschreibung bleibt auch nach erneutem Fund erhalten
	if err := db.SpeichereDetail(anzeige.Kleinanzeigen, "1", anzeige.Anzeige{Titel: "RTX 2080", Beschreibung: "lang und vollständig",
		Status: anzeige.StatusAktiv, Lat: 50.8, Lon: 7.1}, t0); err != nil {
		t.Fatal(err)
	}
	ohne, _ := db.Treffer(a.ID, TrefferFilter{OhneDetail: true})
	if len(ohne) != 1 || ohne[0].ID != "2" {
		t.Errorf("ohne Detail: %+v", ohne)
	}

	// Lauf 2: Anzeige 1 erneut, Anzeige 2 verworfen, Anzeige 3 neu
	if err := db.Verwerfe(a.ID, anzeige.Kleinanzeigen, "2"); err != nil {
		t.Fatal(err)
	}
	db.BeginneLauf(a.ID)
	t1 := t0.Add(time.Hour)
	for id, erwartet := range map[string]bool{"1": false, "2": false, "3": true} {
		neu, err := db.MerkeTreffer(a.ID, beispielAnzeige(id, 9), "Köln", t1)
		if err != nil {
			t.Fatal(err)
		}
		if neu != erwartet {
			t.Errorf("Lauf 2, %s: neu=%v, erwartet %v", id, neu, erwartet)
		}
	}
	tr, _ = db.Treffer(a.ID, TrefferFilter{})
	ids := map[string]Treffer{}
	for _, x := range tr {
		ids[x.ID] = x
	}
	if _, da := ids["2"]; da || len(tr) != 2 {
		t.Fatalf("verworfene Anzeige wieder da: %+v", tr)
	}
	if ids["1"].Neu || !ids["3"].Neu {
		t.Errorf("Kennung neu: 1=%v 3=%v", ids["1"].Neu, ids["3"].Neu)
	}
	if ids["1"].Beschreibung != "lang und vollständig" || ids["1"].Lat != 50.8 {
		t.Errorf("Detaildaten überschrieben: %q %.1f", ids["1"].Beschreibung, ids["1"].Lat)
	}
	if ids["1"].EntfernungKm != 5 {
		t.Errorf("kürzeste Entfernung nicht behalten: %.0f", ids["1"].EntfernungKm)
	}

	// Status
	db.SetzeStatus(anzeige.Kleinanzeigen, "1", anzeige.StatusGeloescht, t1)
	tr, _ = db.Treffer(a.ID, TrefferFilter{})
	for _, x := range tr {
		if x.ID == "1" && x.Status != anzeige.StatusGeloescht {
			t.Errorf("Status %q", x.Status)
		}
	}
}

func TestEinstellungen(t *testing.T) {
	db, _ := oeffne(t)
	if w, _ := db.Einstellung("ton", "an"); w != "an" {
		t.Errorf("Standardwert %q", w)
	}
	db.SetzeEinstellung("ton", "aus")
	db.SetzeEinstellung("ton", "aus2")
	if w, _ := db.Einstellung("ton", "an"); w != "aus2" {
		t.Errorf("Wert %q", w)
	}
}

func TestAuftragPruefe(t *testing.T) {
	a := beispielAuftrag()
	if err := a.Pruefe(); err != nil {
		t.Fatalf("gültiger Auftrag abgelehnt: %v", err)
	}
	for name, aendere := range map[string]func(*auftrag.Auftrag){
		"ohne Suchbegriff":       func(a *auftrag.Auftrag) { a.Suchbegriff = " " },
		"ohne Orte":              func(a *auftrag.Auftrag) { a.OrteDE, a.OrteNL = nil, nil },
		"NL-Postleitzahl falsch": func(a *auftrag.Auftrag) { a.OrteNL[0].Postleitzahl = "41372" },
		"Ort ohne ID":            func(a *auftrag.Auftrag) { a.OrteDE[0].ID = "" },
		"Radius 0":               func(a *auftrag.Auftrag) { a.OrteDE[0].RadiusKm = 0 },
		"Intervall 5":            func(a *auftrag.Auftrag) { a.IntervallMin = 5 },
		"Mindest über Such":      func(a *auftrag.Auftrag) { a.Preis.MindestpreisCent = 60000 },
		"VB 100 %":               func(a *auftrag.Auftrag) { a.Preis.VBProzent = 100 },
	} {
		b := beispielAuftrag()
		aendere(&b)
		if err := b.Pruefe(); err == nil {
			t.Errorf("%s: angenommen", name)
		}
	}
}
