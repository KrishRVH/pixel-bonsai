package art

import "math"

// Noise returns a fixed pseudo-random value in [0, 1) for a pixel or a position and a salt, so texture and blossoms
// stay put as a tree grows. It is a 32-bit integer hash and gives the same value on every platform.
func Noise(x, y, salt int) float64 {
	h := uint32(int64(x)*374761393 + int64(y)*668265263 + int64(salt)*2246822519) // #nosec G115 -- The hash wraps on purpose.
	h = (h ^ h>>13) * 1274126177
	return float64((h^h>>16)&0xffff) / 65536
}

// Wobble returns a smooth fixed wander in [-1, 1] along t, for a trunk's irregularity or an uneven edge.
func Wobble(t float64, salt int) float64 {
	a, b := Noise(salt, 1, 0)*2*math.Pi, Noise(salt, 2, 0)*2*math.Pi
	return 0.6*math.Sin(t*1.7+a) + 0.4*math.Sin(t*3.9+b)
}
