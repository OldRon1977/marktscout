// Package protokoll schreibt die Logdatei mit Rotation nach Größe.
package protokoll

import (
	"fmt"
	"io"
	"log/slog"
	"os"
	"path/filepath"
	"sync"
)

// Dateiname der aktuellen Logdatei.
const Dateiname = "marktscout.log"

// Datei ist eine rotierende Logdatei. Ist die aktuelle Datei größer als MaxBytes,
// wird sie zu .1 umbenannt (.1 zu .2 usw., höchstens Behalten ältere Dateien).
type Datei struct {
	mu       sync.Mutex
	verz     string
	maxBytes int64
	behalten int
	f        *os.File
	groesse  int64
}

// Oeffne öffnet die Logdatei im Verzeichnis verz.
func Oeffne(verz string, maxBytes int64, behalten int) (*Datei, error) {
	if err := os.MkdirAll(verz, 0o755); err != nil {
		return nil, err
	}
	d := &Datei{verz: verz, maxBytes: maxBytes, behalten: behalten}
	if err := d.oeffne(); err != nil {
		return nil, err
	}
	return d, nil
}

func (d *Datei) pfad(n int) string {
	if n == 0 {
		return filepath.Join(d.verz, Dateiname)
	}
	return filepath.Join(d.verz, fmt.Sprintf("%s.%d", Dateiname, n))
}

func (d *Datei) oeffne() error {
	f, err := os.OpenFile(d.pfad(0), os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0o644)
	if err != nil {
		return err
	}
	info, err := f.Stat()
	if err != nil {
		f.Close()
		return err
	}
	d.f, d.groesse = f, info.Size()
	return nil
}

func (d *Datei) rotiere() error {
	if err := d.f.Close(); err != nil {
		return err
	}
	os.Remove(d.pfad(d.behalten))
	for n := d.behalten - 1; n >= 0; n-- {
		if _, err := os.Stat(d.pfad(n)); err == nil {
			if err := os.Rename(d.pfad(n), d.pfad(n+1)); err != nil {
				return err
			}
		}
	}
	return d.oeffne()
}

// Write schreibt in die Logdatei und rotiert bei Bedarf vorher.
func (d *Datei) Write(p []byte) (int, error) {
	d.mu.Lock()
	defer d.mu.Unlock()
	if d.groesse > 0 && d.groesse+int64(len(p)) > d.maxBytes {
		if err := d.rotiere(); err != nil {
			return 0, err
		}
	}
	n, err := d.f.Write(p)
	d.groesse += int64(n)
	return n, err
}

// Exportiere schreibt alle Logdateien, älteste zuerst, nach w.
func (d *Datei) Exportiere(w io.Writer) error {
	d.mu.Lock()
	defer d.mu.Unlock()
	for n := d.behalten; n >= 0; n-- {
		f, err := os.Open(d.pfad(n))
		if os.IsNotExist(err) {
			continue
		}
		if err != nil {
			return err
		}
		_, err = io.Copy(w, f)
		f.Close()
		if err != nil {
			return err
		}
	}
	return nil
}

// Schliesse schließt die Datei.
func (d *Datei) Schliesse() error {
	d.mu.Lock()
	defer d.mu.Unlock()
	return d.f.Close()
}

// Logger liefert einen Logger, der in die Datei schreibt.
func (d *Datei) Logger(level slog.Level) *slog.Logger {
	return slog.New(slog.NewTextHandler(d, &slog.HandlerOptions{Level: level}))
}
