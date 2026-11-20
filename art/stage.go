package art

// A Stage is a named step on the ladder of growth. Features key off stage names through [Since], [From], [During],
// [Steps], and [Lerp], so the ladder and the drawings cannot drift apart.
type Stage int

// The twenty stages, from a seed to a venerable tree.
const (
	Seed Stage = iota
	Sprout
	Seedling
	Sapling
	Whip
	YoungTree
	FirstBranches
	FirstWiring
	ThickeningTrunk
	FirstStyling
	Shaped
	LayeredPads
	Ramified
	Refined
	Established
	Mature
	ShowTree
	Old
	Ancient
	Venerable
)

// Stages is the number of stages on the ladder.
const Stages = int(Venerable) + 1

// Plus is how far past a stage's start a frame of it is judged, where its new feature has arrived.
const Plus = 0.005

// stepSpan is the age over which [Steps] moves to a stage's value. It stays below [Plus], so a step shows in the
// frame judged at its stage.
const stepSpan = 0.004

var ladder = [Stages]struct {
	start float64
	name  string
}{
	{0.000, "seed"},
	{0.015, "sprout"},
	{0.040, "seedling"},
	{0.070, "sapling"},
	{0.100, "whip"},
	{0.140, "young tree"},
	{0.190, "first branches"},
	{0.240, "first wiring"},
	{0.290, "thickening trunk"},
	{0.340, "first styling"},
	{0.390, "shaped"},
	{0.450, "layered pads"},
	{0.510, "ramified"},
	{0.570, "refined"},
	{0.630, "established"},
	{0.690, "mature"},
	{0.750, "show tree"},
	{0.810, "old"},
	{0.870, "ancient"},
	{0.940, "venerable"},
}

// Start returns the age at which the stage begins.
func (s Stage) Start() float64 {
	return ladder[s].start
}

// String returns the stage's name as the caption shows it, such as "young tree".
func (s Stage) String() string {
	return ladder[s].name
}

// StageAt returns the stage an age is in.
func StageAt(g float64) Stage {
	s := Seed
	for k := 1; k < Stages; k++ {
		if g >= ladder[k].start {
			s = Stage(k)
		}
	}
	return s
}

// Ramp returns 0 before start, rising linearly to 1 at start + span.
func Ramp(g, start, span float64) float64 {
	return Clamp((g - start) / span)
}

// Since returns 0 before stage s starts, rising to 1 over span of age.
func Since(g float64, s Stage, span float64) float64 {
	return Ramp(g, s.Start(), span)
}

// From reports whether g has reached stage s.
func From(g float64, s Stage) bool {
	return g >= s.Start()
}

// During reports whether g is from the start of stage first until the start of stage last, for a feature that is only
// there a while, such as wire or the nursery pot.
func During(g float64, first, last Stage) bool {
	return g >= first.Start() && g < last.Start()
}

// A Step is a value a feature takes when a stage starts.
type Step struct {
	Stage Stage
	Value float64
}

// At returns the step to value at stage s, for writing step tables.
func At(s Stage, value float64) Step {
	return Step{Stage: s, Value: value}
}

// Steps returns a value that is 0 until the table's first stage and moves to each step's value in turn as its stage
// starts.
func Steps(g float64, table []Step) float64 {
	v := 0.0
	for _, step := range table {
		v += (step.Value - v) * Since(g, step.Stage, stepSpan)
	}
	return v
}

// Lerp returns a value that moves linearly through the table, reaching each step's value at its stage's start. It
// holds the first value before the first stage and the last after the last.
func Lerp(g float64, table []Step) float64 {
	if g <= table[0].Stage.Start() {
		return table[0].Value
	}
	for k := 1; k < len(table); k++ {
		a, b := table[k-1], table[k]
		if g <= b.Stage.Start() {
			return a.Value + (b.Value-a.Value)*(g-a.Stage.Start())/(b.Stage.Start()-a.Stage.Start())
		}
	}
	return table[len(table)-1].Value
}
