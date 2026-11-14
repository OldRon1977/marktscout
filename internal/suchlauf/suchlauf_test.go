package suchlauf

import (
	"context"
	"path/filepath"
	"testing"
	"time"

	"github.com/OldRon1977/marktscout/internal/abruf"
	"github.com/OldRon1977/marktscout/internal/anzeige"
	"github.com/OldRon1977/marktscout/internal/auftrag"
	"github.com/OldRon1977/marktscout/internal/kleinanzeigen"
	"github.com/OldRon1977/marktscout/internal/marktplaats"
	"github.com/OldRon1977/marktscout/internal/speicher"
	"github.com/OldRon1977/marktscout/internal/textpruefung"
)

func lauf(t *testing.T) *Lauf {
	db, err := speicher.Oeffne(filepath.Join(t.TempDir(), "t.db"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { db.Schliesse() })
	c := abruf.Standard(nil)
	return &Lauf{DB: db, KA: kleinanzeigen.Neu(c), MP: marktplaats.Neu(c), Bewerter: textpruefung.Regelwerk{}}
}

// Live-Test: echter Suchlauf über beide Seiten mit zwei überlappenden Orten.
func TestSuchlaufLive(t *testing.T) {
	l := lauf(t)
	ctx, abbruch := context.WithTimeout(context.Background(), 8*time.Minute)
	defer abbruch()

	a := auftrag.Neu()
	a.Name, a.Suchbegriff = "RTX", "rtx"
	a.OrteDE = []auftrag.OrtDE{{ID: "945", Name: "Köln", RadiusKm: 15}, {ID: "977", Name: "Innenstadt - Köln", RadiusKm: 15}}
	a.OrteNL = []auftrag.OrtNL{{Postleitzahl: "6041", RadiusKm: 40}}
	// Enges Preisfenster, damit nur wenige Detailseiten geladen werden.
	a.Preis.SuchpreisCent, a.Preis.MindestpreisCent = 15000, 12000
	a.MaxSeiten = 1
	if err := a.Pruefe(); err != nil {
		t.Fatal(err)
	}
	if err := l.DB.SpeichereAuftrag(&a); err != nil {
		t.Fatal(err)
	}

	b, err := l.Suche(ctx, a.ID, nil)
	if err != nil {
		t.Fatal(err)
	}
	if len(b.Fehler) > 0 {
		t.Fatalf("Fehler im Lauf: %v", b.Fehler)
	}
	if b.Gefunden == 0 || b.Neu != b.Gefunden {
		t.Fatalf("Lauf 1: gefunden %d, neu %d", b.Gefunden, b.Neu)
	}
	tr, _ := l.DB.Treffer(a.ID, speicher.TrefferFilter{})
	if len(tr) != b.Gefunden {
		t.Errorf("%d Treffer gespeichert, %d gefunden", len(tr), b.Gefunden)
	}
	quellen := map[anzeige.Quelle]int{}
	for _, x := range tr {
		quellen[x.Quelle]++
	}
	if quellen[anzeige.Kleinanzeigen] == 0 || quellen[anzeige.Marktplaats] == 0 {
		t.Errorf("Treffer je Quelle: %v", quellen)
	}
	// Für jede sichtbare Anzeige mit Preis im Fenster wurde die Detailseite geladen.
	for _, e := range Bewerte(a, tr, l.Bewerter) {
		if e.Sichtbar && e.Status != anzeige.StatusGeloescht && !e.BeschreibungVollstaendig {
			t.Errorf("%s %s: sichtbar, aber ohne vollständige Beschreibung", e.Quelle, e.ID)
		}
	}
	if b.Details == 0 {
		t.Error("keine Detailseite geladen")
	}

	// Lauf 2: dieselben Anzeigen sind nicht mehr neu.
	b2, err := l.Suche(ctx, a.ID, nil)
	if err != nil {
		t.Fatal(err)
	}
	if b2.Neu > 2 || b2.Neu >= b.Neu {
		t.Errorf("Lauf 2: %d neue von %d (Lauf 1: %d)", b2.Neu, b2.Gefunden, b.Neu)
	}
	if b2.Details > 2 {
		t.Errorf("Lauf 2 lädt %d Detailseiten erneut", b2.Details)
	}
	g, _ := l.DB.Auftrag(a.ID)
	if g.LetzterLauf == nil {
		t.Error("letzter Lauf nicht festgehalten")
	}

	// Statusprüfung einer Anzeige
	st, err := l.PruefeEine(ctx, a.ID, tr[0].Quelle, tr[0].ID)
	if err != nil {
		t.Fatal(err)
	}
	if st != anzeige.StatusAktiv && st != anzeige.StatusReserviert && st != anzeige.StatusGeloescht {
		t.Errorf("Status %q", st)
	}
}

func treffer(id, ort, land string, typ anzeige.PreisTyp, cent int64, titel string) speicher.Treffer {
	return speicher.Treffer{Anzeige: anzeige.Anzeige{Quelle: anzeige.Kleinanzeigen, ID: id, Ort: ort, Land: land,
		PreisTyp: typ, PreisCent: cent, Titel: titel, Status: anzeige.StatusAktiv}}
}

func TestErgebnisseSortierungUndGruppen(t *testing.T) {
	a := auftrag.Neu()
	a.Suchbegriff = "rtx 2080"
	a.Preis.SuchpreisCent = 50000
	a.Preis.ToleranzWert = 10
	tr := []speicher.Treffer{
		treffer("1", "Köln", "DE", anzeige.PreisFest, 45000, "RTX 2080"),
		treffer("2", "Köln", "DE", anzeige.PreisVBOhne, 0, "RTX 2080"),
		treffer("3", "Köln", "DE", anzeige.PreisFest, 30000, "RTX 2080"),
		treffer("4", "Venlo", "NL", anzeige.PreisFest, 20000, "RTX 2080"),
		treffer("5", "Bonn", "DE", anzeige.PreisFest, 52000, "RTX 2080"),
		treffer("6", "Bonn", "DE", anzeige.PreisFest, 70000, "RTX 2080"),        // über Obergrenze
		treffer("7", "Bonn", "DE", anzeige.PreisFest, 40000, "RTX 2080 defekt"), // Ausschluss im Titel
	}
	l := Ergebnisse(a, tr, textpruefung.Regelwerk{}, false)
	var reihenfolge []string
	for _, g := range l.Gruppen {
		reihenfolge = append(reihenfolge, g.Land+":"+g.Ort)
		for _, e := range g.Eintraege {
			reihenfolge = append(reihenfolge, e.ID)
		}
	}
	erwartet := []string{"DE:Bonn", "5", "DE:Köln", "3", "1", "2", "NL:Venlo", "4"}
	if len(reihenfolge) != len(erwartet) {
		t.Fatalf("Reihenfolge %v, erwartet %v", reihenfolge, erwartet)
	}
	for i := range erwartet {
		if reihenfolge[i] != erwartet[i] {
			t.Fatalf("Reihenfolge %v, erwartet %v", reihenfolge, erwartet)
		}
	}
	if l.Sichtbar != 5 || l.Ausgeblendet != 2 {
		t.Errorf("sichtbar %d, ausgeblendet %d", l.Sichtbar, l.Ausgeblendet)
	}
	mit := Ergebnisse(a, tr, textpruefung.Regelwerk{}, true)
	n := 0
	for _, g := range mit.Gruppen {
		for _, e := range g.Eintraege {
			n++
			if !e.Sichtbar && len(e.Gruende) == 0 {
				t.Errorf("%s ausgeblendet ohne Grund", e.ID)
			}
		}
	}
	if n != 7 {
		t.Errorf("mit Ausgeblendeten: %d Einträge", n)
	}
}

func TestZerteile(t *testing.T) {
	a := auftrag.Neu()
	a.Suchbegriff = "2080"
	a.StandardModus = textpruefung.Markieren
	e := Bewerte(a, []speicher.Treffer{treffer("1", "Köln", "DE", anzeige.PreisFest, 1, "RTX 2080 – defekt")}, textpruefung.Regelwerk{})[0]
	var zusammen string
	arten := map[textpruefung.Art]string{}
	for _, t := range e.TitelTeile {
		zusammen += t.T
		if t.Art != "" {
			arten[t.Art] = t.T
		}
	}
	if zusammen != "RTX 2080 – defekt" || arten[textpruefung.ArtSuchbegriff] != "2080" || arten[textpruefung.ArtAusschluss] != "defekt" {
		t.Errorf("Teile %+v", e.TitelTeile)
	}
	if !e.Markiert || !e.Sichtbar {
		t.Errorf("markiert=%v sichtbar=%v", e.Markiert, e.Sichtbar)
	}
}

func TestFaellig(t *testing.T) {
	jetzt := time.Now()
	vor := jetzt.Add(-31 * time.Minute)
	kurz := jetzt.Add(-5 * time.Minute)
	for _, f := range []struct {
		intervall int
		lauf      *time.Time
		faellig   bool
	}{{0, nil, false}, {30, nil, true}, {30, &vor, true}, {30, &kurz, false}} {
		a := auftrag.Auftrag{IntervallMin: f.intervall, LetzterLauf: f.lauf}
		if Faellig(a, jetzt) != f.faellig {
			t.Errorf("Intervall %d, Lauf %v: %v", f.intervall, f.lauf, !f.faellig)
		}
	}
}
