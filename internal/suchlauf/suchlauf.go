// Package suchlauf führt Suchaufträge aus und prüft den Status gespeicherter Anzeigen.
package suchlauf

import (
	"context"
	"fmt"
	"log/slog"
	"strings"
	"time"

	"github.com/OldRon1977/marktscout/internal/anzeige"
	"github.com/OldRon1977/marktscout/internal/auftrag"
	"github.com/OldRon1977/marktscout/internal/kleinanzeigen"
	"github.com/OldRon1977/marktscout/internal/marktplaats"
	"github.com/OldRon1977/marktscout/internal/speicher"
	"github.com/OldRon1977/marktscout/internal/textpruefung"
)

// Lauf bündelt alles, was ein Suchlauf braucht.
type Lauf struct {
	DB       *speicher.DB
	KA       *kleinanzeigen.Anbindung
	MP       *marktplaats.Anbindung
	Bewerter textpruefung.Bewerter
	Log      *slog.Logger
}

// Bericht über einen Suchlauf.
type Bericht struct {
	Gefunden    int      `json:"gefunden"`
	Neu         int      `json:"neu"`
	NeuSichtbar int      `json:"neuSichtbar"`
	Details     int      `json:"details"`
	Fehler      []string `json:"fehler,omitempty"`
}

func (l *Lauf) log() *slog.Logger {
	if l.Log == nil {
		return slog.New(slog.DiscardHandler)
	}
	return l.Log
}

func melde(f func(string), format string, args ...any) {
	if f != nil {
		f(fmt.Sprintf(format, args...))
	}
}

// Suche führt einen Suchauftrag aus: alle Orte beider Seiten, danach Detailseiten
// der neuen Anzeigen, die Preis- und Titelprüfung bestehen.
func (l *Lauf) Suche(ctx context.Context, id int64, fortschritt func(string)) (Bericht, error) {
	var b Bericht
	a, err := l.DB.Auftrag(id)
	if err != nil {
		return b, err
	}
	if err := l.DB.BeginneLauf(id); err != nil {
		return b, err
	}
	l.log().Info("suchlauf start", "auftrag", id, "name", a.Name)
	minC, maxC := a.Preis.SeitenGrenzenCent()
	jetzt := time.Now()
	gefunden := map[string]bool{}

	merke := func(an anzeige.Anzeige, suchort string) error {
		neu, err := l.DB.MerkeTreffer(id, an, suchort, jetzt)
		if err != nil {
			return err
		}
		gefunden[string(an.Quelle)+"/"+an.ID] = true
		if neu {
			b.Neu++
		}
		return nil
	}

	for _, o := range a.OrteDE {
		melde(fortschritt, "kleinanzeigen.de: %s (%d km)", o.Name, o.RadiusKm)
		erg, err := l.KA.Suche(ctx, kleinanzeigen.Suche{
			Suchbegriff: a.Suchbegriff, OrtID: o.ID, RadiusKm: o.RadiusKm,
			PreisMin: minC / 100, PreisMax: (maxC + 99) / 100, MaxSeiten: a.MaxSeiten,
		})
		if ctx.Err() != nil {
			return b, ctx.Err()
		}
		if err != nil {
			b.Fehler = append(b.Fehler, fmt.Sprintf("kleinanzeigen.de %s: %v", o.Name, err))
			l.log().Error("suche kleinanzeigen", "ort", o.Name, "fehler", err)
			continue
		}
		l.log().Info("suche kleinanzeigen", "ort", o.Name, "radius", o.RadiusKm, "gesamt", erg.Gesamt, "gelesen", len(erg.Anzeigen))
		for _, an := range erg.Anzeigen {
			if err := merke(an, o.Name); err != nil {
				return b, err
			}
		}
	}
	for _, o := range a.OrteNL {
		melde(fortschritt, "marktplaats.nl: %s (%d km)", o.Postleitzahl, o.RadiusKm)
		erg, err := l.MP.Suche(ctx, marktplaats.Suche{
			Suchbegriff: a.Suchbegriff, Postleitzahl: o.Postleitzahl, RadiusKm: o.RadiusKm,
			PreisMinCent: minC, PreisMaxCent: maxC, MaxSeiten: a.MaxSeiten,
		})
		if ctx.Err() != nil {
			return b, ctx.Err()
		}
		if err != nil {
			b.Fehler = append(b.Fehler, fmt.Sprintf("marktplaats.nl %s: %v", o.Postleitzahl, err))
			l.log().Error("suche marktplaats", "postleitzahl", o.Postleitzahl, "fehler", err)
			continue
		}
		l.log().Info("suche marktplaats", "postleitzahl", o.Postleitzahl, "radius", o.RadiusKm,
			"gesamt", erg.Gesamt, "gelesen", len(erg.Anzeigen), "gesuche", erg.Gesuche)
		for _, an := range erg.Anzeigen {
			if err := merke(an, o.Postleitzahl); err != nil {
				return b, err
			}
		}
	}
	b.Gefunden = len(gefunden)

	// Detailseiten: nur für Anzeigen, die Preis- und Titelprüfung bestehen.
	offen, err := l.DB.Treffer(id, speicher.TrefferFilter{OhneDetail: true})
	if err != nil {
		return b, err
	}
	var laden []speicher.Treffer
	for _, t := range offen {
		if a.Preis.Bewerte(t.PreisTyp, t.PreisCent).Sichtbar && l.Bewerter.Bewerte(t.Titel, "", a.Regeln()).Sichtbar {
			laden = append(laden, t)
		}
	}
	for i, t := range laden {
		melde(fortschritt, "Anzeigentexte: %d von %d", i+1, len(laden))
		if err := l.ladeDetail(ctx, t.Anzeige); err != nil {
			if ctx.Err() != nil {
				return b, ctx.Err()
			}
			b.Fehler = append(b.Fehler, fmt.Sprintf("Anzeige %s: %v", t.ID, err))
			continue
		}
		b.Details++
	}

	// Neue sichtbare Treffer zählen
	alle, err := l.DB.Treffer(id, speicher.TrefferFilter{})
	if err != nil {
		return b, err
	}
	for _, e := range Bewerte(a, alle, l.Bewerter) {
		if e.Neu && e.Sichtbar {
			b.NeuSichtbar++
		}
	}
	if err := l.DB.BeendeLauf(id, time.Now(), strings.Join(b.Fehler, "\n")); err != nil {
		return b, err
	}
	l.log().Info("suchlauf ende", "auftrag", id, "gefunden", b.Gefunden, "neu", b.Neu, "neuSichtbar", b.NeuSichtbar,
		"details", b.Details, "fehler", len(b.Fehler))
	return b, nil
}

