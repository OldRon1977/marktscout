// Package web stellt die Oberfläche und die JSON-Schnittstelle auf 127.0.0.1 bereit.
package web

import (
	"context"
	"embed"
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"log/slog"
	"net"
	"net/http"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/OldRon1977/marktscout/internal/anzeige"
	"github.com/OldRon1977/marktscout/internal/auftrag"
	"github.com/OldRon1977/marktscout/internal/kleinanzeigen"
	"github.com/OldRon1977/marktscout/internal/melden"
	"github.com/OldRon1977/marktscout/internal/protokoll"
	"github.com/OldRon1977/marktscout/internal/speicher"
	"github.com/OldRon1977/marktscout/internal/suchlauf"
	"github.com/OldRon1977/marktscout/internal/symbol"
	"github.com/OldRon1977/marktscout/internal/textpruefung"
)

//go:embed ui
var uiDateien embed.FS

// Kopfzeile, die jede ändernde Anfrage tragen muss. Fremde Webseiten können sie
// ohne CORS-Freigabe nicht setzen.
const SchutzKopf = "X-Marktscout"

// Schlüssel der Einstellungen.
const (
	EinstZeitsteuerung = "zeitsteuerung"
	EinstDesktop       = "desktop"
	EinstTon           = "ton"
)

// Melder gibt Benachrichtigungen aus. Desktop und Ton sind austauschbar, damit der
// Server ohne Desktopsitzung laufen kann.
type Melder struct {
	Desktop func(titel, text string) error
	Ton     func() error
}

// StandardMelder nutzt Desktopmeldung und Hinweiston des Betriebssystems.
func StandardMelder() Melder { return Melder{Desktop: melden.Desktop, Ton: melden.Ton} }

// Server ist die Anwendung hinter der Oberfläche.
type Server struct {
	DB      *speicher.DB
	Lauf    *suchlauf.Lauf
	Orte    *kleinanzeigen.Anbindung // eigene Anbindung mit kurzer Drosselung für Vorschläge
	Log     *slog.Logger
	Datei   *protokoll.Datei
	Version string
	Melder  Melder
	Beenden func()

	adresse string
	mu      sync.Mutex
	aktuell *JobStatus
	warte   []job
	bericht map[int64]Laufbericht
	signal  chan struct{}
}

type job struct {
	Art       string `json:"art"` // "suche" | "status"
	AuftragID int64  `json:"auftragId"`
	Geplant   bool   `json:"geplant"`
}

// JobStatus beschreibt den laufenden Auftrag.
type JobStatus struct {
	job
	Name string    `json:"name"`
	Text string    `json:"text"`
	Seit time.Time `json:"seit"`
}

// Laufbericht ist das Ergebnis des letzten Laufs eines Auftrags.
type Laufbericht struct {
	Art     string           `json:"art"`
	Ende    time.Time        `json:"ende"`
	Bericht suchlauf.Bericht `json:"bericht"`
	Fehler  string           `json:"fehler,omitempty"`
}

// Starte bindet den Server an adresse (z. B. 127.0.0.1:47811), startet Arbeiter und
// Zeitsteuerung und liefert den Listener. Stopp über ctx.
func (s *Server) Starte(ctx context.Context, adresse string) (net.Listener, *http.Server, error) {
	ln, err := net.Listen("tcp", adresse)
	if err != nil {
		return nil, nil, err
	}
	s.adresse = ln.Addr().String()
	s.bericht = map[int64]Laufbericht{}
	s.signal = make(chan struct{}, 1)
	if s.Log == nil {
		s.Log = slog.New(slog.DiscardHandler)
	}
	srv := &http.Server{Handler: s.Handler(), ReadHeaderTimeout: 10 * time.Second}
	go srv.Serve(ln)
	go s.arbeite(ctx)
	go s.zeitsteuerung(ctx)
	go func() {
		<-ctx.Done()
		c, abbruch := context.WithTimeout(context.Background(), 3*time.Second)
		defer abbruch()
		srv.Shutdown(c)
	}()
	return ln, srv, nil
}

// Adresse liefert die gebundene Adresse.
func (s *Server) Adresse() string { return s.adresse }

