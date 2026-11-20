# Pixel Bonsai

Grow a pixel-art sakura, Japanese maple, or Japanese black pine in your terminal with the token usage of your coding
agents. Each variety has a tree for today and a tree for its lifetime, and grows through twenty named stages.

Pixel Bonsai is one self-contained binary for Windows, macOS, and Linux. It reads the usage logs that Claude Code,
Codex, Antigravity, Pi, and OpenCode already write, and any other harness can report its usage with one command.

## Install

Download the archive for your system from the
[releases page](https://github.com/KrishRVH/pixel-bonsai/releases), unpack it, and put `pixel-bonsai` (or
`pixel-bonsai.exe`) on your `PATH`. To build from source with Go 1.27 or later:

```sh
go install github.com/KrishRVH/pixel-bonsai@latest
```

## Run the tree

```sh
pixel-bonsai
```

The tree fills the terminal, as large as fits, over a caption that names the variety, its stage, its tokens, and the
view. It needs a terminal that understands escape codes: Windows Terminal, the Windows 10 or later console, and every
current macOS and Linux terminal do.

| Key or click | Action |
| --- | --- |
| **V**, **Tab**, **→**, or a click on the variety's name | Select the next variety |
| **←** | Select the previous variety |
| **Space**, any other key, or any other click | Switch between today and lifetime |
| **Q**, **Ctrl+C**, or **Ctrl+D** | Quit |

The tree counts usage when it starts and every minute after. New tokens go to the variety selected when they are
counted, for both its trees. The day's tree starts afresh at local midnight.

Colors follow the environment: `NO_COLOR` or `TERM=dumb` draws a plain silhouette, Apple's Terminal gets 256 colors,
and every other terminal gets 24-bit color. Choose one yourself with `--color truecolor`, `--color 256`, or
`--color none`.

The garden, which records each variety's growth, lives in the state folder:

| System | State folder |
| --- | --- |
| Windows | `%LOCALAPPDATA%\pixel-bonsai` |
| macOS | `~/Library/Application Support/pixel-bonsai` |
| Linux and others | `$XDG_STATE_HOME/pixel-bonsai`, by default `~/.local/state/pixel-bonsai` |

Set `BONSAI_STATE_DIR` to use another folder. Several panes can share one garden.

## Connect a harness

The tree reads these sources on its own when they are present, several at once:

| Harness | Usage source |
| --- | --- |
| Claude Code | Assistant messages in the transcripts under `~/.claude*/projects` and `$CLAUDE_CONFIG_DIR/projects` |
| Codex | Token-count events in the active and archived rollouts under `$CODEX_HOME`, by default `~/.codex` |
| Antigravity | Completed model steps in the conversation transcripts under `~/.gemini/antigravity*/brain` |
| Pi and Oh My Pi | Assistant messages in the sessions under `~/.pi/agent/sessions`, `~/.omp/agent/sessions`, and `$PI_CODING_AGENT_SESSION_DIR` or `$PI_CODING_AGENT_DIR/sessions` |
| OpenCode | Completed assistant messages in its SQLite database, `$XDG_DATA_HOME/opencode/opencode.db` by default, or `$OPENCODE_DB` |

Any other harness records one event for each completed model call:

```sh
pixel-bonsai record --source my-harness --id session42/turn7 --tokens 1250
```

Give the call's own tokens, and the same ID when a call is retried; IDs are scoped to the source. A hook or a remote
worker can instead append JSON lines to a file and point `BONSAI_USAGE_FILE` at it. The
[integration contracts](docs/integrations.md) define both.

To see which sources the tree finds and what each counts:

```sh
pixel-bonsai doctor
```

## Draw a tree in another program

A program that keeps its own idea of progress can draw one frame, with no terminal, usage logs, or garden:

```sh
pixel-bonsai render --variety maple --growth 0.7
pixel-bonsai render --variety maple --growth 0.7 --format json
pixel-bonsai render --variety maple --growth 0.7 --format png --zoom 8 > maple.png
```

Growth runs from 0, a seed, to 1, a venerable tree. The [integration contracts](docs/integrations.md) define the
output formats. Go programs can import the drawing kit, `github.com/KrishRVH/pixel-bonsai/art`, and the varieties in
`github.com/KrishRVH/pixel-bonsai/bonsai` directly.

## Change the art

Each variety is a Go package under [`bonsai/`](bonsai) that draws with the kit in [`art/`](art). To see every stage
of a variety as labelled PNG contact sheets and terminal frames:

```sh
pixel-bonsai sheet artifacts/sheets maple
```

[`AGENTS.md`](AGENTS.md) explains how to change a variety, add one, or add a harness, and the rules every drawing
follows.

## Develop

[mise](https://mise.jdx.dev/) pins the toolchain and runs the project's tasks:

```sh
mise install                 # install Go and the linters
mise run standards           # format and tidy
mise run standards:check     # the full gate: secrets, hygiene, formatting, lint, and tests with the race detector
mise run go:bench            # benchmarks
mise run release v1.2.3      # build every platform's archive from main and publish a GitHub release
```

## Restricted Go dialect

These guards keep the project's Go dialect direct and self-contained. Local data and control flow should remain
visible without extra ceremony or language machinery.

Project code rejects two language features:

- Go 1.23 range-over-function iterators. `boringlint/noiterator` rejects direct `iter` imports, the language
  construct, and iterator-shaped project type and function declarations.
- Go 1.27 generic methods. `boringlint/nogenericmethod` rejects declarations and dependency method selections even
  though the compiler accepts them.

Methods on generic types that only use their receiver's type parameters and package-level generic functions remain
allowed. Dependencies may still return iterator values; materialize them immediately at the call boundary, for
example with `slices.Collect`.

## License

[MIT](LICENSE).
