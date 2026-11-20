// Package bonsai lists the varieties in switch order and maps token counts to a tree's age.
package bonsai

import (
	"math"

	"github.com/KrishRVH/pixel-bonsai/art"
	"github.com/KrishRVH/pixel-bonsai/bonsai/maple"
	"github.com/KrishRVH/pixel-bonsai/bonsai/pine"
	"github.com/KrishRVH/pixel-bonsai/bonsai/sakura"
)

// The token counts that grow a tree to its full age. DayTokens fills a day's tree: 100M tokens is a young tree, a
// median autumn 2026 day of about 0.25B thickens its trunk, a busy 1B is refined, and about 2.6B is venerable.
// MatureTokens fills a variety's lifetime tree, about two years at autumn 2026's pace of 0.4B tokens a day.
const (
	DayTokens    = 3e9
	MatureTokens = 250e9
)

// Varieties returns the varieties in the order a switch steps through them. The first is the default.
func Varieties() []art.Variety {
	return []art.Variety{sakura.Variety(), maple.Variety(), pine.Variety()}
}

// Default returns the variety a new garden starts with.
func Default() art.Variety {
	return Varieties()[0]
}

// Lookup returns the variety with an id.
func Lookup(id string) (art.Variety, bool) {
	for _, v := range Varieties() {
		if v.ID == id {
			return v, true
		}
	}
	return art.Variety{}, false
}

// IDs returns the variety ids in switch order.
func IDs() []string {
	vs := Varieties()
	ids := make([]string, len(vs))
	for k, v := range vs {
		ids[k] = v.ID
	}
	return ids
}

// A View is which of a variety's trees the pane shows.
type View string

// The two views: the tree grown from today's tokens, and the one grown from every token the variety has received.
const (
	Today    View = "today"
	Lifetime View = "lifetime"
)

// Growth returns the age of the view's tree for a token count.
func (v View) Growth(tokens int64) float64 {
	if v == Lifetime {
		return LifeGrowth(tokens)
	}
	return DayGrowth(tokens)
}

// Next returns the other view.
func (v View) Next() View {
	if v == Today {
		return Lifetime
	}
	return Today
}

// DayGrowth returns the age of a day's tree, (tokens/DayTokens)^0.47: quick early in the day and never capped short
// of the ladder's end.
func DayGrowth(tokens int64) float64 {
	return art.Clamp(math.Pow(max(0, float64(tokens))/DayTokens, 0.47))
}

// LifeGrowth returns the age of a variety's lifetime tree, (tokens/MatureTokens)^0.3: quick while young and slower
// with age.
func LifeGrowth(tokens int64) float64 {
	return art.Clamp(math.Pow(max(0, float64(tokens))/MatureTokens, 0.3))
}