// Handler liefert alle Routen.
func (s *Server) Handler() http.Handler {
	m := http.NewServeMux()
	ui, _ := fs.Sub(uiDateien, "ui")
	m.Handle("GET /", http.FileServerFS(ui))

	m.HandleFunc("GET /favicon.png", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "image/png")
		w.Write(symbol.PNG(64))
	})
	m.HandleFunc("GET /api/version", func(w http.ResponseWriter, r *http.Request) {
		schreibe(w, map[string]string{"name": "marktscout", "version": s.Version})
	})
	m.HandleFunc("GET /api/status", s.status)
	m.HandleFunc("GET /api/auftraege", s.auftraege)
	m.HandleFunc("POST /api/auftraege", s.speichere)
	m.HandleFunc("GET /api/auftraege/{id}", s.einAuftrag)
	m.HandleFunc("PUT /api/auftraege/{id}", s.speichere)
	m.HandleFunc("DELETE /api/auftraege/{id}", s.loesche)
	m.HandleFunc("POST /api/auftraege/{id}/suchen", s.reihe("suche"))
	m.HandleFunc("POST /api/auftraege/{id}/pruefen", s.reihe("status"))
	m.HandleFunc("GET /api/auftraege/{id}/ergebnisse", s.ergebnisse)
	m.HandleFunc("DELETE /api/auftraege/{id}/treffer/{quelle}/{aid}", s.verwerfe)
	m.HandleFunc("POST /api/auftraege/{id}/treffer/{quelle}/{aid}/pruefen", s.pruefeEine)
	m.HandleFunc("GET /api/orte", s.orte)
	m.HandleFunc("GET /api/standardliste", func(w http.ResponseWriter, r *http.Request) { schreibe(w, textpruefung.Standard) })
	m.HandleFunc("GET /api/einstellungen", s.einstellungen)
	m.HandleFunc("PUT /api/einstellungen", s.setzeEinstellungen)
	m.HandleFunc("POST /api/meldung-test", s.meldungTest)
	m.HandleFunc("GET /api/log", s.log)
	m.HandleFunc("POST /api/beenden", func(w http.ResponseWriter, r *http.Request) {
		schreibe(w, map[string]bool{"ok": true})
		if s.Beenden != nil {
			go func() { time.Sleep(200 * time.Millisecond); s.Beenden() }()
		}
	})
	return s.schutz(m)
}

// schutz lässt nur Anfragen an 127.0.0.1/localhost zu (gegen DNS-Rebinding) und verlangt
// für ändernde Anfragen den Schutzkopf (gegen Anfragen fremder Webseiten).
func (s *Server) schutz(n http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		host, _, err := net.SplitHostPort(r.Host)
		if err != nil {
			host = r.Host
		}
		if host != "127.0.0.1" && host != "localhost" {
			http.Error(w, "nur lokal", http.StatusForbidden)
			return
		}
		if r.Method != http.MethodGet && r.Method != http.MethodHead && r.Header.Get(SchutzKopf) != "1" {
			http.Error(w, "Schutzkopf fehlt", http.StatusForbidden)
			return
		}
		w.Header().Set("Cache-Control", "no-store")
		w.Header().Set("X-Content-Type-Options", "nosniff")
		w.Header().Set("Content-Security-Policy", "default-src 'self'; img-src 'self' https: data:; style-src 'self'; script-src 'self'")
		n.ServeHTTP(w, r)
	})
}

func schreibe(w http.ResponseWriter, v any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	json.NewEncoder(w).Encode(v)
}

func fehler(w http.ResponseWriter, code int, err error) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(code)
	json.NewEncoder(w).Encode(map[string]string{"fehler": err.Error()})
}

func (s *Server) id(w http.ResponseWriter, r *http.Request) (int64, bool) {
	id, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if err != nil || id <= 0 {
		fehler(w, http.StatusBadRequest, errors.New("ungültige ID"))
		return 0, false
	}
	return id, true
}

func dbFehler(w http.ResponseWriter, err error) {
	if errors.Is(err, speicher.ErrNichtGefunden) {
		fehler(w, http.StatusNotFound, err)
		return
	}
	fehler(w, http.StatusInternalServerError, err)
}

// --- Aufträge ---

// AuftragInfo ist ein Auftrag mit Zählern für die Seitenleiste.
type AuftragInfo struct {
	auftrag.Auftrag
	Sichtbar int          `json:"sichtbar"`
	Neu      int          `json:"neu"`
	Bericht  *Laufbericht `json:"bericht,omitempty"`
}

func (s *Server) info(a auftrag.Auftrag) (AuftragInfo, error) {
	tr, err := s.DB.Treffer(a.ID, speicher.TrefferFilter{})
	if err != nil {
		return AuftragInfo{}, err
	}
	l := suchlauf.Ergebnisse(a, tr, s.Lauf.Bewerter, false)
	info := AuftragInfo{Auftrag: a, Sichtbar: l.Sichtbar, Neu: l.Neu}
	s.mu.Lock()
	if b, ok := s.bericht[a.ID]; ok {
		info.Bericht = &b
	}
	s.mu.Unlock()
	return info, nil
}