// ladeDetail ruft die Detailseite ab und speichert Beschreibung und Status.
func (l *Lauf) ladeDetail(ctx context.Context, an anzeige.Anzeige) error {
	var d *anzeige.Anzeige
	var err error
	switch an.Quelle {
	case anzeige.Kleinanzeigen:
		d, err = l.KA.Detail(ctx, an.ID)
	case anzeige.Marktplaats:
		d, err = l.MP.Detail(ctx, an.URL)
	default:
		return fmt.Errorf("unbekannte Quelle %q", an.Quelle)
	}
	if err != nil {
		l.log().Warn("detail", "quelle", an.Quelle, "id", an.ID, "fehler", err)
		return err
	}
	return l.DB.SpeichereDetail(an.Quelle, an.ID, *d, time.Now())
}

// PruefeEine prüft den Status einer Anzeige eines Auftrags.
func (l *Lauf) PruefeEine(ctx context.Context, auftragID int64, q anzeige.Quelle, id string) (anzeige.Status, error) {
	tr, err := l.DB.Treffer(auftragID, speicher.TrefferFilter{})
	if err != nil {
		return "", err
	}
	for _, t := range tr {
		if t.Quelle == q && t.ID == id {
			if err := l.ladeDetail(ctx, t.Anzeige); err != nil {
				return "", err
			}
			tr2, err := l.DB.Treffer(auftragID, speicher.TrefferFilter{})
			if err != nil {
				return "", err
			}
			for _, x := range tr2 {
				if x.Quelle == q && x.ID == id {
					return x.Status, nil
				}
			}
		}
	}
	return "", speicher.ErrNichtGefunden
}

// PruefeStatus prüft alle nicht gelöschten Anzeigen eines Auftrags.
func (l *Lauf) PruefeStatus(ctx context.Context, auftragID int64, fortschritt func(string)) (geprueft, geloescht int, err error) {
	tr, err := l.DB.Treffer(auftragID, speicher.TrefferFilter{})
	if err != nil {
		return 0, 0, err
	}
	var offen []speicher.Treffer
	for _, t := range tr {
		if t.Status != anzeige.StatusGeloescht {
			offen = append(offen, t)
		}
	}
	for i, t := range offen {
		melde(fortschritt, "Status: %d von %d", i+1, len(offen))
		if err := l.ladeDetail(ctx, t.Anzeige); err != nil {
			if ctx.Err() != nil {
				return geprueft, geloescht, ctx.Err()
			}
			continue
		}
		geprueft++
	}
	tr, err = l.DB.Treffer(auftragID, speicher.TrefferFilter{})
	if err != nil {
		return geprueft, 0, err
	}
	for _, t := range tr {
		if t.Status == anzeige.StatusGeloescht {
			geloescht++
		}
	}
	l.log().Info("statusprüfung", "auftrag", auftragID, "geprueft", geprueft, "geloescht", geloescht)
	return geprueft, geloescht, nil
}

// Faellig meldet, ob ein Auftrag laut Intervall wieder laufen soll.
func Faellig(a auftrag.Auftrag, jetzt time.Time) bool {
	if a.IntervallMin <= 0 {
		return false
	}
	return a.LetzterLauf == nil || jetzt.Sub(*a.LetzterLauf) >= time.Duration(a.IntervallMin)*time.Minute
}
