package main

import (
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

// Baut das echte Programm mit gesetzter Version und ruft es auf.
func TestVersionAusgabe(t *testing.T) {
	bin := filepath.Join(t.TempDir(), "marktscout")
	if runtime.GOOS == "windows" {
		bin += ".exe"
	}
	bau := exec.Command("go", "build", "-ldflags", "-X main.version=9.8.7-test", "-o", bin, ".")
	if out, err := bau.CombinedOutput(); err != nil {
		t.Fatalf("Bauen fehlgeschlagen: %v\n%s", err, out)
	}

	out, err := exec.Command(bin, "-version").CombinedOutput()
	if err != nil {
		t.Fatalf("Aufruf mit -version fehlgeschlagen: %v\n%s", err, out)
	}
	if got, want := strings.TrimSpace(string(out)), "marktscout 9.8.7-test"; got != want {
		t.Fatalf("Ausgabe %q, erwartet %q", got, want)
	}
}
