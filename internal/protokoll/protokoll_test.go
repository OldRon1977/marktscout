package protokoll

import (
	"bytes"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestRotationUndExport(t *testing.T) {
	verz := t.TempDir()
	d, err := Oeffne(verz, 1000, 2)
	if err != nil {
		t.Fatal(err)
	}
	log := d.Logger(slog.LevelInfo)
	for i := 0; i < 100; i++ {
		log.Info("zeile", "nr", i)
	}
	d.Schliesse()

	dateien, _ := filepath.Glob(filepath.Join(verz, Dateiname+"*"))
	if len(dateien) != 3 {
		t.Fatalf("%d Dateien, erwartet 3 (aktuell + 2 ältere): %v", len(dateien), dateien)
	}
	for _, f := range dateien {
		info, _ := os.Stat(f)
		if info.Size() > 1000 {
			t.Errorf("%s: %d Bytes, Grenze 1000", f, info.Size())
		}
	}

	d, _ = Oeffne(verz, 1000, 2)
	defer d.Schliesse()
	var b bytes.Buffer
	if err := d.Exportiere(&b); err != nil {
		t.Fatal(err)
	}
	export := b.String()
	if !strings.Contains(export, "nr=99") {
		t.Error("letzte Zeile fehlt im Export")
	}
	// Älteste zuerst: Zeilennummern steigen
	letzte := -1
	for _, z := range strings.Split(strings.TrimSpace(export), "\n") {
		var nr int
		i := strings.Index(z, "nr=")
		if i < 0 {
			t.Fatalf("Zeile ohne nr: %q", z)
		}
		fmt.Sscanf(z[i:], "nr=%d", &nr)
		if nr <= letzte {
			t.Fatalf("Reihenfolge: %d nach %d", nr, letzte)
		}
		letzte = nr
	}
	if strings.Contains(export, "nr=0 ") {
		t.Error("älteste Zeilen hätten wegrotiert sein müssen")
	}
}
