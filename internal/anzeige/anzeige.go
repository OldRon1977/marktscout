// Package anzeige enthält die gemeinsamen Datentypen beider Seitenanbindungen.
package anzeige

// Quelle bezeichnet die Seite, von der eine Anzeige stammt.
type Quelle string

const (
	Kleinanzeigen Quelle = "kleinanzeigen"
	Marktplaats   Quelle = "marktplaats"
)

// PreisTyp beschreibt, wie der Preis einer Anzeige zu verstehen ist.
type PreisTyp string

const (
	PreisFest        PreisTyp = "fest"        // Festpreis
	PreisVB          PreisTyp = "vb"          // Verhandlungsbasis mit Betrag
	PreisVBOhne      PreisTyp = "vb_ohne"     // Verhandlungsbasis ohne Betrag
	PreisVerschenken PreisTyp = "verschenken" // zu verschenken
	PreisOhne        PreisTyp = "ohne"        // kein Preis (auf Anfrage, siehe Beschreibung)
)

// HatBetrag meldet, ob der Preistyp einen auswertbaren Betrag trägt.
func (p PreisTyp) HatBetrag() bool { return p == PreisFest || p == PreisVB }

// Status einer Anzeige auf der Seite.
type Status string

const (
	StatusAktiv      Status = "aktiv"
	StatusReserviert Status = "reserviert"
	StatusGeloescht  Status = "geloescht"
)

// Anzeige ist eine Anzeige, wie sie aus Suche oder Detailseite gelesen wurde.
type Anzeige struct {
	Quelle                   Quelle   `json:"quelle"`
	ID                       string   `json:"id"`
	URL                      string   `json:"url"`
	Titel                    string   `json:"titel"`
	Beschreibung             string   `json:"beschreibung"`
	BeschreibungVollstaendig bool     `json:"beschreibungVollstaendig"`
	PreisCent                int64    `json:"preisCent"`
	PreisTyp                 PreisTyp `json:"preisTyp"`
	Ort                      string   `json:"ort"`
	PLZ                      string   `json:"plz"`
	Land                     string   `json:"land"`
	Lat                      float64  `json:"lat"`
	Lon                      float64  `json:"lon"`
	EntfernungKm             float64  `json:"entfernungKm"` // -1 = unbekannt
	BildURL                  string   `json:"bildUrl"`
	Eingestellt              string   `json:"eingestellt"`
	Status                   Status   `json:"status"`
	Hervorgehoben            bool     `json:"hervorgehoben"`
}
