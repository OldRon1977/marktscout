// Package melden gibt Desktopmeldungen und einen Hinweiston aus.
package melden

import (
	"bytes"
	"encoding/binary"
	"math"
	"os"
	"path/filepath"

	"github.com/OldRon1977/marktscout/internal/symbol"
	"github.com/gen2brain/beeep"
)

// Desktop zeigt eine Desktopmeldung.
func Desktop(titel, text string) error {
	beeep.AppName = "Marktscout"
	return beeep.Notify(titel, text, symbol.PNG(64))
}

// Klang erzeugt den Hinweiston als WAV (44,1 kHz, 16 Bit, mono): zwei aufsteigende Töne.
func Klang() []byte {
	const rate = 44100
	toene := []struct {
		hz    float64
		dauer float64
	}{{880, 0.14}, {1320, 0.26}}
	var proben []int16
	for _, t := range toene {
		n := int(t.dauer * rate)
		for i := 0; i < n; i++ {
			x := float64(i) / rate
			huelle := math.Min(1, float64(i)/400) * math.Exp(-3*x/t.dauer)
			proben = append(proben, int16(0.35*32767*huelle*math.Sin(2*math.Pi*t.hz*x)))
		}
	}
	var b bytes.Buffer
	daten := len(proben) * 2
	b.WriteString("RIFF")
	binary.Write(&b, binary.LittleEndian, uint32(36+daten))
	b.WriteString("WAVEfmt ")
	binary.Write(&b, binary.LittleEndian, uint32(16))
	binary.Write(&b, binary.LittleEndian, uint16(1)) // PCM
	binary.Write(&b, binary.LittleEndian, uint16(1)) // mono
	binary.Write(&b, binary.LittleEndian, uint32(rate))
	binary.Write(&b, binary.LittleEndian, uint32(rate*2))
	binary.Write(&b, binary.LittleEndian, uint16(2))
	binary.Write(&b, binary.LittleEndian, uint16(16))
	b.WriteString("data")
	binary.Write(&b, binary.LittleEndian, uint32(daten))
	binary.Write(&b, binary.LittleEndian, proben)
	return b.Bytes()
}

// klangDatei legt den Klang einmalig im Temp-Verzeichnis ab.
func klangDatei() (string, error) {
	pfad := filepath.Join(os.TempDir(), "marktscout-hinweis.wav")
	if info, err := os.Stat(pfad); err == nil && info.Size() == int64(len(Klang())) {
		return pfad, nil
	}
	return pfad, os.WriteFile(pfad, Klang(), 0o644)
}

// Ton spielt den Hinweiston ab.
func Ton() error {
	pfad, err := klangDatei()
	if err != nil {
		return err
	}
	return spiele(pfad)
}
