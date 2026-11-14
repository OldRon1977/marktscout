//go:build windows

package melden

import (
	"os/exec"
	"strings"
	"syscall"
)

// Spieler liefert den Abspieler (PowerShell mit System.Media.SoundPlayer).
func Spieler() []string { return []string{"powershell", "-NoProfile", "-NonInteractive", "-Command"} }

func spiele(pfad string) error {
	s := Spieler()
	befehl := "(New-Object Media.SoundPlayer '" + strings.ReplaceAll(pfad, "'", "''") + "').PlaySync()"
	cmd := exec.Command(s[0], append(s[1:], befehl)...)
	cmd.SysProcAttr = &syscall.SysProcAttr{HideWindow: true, CreationFlags: 0x08000000} // CREATE_NO_WINDOW
	return cmd.Run()
}
