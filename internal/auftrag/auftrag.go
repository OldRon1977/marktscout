// Package auftrag beschreibt einen gespeicherten Suchauftrag.
package auftrag

import (
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/OldRon1977/marktscout/internal/marktplaats"
	"github.com/OldRon1977/marktscout/internal/preis"
	"github.com/OldRon1977/marktscout/internal/textpruefung"
)

// OrtDE ist ein Suchort bei kleinanzeigen.de.
type OrtDE struct {
	ID       string `json:"id"`
	Name     string `json:"name"`
	RadiusKm int    `json:"radiusKm"`
}

// OrtNL ist ein Suchort bei marktplaats.nl.
type OrtNL struct {
	Postleitzahl string `json:"postleitzahl"`
	RadiusKm     int    `json:"radiusKm"`
}

// Auftrag ist ein Suchauftrag.
type Auftrag struct {
	ID              int64              `json:"id"`
	Name            string             `json:"name"`
	Suchbegriff     string             `json:"suchbegriff"`
	OrteDE          []OrtDE            `json:"orteDE"`
	OrteNL          []OrtNL            `json:"orteNL"`
	Pflicht         []string           `json:"pflicht"`
	Ausschluss      []string           `json:"ausschluss"`
	AusschlussModus textpruefung.Modus `json:"ausschlussModus"`
	StandardAktiv   bool               `json:"standardAktiv"`
	StandardModus   textpruefung.Modus `json:"standardModus"`
	Preis           preis.Regel        `json:"preis"`
	IntervallMin    int                `json:"intervallMin"` // 0 = keine Zeitsteuerung
	MaxSeiten       int                `json:"maxSeiten"`    // je Ort und Seite
	LetzterLauf     *time.Time         `json:"letzterLauf"`
	LetzterFehler   string             `json:"letzterFehler"`
}

// Grenzen für Eingaben.
const (
	MaxRadiusKm       = 200
	MinIntervallMin   = 10
	StandardMaxSeiten = 4
)

// Neu liefert einen Auftrag mit Standardwerten.
func Neu() Auftrag {
	return Auftrag{
		AusschlussModus: textpruefung.Titel,
		StandardAktiv:   true,
		StandardModus:   textpruefung.Titel,
		Preis:           preis.Regel{ToleranzArt: preis.Prozent},
		MaxSeiten:       StandardMaxSeiten,
	}
}

// Regeln liefert die Regeln für die Textprüfung.
func (a Auftrag) Regeln() textpruefung.Regeln {
	return textpruefung.Regeln{
		Suchbegriff:     a.Suchbegriff,
		Pflicht:         a.Pflicht,
		Ausschluss:      a.Ausschluss,
		AusschlussModus: a.AusschlussModus,
		StandardAktiv:   a.StandardAktiv,
		StandardModus:   a.StandardModus,
	}
}

func modusGueltig(m textpruefung.Modus) bool {
	return m == textpruefung.Markieren || m == textpruefung.Titel || m == textpruefung.Hart
}

func liste(ws []string) []string {
	var aus []string
	gesehen := map[string]bool{}
	for _, w := range ws {
		w = strings.TrimSpace(w)
		if w != "" && !gesehen[strings.ToLower(w)] {
			gesehen[strings.ToLower(w)] = true
			aus = append(aus, w)
		}
	}
	return aus
}

// Pruefe bereinigt Eingaben und meldet ungültige Angaben.
func (a *Auftrag) Pruefe() error {
	var fehler []string
	a.Name = strings.TrimSpace(a.Name)
	a.Suchbegriff = strings.Join(strings.Fields(a.Suchbegriff), " ")
	a.Pflicht, a.Ausschluss = liste(a.Pflicht), liste(a.Ausschluss)
	if a.Name == "" {
		a.Name = a.Suchbegriff
	}
	if a.Suchbegriff == "" {
		fehler = append(fehler, "Suchbegriff fehlt")
	}
	if len(a.OrteDE) == 0 && len(a.OrteNL) == 0 {
		fehler = append(fehler, "mindestens ein Ort nötig")
	}
	for i, o := range a.OrteDE {
		if o.ID == "" {
			fehler = append(fehler, fmt.Sprintf("Ort %q: nicht aus den Vorschlägen gewählt", o.Name))
		}
		if o.RadiusKm < 1 || o.RadiusKm > MaxRadiusKm {
			fehler = append(fehler, fmt.Sprintf("Ort %q: Umkreis 1–%d km", o.Name, MaxRadiusKm))
		}
		a.OrteDE[i].Name = strings.TrimSpace(o.Name)
	}
	for i, o := range a.OrteNL {
		p, err := marktplaats.NormalisierePostleitzahl(o.Postleitzahl)
		if err != nil {
			fehler = append(fehler, fmt.Sprintf("Postleitzahl %q ungültig", o.Postleitzahl))
		}
		a.OrteNL[i].Postleitzahl = p
		if o.RadiusKm < 1 || o.RadiusKm > MaxRadiusKm {
			fehler = append(fehler, fmt.Sprintf("Postleitzahl %s: Umkreis 1–%d km", o.Postleitzahl, MaxRadiusKm))
		}
	}
	if !modusGueltig(a.AusschlussModus) || !modusGueltig(a.StandardModus) {
		fehler = append(fehler, "Modus für Ausschlusswörter ungültig")
	}
	p := a.Preis
	if p.SuchpreisCent < 0 || p.MindestpreisCent < 0 || p.ToleranzWert < 0 || p.VBProzent < 0 || p.VBProzent >= 100 {
		fehler = append(fehler, "Preisangaben ungültig")
	}
	if p.ToleranzArt != preis.Prozent && p.ToleranzArt != preis.Euro {
		fehler = append(fehler, "Toleranz: Prozent oder Euro")
	}
	if p.SuchpreisCent > 0 && p.MindestpreisCent > p.SuchpreisCent {
		fehler = append(fehler, "Mindestpreis über Suchpreis")
	}
	if a.IntervallMin != 0 && a.IntervallMin < MinIntervallMin {
		fehler = append(fehler, fmt.Sprintf("Intervall mindestens %d Minuten", MinIntervallMin))
	}
	if a.MaxSeiten < 1 || a.MaxSeiten > 20 {
		fehler = append(fehler, "Seiten je Ort: 1–20")
	}
	if len(fehler) > 0 {
		return errors.New(strings.Join(fehler, "; "))
	}
	return nil
}
