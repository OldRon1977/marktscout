// Package textpruefung prüft Titel und Beschreibung einer Anzeige auf Pflicht- und
// Ausschlusswörter und stuft ein, ob der Suchbegriff tatsächlich angeboten oder nur
// zum Vergleich genannt wird.
package textpruefung

import (
	"regexp"
	"sort"
	"strings"
	"sync"
	"unicode"
	"unicode/utf8"
)

// Modus legt fest, wie Ausschlusswörter wirken.
type Modus string

const (
	Markieren Modus = "markieren" // nie ausblenden, nur hervorheben
	Titel     Modus = "titel"     // Treffer im Titel blendet aus, im Text wird markiert
	Hart      Modus = "hart"      // Treffer in Titel oder Text blendet aus
)

// Standard ist die Standard-Ausschlussliste.
var Standard = []string{
	// Deutsch
	"defekt", "Bastler", "für Bastler", "Tausch", "tausche", "nur OVP", "leere Verpackung", "Ersatzteile", "kaputt",
	// Niederländisch
	"defect", "kapot", "ruilen", "onderdelen", "voor onderdelen", "alleen doos",
}

// Regeln eines Suchauftrags für die Textprüfung.
type Regeln struct {
	Suchbegriff     string   `json:"suchbegriff"`
	Pflicht         []string `json:"pflicht"`
	Ausschluss      []string `json:"ausschluss"`
	AusschlussModus Modus    `json:"ausschlussModus"`
	StandardAktiv   bool     `json:"standardAktiv"`
	StandardModus   Modus    `json:"standardModus"`
}

// Feld, in dem eine Fundstelle liegt.
type Feld string

const (
	FeldTitel Feld = "titel"
	FeldText  Feld = "text"
)

// Art einer Fundstelle.
type Art string

const (
	ArtAusschluss  Art = "ausschluss"  // zählt als Ausschluss
	ArtVerneint    Art = "verneint"    // Ausschlusswort, aber verneint („kein Tausch“)
	ArtSuchbegriff Art = "suchbegriff" // Suchbegriff als Angebot
	ArtVergleich   Art = "vergleich"   // Suchbegriff als Vergleich („fast wie eine RTX 2080“)
	ArtPflicht     Art = "pflicht"     // Pflichtwort
)

// Fundstelle im Titel oder Text, Start/Ende als Byte-Position.
type Fundstelle struct {
	Wort  string `json:"wort"`
	Feld  Feld   `json:"feld"`
	Start int    `json:"start"`
	Ende  int    `json:"ende"`
	Art   Art    `json:"art"`
}

// Einstufung des Suchbegriffs.
type Einstufung string

const (
	Echt        Einstufung = "echt"
	Zweifelhaft Einstufung = "zweifelhaft"
	Vergleich   Einstufung = "vergleich"
)

// Ergebnis der Textprüfung.
type Ergebnis struct {
	Sichtbar    bool         `json:"sichtbar"`
	Gruende     []string     `json:"gruende,omitempty"`
	Einstufung  Einstufung   `json:"einstufung,omitempty"`
	Fundstellen []Fundstelle `json:"fundstellen,omitempty"`
}

// Bewerter prüft eine Anzeige. Regelwerk ist die eingebaute Umsetzung; weitere
// Umsetzungen erfüllen dieselbe Schnittstelle.
type Bewerter interface {
	Bewerte(titel, text string, r Regeln) Ergebnis
}

// Regelwerk ist der regelbasierte Bewerter.
type Regelwerk struct{}

var _ Bewerter = Regelwerk{}

var musterCache sync.Map

// muster liefert den regulären Ausdruck für ein Wort oder eine Wortfolge.
// Teile dürfen durch Leerzeichen, Bindestrich, Unterstrich, Punkt oder Schrägstrich
// getrennt oder zusammengeschrieben sein („RTX 2080“ = „rtx-2080“ = „RTX2080“).
// Endet das Wort auf einen Buchstaben und ist endung=true, sind weitere Buchstaben
// erlaubt („defekt“ trifft „defekte“).
func muster(wort string, endung bool) *regexp.Regexp {
	schluessel := wort
	if endung {
		schluessel += "\x00+"
	}
	if m, ok := musterCache.Load(schluessel); ok {
		return m.(*regexp.Regexp)
	}
	teile := zerlege(wort)
	if len(teile) == 0 {
		return nil
	}
	for i, t := range teile {
		teile[i] = regexp.QuoteMeta(t)
	}
	kern := strings.Join(teile, `[\s\-_./]*`)
	letztes, _ := utf8.DecodeLastRuneInString(wort)
	if endung && unicode.IsLetter(letztes) {
		kern += `\pL*`
	}
	m := regexp.MustCompile(`(?i)(` + kern + `)`)
	musterCache.Store(schluessel, m)
	return m
}

