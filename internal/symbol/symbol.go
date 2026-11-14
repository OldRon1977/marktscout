// Package symbol erzeugt das Programmsymbol (Lupe) als PNG und ICO.
package symbol

import (
	"bytes"
	"encoding/binary"
	"image"
	"image/color"
	"image/png"
	"math"
)

var (
	farbe   = color.NRGBA{0x1f, 0x6f, 0xb2, 0xff}
	flaeche = color.NRGBA{0xff, 0xff, 0xff, 0xff}
)

// Bild zeichnet das Symbol in der Größe n×n.
func Bild(n int) *image.NRGBA {
	img := image.NewNRGBA(image.Rect(0, 0, n, n))
	f := float64(n)
	cx, cy, r := f*0.42, f*0.42, f*0.30
	staerke := f * 0.09
	for y := 0; y < n; y++ {
		for x := 0; x < n; x++ {
			px, py := float64(x)+0.5, float64(y)+0.5
			d := math.Hypot(px-cx, py-cy)
			switch {
			case d <= r-staerke/2:
				img.SetNRGBA(x, y, flaeche)
			case d <= r+staerke/2:
				img.SetNRGBA(x, y, farbe)
			}
			// Griff: Strecke von Kreisrand nach unten rechts
			ax, ay := cx+r*0.72, cy+r*0.72
			bx, by := f*0.92, f*0.92
			t := ((px-ax)*(bx-ax) + (py-ay)*(by-ay)) / ((bx-ax)*(bx-ax) + (by-ay)*(by-ay))
			if t >= 0 && t <= 1 {
				qx, qy := ax+t*(bx-ax), ay+t*(by-ay)
				if math.Hypot(px-qx, py-qy) <= staerke*0.8 {
					img.SetNRGBA(x, y, farbe)
				}
			}
		}
	}
	return img
}

// PNG liefert das Symbol als PNG.
func PNG(n int) []byte {
	var b bytes.Buffer
	png.Encode(&b, Bild(n))
	return b.Bytes()
}

// ICO liefert das Symbol als ICO-Datei mit eingebettetem PNG (ab Windows Vista lesbar).
func ICO(n int) []byte {
	p := PNG(n)
	var b bytes.Buffer
	groesse := byte(n)
	if n >= 256 {
		groesse = 0
	}
	binary.Write(&b, binary.LittleEndian, []uint16{0, 1, 1}) // reserviert, Typ Icon, 1 Bild
	b.Write([]byte{groesse, groesse, 0, 0})
	binary.Write(&b, binary.LittleEndian, uint16(1))  // Farbebenen
	binary.Write(&b, binary.LittleEndian, uint16(32)) // Bit je Pixel
	binary.Write(&b, binary.LittleEndian, uint32(len(p)))
	binary.Write(&b, binary.LittleEndian, uint32(6+16))
	b.Write(p)
	return b.Bytes()
}
