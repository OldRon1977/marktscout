// Marktscout: Anzeigensuche für kleinanzeigen.de und marktplaats.nl.
package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"os/exec"
	"os/signal"
	"path/filepath"
	"runtime"
	"strconv"
	"syscall"
	"time"

	"fyne.io/systray"
	"github.com/OldRon1977/marktscout/internal/abruf"
	"github.com/OldRon1977/marktscout/internal/kleinanzeigen"
	"github.com/OldRon1977/marktscout/internal/marktplaats"
	"github.com/OldRon1977/marktscout/internal/protokoll"
	"github.com/OldRon1977/marktscout/internal/speicher"
	"github.com/OldRon1977/marktscout/internal/suchlauf"
	"github.com/OldRon1977/marktscout/internal/symbol"
	"github.com/OldRon1977/marktscout/internal/textpruefung"
	"github.com/OldRon1977/marktscout/internal/web"
)

// version wird beim Bauen über -ldflags "-X main.version=…" gesetzt.
var version = "dev"

// StandardPort der Oberfläche.
const StandardPort = 47811

func main() {
	zeigeVersion := flag.Bool("version", false, "Version ausgeben und beenden")
	datenVerz := flag.String("daten", "", "Datenverzeichnis (Standard: Ordner „daten“ neben dem Programm)")
	port := flag.Int("port", StandardPort, "Port der Oberfläche auf 127.0.0.1")
	keinBrowser := flag.Bool("kein-browser", false, "Browser beim Start nicht öffnen")
	keinTray := flag.Bool("kein-tray", false, "kein Symbol im Infobereich")
	flag.Parse()

	if *zeigeVersion {
		fmt.Println("marktscout", version)
		return
	}
	if err := starte(*datenVerz, *port, !*keinBrowser, !*keinTray); err != nil {
		fmt.Fprintln(os.Stderr, "marktscout:", err)
		os.Exit(1)
	}
}

func standardDaten() (string, error) {
	exe, err := os.Executable()
	if err != nil {
		return "", err
	}
	if p, err := filepath.EvalSymlinks(exe); err == nil {
		exe = p
	}
	return filepath.Join(filepath.Dir(exe), "daten"), nil
}

// laeuftSchon prüft, ob unter der Adresse bereits ein Marktscout antwortet.
func laeuftSchon(url string) bool {
	c := http.Client{Timeout: 2 * time.Second}
	r, err := c.Get(url + "/api/version")
	if err != nil {
		return false
	}
	defer r.Body.Close()
	var v struct{ Name string }
	return json.NewDecoder(r.Body).Decode(&v) == nil && v.Name == "marktscout"
}

func oeffneBrowser(url string) {
	var cmd *exec.Cmd
	switch runtime.GOOS {
	case "windows":
		cmd = exec.Command("rundll32", "url.dll,FileProtocolHandler", url)
	case "darwin":
		cmd = exec.Command("open", url)
	default:
		cmd = exec.Command("xdg-open", url)
	}
	cmd.Start()
}

func starte(datenVerz string, port int, browser, tray bool) error {
	if datenVerz == "" {
		var err error
		if datenVerz, err = standardDaten(); err != nil {
			return err
		}
	}
	if err := os.MkdirAll(datenVerz, 0o755); err != nil {
		return err
	}
	adresse := "127.0.0.1:" + strconv.Itoa(port)
	url := "http://" + adresse

	datei, err := protokoll.Oeffne(filepath.Join(datenVerz, "logs"), 2<<20, 3)
	if err != nil {
		return err
	}
	defer datei.Schliesse()
	log := datei.Logger(slog.LevelInfo)

	db, err := speicher.Oeffne(filepath.Join(datenVerz, "marktscout.db"))
	if err != nil {
		return err
	}
	defer db.Schliesse()

	client := abruf.Standard(log)
	ortClient := abruf.Neu(300*time.Millisecond, 800*time.Millisecond, log)
	ctx, beende := context.WithCancel(context.Background())
	defer beende()

	srv := &web.Server{
		DB: db,
		Lauf: &suchlauf.Lauf{DB: db, KA: kleinanzeigen.Neu(client), MP: marktplaats.Neu(client),
			Bewerter: textpruefung.Regelwerk{}, Log: log},
		Orte:    kleinanzeigen.Neu(ortClient),
		Log:     log,
		Datei:   datei,
		Version: version,
		Melder:  web.StandardMelder(),
		Beenden: beende,
	}
	if _, _, err := srv.Starte(ctx, adresse); err != nil {
		if laeuftSchon(url) {
			if browser {
				oeffneBrowser(url)
			}
			return nil
		}
		return fmt.Errorf("Port %d belegt: %w", port, err)
	}
	log.Info("gestartet", "version", version, "adresse", url, "daten", datenVerz)
	fmt.Println("Marktscout läuft auf", url)
	if browser {
		oeffneBrowser(url)
	}

	sig := make(chan os.Signal, 1)
	signal.Notify(sig, os.Interrupt, syscall.SIGTERM)
	go func() {
		select {
		case <-sig:
			beende()
		case <-ctx.Done():
		}
	}()

	if tray {
		go func() {
			<-ctx.Done()
			systray.Quit()
		}()
		systray.Run(func() { trayMenue(url, beende) }, func() {})
		beende()
	} else {
		<-ctx.Done()
	}
	log.Info("beendet")
	time.Sleep(300 * time.Millisecond)
	return nil
}

func trayMenue(url string, beende func()) {
	if runtime.GOOS == "windows" {
		systray.SetIcon(symbol.ICO(32))
	} else {
		systray.SetIcon(symbol.PNG(64))
	}
	systray.SetTitle("Marktscout")
	systray.SetTooltip("Marktscout")
	oeffnen := systray.AddMenuItem("Öffnen", "Oberfläche im Browser öffnen")
	systray.AddSeparator()
	ende := systray.AddMenuItem("Beenden", "Marktscout beenden")
	go func() {
		for {
			select {
			case <-oeffnen.ClickedCh:
				oeffneBrowser(url)
			case <-ende.ClickedCh:
				beende()
				return
			}
		}
	}()
}