// zerlege trennt an Nicht-Buchstaben/Ziffern und am Wechsel zwischen Ziffer und Buchstabe
// („RTX2080“ → „RTX“, „2080“; „8GB“ → „8“, „GB“).
func zerlege(wort string) []string {
	var teile []string
	var akt []rune
	var vorher rune
	for _, r := range wort {
		if !unicode.IsLetter(r) && !unicode.IsDigit(r) {
			if len(akt) > 0 {
				teile = append(teile, string(akt))
				akt = nil
			}
			continue
		}
		if len(akt) > 0 && unicode.IsDigit(r) != unicode.IsDigit(vorher) {
			teile = append(teile, string(akt))
			akt = nil
		}
		akt = append(akt, r)
		vorher = r
	}
	if len(akt) > 0 {
		teile = append(teile, string(akt))
	}
	return teile
}

type treffer struct{ start, ende int }

// finde liefert alle Fundstellen von wort in s.
// Links darf kein Zeichen derselben Art stehen (vor einer Ziffer keine Ziffer, vor einem
// Buchstaben kein Buchstabe): „2080“ trifft „RTX2080“, nicht „12080“.
// Rechts darf weder Buchstabe noch Ziffer folgen (außer der erlaubten Endung).
func finde(s, wort string, endung bool) []treffer {
	m := muster(wort, endung)
	if m == nil {
		return nil
	}
	var aus []treffer
	for _, idx := range m.FindAllStringSubmatchIndex(s, -1) {
		start, ende := idx[2], idx[3]
		if start > 0 {
			vor, _ := utf8.DecodeLastRuneInString(s[:start])
			erstes, _ := utf8.DecodeRuneInString(s[start:])
			if (unicode.IsDigit(erstes) && unicode.IsDigit(vor)) || (unicode.IsLetter(erstes) && unicode.IsLetter(vor)) {
				continue
			}
		}
		if ende < len(s) {
			r, _ := utf8.DecodeRuneInString(s[ende:])
			if unicode.IsLetter(r) || unicode.IsDigit(r) {
				continue
			}
		}
		aus = append(aus, treffer{start, ende})
	}
	return aus
}

const satzZeichen = ".!?;\n"

// davor liefert den Text zwischen Satzanfang und start, kleingeschrieben.
func davor(s string, start int) string {
	i := strings.LastIndexAny(s[:start], satzZeichen)
	return strings.ToLower(s[i+1 : start])
}

// danach liefert den Text zwischen ende und Satzende, kleingeschrieben.
func danach(s string, ende int) string {
	rest := s[ende:]
	if i := strings.IndexAny(rest, satzZeichen); i >= 0 {
		rest = rest[:i]
	}
	return strings.ToLower(rest)
}

var verneinungen = map[string]bool{
	"kein": true, "keine": true, "keinen": true, "keiner": true, "keinem": true, "keines": true,
	"nicht": true, "ohne": true, "nie": true, "niemals": true,
	"geen": true, "niet": true, "zonder": true, "nooit": true, "no": true,
}

var (
	nachVerneinung    = regexp.MustCompile(`^[\s,:–-]*(?:(?:ist|sind|wird|werden|is|zijn|wordt|worden)\s+)?(?:nicht|ausgeschlossen|unerwünscht|niet|uitgesloten)(?:[^\pL]|$)`)
	doppeltVerneint   = regexp.MustCompile(`^[\s,:–-]*(?:(?:ist|sind|wird|werden)\s+)?nicht\s+ausgeschlossen`)
	wortGrenze        = regexp.MustCompile(`[^\pL\pN]+`)
	vergleichVorTreff = regexp.MustCompile(`(?:^|[^\pL])(?:(?:fast|beinahe|nahezu|ungefähr|etwa|quasi|genau|genauso|so gut)\s+)?` +
		`(?:wie|ähnlich(?:\s+wie)?|vergleichbar(?:\s+mit)?|statt|anstatt|anstelle(?:\s+von)?|ersatz\s+für|alternative\s+zu|` +
		`niveau(?:\s+(?:einer?|von|der|des))?|leistung\s+(?:einer?|von)|` +
		`(?:besser|schneller|stärker|leistungsstärker|langsamer|schwächer|mehr|weniger|günstiger|billiger|teurer)\s+als|` +
		`vs\.?|versus|gegenüber|vergelijkbaar(?:\s+met)?|zoals|net\s+als|` +
		`(?:beter|sneller|sterker|trager|langzamer|goedkoper|duurder)\s+dan|in\s+plaats\s+van|i\.?p\.?v\.?)` +
		`\s+(?:(?:eine?[rsmn]?|die|der|das|den|dem|de|het|een)\s+)?` +
		`(?:(?:nvidia|geforce|amd|radeon|intel|asus|msi|gigabyte|zotac|evga|palit|gainward|kfa2|pny|sapphire|xfx|powercolor|inno3d)\s+)?$`)
)