func (s *Server) auftraege(w http.ResponseWriter, r *http.Request) {
	as, err := s.DB.Auftraege()
	if err != nil {
		dbFehler(w, err)
		return
	}
	aus := []AuftragInfo{}
	for _, a := range as {
		i, err := s.info(a)
		if err != nil {
			dbFehler(w, err)
			return
		}
		aus = append(aus, i)
	}
	schreibe(w, aus)
}

func (s *Server) einAuftrag(w http.ResponseWriter, r *http.Request) {
	id, ok := s.id(w, r)
	if !ok {
		return
	}
	a, err := s.DB.Auftrag(id)
	if err != nil {
		dbFehler(w, err)
		return
	}
	i, err := s.info(a)
	if err != nil {
		dbFehler(w, err)
		return
	}
	schreibe(w, i)
}

func (s *Server) speichere(w http.ResponseWriter, r *http.Request) {
	a := auftrag.Neu()
	if err := json.NewDecoder(http.MaxBytesReader(w, r.Body, 1<<20)).Decode(&a); err != nil {
		fehler(w, http.StatusBadRequest, fmt.Errorf("Eingabe nicht lesbar: %w", err))
		return
	}
	a.ID = 0
	if r.Method == http.MethodPut {
		id, ok := s.id(w, r)
		if !ok {
			return
		}
		if _, err := s.DB.Auftrag(id); err != nil {
			dbFehler(w, err)
			return
		}
		a.ID = id
	}
	if err := a.Pruefe(); err != nil {
		fehler(w, http.StatusUnprocessableEntity, err)
		return
	}
	if err := s.DB.SpeichereAuftrag(&a); err != nil {
		dbFehler(w, err)
		return
	}
	s.Log.Info("auftrag gespeichert", "id", a.ID, "name", a.Name)
	g, err := s.DB.Auftrag(a.ID)
	if err != nil {
		dbFehler(w, err)
		return
	}
	schreibe(w, g)
}

func (s *Server) loesche(w http.ResponseWriter, r *http.Request) {
	id, ok := s.id(w, r)
	if !ok {
		return
	}
	if err := s.DB.LoescheAuftrag(id); err != nil {
		dbFehler(w, err)
		return
	}
	s.mu.Lock()
	delete(s.bericht, id)
	s.mu.Unlock()
	s.Log.Info("auftrag gelöscht", "id", id)
	schreibe(w, map[string]bool{"ok": true})
}

func (s *Server) ergebnisse(w http.ResponseWriter, r *http.Request) {
	id, ok := s.id(w, r)
	if !ok {
		return
	}
	a, err := s.DB.Auftrag(id)
	if err != nil {
		dbFehler(w, err)
		return
	}
	tr, err := s.DB.Treffer(id, speicher.TrefferFilter{})
	if err != nil {
		dbFehler(w, err)
		return
	}
	schreibe(w, suchlauf.Ergebnisse(a, tr, s.Lauf.Bewerter, r.URL.Query().Get("ausgeblendete") == "1"))
}

func quelle(w http.ResponseWriter, r *http.Request) (anzeige.Quelle, string, bool) {
	q := anzeige.Quelle(r.PathValue("quelle"))
	if q != anzeige.Kleinanzeigen && q != anzeige.Marktplaats {
		fehler(w, http.StatusBadRequest, errors.New("unbekannte Quelle"))
		return "", "", false
	}
	return q, r.PathValue("aid"), true
}

func (s *Server) verwerfe(w http.ResponseWriter, r *http.Request) {
	id, ok := s.id(w, r)
	if !ok {
		return
	}
	q, aid, ok := quelle(w, r)
	if !ok {
		return
	}
	if err := s.DB.Verwerfe(id, q, aid); err != nil {
		dbFehler(w, err)
		return
	}
	schreibe(w, map[string]bool{"ok": true})
}

func (s *Server) pruefeEine(w http.ResponseWriter, r *http.Request) {
	id, ok := s.id(w, r)
	if !ok {
		return
	}
	q, aid, ok := quelle(w, r)
	if !ok {
		return
	}
	c, abbruch := context.WithTimeout(r.Context(), 90*time.Second)
	defer abbruch()
	st, err := s.Lauf.PruefeEine(c, id, q, aid)
	if err != nil {
		if errors.Is(err, speicher.ErrNichtGefunden) {
			dbFehler(w, err)
			return
		}
		fehler(w, http.StatusBadGateway, err)
		return
	}
	schreibe(w, map[string]anzeige.Status{"status": st})
}

