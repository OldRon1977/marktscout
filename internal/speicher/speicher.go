// Package speicher legt Suchaufträge, Anzeigen und Treffer in einer SQLite-Datei ab.
package speicher

import (
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/OldRon1977/marktscout/internal/anzeige"
	"github.com/OldRon1977/marktscout/internal/auftrag"
	_ "modernc.org/sqlite"
)

// ErrNichtGefunden: Datensatz existiert nicht.
var ErrNichtGefunden = errors.New("nicht gefunden")

// DB ist die geöffnete Datenbank.
type DB struct{ db *sql.DB }

const schema = `
CREATE TABLE IF NOT EXISTS auftraege (
	id INTEGER PRIMARY KEY AUTOINCREMENT,
	daten TEXT NOT NULL,
	letzter_lauf INTEGER,
	letzter_fehler TEXT NOT NULL DEFAULT ''
);
CREATE TABLE IF NOT EXISTS anzeigen (
	quelle TEXT NOT NULL,
	id TEXT NOT NULL,
	url TEXT NOT NULL,
	titel TEXT NOT NULL,
	beschreibung TEXT NOT NULL DEFAULT '',
	beschreibung_voll INTEGER NOT NULL DEFAULT 0,
	preis_cent INTEGER NOT NULL DEFAULT 0,
	preis_typ TEXT NOT NULL,
	ort TEXT NOT NULL DEFAULT '',
	plz TEXT NOT NULL DEFAULT '',
	land TEXT NOT NULL DEFAULT '',
	lat REAL NOT NULL DEFAULT 0,
	lon REAL NOT NULL DEFAULT 0,
	bild TEXT NOT NULL DEFAULT '',
	eingestellt TEXT NOT NULL DEFAULT '',
	hervorgehoben INTEGER NOT NULL DEFAULT 0,
	status TEXT NOT NULL,
	status_geprueft INTEGER NOT NULL,
	PRIMARY KEY (quelle, id)
);
CREATE TABLE IF NOT EXISTS treffer (
	auftrag_id INTEGER NOT NULL REFERENCES auftraege(id) ON DELETE CASCADE,
	quelle TEXT NOT NULL,
	anzeige_id TEXT NOT NULL,
	erstmals INTEGER NOT NULL,
	zuletzt INTEGER NOT NULL,
	entfernung_km REAL NOT NULL DEFAULT -1,
	suchort TEXT NOT NULL DEFAULT '',
	neu INTEGER NOT NULL DEFAULT 1,
	verworfen INTEGER NOT NULL DEFAULT 0,
	PRIMARY KEY (auftrag_id, quelle, anzeige_id),
	FOREIGN KEY (quelle, anzeige_id) REFERENCES anzeigen(quelle, id)
);
CREATE TABLE IF NOT EXISTS einstellungen (
	schluessel TEXT PRIMARY KEY,
	wert TEXT NOT NULL
);
`

// Oeffne öffnet oder erzeugt die Datenbank unter pfad.
func Oeffne(pfad string) (*DB, error) {
	db, err := sql.Open("sqlite", "file:"+pfad+"?_pragma=foreign_keys(1)&_pragma=journal_mode(WAL)&_pragma=busy_timeout(5000)")
	if err != nil {
		return nil, err
	}
	db.SetMaxOpenConns(1)
	if _, err := db.Exec(schema); err != nil {
		db.Close()
		return nil, fmt.Errorf("schema: %w", err)
	}
	return &DB{db: db}, nil
}

// Schliesse schließt die Datenbank.
func (d *DB) Schliesse() error { return d.db.Close() }

func unix(t time.Time) int64 { return t.Unix() }

// --- Aufträge ---

func (d *DB) leseAuftrag(id int64, daten string, lauf sql.NullInt64, fehler string) (auftrag.Auftrag, error) {
	a := auftrag.Neu()
	if err := json.Unmarshal([]byte(daten), &a); err != nil {
		return a, fmt.Errorf("auftrag %d: %w", id, err)
	}
	a.ID = id
	a.LetzterFehler = fehler
	a.LetzterLauf = nil
	if lauf.Valid {
		t := time.Unix(lauf.Int64, 0)
		a.LetzterLauf = &t
	}
	return a, nil
}

