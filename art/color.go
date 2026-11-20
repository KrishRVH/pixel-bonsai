package art

import "math"

// A Color is an opaque 24-bit RGB color, written 0xRRGGBB.
type Color uint32

// Tones are colors ordered from shadow to lit.
type Tones []Color

// RGB returns the color made of three channels.
func RGB(r, g, b uint8) Color {
	return Color(r)<<16 | Color(g)<<8 | Color(b)
}

// RGB returns the color's red, green, and blue channels.
func (c Color) RGB() (r, g, b uint8) {
	return uint8(c >> 16 & 0xff), uint8(c >> 8 & 0xff), uint8(c & 0xff)
}

// Hex returns the color as six lowercase hex digits, "rrggbb".
func (c Color) Hex() string {
	const digits = "0123456789abcdef"
	var out [6]byte
	for k := range out {
		out[k] = digits[(c>>(20-4*k))&0xf]
	}
	return string(out[:])
}

// String returns [Color.Hex], so colors read as hex in test failures and logs.
func (c Color) String() string {
	return c.Hex()
}

// Mix returns the color a fraction t of the way from a to b.
func Mix(a, b Color, t float64) Color {
	ar, ag, ab := a.RGB()
	br, bg, bb := b.RGB()
	return RGB(mixChannel(ar, br, t), mixChannel(ag, bg, t), mixChannel(ab, bb, t))
}

func mixChannel(p, q uint8, t float64) uint8 {
	v := math.RoundToEven(float64(p) + (float64(q)-float64(p))*t)
	return uint8(max(0, min(255, v)))
}

// Shade returns the tone for shade s in [0, 1], 0 shadow and 1 lit. The noise value n in [0, 1) moves s by up to
// jitter/2 either way, so the edge between two tones breaks up into dither.
func (t Tones) Shade(s, jitter, n float64) Color {
	k := int((s + (n-0.5)*jitter) * float64(len(t)))
	return t[max(0, min(len(t)-1, k))]
}