func (s *Server) orte(w http.ResponseWriter, r *http.Request) {
	q := strings.TrimSpace(r.URL.Query().Get("q"))
	if len([]rune(q)) < 2 {
		schreibe(w, []kleinanzeigen.Ort{})
		return
	}
	c, abbruch := context.WithTimeout(r.Context(), 20*time.Second)
	defer abbruch()
	orte, err := s.Orte.Orte(c, q)
	if err != nil {
		fehler(w, http.StatusBadGateway, err)
		return
	}
	if orte == nil {
		orte = []kleinanzeigen.Ort{}
	}
	schreibe(w, orte)
}

// --- Einstellungen und Log ---

// Einstellungen der Oberfläche.
type Einstellungen struct {
	Zeitsteuerung bool `json:"zeitsteuerung"`
	Desktop       bool `json:"desktop"`
	Ton           bool `json:"ton"`
}

func (s *Server) leseEinstellungen() Einstellungen {
	b := func(k string) bool { v, _ := s.DB.Einstellung(k, "1"); return v == "1" }
	return Einstellungen{Zeitsteuerung: b(EinstZeitsteuerung), Desktop: b(EinstDesktop), Ton: b(EinstTon)}
}

func (s *Server) einstellungen(w http.ResponseWriter, r *http.Request) {
	schreibe(w, s.leseEinstellungen())
}

func (s *Server) setzeEinstellungen(w http.ResponseWriter, r *http.Request) {
	var e Einstellungen
	if err := json.NewDecoder(http.MaxBytesReader(w, r.Body, 1<<16)).Decode(&e); err != nil {
		fehler(w, http.StatusBadRequest, err)
		return
	}
	wert := func(b bool) string {
		if b {
			return "1"
		}
		return "0"
	}
	for k, v := range map[string]bool{EinstZeitsteuerung: e.Zeitsteuerung, EinstDesktop: e.Desktop, EinstTon: e.Ton} {
		if err := s.DB.SetzeEinstellung(k, wert(v)); err != nil {
			dbFehler(w, err)
			return
		}
	}
	s.Log.Info("einstellungen", "zeitsteuerung", e.Zeitsteuerung, "desktop", e.Desktop, "ton", e.Ton)
	schreibe(w, s.leseEinstellungen())
}

func (s *Server) meldungTest(w http.ResponseWriter, r *http.Request) {
	var fehlerTexte []string
	if s.Melder.Desktop != nil {
		if err := s.Melder.Desktop("Marktscout", "Testmeldung"); err != nil {
			fehlerTexte = append(fehlerTexte, "Desktopmeldung: "+err.Error())
		}
	}
	if s.Melder.Ton != nil {
		if err := s.Melder.Ton(); err != nil {
			fehlerTexte = append(fehlerTexte, "Ton: "+err.Error())
		}
	}
	schreibe(w, map[string]any{"ok": len(fehlerTexte) == 0, "fehler": fehlerTexte})
}

func (s *Server) log(w http.ResponseWriter, r *http.Request) {
	if s.Datei == nil {
		fehler(w, http.StatusNotFound, errors.New("keine Logdatei"))
		return
	}
	name := "marktscout-log-" + time.Now().Format("2006-01-02-1504") + ".txt"
	w.Header().Set("Content-Type", "text/plain; charset=utf-8")
	w.Header().Set("Content-Disposition", `attachment; filename="`+name+`"`)
	fmt.Fprintf(w, "Marktscout %s – Logdatei exportiert %s\n\n", s.Version, time.Now().Format(time.RFC3339))
	s.Datei.Exportiere(w)
}

// --- Warteschlange ---

func (s *Server) reihe(art string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		id, ok := s.id(w, r)
		if !ok {
			return
		}
		if _, err := s.DB.Auftrag(id); err != nil {
			dbFehler(w, err)
			return
		}
		s.Einreihen(art, id, false)
		s.status(w, r)
	}
}

// Einreihen fügt einen Job hinzu, wenn derselbe nicht schon läuft oder wartet.
func (s *Server) Einreihen(art string, id int64, geplant bool) bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.aktuell != nil && s.aktuell.Art == art && s.aktuell.AuftragID == id {
		return false
	}
	for _, j := range s.warte {
		if j.Art == art && j.AuftragID == id {
			return false
		}
	}
	s.warte = append(s.warte, job{Art: art, AuftragID: id, Geplant: geplant})
	if s.signal != nil {
		select {
		case s.signal <- struct{}{}:
		default:
		}
	}
	return true
}