// Auftraege liefert alle Suchaufträge, nach Name sortiert.
func (d *DB) Auftraege() ([]auftrag.Auftrag, error) {
	rows, err := d.db.Query(`SELECT id, daten, letzter_lauf, letzter_fehler FROM auftraege`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var aus []auftrag.Auftrag
	for rows.Next() {
		var id int64
		var daten, fehler string
		var lauf sql.NullInt64
		if err := rows.Scan(&id, &daten, &lauf, &fehler); err != nil {
			return nil, err
		}
		a, err := d.leseAuftrag(id, daten, lauf, fehler)
		if err != nil {
			return nil, err
		}
		aus = append(aus, a)
	}
	return aus, rows.Err()
}

// Auftrag liefert einen Suchauftrag.
func (d *DB) Auftrag(id int64) (auftrag.Auftrag, error) {
	var daten, fehler string
	var lauf sql.NullInt64
	err := d.db.QueryRow(`SELECT daten, letzter_lauf, letzter_fehler FROM auftraege WHERE id = ?`, id).Scan(&daten, &lauf, &fehler)
	if errors.Is(err, sql.ErrNoRows) {
		return auftrag.Auftrag{}, ErrNichtGefunden
	}
	if err != nil {
		return auftrag.Auftrag{}, err
	}
	return d.leseAuftrag(id, daten, lauf, fehler)
}

// SpeichereAuftrag legt einen Auftrag an (ID 0) oder ändert ihn. Laufdaten bleiben unverändert.
func (d *DB) SpeichereAuftrag(a *auftrag.Auftrag) error {
	kopie := *a
	kopie.ID, kopie.LetzterLauf, kopie.LetzterFehler = 0, nil, ""
	daten, err := json.Marshal(kopie)
	if err != nil {
		return err
	}
	if a.ID == 0 {
		r, err := d.db.Exec(`INSERT INTO auftraege (daten) VALUES (?)`, string(daten))
		if err != nil {
			return err
		}
		a.ID, err = r.LastInsertId()
		return err
	}
	r, err := d.db.Exec(`UPDATE auftraege SET daten = ? WHERE id = ?`, string(daten), a.ID)
	if err != nil {
		return err
	}
	if n, _ := r.RowsAffected(); n == 0 {
		return ErrNichtGefunden
	}
	return nil
}

// LoescheAuftrag löscht einen Auftrag samt Treffern.
func (d *DB) LoescheAuftrag(id int64) error {
	r, err := d.db.Exec(`DELETE FROM auftraege WHERE id = ?`, id)
	if err != nil {
		return err
	}
	if n, _ := r.RowsAffected(); n == 0 {
		return ErrNichtGefunden
	}
	_, err = d.db.Exec(`DELETE FROM anzeigen WHERE NOT EXISTS (SELECT 1 FROM treffer t WHERE t.quelle = anzeigen.quelle AND t.anzeige_id = anzeigen.id)`)
	return err
}

// BeginneLauf setzt die Kennung „neu“ der bisherigen Treffer zurück.
func (d *DB) BeginneLauf(id int64) error {
	_, err := d.db.Exec(`UPDATE treffer SET neu = 0 WHERE auftrag_id = ?`, id)
	return err
}

// BeendeLauf hält Zeitpunkt und Fehlertext des Laufs fest.
func (d *DB) BeendeLauf(id int64, zeit time.Time, fehler string) error {
	_, err := d.db.Exec(`UPDATE auftraege SET letzter_lauf = ?, letzter_fehler = ? WHERE id = ?`, unix(zeit), fehler, id)
	return err
}

// --- Anzeigen und Treffer ---

// MerkeTreffer speichert eine Anzeige aus der Suche und ordnet sie dem Auftrag zu.
// neu meldet, ob die Anzeige für diesen Auftrag zum ersten Mal gefunden wurde.
// Vom Nutzer gelöschte Treffer bleiben gelöscht.
func (d *DB) MerkeTreffer(auftragID int64, a anzeige.Anzeige, suchort string, zeit time.Time) (neu bool, err error) {
	tx, err := d.db.Begin()
	if err != nil {
		return false, err
	}
	defer tx.Rollback()
	_, err = tx.Exec(`
INSERT INTO anzeigen (quelle, id, url, titel, beschreibung, preis_cent, preis_typ, ort, plz, land, lat, lon, bild, eingestellt, hervorgehoben, status, status_geprueft)
VALUES (?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?)
ON CONFLICT (quelle, id) DO UPDATE SET
	url = excluded.url, titel = excluded.titel,
	beschreibung = CASE WHEN anzeigen.beschreibung_voll = 1 THEN anzeigen.beschreibung ELSE excluded.beschreibung END,
	preis_cent = excluded.preis_cent, preis_typ = excluded.preis_typ,
	ort = excluded.ort, plz = CASE WHEN excluded.plz <> '' THEN excluded.plz ELSE anzeigen.plz END,
	land = excluded.land,
	lat = CASE WHEN excluded.lat <> 0 THEN excluded.lat ELSE anzeigen.lat END,
	lon = CASE WHEN excluded.lon <> 0 THEN excluded.lon ELSE anzeigen.lon END,
	bild = excluded.bild, hervorgehoben = excluded.hervorgehoben,
	status = excluded.status, status_geprueft = excluded.status_geprueft`,
		a.Quelle, a.ID, a.URL, a.Titel, a.Beschreibung, a.PreisCent, a.PreisTyp, a.Ort, a.PLZ, a.Land,
		a.Lat, a.Lon, a.BildURL, a.Eingestellt, a.Hervorgehoben, a.Status, unix(zeit))
	if err != nil {
		return false, err
	}
	r, err := tx.Exec(`INSERT OR IGNORE INTO treffer (auftrag_id, quelle, anzeige_id, erstmals, zuletzt, entfernung_km, suchort)
VALUES (?,?,?,?,?,?,?)`, auftragID, a.Quelle, a.ID, unix(zeit), unix(zeit), a.EntfernungKm, suchort)
	if err != nil {
		return false, err
	}
	n, _ := r.RowsAffected()
	neu = n == 1
	if !neu {
		_, err = tx.Exec(`UPDATE treffer SET zuletzt = ?,
	suchort = CASE WHEN ? >= 0 AND (entfernung_km < 0 OR ? < entfernung_km) THEN ? ELSE suchort END,
	entfernung_km = CASE WHEN ? >= 0 AND (entfernung_km < 0 OR ? < entfernung_km) THEN ? ELSE entfernung_km END
WHERE auftrag_id = ? AND quelle = ? AND anzeige_id = ?`,
			unix(zeit), a.EntfernungKm, a.EntfernungKm, suchort, a.EntfernungKm, a.EntfernungKm, a.EntfernungKm,
			auftragID, a.Quelle, a.ID)
		if err != nil {
			return false, err
		}
	}
	return neu, tx.Commit()
}

// SpeichereDetail übernimmt Daten der Detailseite (vollständige Beschreibung, Koordinaten, Status).
func (d *DB) SpeichereDetail(q anzeige.Quelle, id string, a anzeige.Anzeige, zeit time.Time) error {
	if a.Status == anzeige.StatusGeloescht {
		return d.SetzeStatus(q, id, a.Status, zeit)
	}
	_, err := d.db.Exec(`UPDATE anzeigen SET
	beschreibung = ?, beschreibung_voll = 1,
	titel = CASE WHEN ? <> '' THEN ? ELSE titel END,
	plz = CASE WHEN ? <> '' THEN ? ELSE plz END,
	ort = CASE WHEN ? <> '' THEN ? ELSE ort END,
	lat = CASE WHEN ? <> 0 THEN ? ELSE lat END,
	lon = CASE WHEN ? <> 0 THEN ? ELSE lon END,
	status = ?, status_geprueft = ?
WHERE quelle = ? AND id = ?`,
		a.Beschreibung, a.Titel, a.Titel, a.PLZ, a.PLZ, a.Ort, a.Ort, a.Lat, a.Lat, a.Lon, a.Lon,
		a.Status, unix(zeit), q, id)
	return err
}

// SetzeStatus hält den geprüften Status einer Anzeige fest.
func (d *DB) SetzeStatus(q anzeige.Quelle, id string, s anzeige.Status, zeit time.Time) error {
	_, err := d.db.Exec(`UPDATE anzeigen SET status = ?, status_geprueft = ? WHERE quelle = ? AND id = ?`, s, unix(zeit), q, id)
	return err
}

// Treffer ist eine Anzeige mit ihrer Zuordnung zu einem Auftrag.
type Treffer struct {
	anzeige.Anzeige
	BeschreibungVoll bool      `json:"-"`
	Erstmals         time.Time `json:"erstmals"`
	Zuletzt          time.Time `json:"zuletzt"`
	Suchort          string    `json:"suchort"`
	Neu              bool      `json:"neu"`
	StatusGeprueft   time.Time `json:"statusGeprueft"`
}

// TrefferFilter wählt Treffer aus.
type TrefferFilter struct {
	OhneDetail bool // nur Anzeigen ohne vollständige Beschreibung, nicht gelöscht
}

// Treffer liefert die nicht verworfenen Treffer eines Auftrags.
func (d *DB) Treffer(auftragID int64, f TrefferFilter) ([]Treffer, error) {
	q := `SELECT a.quelle, a.id, a.url, a.titel, a.beschreibung, a.beschreibung_voll, a.preis_cent, a.preis_typ,
	a.ort, a.plz, a.land, a.lat, a.lon, a.bild, a.eingestellt, a.hervorgehoben, a.status, a.status_geprueft,
	t.erstmals, t.zuletzt, t.entfernung_km, t.suchort, t.neu
FROM treffer t JOIN anzeigen a ON a.quelle = t.quelle AND a.id = t.anzeige_id
WHERE t.auftrag_id = ? AND t.verworfen = 0`
	if f.OhneDetail {
		q += ` AND a.beschreibung_voll = 0 AND a.status <> 'geloescht'`
	}
	q += ` ORDER BY t.erstmals, a.id`
	rows, err := d.db.Query(q, auftragID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var aus []Treffer
	for rows.Next() {
		var t Treffer
		var erst, zul, gepr int64
		if err := rows.Scan(&t.Quelle, &t.ID, &t.URL, &t.Titel, &t.Beschreibung, &t.BeschreibungVoll, &t.PreisCent, &t.PreisTyp,
			&t.Ort, &t.PLZ, &t.Land, &t.Lat, &t.Lon, &t.BildURL, &t.Eingestellt, &t.Hervorgehoben, &t.Status, &gepr,
			&erst, &zul, &t.EntfernungKm, &t.Suchort, &t.Neu); err != nil {
			return nil, err
		}
		t.BeschreibungVollstaendig = t.BeschreibungVoll
		t.Erstmals, t.Zuletzt, t.StatusGeprueft = time.Unix(erst, 0), time.Unix(zul, 0), time.Unix(gepr, 0)
		aus = append(aus, t)
	}
	return aus, rows.Err()
}

// Verwerfe löscht einen Treffer aus der Liste des Auftrags. Er erscheint bei späteren
// Suchläufen nicht wieder.
func (d *DB) Verwerfe(auftragID int64, q anzeige.Quelle, id string) error {
	r, err := d.db.Exec(`UPDATE treffer SET verworfen = 1, neu = 0 WHERE auftrag_id = ? AND quelle = ? AND anzeige_id = ?`, auftragID, q, id)
	if err != nil {
		return err
	}
	if n, _ := r.RowsAffected(); n == 0 {
		return ErrNichtGefunden
	}
	return nil
}

// --- Einstellungen ---

// Einstellung liefert einen gespeicherten Wert oder standard.
func (d *DB) Einstellung(schluessel, standard string) (string, error) {
	var w string
	err := d.db.QueryRow(`SELECT wert FROM einstellungen WHERE schluessel = ?`, schluessel).Scan(&w)
	if errors.Is(err, sql.ErrNoRows) {
		return standard, nil
	}
	return w, err
}

// SetzeEinstellung speichert einen Wert.
func (d *DB) SetzeEinstellung(schluessel, wert string) error {
	_, err := d.db.Exec(`INSERT INTO einstellungen (schluessel, wert) VALUES (?, ?)
ON CONFLICT (schluessel) DO UPDATE SET wert = excluded.wert`, schluessel, wert)
	return err
}
