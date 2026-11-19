# Pixel Bonsai

Grow a pixel-art sakura, Japanese maple, or Japanese black pine in your terminal with your Claude Code token usage. Each variety has its own daily and lifetime tree and grows through twenty named stages.

## Run the tree

On macOS or Linux, with Python 3.9 or later and a UTF-8 terminal with true color support:

```sh
git clone https://github.com/KrishRVH/pixel-bonsai.git
python3 pixel-bonsai
```

Press **V**, **Tab**, or **→** to select the next variety, or **←** for the previous one. Clicking the variety name also cycles through the trees. Press **Space** or click the tree to switch between today and lifetime. **Ctrl+C** exits.

The live view reads local Claude Code transcripts under `~/.claude*/projects`. It starts at seed if there are no transcripts. The selected variety receives new tokens when they are counted. Daily growth resets at local midnight. Lifetime growth survives transcript pruning. The garden and token history use the original minimap files under `${XDG_STATE_HOME:-$HOME/.local/state}/qd`, so an existing garden carries over.

## Draw a tree in another program

Each variety exposes `pixels(growth, scale)`. Growth ranges from 0 (seed) to 1 (venerable). The result maps pixel coordinates to six-digit RGB values. Missing coordinates are transparent. For example, from a Python process with this checkout on its module path:

```python
import art
import maple

width, height = art.size(1)
pixels = maple.pixels(0.7, scale=1)
```

Edit `sakura.py`, `maple.py`, or `pine.py` to customize a variety. To see every stage, export a labelled PNG contact sheet and terminal frames from the checkout:

```sh
python3 . sheet /tmp/bonsai-preview maple
python3 -m unittest discover -s tests
```

The drawing code and tests were extracted from [herdr-minimap](https://github.com/KrishRVH/herdr-minimap). Herdr uses the same tree in its navigation column.
