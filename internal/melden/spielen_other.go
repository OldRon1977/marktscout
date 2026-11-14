//go:build !windows

package melden

import (
	"errors"
	"os/exec"
)

// Spieler in der Reihenfolge, in der sie versucht werden (PipeWire, PulseAudio, ALSA).
var spielerListe = [][]string{{"pw-play"}, {"paplay"}, {"aplay", "-q"}}

// Spieler liefert den ersten vorhandenen Abspieler oder nil.
func Spieler() []string {
	for _, s := range spielerListe {
		if _, err := exec.LookPath(s[0]); err == nil {
			return s
		}
	}
	return nil
}

func spiele(pfad string) error {
	s := Spieler()
	if s == nil {
		return errors.New("kein Abspieler gefunden (pw-play, paplay, aplay)")
	}
	cmd := exec.Command(s[0], append(s[1:], pfad)...)
	return cmd.Run()
}
