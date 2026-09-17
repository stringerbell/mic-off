package icon

import (
	"bytes"
	"encoding/binary"
	"image"
	"image/color"
	"math"
)

// App renders the application icon (Finder / Dock / Explorer) at the given
// pixel size: a dark rounded square, a white microphone and a red slash.
// Geometry is defined on a 1024-unit canvas and supersampled.
func App(size int) *image.NRGBA {
	img := image.NewNRGBA(image.Rect(0, 0, size, size))
	ss := 4
	if size < 128 {
		ss = 8 // small icons need more samples per pixel to stay smooth
	}
	scale := 1024.0 / float64(size)
	inv := 1.0 / float64(ss*ss)
	for py := 0; py < size; py++ {
		for px := 0; px < size; px++ {
			var r, g, b, a float64
			for sy := 0; sy < ss; sy++ {
				for sx := 0; sx < ss; sx++ {
					x := (float64(px) + (float64(sx)+0.5)/float64(ss)) * scale
					y := (float64(py) + (float64(sy)+0.5)/float64(ss)) * scale
					c := shadeApp(x, y)
					if c.A == 0 {
						continue
					}
					fa := float64(c.A) / 255
					r += float64(c.R) * fa
					g += float64(c.G) * fa
					b += float64(c.B) * fa
					a += fa
				}
			}
			if a == 0 {
				continue
			}
			img.SetNRGBA(px, py, color.NRGBA{
				R: uint8(math.Round(r / a)),
				G: uint8(math.Round(g / a)),
				B: uint8(math.Round(b / a)),
				A: uint8(math.Round(a * inv * 255)),
			})
		}
	}
	return img
}

var (
	appBGTop    = color.NRGBA{0x3d, 0x43, 0x55, 255}
	appBGBottom = color.NRGBA{0x16, 0x19, 0x21, 255}
	appSlash    = color.NRGBA{0xff, 0x3b, 0x30, 255}
)

// shadeApp returns the colour of one sample on the 1024-unit canvas.
func shadeApp(x, y float64) color.NRGBA {
	const cx, cy, half, n = 512.0, 512.0, 416.0, 5.0
	// macOS-style squircle (superellipse) occupying the central 832 units.
	if math.Pow(math.Abs((x-cx)/half), n)+math.Pow(math.Abs((y-cy)/half), n) > 1 {
		return color.NRGBA{}
	}
	t := clamp((y-(cy-half))/(2*half), 0, 1)
	bg := lerp(appBGTop, appBGBottom, t)

	slash := segDist(x, y, 292, 292, 732, 732)
	if slash <= 36 {
		return appSlash
	}
	mic := math.Min(
		math.Min(segDist(x, y, cx, 300, cx, 520)-110, segDist(x, y, cx, 710, cx, 790)-22),
		segDist(x, y, cx-120, 800, cx+120, 800)-22,
	)
	if y >= 520 {
		mic = math.Min(mic, math.Abs(math.Hypot(x-cx, y-520)-190)-22)
	}
	if mic <= 0 {
		if slash <= 58 {
			return bg // gap so the slash reads as separate from the mic
		}
		return White
	}
	return bg
}

func lerp(a, b color.NRGBA, t float64) color.NRGBA {
	f := func(x, y uint8) uint8 { return uint8(math.Round(float64(x) + (float64(y)-float64(x))*t)) }
	return color.NRGBA{f(a.R, b.R), f(a.G, b.G), f(a.B, b.B), 255}
}

// ICNS builds an Apple icon container holding PNG images at every size
// modern macOS asks for. render is called once per distinct pixel size.
func ICNS(render func(size int) *image.NRGBA) []byte {
	entries := []struct {
		typ  string
		size int
	}{
		{"ic11", 32}, {"ic12", 64}, {"ic07", 128}, {"ic13", 256},
		{"ic08", 256}, {"ic14", 512}, {"ic09", 512}, {"ic10", 1024},
	}
	cache := map[int][]byte{}
	var body bytes.Buffer
	for _, e := range entries {
		data, ok := cache[e.size]
		if !ok {
			data = PNG(render(e.size))
			cache[e.size] = data
		}
		body.WriteString(e.typ)
		binary.Write(&body, binary.BigEndian, uint32(8+len(data)))
		body.Write(data)
	}
	var out bytes.Buffer
	out.WriteString("icns")
	binary.Write(&out, binary.BigEndian, uint32(8+body.Len()))
	out.Write(body.Bytes())
	return out.Bytes()
}

// AppICO builds a multi-resolution Windows icon.
func AppICO() []byte {
	return ICOSet(App(16), App(32), App(48), App(256))
}
