// Package abruf stellt einen gedrosselten HTTP-Client für die Seitenanbindungen bereit.
package abruf

import (
	"context"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"math/rand/v2"
	"net/http"
	"net/url"
	"sync"
	"time"
)

// UserAgent wird bei allen Abrufen gesendet.
const UserAgent = "Mozilla/5.0 (X11; Linux x86_64; rv:140.0) Gecko/20100101 Firefox/140.0"

const maxGroesse = 8 << 20

// Antwort eines Abrufs.
type Antwort struct {
	Status int
	URL    string // endgültige URL nach Umleitungen
	Ziel   string // Location-Header, wenn nicht umgeleitet wurde
	Body   []byte
}

// Client ruft Seiten ab und hält je Host einen Mindestabstand zwischen zwei Abrufen ein.
type Client struct {
	http       *http.Client
	minAbstand time.Duration
	maxAbstand time.Duration
	log        *slog.Logger

	mu     sync.Mutex
	naechs map[string]time.Time // frühester Zeitpunkt des nächsten Abrufs je Host
}

// Neu erzeugt einen Client. Der Abstand je Host liegt zufällig zwischen min und max.
func Neu(min, max time.Duration, log *slog.Logger) *Client {
	if log == nil {
		log = slog.New(slog.DiscardHandler)
	}
	return &Client{
		http:       &http.Client{Timeout: 30 * time.Second},
		minAbstand: min,
		maxAbstand: max,
		log:        log,
		naechs:     map[string]time.Time{},
	}
}

// Standard erzeugt einen Client mit 2–4 s Abstand je Host.
func Standard(log *slog.Logger) *Client { return Neu(2*time.Second, 4*time.Second, log) }

func (c *Client) warte(ctx context.Context, host string) error {
	c.mu.Lock()
	jetzt := time.Now()
	start := c.naechs[host]
	if start.Before(jetzt) {
		start = jetzt
	}
	abstand := c.minAbstand
	if c.maxAbstand > c.minAbstand {
		abstand += time.Duration(rand.Int64N(int64(c.maxAbstand - c.minAbstand)))
	}
	c.naechs[host] = start.Add(abstand)
	c.mu.Unlock()

	if d := time.Until(start); d > 0 {
		t := time.NewTimer(d)
		defer t.Stop()
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-t.C:
		}
	}
	return nil
}

// Hole ruft adresse per GET ab. Bei folgeUmleitung=false wird eine Umleitung nicht verfolgt,
// sondern als Antwort mit Status 3xx und Ziel zurückgegeben.
// Netzwerkfehler, 429 und 5xx werden bis zu zweimal wiederholt.
func (c *Client) Hole(ctx context.Context, adresse string, folgeUmleitung bool) (*Antwort, error) {
	u, err := url.Parse(adresse)
	if err != nil {
		return nil, err
	}
	var letzterFehler error
	for versuch := 0; versuch < 3; versuch++ {
		if versuch > 0 {
			pause := time.Duration(versuch*5) * time.Second
			select {
			case <-ctx.Done():
				return nil, ctx.Err()
			case <-time.After(pause):
			}
		}
		if err := c.warte(ctx, u.Host); err != nil {
			return nil, err
		}
		a, err := c.einmal(ctx, adresse, folgeUmleitung)
		if err == nil && a.Status != http.StatusTooManyRequests && a.Status < 500 {
			c.log.Debug("abruf", "url", adresse, "status", a.Status, "bytes", len(a.Body))
			return a, nil
		}
		if err == nil {
			err = fmt.Errorf("HTTP %d", a.Status)
		}
		if errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
			return nil, err
		}
		letzterFehler = err
		c.log.Warn("abruf fehlgeschlagen", "url", adresse, "versuch", versuch+1, "fehler", err)
	}
	return nil, fmt.Errorf("abruf %s: %w", adresse, letzterFehler)
}

func (c *Client) einmal(ctx context.Context, adresse string, folgeUmleitung bool) (*Antwort, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, adresse, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("User-Agent", UserAgent)
	req.Header.Set("Accept", "text/html,application/json;q=0.9,*/*;q=0.8")
	req.Header.Set("Accept-Language", "de-DE,de;q=0.9,nl;q=0.8,en;q=0.7")

	hc := c.http
	if !folgeUmleitung {
		kopie := *c.http
		kopie.CheckRedirect = func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse }
		hc = &kopie
	}
	resp, err := hc.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(io.LimitReader(resp.Body, maxGroesse))
	if err != nil {
		return nil, err
	}
	return &Antwort{
		Status: resp.StatusCode,
		URL:    resp.Request.URL.String(),
		Ziel:   resp.Header.Get("Location"),
		Body:   body,
	}, nil
}
