// Package scripts enthält nur Tests für die Skripte in diesem Ordner.
package scripts

import (
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

func TestMain(m *testing.M) {
	if runtime.GOOS == "windows" {
		// Skript und Hook brauchen ein POSIX-sh; die Prüfung läuft in der CI unter Linux.
		os.Exit(0)
	}
	os.Exit(m.Run())
}

// Testwerte werden zusammengesetzt, damit sie nicht wörtlich im Repository stehen
// (sonst schlüge die Prüfung --alle auf diese Datei an).
var (
	adresse  = "max.mustermann" + "@" + "web.de"
	beispiel = "info" + "@" + "example.com"
	noreply  = "1+test" + "@" + "users.noreply.github.com"
)

func git(t *testing.T, verz string, args ...string) {
	t.Helper()
	c := exec.Command("git", args...)
	c.Dir = verz
	if out, err := c.CombinedOutput(); err != nil {
		t.Fatalf("git %v: %v\n%s", args, err, out)
	}
}

// repo legt ein echtes git-Repository mit dem Skript und dem Hook an.
func repo(t *testing.T) string {
	t.Helper()
	verz := t.TempDir()
	git(t, verz, "init", "-q")
	git(t, verz, "config", "user.name", "Test")
	git(t, verz, "config", "user.email", noreply)
	for _, f := range []string{"scripts/datenschutz.sh", ".githooks/pre-commit", ".githooks/commit-msg"} {
		inhalt, err := os.ReadFile(filepath.Join("..", f))
		if err != nil {
			t.Fatal(err)
		}
		os.MkdirAll(filepath.Join(verz, filepath.Dir(f)), 0o755)
		if err := os.WriteFile(filepath.Join(verz, f), inhalt, 0o755); err != nil {
			t.Fatal(err)
		}
	}
	git(t, verz, "config", "core.hooksPath", ".githooks")
	git(t, verz, "add", ".")
	git(t, verz, "commit", "-q", "-m", "start")
	return verz
}

// commit legt eine Datei an und versucht den Commit über den echten Hook.
func commit(t *testing.T, verz, datei, inhalt string) (bool, string) {
	t.Helper()
	p := filepath.Join(verz, datei)
	os.MkdirAll(filepath.Dir(p), 0o755)
	if err := os.WriteFile(p, []byte(inhalt), 0o644); err != nil {
		t.Fatal(err)
	}
	git(t, verz, "add", "-f", datei)
	c := exec.Command("git", "commit", "-q", "-m", "test")
	c.Dir = verz
	out, err := c.CombinedOutput()
	if err != nil {
		git(t, verz, "rm", "-q", "--cached", datei)
		os.Remove(p)
	}
	return err == nil, string(out)
}

func TestHookSperrt(t *testing.T) {
	verz := repo(t)
	for _, f := range []struct{ datei, inhalt, art string }{
		{"notiz.md", "Kontakt: " + adresse + "\n", "E-Mail-Adresse"},
		{"notiz.md", "Beispiel " + beispiel + " und echt " + adresse + "\n", "E-Mail-Adresse"},
		{"config.yml", "passwort: " + "Geheim123\n", "Passwort"},
		{"login.json", `{"password": "` + "hunter22" + `"}` + "\n", "Passwort"},
		{"abruf.txt", "Cookie: " + "sessionid=abc\n", "Cookie"},
		{"a.go", `token := "` + strings.Repeat("a1B2", 6) + `"` + "\n", "Sitzungs-ID oder Token"},
		{"k.txt", "-----BEGIN " + "RSA PRIVATE KEY-----\n", "Privater Schlüssel"},
		{"k.txt", "AIza" + strings.Repeat("x", 35) + "\n", "API-Schlüssel"},
		{"k.txt", "Authorization: Bearer " + strings.Repeat("z", 24) + "\n", "Bearer-Token"},
		{"browser-profil/Default/Preferences", "{}\n", "gesperrte Datei"},
		{"auth/storage-state.json", "{}\n", "gesperrte Datei"},
		{".env", "X=1\n", "gesperrte Datei"},
		{"daten/marktscout.db", "x\n", "gesperrte Datei"},
	} {
		ok, out := commit(t, verz, f.datei, f.inhalt)
		if ok {
			t.Errorf("%s (%s): Commit nicht gesperrt", f.datei, f.art)
			continue
		}
		if !strings.Contains(out, f.art) {
			t.Errorf("%s: Meldung ohne %q:\n%s", f.datei, f.art, out)
		}
		// Der Wert selbst wird nie ausgegeben.
		for _, w := range []string{adresse, "Geheim123", "hunter22"} {
			if strings.Contains(out, w) {
				t.Errorf("%s: Meldung enthält den Wert %q", f.datei, w)
			}
		}
	}
}

func TestHookLaesstDurch(t *testing.T) {
	verz := repo(t)
	for _, f := range []struct{ datei, inhalt string }{
		{"README.md", "Kontakt: " + beispiel + ", Testadresse a" + "@" + "b.invalid\n"},
		{"doku.md", "Das Passwort-Feld bleibt leer.\nEin Login ist nicht nötig.\n"},
		{"package.json", `{"devDependencies": {"` + "@playwright/test" + `": "1.0.0"}}` + "\n"},
		{"a.go", "token := os.Getenv(\"TOKEN\")\n"},
	} {
		if ok, out := commit(t, verz, f.datei, f.inhalt); !ok {
			t.Errorf("%s gesperrt:\n%s", f.datei, out)
		}
	}
}

// commitMit versucht einen Commit mit Nachricht und Autor-Adresse über die echten Hooks.
func commitMit(t *testing.T, verz, nachricht, email string) (bool, string) {
	t.Helper()
	os.WriteFile(filepath.Join(verz, "x.txt"), []byte(strings.Repeat("x", len(nachricht))), 0o644)
	git(t, verz, "add", "x.txt")
	c := exec.Command("git", "-c", "user.email="+email, "commit", "-q", "-m", nachricht)
	c.Dir = verz
	out, err := c.CombinedOutput()
	if err != nil {
		git(t, verz, "reset", "-q", "HEAD", "x.txt")
	}
	return err == nil, string(out)
}

func TestHookAutorUndNachricht(t *testing.T) {
	verz := repo(t)
	if ok, out := commitMit(t, verz, "Echte Adresse als Autor", adresse); ok || !strings.Contains(out, "Autor-Adresse ist keine GitHub-noreply-Adresse") {
		t.Errorf("Autor mit echter Adresse: ok=%v\n%s", ok, out)
	}
	// Echte Adresse vorn, noreply-Adresse angehängt
	if ok, out := commitMit(t, verz, "Getarnte Adresse", adresse+"."+noreply); ok || !strings.Contains(out, "Autor-Adresse ist keine GitHub-noreply-Adresse") {
		t.Errorf("getarnte Autor-Adresse: ok=%v\n%s", ok, out)
	}
	if ok, out := commitMit(t, verz, "Kontakt "+adresse, noreply); ok || !strings.Contains(out, "E-Mail-Adresse in Commit-Nachricht") {
		t.Errorf("Adresse in Nachricht: ok=%v\n%s", ok, out)
	}
	if ok, out := commitMit(t, verz, "Login mit passwort="+"Geheim123", noreply); ok || !strings.Contains(out, "Passwort in Commit-Nachricht") || strings.Contains(out, "Geheim123") {
		t.Errorf("Passwort in Nachricht: ok=%v\n%s", ok, out)
	}
	if ok, out := commitMit(t, verz, "Suche verbessert\n\nCo-authored-by: Test <"+noreply+">", noreply); !ok {
		t.Errorf("zulässiger Commit gesperrt:\n%s", out)
	}
}

// Prüfung aller versionierten Dateien: das echte Repository ist sauber, ein
// eingecheckter Fund wird erkannt.
func TestAlle(t *testing.T) {
	verz := repo(t)
	c := exec.Command("sh", "scripts/datenschutz.sh", "--alle")
	c.Dir = verz
	if out, err := c.CombinedOutput(); err != nil {
		t.Fatalf("sauberes Repository gemeldet: %v\n%s", err, out)
	}
	os.WriteFile(filepath.Join(verz, "alt.md"), []byte(adresse+"\n"), 0o644)
	git(t, verz, "add", "alt.md")
	git(t, verz, "commit", "-q", "--no-verify", "-m", "an der Prüfung vorbei")
	c = exec.Command("sh", "scripts/datenschutz.sh", "--alle")
	c.Dir = verz
	out, err := c.CombinedOutput()
	if err == nil || !strings.Contains(string(out), "E-Mail-Adresse in alt.md") {
		t.Fatalf("Fund nicht erkannt: %v\n%s", err, out)
	}
	// Commit mit echter Autor-Adresse und Adresse in der Nachricht, an den Hooks vorbei
	c = exec.Command("git", "-c", "user.email="+adresse, "commit", "-q", "--no-verify", "--allow-empty", "-m", "von "+adresse)
	c.Dir = verz
	if out, err := c.CombinedOutput(); err != nil {
		t.Fatalf("%v\n%s", err, out)
	}
	c = exec.Command("sh", "scripts/datenschutz.sh", "--alle")
	c.Dir = verz
	out, _ = c.CombinedOutput()
	for _, w := range []string{"Autor-Adresse ist keine GitHub-noreply-Adresse", "E-Mail-Adresse in Commit "} {
		if !strings.Contains(string(out), w) {
			t.Errorf("Historie: %q fehlt\n%s", w, out)
		}
	}
	if strings.Contains(string(out), adresse) {
		t.Errorf("Ausgabe enthält die Adresse")
	}
}
