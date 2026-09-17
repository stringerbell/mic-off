// Package icon draws the tray icon at runtime so no image assets need to be
// shipped. It produces PNG (macOS) and ICO (Windows) bytes.
package icon

import (
	"bytes"
	"encoding/binary"
	"image"
	"image/color"
	"image/png"
	"math"
)

const Size = 32

// Colors used for the non-template variants.
var (
	Black = color.NRGBA{0, 0, 0, 255}
	White = color.NRGBA{255, 255, 255, 255}
	Blue  = color.NRGBA{0x3b, 0x82, 0xf6, 255}
	Red   = color.NRGBA{0xe0, 0x3a, 0x2f, 255}
)

// Draw renders a microphone. When muted is true a diagonal slash is added.
func Draw(muted bool, fg color.NRGBA) *image.NRGBA {
	img := image.NewNRGBA(image.Rect(0, 0, Size, Size))
	const s = float64(Size) / 32.0 // shapes are laid out on a 32-unit grid
	cx := 16.0 * s
	shapes := []func(x, y float64) float64{
		// capsule body
		func(x, y float64) float64 { return segDist(x, y, cx, 8*s, cx, 15*s) - 4.5*s },
		// U-shaped cradle: lower half of a ring
		func(x, y float64) float64 {
			if y < 15*s {
				return 1e9
			}
			return math.Abs(math.Hypot(x-cx, y-15*s)-7.5*s) - 1.1*s
		},
		// stem and base
		func(x, y float64) float64 { return segDist(x, y, cx, 22*s, cx, 26*s) - 1.1*s },
		func(x, y float64) float64 { return segDist(x, y, cx-5*s, 27*s, cx+5*s, 27*s) - 1.1*s },
	}
	if muted {
		shapes = append(shapes, func(x, y float64) float64 {
			return segDist(x, y, 6*s, 6*s, 26*s, 26*s) - 1.6*s
		})
	}
	for py := 0; py < Size; py++ {
		for px := 0; px < Size; px++ {
			x, y := float64(px)+0.5, float64(py)+0.5
			d := 1e9
			for _, f := range shapes {
				d = math.Min(d, f(x, y))
			}
			cov := clamp(0.5-d, 0, 1) // 1px anti-aliased edge
			if cov <= 0 {
				continue
			}
			c := fg
			c.A = uint8(math.Round(float64(fg.A) * cov))
			img.SetNRGBA(px, py, c)
		}
	}
	return img
}

func segDist(px, py, ax, ay, bx, by float64) float64 {
	vx, vy := bx-ax, by-ay
	wx, wy := px-ax, py-ay
	t := clamp((wx*vx+wy*vy)/(vx*vx+vy*vy), 0, 1)
	return math.Hypot(px-(ax+t*vx), py-(ay+t*vy))
}

func clamp(v, lo, hi float64) float64 { return math.Max(lo, math.Min(hi, v)) }

// PNG encodes an image as PNG bytes.
func PNG(img image.Image) []byte {
	var buf bytes.Buffer
	if err := png.Encode(&buf, img); err != nil {
		panic(err) // encoding an in-memory image cannot fail
	}
	return buf.Bytes()
}

// ICO wraps an image in a single-entry 32-bit ICO container, the format the
// Windows tray requires.
func ICO(img *image.NRGBA) []byte { return ICOSet(img) }

// ICOSet packs several sizes of the same icon into one ICO file, each as a
// 32-bit BGRA bitmap with an empty AND mask (alpha handles transparency).
func ICOSet(imgs ...*image.NRGBA) []byte {
	var buf bytes.Buffer
	le := binary.LittleEndian
	binary.Write(&buf, le, uint16(0))
	binary.Write(&buf, le, uint16(1))
	binary.Write(&buf, le, uint16(len(imgs)))
	offset := 6 + 16*len(imgs)
	var images bytes.Buffer
	for _, img := range imgs {
		w, h := img.Rect.Dx(), img.Rect.Dy()
		maskRow := ((w + 31) / 32) * 4
		xorSize := w * h * 4
		andSize := maskRow * h
		// ICONDIRENTRY
		buf.WriteByte(byte(w % 256))
		buf.WriteByte(byte(h % 256))
		buf.WriteByte(0)
		buf.WriteByte(0)
		binary.Write(&buf, le, uint16(1))
		binary.Write(&buf, le, uint16(32))
		binary.Write(&buf, le, uint32(40+xorSize+andSize))
		binary.Write(&buf, le, uint32(offset+images.Len()))
		// BITMAPINFOHEADER
		binary.Write(&images, le, uint32(40))
		binary.Write(&images, le, int32(w))
		binary.Write(&images, le, int32(h*2))
		binary.Write(&images, le, uint16(1))
		binary.Write(&images, le, uint16(32))
		binary.Write(&images, le, uint32(0))
		binary.Write(&images, le, uint32(xorSize+andSize))
		binary.Write(&images, le, [4]int32{0, 0, 0, 0})
		for y := h - 1; y >= 0; y-- {
			for x := 0; x < w; x++ {
				c := img.NRGBAAt(x, y)
				images.Write([]byte{c.B, c.G, c.R, c.A})
			}
		}
		images.Write(make([]byte, andSize))
	}
	buf.Write(images.Bytes())
	return buf.Bytes()
}

// Set is the pair of icons an app needs for one platform.
type Set struct {
	Live, Muted []byte
	// LiveIsTemplate is true when Live should be shown as a macOS template
	// image (monochrome, adapts to the menu bar theme).
	LiveIsTemplate bool
}

// ForPlatform returns ready-to-use icon bytes for "darwin" or "windows".
func ForPlatform(platform string) Set {
	if platform == "darwin" {
		return Set{
			Live:           PNG(Draw(false, Black)),
			Muted:          PNG(Draw(true, Red)),
			LiveIsTemplate: true,
		}
	}
	return Set{
		Live:  ICO(Draw(false, Blue)),
		Muted: ICO(Draw(true, Red)),
	}
}
