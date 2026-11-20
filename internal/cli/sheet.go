package cli

import (
	"errors"
	"fmt"
	"strings"

	"github.com/KrishRVH/pixel-bonsai/art"
	"github.com/KrishRVH/pixel-bonsai/bonsai"
	"github.com/KrishRVH/pixel-bonsai/internal/sheet"
)

// sheet writes contact sheets of varieties, for reviewing their art without a terminal pane.
func (a App) sheet(args []string) error {
	fs := a.flags("sheet", "pixel-bonsai sheet DIR [VARIETY ...] [--scale S] [--zoom N]")
	scale := fs.Float64("scale", 1, "canvas scale of the stage grid")
	zoom := fs.Int("zoom", 6, "PNG pixels per tree pixel")
	positional, err := parse(fs, args)
	if err != nil {
		return err
	}
	if len(positional) == 0 {
		return usageError{errors.New("sheet needs an output folder")}
	}
	if !(*scale >= art.MinScale && *scale <= art.MaxScale) || *zoom < 1 || *zoom > 64 { // also refuses NaN
		return usageError{fmt.Errorf("--scale must be from %g to %g and --zoom from 1 to 64", art.MinScale, art.MaxScale)}
	}
	varieties, err := pick(positional[1:])
	if err != nil {
		return err
	}
	for _, v := range varieties {
		files, err := sheet.Write(positional[0], v, *scale, *zoom)
		for _, f := range files {
			if _, err := fmt.Fprintln(a.Stdout, f); err != nil {
				return err
			}
		}
		if err != nil {
			return err
		}
	}
	return nil
}

// pick returns the varieties with the ids given, or every variety for none.
func pick(ids []string) ([]art.Variety, error) {
	if len(ids) == 0 {
		return bonsai.Varieties(), nil
	}
	out := make([]art.Variety, 0, len(ids))
	for _, id := range ids {
		v, ok := bonsai.Lookup(id)
		if !ok {
			return nil, usageError{fmt.Errorf("unknown variety %q: use %s", id, strings.Join(bonsai.IDs(), ", "))}
		}
		out = append(out, v)
	}
	return out, nil
}
