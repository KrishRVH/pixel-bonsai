# Context

Pixel Bonsai grows a pixel-art tree in the terminal from the tokens coding agents spend. This page defines the words
the code, tests, and docs use.

## Trees

- **Variety**: a kind of tree with its own drawing, palette, and growth: `sakura`, `maple`, or `black-pine`. A
  variety is an `art.Variety`, drawn by one package under `bonsai/`.
- **Age** or **growth**: a tree's maturity, `g`, from 0 (a seed) to 1 (venerable). A drawing is a function of age and
  scale alone.
- **Stage**: one of the twenty named steps on the **ladder**, from `seed` to `venerable`. Each stage starts at a fixed
  age, and every variety shows something new at each one.
- **View**: which of a variety's two trees the pane shows. The **today** tree grows from the tokens the variety
  received since local midnight; the **lifetime** tree from every token it has received.
- **Signature color**: a color only one variety uses, which makes it recognizable even small.

## Drawing

- **Kit**: the `art` package, the shapes and helpers every variety draws with.
- **Canvas**: the pixels a variety paints, 48 by 30 at scale 1.
- **Scale**: how many pixels a canvas unit takes, from 0.5 to 3. A larger scale draws more detail, not bigger pixels.
- **Frame**: a finished canvas, the pixels a variety drew at one age and scale.
- **Cover**: the pixels a shape painted, in drawing order, for texturing or clipping later shapes against it.
- **Half blocks**: how a frame reaches the terminal, two pixels stacked in each character cell.
- **Contact sheet**: PNG grids and terminal frames of a variety's stages, for reviewing art.

## Usage

- **Harness**: a coding agent that spends tokens, such as Claude Code or Codex.
- **Source**: where usage comes from: a harness whose logs the bonsai reads natively, or the event feed.
- **Event**: one model call's tokens and when it completed, with a **key** that identifies the call within its
  source, so a retried, resumed, or copied call counts once.
- **Ledger**: one source's events, counted once each and totaled by quarter hour. A ledger reads only what its source
  added since its last read.
- **Event feed**: JSON lines any harness writes, directly or with `pixel-bonsai record`.

## Garden

- **Garden**: the saved state in the state folder: the selected variety and view, each variety's growth, and the
  daily highs.
- **Daily high**: a source's highest token count for a UTC day. The lifetime total is the sum of the daily highs, so it
  never shrinks when a harness prunes its logs.
- **Accrual**: handing the selected variety the tokens counted since the last accrual. Tokens go only to the variety
  selected when they are counted.
- **Day record**: the day tree's share of each variety's tokens for one local date.

## Pane

- **Pane**: the live view: the selected variety's tree as large as fits, over the **caption** that names the variety,
  its stage, its tokens, and the view.
