package melden

import (
	"bytes"
	"encoding/binary"
	"os"
	"os/exec"
	"testing"
)

func TestKlangIstGueltigesWAV(t *testing.T) {
	k := Klang()
	if !bytes.HasPrefix(k, []byte("RIFF")) || string(k[8:16]) != "WAVEfmt " || string(k[36:40]) != "data" {
		t.Fatal("WAV-Kopf ungültig")
	}
	if got := binary.LittleEndian.Uint32(k[4:8]); int(got) != len(k)-8 {
		t.Errorf("RIFF-Länge %d, Datei %d", got, len(k)-8)
	}
	if got := binary.LittleEndian.Uint32(k[40:44]); int(got) != len(k)-44 {
		t.Errorf("data-Länge %d, Daten %d", got, len(k)-44)
	}
	if rate := binary.LittleEndian.Uint32(k[24:28]); rate != 44100 {
		t.Errorf("Abtastrate %d", rate)
	}
	// Dauer 0,4 s
	if n := (len(k) - 44) / 2; n < 17000 || n > 18000 {
		t.Errorf("%d Proben, erwartet ca. 17640", n)
	}
	// Nicht stumm
	var max int16
	for i := 44; i+1 < len(k); i += 2 {
		v := int16(binary.LittleEndian.Uint16(k[i:]))
		if v > max {
			max = v
		}
	}
	if max < 5000 {
		t.Errorf("Spitzenwert %d, Ton zu leise", max)
	}
}

// Spielt den Ton tatsächlich ab, wenn ein Abspieler und eine Audioausgabe vorhanden sind.
// In CI ohne Audiogerät gibt es keinen Abspieler.
func TestTonAbspielen(t *testing.T) {
	s := Spieler()
	if s == nil {
		t.Skip("kein Abspieler vorhanden")
	}
	if _, err := exec.LookPath(s[0]); err != nil {
		t.Skip("Abspieler nicht im Pfad")
	}
	if os.Getenv("MARKTSCOUT_TON") != "1" {
		t.Skip("Abspielen nur mit MARKTSCOUT_TON=1")
	}
	if err := Ton(); err != nil {
		t.Fatal(err)
	}
}
