package art

// A Cell is one pixel of a shape and its shade, 0 in shadow to 1 lit.
type Cell struct {
	I, J  int
	Shade float64
}

type cellKey struct{ i, j int }

// A Cover is the pixels a shape drew and their shades, in drawing order, for detail that follows the shape: blossoms
// on a pad, flecks on bark. Its order is fixed, so detail drawn over it is the same on every call.
type Cover struct {
	cells []Cell
	index map[cellKey]int // position in cells; built on the first lookup
}

// NewCover returns an empty cover, for a variety to collect cells of its own.
func NewCover() *Cover {
	return &Cover{}
}

// push appends a cell the cover does not hold yet.
func (cv *Cover) push(i, j int, shade float64) {
	if cv.index != nil {
		cv.index[cellKey{i, j}] = len(cv.cells)
	}
	cv.cells = append(cv.cells, Cell{i, j, shade})
}

func (cv *Cover) find(i, j int) (int, bool) {
	if cv.index == nil {
		cv.index = make(map[cellKey]int, len(cv.cells))
		for k, cell := range cv.cells {
			cv.index[cellKey{cell.I, cell.J}] = k
		}
	}
	k, ok := cv.index[cellKey{i, j}]
	return k, ok
}

// Cells returns the cover's cells in drawing order. The slice belongs to the cover; do not modify it.
func (cv *Cover) Cells() []Cell {
	return cv.cells
}

// Len returns how many pixels the cover holds.
func (cv *Cover) Len() int {
	return len(cv.cells)
}

// Has reports whether the cover holds pixel (i, j).
func (cv *Cover) Has(i, j int) bool {
	_, ok := cv.find(i, j)
	return ok
}

// Shade returns the shade of pixel (i, j) and whether the cover holds it.
func (cv *Cover) Shade(i, j int) (float64, bool) {
	k, ok := cv.find(i, j)
	if !ok {
		return 0, false
	}
	return cv.cells[k].Shade, true
}

// Set gives pixel (i, j) a shade: a pixel the cover holds keeps its place, and a new one goes last.
func (cv *Cover) Set(i, j int, shade float64) {
	if k, ok := cv.find(i, j); ok {
		cv.cells[k].Shade = shade
		return
	}
	cv.push(i, j, shade)
}

// Merge sets each of other's cells in this cover, in other's order.
func (cv *Cover) Merge(other *Cover) {
	for _, cell := range other.cells {
		cv.Set(cell.I, cell.J, cell.Shade)
	}
}

// Filter returns a new cover of the cells keep reports true for, in the same order.
func (cv *Cover) Filter(keep func(Cell) bool) *Cover {
	out := NewCover()
	for _, cell := range cv.cells {
		if keep(cell) {
			out.push(cell.I, cell.J, cell.Shade)
		}
	}
	return out
}
