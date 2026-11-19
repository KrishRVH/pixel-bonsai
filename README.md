# Pixel Bonsai

Grow a pixel-art sakura, Japanese maple, or Japanese black pine in your terminal with token usage from your coding agents. Each variety has its own daily and lifetime tree and grows through twenty named stages.

## Run the tree

On macOS or Linux, with Python 3.9 or later and a UTF-8 terminal with true color support:

```sh
git clone https://github.com/KrishRVH/pixel-bonsai.git
python3 pixel-bonsai
```

Press **V**, **Tab**, or **→** to select the next variety, or **←** for the previous one. Clicking the variety name also cycles through the trees. Press **Space** or click the tree to switch between today and lifetime. **Ctrl+C** exits.

The live view starts at seed and checks for new usage every minute. The selected variety receives new tokens when they are counted. Daily growth resets at local midnight. The garden and token history use the original minimap directory at `${XDG_STATE_HOME:-$HOME/.local/state}/qd`, so an existing garden carries over.

## Connect a harness

The tree automatically reads these local sources when present. You can use several at once.

| Harness | Local usage source |
| --- | --- |
| Claude Code | Assistant message usage under `~/.claude*/projects`. |
| Codex | Token-count events from active and archived rollout JSONL in `$CODEX_HOME` (default `~/.codex`). |
| Antigravity | Completed model steps in the App, CLI, and IDE conversation transcripts under `~/.gemini/antigravity*/brain`. |
| Pi and Oh My Pi | Assistant usage from session JSONL under `~/.pi/agent/sessions` and `~/.omp/agent/sessions`. Also reads `PI_CODING_AGENT_SESSION_DIR` or the sessions folder in `PI_CODING_AGENT_DIR`. |
| OpenCode | Completed assistant messages in its local SQLite database under `$XDG_DATA_HOME/opencode` (default `~/.local/share/opencode`). Set `OPENCODE_DB` for a custom database. |

**Any other harness** can send one event per completed model call. From a checkout:

```sh
python3 . record --source my-harness --id session42/turn7 --tokens 1250
```

Use the token count **for that call** and reuse its ID when retrying. IDs are scoped to the source. The command timestamps the event when received. Supply `--timestamp 2026-11-19T12:00:00Z` to import historical usage. A hook or a remote worker can also append one JSON object per line to a file the viewer can read. Point `BONSAI_USAGE_FILE` at that file when launching the tree:

```json
{"source":"my-harness","id":"session42/turn7","timestamp":"2026-11-19T12:00:00Z","tokens":1250}
```

Choose one reporting path for each call. A separately reported event and an automatically discovered provider record have different identities and both count.

## Draw a tree in another program

A host that already knows what progress means can render one frame directly. The command works with redirected output and needs no usage logs or garden:

```sh
python3 . render --variety maple --growth 0.7
python3 . render --variety maple --growth 0.7 --format json
```

Growth ranges from 0 to 1. The default output is ANSI half-block text. JSON returns `width`, `height`, and a list of `[x, y, "rrggbb"]` pixels with the origin at the top left.

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

## License

[MIT](LICENSE).