// Status der Warteschlange.
type Status struct {
	Aktuell       *JobStatus            `json:"aktuell"`
	Warte         []job                 `json:"warte"`
	Berichte      map[int64]Laufbericht `json:"berichte"`
	Einstellungen Einstellungen         `json:"einstellungen"`
}

func (s *Server) status(w http.ResponseWriter, r *http.Request) {
	s.mu.Lock()
	st := Status{Warte: append([]job{}, s.warte...), Berichte: map[int64]Laufbericht{}}
	if s.aktuell != nil {
		k := *s.aktuell
		st.Aktuell = &k
	}
	for k, v := range s.bericht {
		st.Berichte[k] = v
	}
	s.mu.Unlock()
	st.Einstellungen = s.leseEinstellungen()
	schreibe(w, st)
}

func (s *Server) arbeite(ctx context.Context) {
	for {
		s.mu.Lock()
		var j *job
		if len(s.warte) > 0 {
			j = &s.warte[0]
			s.warte = s.warte[1:]
		}
		s.mu.Unlock()
		if j == nil {
			select {
			case <-ctx.Done():
				return
			case <-s.signal:
			case <-time.After(5 * time.Second):
			}
			continue
		}
		s.fuehreAus(ctx, *j)
		if ctx.Err() != nil {
			return
		}
	}
}

func (s *Server) fuehreAus(ctx context.Context, j job) {
	a, err := s.DB.Auftrag(j.AuftragID)
	if err != nil {
		return
	}
	s.mu.Lock()
	s.aktuell = &JobStatus{job: j, Name: a.Name, Seit: time.Now(), Text: "startet"}
	s.mu.Unlock()
	fortschritt := func(t string) {
		s.mu.Lock()
		if s.aktuell != nil {
			s.aktuell.Text = t
		}
		s.mu.Unlock()
	}
	lb := Laufbericht{Art: j.Art}
	switch j.Art {
	case "suche":
		b, err := s.Lauf.Suche(ctx, j.AuftragID, fortschritt)
		lb.Bericht = b
		if err != nil {
			lb.Fehler = err.Error()
			s.Log.Error("suchlauf", "auftrag", j.AuftragID, "fehler", err)
			// Auch ein gescheiterter Lauf zählt für das Intervall, sonst wird er sofort wiederholt.
			if ctx.Err() == nil {
				s.DB.BeendeLauf(j.AuftragID, time.Now(), err.Error())
			}
		}
		if err == nil && b.NeuSichtbar > 0 {
			s.melde(a.Name, b.NeuSichtbar)
		}
	case "status":
		gepr, gel, err := s.Lauf.PruefeStatus(ctx, j.AuftragID, fortschritt)
		lb.Bericht = suchlauf.Bericht{Gefunden: gepr, Details: gel}
		if err != nil {
			lb.Fehler = err.Error()
		}
	}
	lb.Ende = time.Now()
	s.mu.Lock()
	s.aktuell = nil
	s.bericht[j.AuftragID] = lb
	s.mu.Unlock()
}

func (s *Server) melde(name string, neu int) {
	e := s.leseEinstellungen()
	text := fmt.Sprintf("%d neue Anzeige(n) für „%s“", neu, name)
	if neu == 1 {
		text = fmt.Sprintf("1 neue Anzeige für „%s“", name)
	}
	if e.Desktop && s.Melder.Desktop != nil {
		if err := s.Melder.Desktop("Marktscout", text); err != nil {
			s.Log.Warn("desktopmeldung", "fehler", err)
		}
	}
	if e.Ton && s.Melder.Ton != nil {
		if err := s.Melder.Ton(); err != nil {
			s.Log.Warn("ton", "fehler", err)
		}
	}
}

// PruefeZeitplan reiht fällige Aufträge ein, wenn die Zeitsteuerung an ist.
func (s *Server) PruefeZeitplan(jetzt time.Time) int {
	if !s.leseEinstellungen().Zeitsteuerung {
		return 0
	}
	as, err := s.DB.Auftraege()
	if err != nil {
		s.Log.Error("zeitsteuerung", "fehler", err)
		return 0
	}
	n := 0
	for _, a := range as {
		if suchlauf.Faellig(a, jetzt) && s.Einreihen("suche", a.ID, true) {
			s.Log.Info("zeitsteuerung", "auftrag", a.ID, "name", a.Name)
			n++
		}
	}
	return n
}

func (s *Server) zeitsteuerung(ctx context.Context) {
	t := time.NewTicker(30 * time.Second)
	defer t.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case jetzt := <-t.C:
			s.PruefeZeitplan(jetzt)
		}
	}
}
