package icon

import (
	"bytes"
	"encoding/binary"
	"image/png"
	"testing"
)

func TestAppIconShapeAndColours(t *testing.T) {
	img := App(128)
	// Corners are outside the squircle: fully transparent.
	for _, p := range [][2]int{{0, 0}, {127, 0}, {0, 127}, {127, 127}} {
		if c := img.NRGBAAt(p[0], p[1]); c.A != 0 {
			t.Errorf("corner %v should be transparent, got %v", p, c)
		}
	}
	// Centre of the mic body is white; a point on the slash is red.
	if c := img.NRGBAAt(64, 48); c.R < 250 || c.G < 250 || c.B < 250 || c.A != 255 {
		t.Errorf("mic body pixel = %v, want white", c)
	}
	if c := img.NRGBAAt(48, 48); c != appSlash {
		t.Errorf("slash pixel = %v, want %v", c, appSlash)
	}
	// Background is dark and opaque.
	if c := img.NRGBAAt(20, 100); c.A != 255 || c.R > 0x40 {
		t.Errorf("background pixel = %v", c)
	}
}

func TestICNSContainer(t *testing.T) {
	data := ICNS(App)
	if string(data[:4]) != "icns" || binary.BigEndian.Uint32(data[4:]) != uint32(len(data)) {
		t.Fatalf("bad header % x (len %d)", data[:8], len(data))
	}
	seen := map[string]int{}
	for off := 8; off < len(data); {
		typ := string(data[off : off+4])
		n := int(binary.BigEndian.Uint32(data[off+4:]))
		img, err := png.Decode(bytes.NewReader(data[off+8 : off+n]))
		if err != nil {
			t.Fatalf("entry %s: %v", typ, err)
		}
		seen[typ] = img.Bounds().Dx()
		off += n
	}
	want := map[string]int{"ic07": 128, "ic08": 256, "ic09": 512, "ic10": 1024, "ic11": 32, "ic12": 64, "ic13": 256, "ic14": 512}
	for typ, size := range want {
		if seen[typ] != size {
			t.Errorf("%s: got %d, want %d", typ, seen[typ], size)
		}
	}
}

func TestICOSetOffsetsAreConsistent(t *testing.T) {
	data := ICOSet(App(16), App(32), App(48))
	le := binary.LittleEndian
	if le.Uint16(data[4:]) != 3 {
		t.Fatalf("entry count %d", le.Uint16(data[4:]))
	}
	end := 0
	for i := 0; i < 3; i++ {
		e := 6 + 16*i
		w := int(data[e])
		size, off := int(le.Uint32(data[e+8:])), int(le.Uint32(data[e+12:]))
		if i == 0 && off != 6+16*3 {
			t.Fatalf("first offset %d", off)
		}
		if end != 0 && off != end {
			t.Fatalf("entry %d offset %d, previous ended at %d", i, off, end)
		}
		if le.Uint32(data[off:]) != 40 || int(le.Uint32(data[off+4:])) != w {
			t.Fatalf("entry %d: bad bitmap header", i)
		}
		end = off + size
	}
	if end != len(data) {
		t.Fatalf("last entry ends at %d, file is %d", end, len(data))
	}
}
