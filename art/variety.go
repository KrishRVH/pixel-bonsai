package art

// A Variety is a kind of bonsai: what the caption calls it, the colors that mark it, and how it draws.
type Variety struct {
	ID   string // a stable slug for flags and saved state, such as "black-pine"
	Name string // the name the caption shows, such as "black pine"
	Icon string // an emoji the caption shows before the name

	// Signature holds colors that mark the variety: every frame from YoungTree on holds one at scale 0.6, and no
	// other variety draws one.
	Signature []Color

	// Draw paints the tree at age g, 0 for a seed to 1 for a venerable tree. It must give the same pixels for the
	// same age and scale on every call.
	Draw func(c *Canvas, g float64)
}

// Render draws the variety at age g on a canvas of [Size](scale) and returns the frame.
func (v Variety) Render(g, scale float64) *Frame {
	c := NewCanvas(scale)
	v.Draw(c, g)
	return c.Frame()
}