// istVerneint: Verneinung bis zu drei Wörter davor („kein Tausch“, „nicht defekt“)
// oder direkt danach („Tausch nicht möglich“, „Tausch ausgeschlossen“).
func istVerneint(s string, t treffer) bool {
	worte := wortGrenze.Split(strings.TrimSpace(davor(s, t.start)), -1)
	for i := len(worte) - 1; i >= 0 && i >= len(worte)-3; i-- {
		if verneinungen[worte[i]] {
			return true
		}
	}
	nach := danach(s, t.ende)
	if doppeltVerneint.MatchString(nach) {
		return false
	}
	return nachVerneinung.MatchString(nach)
}

// istVergleich: Der Suchbegriff wird verglichen („fast wie eine RTX 2080“,
// „schneller als eine 2080“) oder verneint („keine RTX 2080“).
func istVergleich(s string, t treffer) bool {
	vor := davor(s, t.start)
	if vergleichVorTreff.MatchString(vor) {
		return true
	}
	worte := wortGrenze.Split(strings.TrimSpace(vor), -1)
	if n := len(worte); n > 0 && verneinungen[worte[n-1]] {
		return true
	}
	return false
}

func bereinige(ws []string) []string {
	var aus []string
	for _, w := range ws {
		if w = strings.TrimSpace(w); w != "" {
			aus = append(aus, w)
		}
	}
	return aus
}

// Bewerte prüft Titel und Text.
func (Regelwerk) Bewerte(titel, text string, r Regeln) Ergebnis {
	erg := Ergebnis{Sichtbar: true}
	felder := []struct {
		feld Feld
		s    string
	}{{FeldTitel, titel}, {FeldText, text}}

	// Ausschlusswörter
	pruefeListe := func(woerter []string, modus Modus, liste string) {
		for _, w := range bereinige(woerter) {
			for _, f := range felder {
				for _, t := range finde(f.s, w, true) {
					art := ArtAusschluss
					if istVerneint(f.s, t) {
						art = ArtVerneint
					}
					erg.Fundstellen = append(erg.Fundstellen, Fundstelle{Wort: w, Feld: f.feld, Start: t.start, Ende: t.ende, Art: art})
					if art != ArtAusschluss {
						continue
					}
					if modus == Hart || (modus == Titel && f.feld == FeldTitel) {
						erg.Sichtbar = false
						erg.Gruende = append(erg.Gruende, liste+": „"+w+"“ im "+map[Feld]string{FeldTitel: "Titel", FeldText: "Text"}[f.feld])
					}
				}
			}
		}
	}
	if r.StandardAktiv {
		pruefeListe(Standard, standardModus(r.StandardModus), "Standard-Ausschluss")
	}
	pruefeListe(r.Ausschluss, standardModus(r.AusschlussModus), "Ausschlusswort")

	// Pflichtwörter: jedes muss unverneint in Titel oder Text stehen.
	for _, w := range bereinige(r.Pflicht) {
		gefunden := false
		for _, f := range felder {
			for _, t := range finde(f.s, w, true) {
				if istVerneint(f.s, t) {
					continue
				}
				gefunden = true
				erg.Fundstellen = append(erg.Fundstellen, Fundstelle{Wort: w, Feld: f.feld, Start: t.start, Ende: t.ende, Art: ArtPflicht})
			}
		}
		if !gefunden {
			erg.Sichtbar = false
			erg.Gruende = append(erg.Gruende, "Pflichtwort fehlt: „"+w+"“")
		}
	}

	// Einstufung des Suchbegriffs
	if sb := strings.TrimSpace(r.Suchbegriff); sb != "" {
		angebot, vergleich, imTitel := 0, 0, false
		for _, f := range felder {
			for _, t := range finde(f.s, sb, false) {
				art := ArtSuchbegriff
				if istVergleich(f.s, t) {
					art = ArtVergleich
					vergleich++
				} else {
					angebot++
					if f.feld == FeldTitel {
						imTitel = true
					}
				}
				erg.Fundstellen = append(erg.Fundstellen, Fundstelle{Wort: sb, Feld: f.feld, Start: t.start, Ende: t.ende, Art: art})
			}
		}
		switch {
		case angebot == 0 && vergleich > 0:
			erg.Einstufung = Vergleich
		case angebot == 0:
			erg.Einstufung = Zweifelhaft
		case vergleich > 0 && !imTitel:
			erg.Einstufung = Zweifelhaft
		default:
			erg.Einstufung = Echt
		}
	}

	sort.SliceStable(erg.Fundstellen, func(i, j int) bool {
		a, b := erg.Fundstellen[i], erg.Fundstellen[j]
		if a.Feld != b.Feld {
			return a.Feld == FeldTitel
		}
		return a.Start < b.Start
	})
	return erg
}

func standardModus(m Modus) Modus {
	switch m {
	case Markieren, Titel, Hart:
		return m
	}
	return Titel
}
