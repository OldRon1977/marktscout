package abruf

import (
	"context"
	"strings"
	"testing"
	"time"
)

// Live-Test: zwei Abrufe desselben Hosts halten den Mindestabstand ein.
func TestDrosselungJeHost(t *testing.T) {
	c := Neu(1500*time.Millisecond, 1600*time.Millisecond, nil)
	ctx := context.Background()
	start := time.Now()
	for i := 0; i < 2; i++ {
		a, err := c.Hole(ctx, "https://www.marktplaats.nl/robots.txt", true)
		if err != nil {
			t.Fatal(err)
		}
		if a.Status != 200 {
			t.Fatalf("HTTP %d", a.Status)
		}
	}
	if d := time.Since(start); d < 1500*time.Millisecond {
		t.Fatalf("zwei Abrufe in %v, Mindestabstand 1,5 s", d)
	}
	// Anderer Host wartet nicht auf den ersten.
	start = time.Now()
	if _, err := c.Hole(ctx, "https://www.kleinanzeigen.de/robots.txt", true); err != nil {
		t.Fatal(err)
	}
	if d := time.Since(start); d > 1400*time.Millisecond {
		t.Errorf("anderer Host wartete %v", d)
	}
}

// Live-Test: Umleitung verfolgen oder als Antwort zurückgeben.
func TestUmleitung(t *testing.T) {
	c := Neu(0, 0, nil)
	ctx := context.Background()
	// Gelöschte Anzeige leitet auf eine Kategorieseite um.
	a, err := c.Hole(ctx, "https://www.kleinanzeigen.de/s-anzeige/2000000000", false)
	if err != nil {
		t.Fatal(err)
	}
	if a.Status < 300 || a.Status >= 400 || a.Ziel == "" {
		t.Fatalf("ohne Verfolgen: HTTP %d, Ziel %q", a.Status, a.Ziel)
	}
	a, err = c.Hole(ctx, "https://www.kleinanzeigen.de/s-anzeige/2000000000", true)
	if err != nil {
		t.Fatal(err)
	}
	if a.Status != 200 || strings.Contains(a.URL, "2000000000") {
		t.Fatalf("mit Verfolgen: HTTP %d, URL %s", a.Status, a.URL)
	}
}

func TestAbbruch(t *testing.T) {
	c := Neu(10*time.Second, 10*time.Second, nil)
	ctx, abbruch := context.WithCancel(context.Background())
	if _, err := c.Hole(ctx, "https://www.marktplaats.nl/robots.txt", true); err != nil {
		t.Fatal(err)
	}
	go func() { time.Sleep(200 * time.Millisecond); abbruch() }()
	start := time.Now()
	if _, err := c.Hole(ctx, "https://www.marktplaats.nl/robots.txt", true); err == nil {
		t.Fatal("abgebrochener Abruf ohne Fehler")
	}
	if d := time.Since(start); d > 2*time.Second {
		t.Errorf("Abbruch wirkte erst nach %v", d)
	}
}
