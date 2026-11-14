package symbol

import (
	"bytes"
	"encoding/binary"
	"image/png"
	"testing"
)

func TestPNGUndICO(t *testing.T) {
	img, err := png.Decode(bytes.NewReader(PNG(32)))
	if err != nil {
		t.Fatal(err)
	}
	if img.Bounds().Dx() != 32 {
		t.Fatalf("Breite %d", img.Bounds().Dx())
	}
	// Rand des Kreises ist gefärbt, Ecke oben rechts transparent
	_, _, _, a := img.At(31, 0).RGBA()
	if a != 0 {
		t.Error("Ecke nicht transparent")
	}
	gefaerbt := 0
	for y := 0; y < 32; y++ {
		for x := 0; x < 32; x++ {
			r, g, b, a := img.At(x, y).RGBA()
			if a > 0 && b > r && b > g {
				gefaerbt++
			}
		}
	}
	if gefaerbt < 60 {
		t.Errorf("nur %d blaue Pixel", gefaerbt)
	}

	ico := ICO(32)
	if binary.LittleEndian.Uint16(ico[2:]) != 1 || binary.LittleEndian.Uint16(ico[4:]) != 1 || ico[6] != 32 {
		t.Fatal("ICO-Kopf ungültig")
	}
	laenge := binary.LittleEndian.Uint32(ico[14:])
	versatz := binary.LittleEndian.Uint32(ico[18:])
	if int(versatz+laenge) != len(ico) {
		t.Fatalf("ICO-Verzeichnis: %d+%d ≠ %d", versatz, laenge, len(ico))
	}
	if _, err := png.Decode(bytes.NewReader(ico[versatz:])); err != nil {
		t.Fatalf("eingebettetes PNG: %v", err)
	}
}
