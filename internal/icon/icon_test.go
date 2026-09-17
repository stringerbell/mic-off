package icon

import (
	"bytes"
	"encoding/binary"
	"image"
	"image/png"
	"testing"
)

func opaquePixels(img image.Image) int {
	n := 0
	for y := 0; y < Size; y++ {
		for x := 0; x < Size; x++ {
			if _, _, _, a := img.At(x, y).RGBA(); a > 0 {
				n++
			}
		}
	}
	return n
}

func TestDrawProducesVisibleShapeAndSlashAddsPixels(t *testing.T) {
	live := Draw(false, Black)
	muted := Draw(true, Red)
	nl, nm := opaquePixels(live), opaquePixels(muted)
	if nl < 100 || nl > Size*Size/2 {
		t.Fatalf("live icon has %d painted pixels; expected a mic-sized shape", nl)
	}
	if nm <= nl {
		t.Fatalf("muted icon (%d px) should paint more than live (%d px)", nm, nl)
	}
	// Symmetric mic: left/right halves of the live icon mirror each other.
	for y := 0; y < Size; y++ {
		for x := 0; x < Size/2; x++ {
			if live.NRGBAAt(x, y) != live.NRGBAAt(Size-1-x, y) {
				t.Fatalf("live icon not symmetric at (%d,%d)", x, y)
			}
		}
	}
}

func TestPNGDecodes(t *testing.T) {
	img, err := png.Decode(bytes.NewReader(PNG(Draw(true, Red))))
	if err != nil {
		t.Fatal(err)
	}
	if img.Bounds().Dx() != Size {
		t.Fatalf("width %d", img.Bounds().Dx())
	}
}

func TestICOLayout(t *testing.T) {
	data := ICO(Draw(false, Blue))
	le := binary.LittleEndian
	if le.Uint16(data[2:]) != 1 || le.Uint16(data[4:]) != 1 {
		t.Fatalf("bad ICONDIR: % x", data[:6])
	}
	if data[6] != Size || data[7] != Size {
		t.Fatalf("entry size %dx%d", data[6], data[7])
	}
	if le.Uint16(data[12:]) != 32 {
		t.Fatalf("bit depth %d", le.Uint16(data[12:]))
	}
	size, off := le.Uint32(data[14:]), le.Uint32(data[18:])
	if off != 22 || int(off+size) != len(data) {
		t.Fatalf("offset %d size %d total %d", off, size, len(data))
	}
	if le.Uint32(data[22:]) != 40 || le.Uint32(data[26:]) != Size || le.Uint32(data[30:]) != Size*2 {
		t.Fatalf("bad BITMAPINFOHEADER: % x", data[22:34])
	}
}

func TestForPlatform(t *testing.T) {
	mac := ForPlatform("darwin")
	if !mac.LiveIsTemplate || !bytes.HasPrefix(mac.Live, []byte("\x89PNG")) {
		t.Fatal("darwin icons should be template PNGs")
	}
	win := ForPlatform("windows")
	if win.LiveIsTemplate || win.Live[2] != 1 || win.Live[3] != 0 {
		t.Fatal("windows icons should be ICO")
	}
}
