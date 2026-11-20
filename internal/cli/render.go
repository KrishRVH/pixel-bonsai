package cli

import (
	"encoding/json"
	"errors"
	"fmt"
	"image"
	"image/color"
	"image/png"
	"strconv"
	"strings"

	"github.com/KrishRVH/pixel-bonsai/art"
	"github.com/KrishRVH/pixel-bonsai/bonsai"
	"github.com/KrishRVH/pixel-bonsai/internal/ansi"
)

// render prints one frame. It needs no terminal, usage logs, or garden, so any program can draw the tree.
func (a App) render(args []string) error {
	fs := a.flags("render", "pixel-bonsai render --growth G [--variety ID] [--scale S] [--format ansi|json|png]")
	variety := fs.String("variety", bonsai.Default().ID, "variety: "+strings.Join(bonsai.IDs(), ", "))
	growth := fs.Float64("growth", -1, "age from 0 (seed) to 1 (venerable); required")
	scale := fs.Float64("scale", 1, "canvas scale from 0.5 to 3")
	format := fs.String("format", "ansi", "ansi (half blocks), json, or png")
	colors := fs.String("color", ansi.TrueColor.String(), "ansi colors: truecolor, 256, or none")
	zoom := fs.Int("zoom", 1, "png pixels per tree pixel, from 1 to 64")
	if err := parseNone(fs, args); err != nil {
		return err
	}
	v, ok := bonsai.Lookup(*variety)
	switch {
	case !ok:
		return usageError{fmt.Errorf("unknown variety %q: use %s", *variety, strings.Join(bonsai.IDs(), ", "))}
	case !(*growth >= 0 && *growth <= 1): // also refuses NaN
		return usageError{errors.New("--growth must be from 0 to 1")}
	case !(*scale >= art.MinScale && *scale <= art.MaxScale):
		return usageError{fmt.Errorf("--scale must be from %g to %g", art.MinScale, art.MaxScale)}
	case *zoom < 1 || *zoom > 64:
		return usageError{errors.New("--zoom must be from 1 to 64")}
	}
	mode, err := ansi.ParseMode(*colors)
	if err != nil {
		return usageError{err}
	}
	f := v.Render(*growth, *scale)
	switch *format {
	case "ansi":
		_, err = fmt.Fprintln(a.Stdout, strings.Join(ansi.HalfBlocks(f, mode), "\n"))
	case "json":
		err = json.NewEncoder(a.Stdout).Encode(frameJSON{f.Width(), f.Height(), v.ID, art.StageAt(*growth).String(), pixelsJSON(f.Pixels())})
	case "png":
		err = png.Encode(a.Stdout, zoomed{f, *zoom})
	default:
		return usageError{fmt.Errorf("unknown format %q: use ansi, json, or png", *format)}
	}
	if err != nil {
		return fmt.Errorf("write frame: %w", err)
	}
	return nil
}

// frameJSON is render's JSON output: the canvas size, the variety and stage, and the set pixels as [x, y, "rrggbb"]
// in row-major order from the top left.
type frameJSON struct {
	Width   int        `json:"width"`
	Height  int        `json:"height"`
	Variety string     `json:"variety"`
	Stage   string     `json:"stage"`
	Pixels  pixelsJSON `json:"pixels"`
}

type pixelsJSON []art.Pixel

// MarshalJSON implements [json.Marshaler], writing each pixel as a compact array.
func (ps pixelsJSON) MarshalJSON() ([]byte, error) {
	out := make([]byte, 0, len(ps)*18+2)
	out = append(out, '[')
	for k, p := range ps {
		if k > 0 {
			out = append(out, ',')
		}
		out = append(out, '[')
		out = strconv.AppendInt(out, int64(p.X), 10)
		out = append(out, ',')
		out = strconv.AppendInt(out, int64(p.Y), 10)
		out = append(out, ",\""...)
		out = append(out, p.Color.Hex()...)
		out = append(out, "\"]"...)
	}
	return append(out, ']'), nil
}

// zoomed shows a frame with each pixel zoom pixels square.
type zoomed struct {
	f    *art.Frame
	zoom int
}

func (z zoomed) ColorModel() color.Model { return z.f.ColorModel() }

func (z zoomed) Bounds() image.Rectangle {
	return image.Rect(0, 0, z.f.Width()*z.zoom, z.f.Height()*z.zoom)
}

func (z zoomed) At(x, y int) color.Color { return z.f.At(x/z.zoom, y/z.zoom) }
