package web

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"net/http"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/OldRon1977/marktscout/internal/abruf"
	"github.com/OldRon1977/marktscout/internal/auftrag"
	"github.com/OldRon1977/marktscout/internal/kleinanzeigen"
	"github.com/OldRon1977/marktscout/internal/marktplaats"
	"github.com/OldRon1977/marktscout/internal/protokoll"
	"github.com/OldRon1977/marktscout/internal/speicher"
	"github.com/OldRon1977/marktscout/internal/suchlauf"
	"github.com/OldRon1977/marktscout/internal/textpruefung"
)

// Tests gegen den echten Server auf 127.0.0.1 mit echter Datenbank.

type umgebung struct {
	s   *Server
	url string
}

func neu(t *testing.T, starten bool) *umgebung {
	verz := t.TempDir()
	db, err := speicher.Oeffne(filepath.Join(verz, "t.db"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { db.Schliesse() })
	d, err := protokoll.Oeffne(filepath.Join(verz, "logs"), 1<<20, 2)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { d.Schliesse() })
	c := abruf.Standard(nil)
	s := &Server{
		DB:      db,
		Lauf:    &suchlauf.Lauf{DB: db, KA: kleinanzeigen.Neu(c), MP: marktplaats.Neu(c), Bewerter: textpruefung.Regelwerk{}},
		Orte:    kleinanzeigen.Neu(abruf.Neu(300*time.Millisecond, 600*time.Millisecond, nil)),
		Log:     d.Logger(0),
		Datei:   d,
		Version: "test",
	}
	u := &umgebung{s: s}
	if starten {
		ctx, abbruch := context.WithCancel(context.Background())
		t.Cleanup(abbruch)
		if _, _, err := s.Starte(ctx, "127.0.0.1:0"); err != nil {
			t.Fatal(err)
		}
		u.url = "http://" + s.Adresse()
	}
	return u
}

func (u *umgebung) anfrage(t *testing.T, methode, pfad string, daten any, kopf bool) (int, []byte, http.Header) {
	t.Helper()
	var body io.Reader
	if daten != nil {
		b, _ := json.Marshal(daten)
		body = bytes.NewReader(b)
	}
	req, _ := http.NewRequest(methode, u.url+pfad, body)
	if kopf {
		req.Header.Set(SchutzKopf, "1")
	}
	r, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	defer r.Body.Close()
	b, _ := io.ReadAll(r.Body)
	return r.StatusCode, b, r.Header
}

func gueltigerAuftrag() auftrag.Auftrag {
	a := auftrag.Neu()
	a.Name, a.Suchbegriff = "RTX 2080", "rtx 2080"
	a.OrteDE = []auftrag.OrtDE{{ID: "945", Name: "Köln", RadiusKm: 10}}
	a.OrteNL = []auftrag.OrtNL{{Postleitzahl: "5911", RadiusKm: 10}}
	return a
}

func TestSchutz(t *testing.T) {
	u := neu(t, true)
	if code, _, _ := u.anfrage(t, "POST", "/api/auftraege", gueltigerAuftrag(), false); code != http.StatusForbidden {
		t.Errorf("POST ohne Schutzkopf: %d", code)
	}
	req, _ := http.NewRequest("GET", u.url+"/api/auftraege", nil)
	req.Host = "boese.example"
	r, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	r.Body.Close()
	if r.StatusCode != http.StatusForbidden {
		t.Errorf("fremder Host: %d", r.StatusCode)
	}
	if code, _, _ := u.anfrage(t, "GET", "/api/auftraege", nil, false); code != http.StatusOK {
		t.Errorf("GET ohne Schutzkopf: %d", code)
	}
}

func TestOberflaecheWirdAusgeliefert(t *testing.T) {
	u := neu(t, true)
	for pfad, inhalt := range map[string]string{"/": "<title>Marktscout</title>", "/app.js": "X-Marktscout", "/style.css": "--akzent"} {
		code, b, _ := u.anfrage(t, "GET", pfad, nil, false)
		if code != 200 || !strings.Contains(string(b), inhalt) {
			t.Errorf("%s: %d, enthält %q nicht", pfad, code, inhalt)
		}
	}
	if code, b, h := u.anfrage(t, "GET", "/favicon.png", nil, false); code != 200 || h.Get("Content-Type") != "image/png" || !bytes.HasPrefix(b, []byte("\x89PNG")) {
		t.Errorf("favicon: %d %s", code, h.Get("Content-Type"))
	}
}

