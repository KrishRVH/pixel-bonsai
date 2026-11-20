package art

import "math"

// A Point is a position in units.
type Point struct {
	X, Y float64
}

// A Path is a run of points in units, such as a trunk's or a branch's line.
type Path []Point

// Spline returns a smooth path through points (Catmull-Rom), with steps segments between each pair.
func Spline(points []Point, steps int) Path {
	padded := make([]Point, 0, len(points)+2)
	padded = append(padded, points[0])
	padded = append(padded, points...)
	padded = append(padded, points[len(points)-1])
	out := make(Path, 0, (len(points)-1)*steps+1)
	for k := 0; k+3 < len(padded); k++ {
		p0, p1, p2, p3 := padded[k], padded[k+1], padded[k+2], padded[k+3]
		for s := range steps {
			t := float64(s) / float64(steps)
			out = append(out, Point{catmullRom(p0.X, p1.X, p2.X, p3.X, t), catmullRom(p0.Y, p1.Y, p2.Y, p3.Y, t)})
		}
	}
	return append(out, points[len(points)-1])
}

func catmullRom(a, b, c, d, t float64) float64 {
	return 0.5 * (2*b + (c-a)*t + (2*a-5*b+4*c-d)*t*t + (3*b-a-3*c+d)*t*t*t)
}

// Along returns the point a share of the way along the path, from 0 at its start to 1 at its end.
func (p Path) Along(share float64) Point {
	return p[int(math.RoundToEven(share*float64(len(p)-1)))]
}