func TestAuftragCRUD(t *testing.T) {
	u := neu(t, true)
	a := gueltigerAuftrag()
	code, b, _ := u.anfrage(t, "POST", "/api/auftraege", a, true)
	if code != 200 {
		t.Fatalf("anlegen: %d %s", code, b)
	}
	var g auftrag.Auftrag
	json.Unmarshal(b, &g)
	if g.ID == 0 || g.Name != "RTX 2080" {
		t.Fatalf("angelegt: %+v", g)
	}

	falsch := gueltigerAuftrag()
	falsch.OrteNL[0].Postleitzahl = "41372"
	code, b, _ = u.anfrage(t, "POST", "/api/auftraege", falsch, true)
	if code != http.StatusUnprocessableEntity || !strings.Contains(string(b), "41372") {
		t.Errorf("ungültige Postleitzahl: %d %s", code, b)
	}

	g.Name = "Neu benannt"
	code, b, _ = u.anfrage(t, "PUT", "/api/auftraege/"+itoa(g.ID), g, true)
	if code != 200 || !strings.Contains(string(b), "Neu benannt") {
		t.Errorf("ändern: %d %s", code, b)
	}
	code, b, _ = u.anfrage(t, "GET", "/api/auftraege", nil, false)
	var liste []AuftragInfo
	json.Unmarshal(b, &liste)
	if code != 200 || len(liste) != 1 || liste[0].Name != "Neu benannt" {
		t.Errorf("Liste: %d %s", code, b)
	}
	code, _, _ = u.anfrage(t, "GET", "/api/auftraege/"+itoa(g.ID)+"/ergebnisse", nil, false)
	if code != 200 {
		t.Errorf("Ergebnisse: %d", code)
	}
	if code, _, _ = u.anfrage(t, "DELETE", "/api/auftraege/"+itoa(g.ID), nil, true); code != 200 {
		t.Errorf("löschen: %d", code)
	}
	if code, _, _ = u.anfrage(t, "GET", "/api/auftraege/"+itoa(g.ID), nil, false); code != http.StatusNotFound {
		t.Errorf("nach Löschen: %d", code)
	}
}

func itoa(i int64) string { b, _ := json.Marshal(i); return string(b) }

func TestEinstellungen(t *testing.T) {
	u := neu(t, true)
	_, b, _ := u.anfrage(t, "GET", "/api/einstellungen", nil, false)
	var e Einstellungen
	json.Unmarshal(b, &e)
	if !e.Zeitsteuerung || !e.Desktop || !e.Ton {
		t.Errorf("Standard: %+v", e)
	}
	u.anfrage(t, "PUT", "/api/einstellungen", Einstellungen{Zeitsteuerung: true, Desktop: false, Ton: false}, true)
	_, b, _ = u.anfrage(t, "GET", "/api/einstellungen", nil, false)
	json.Unmarshal(b, &e)
	if !e.Zeitsteuerung || e.Desktop || e.Ton {
		t.Errorf("gespeichert: %+v", e)
	}
}

func TestLogExport(t *testing.T) {
	u := neu(t, true)
	u.s.Log.Info("eine Testzeile", "x", 42)
	code, b, h := u.anfrage(t, "GET", "/api/log", nil, false)
	if code != 200 || !strings.Contains(string(b), "eine Testzeile") || !strings.HasPrefix(h.Get("Content-Disposition"), "attachment;") {
		t.Errorf("Log: %d %q %s", code, h.Get("Content-Disposition"), b)
	}
}

func TestOrteLive(t *testing.T) {
	u := neu(t, true)
	code, b, _ := u.anfrage(t, "GET", "/api/orte?q=K%C3%B6ln", nil, false)
	var orte []kleinanzeigen.Ort
	json.Unmarshal(b, &orte)
	if code != 200 || len(orte) == 0 || orte[0].ID != "945" {
		t.Errorf("Orte: %d %s", code, b)
	}
}

func TestZeitplan(t *testing.T) {
	u := neu(t, false) // ohne Arbeiter: eingereihte Läufe bleiben in der Warteschlange
	a := gueltigerAuftrag()
	a.IntervallMin = 30
	u.s.DB.SpeichereAuftrag(&a)
	b := gueltigerAuftrag()
	b.IntervallMin = 0
	u.s.DB.SpeichereAuftrag(&b)

	jetzt := time.Now()
	if n := u.s.PruefeZeitplan(jetzt); n != 1 {
		t.Fatalf("eingereiht %d, erwartet 1", n)
	}
	if n := u.s.PruefeZeitplan(jetzt); n != 0 {
		t.Errorf("doppelt eingereiht: %d", n)
	}
	u.s.warte = nil
	u.s.DB.BeendeLauf(a.ID, jetzt.Add(-10*time.Minute), "")
	if n := u.s.PruefeZeitplan(jetzt); n != 0 {
		t.Errorf("vor Ablauf des Intervalls eingereiht: %d", n)
	}
	u.s.DB.BeendeLauf(a.ID, jetzt.Add(-31*time.Minute), "")
	u.s.DB.SetzeEinstellung(EinstZeitsteuerung, "0")
	if n := u.s.PruefeZeitplan(jetzt); n != 0 {
		t.Errorf("Zeitsteuerung aus, trotzdem eingereiht: %d", n)
	}
	u.s.DB.SetzeEinstellung(EinstZeitsteuerung, "1")
	if n := u.s.PruefeZeitplan(jetzt); n != 1 {
		t.Errorf("nach Ablauf: %d", n)
	}
}
