# Integration contracts

These are the interfaces other programs rely on: the event feed, the command line, and the files the bonsai keeps. A
change to any of them is a breaking change for its users.

## Event feed

The event feed carries usage from any harness the bonsai does not read on its own. Each completed model call is one
JSON object on one line:

```json
{"source":"my-harness","id":"session42/turn7","timestamp":"2026-11-19T12:00:00Z","tokens":1250}
```

| Field | Meaning |
| --- | --- |
| `source` | A stable name for the harness. It must not be blank. |
| `id` | The call's ID, unique within the source and the same when the call is retried. It must not be blank. |
| `timestamp` | When the call completed, in RFC 3339 with a time zone offset. A space may stand in for the `T`. |
| `tokens` | The call's tokens, at least 0: input, output, and cache together, not a running total. |

The bonsai counts each `source` and `id` pair once, so a retried or replayed event adds nothing. A line counts once it
ends with a newline; a line still being written waits. Every field is required. A line that is not such an object, or
breaks a rule in the table, is skipped and counted as malformed in `pixel-bonsai doctor`: `tokens` must be a JSON
integer, and `timestamp` a string. Blank lines and fields the table does not name are ignored.

The bonsai reads two kinds of feed file:

- The monthly files that `pixel-bonsai record` appends to, `events/YYYY-MM.jsonl` in the state folder.
- The files that `BONSAI_USAGE_FILE` lists, separated as in `PATH`: by `;` on Windows and `:` elsewhere.

Report each call one way. An event in the feed and the same call in a harness log the bonsai reads natively have
different identities, and both count.

## Commands

Every command exits 0 on success, 1 on a failure, and 2 on a mistake in its command line. Flags may come before or
after positional arguments.

### `pixel-bonsai record`

```sh
pixel-bonsai record --source NAME --id ID --tokens N [--timestamp TIME]
```

Appends one event to this month's feed file, under a lock that other recorders share.
`--timestamp` defaults to now; any offset is stored as UTC. An event that breaks the feed's rules exits 2 and writes
nothing.

### `pixel-bonsai render`

```sh
pixel-bonsai render --growth G [--variety ID] [--scale S] [--format ansi|json|png] [--color MODE] [--zoom N]
```

Writes one frame to standard output. It reads no usage and touches no files.

| Flag | Meaning |
| --- | --- |
| `--growth` | Required. The tree's age, from 0 (seed) to 1 (venerable). |
| `--variety` | `sakura` (the default), `maple`, or `black-pine`. |
| `--scale` | Canvas scale, from 0.5 to 3, default 1. The canvas is 48 by 30 pixels at scale 1, and a larger scale draws more detail, not bigger pixels. |
| `--format` | `ansi` (the default), `json`, or `png`. |
| `--color` | For `ansi`: `truecolor` (the default), `256`, or `none`. |
| `--zoom` | For `png`: image pixels per tree pixel, from 1 to 64, default 1. |

`ansi` is lines of half-block characters, two pixels to a cell, the upper pixel as the character's color and the lower
as its background. `png` is the frame with a transparent background. `json` is one object:

```json
{"width":48,"height":30,"variety":"maple","stage":"mature","pixels":[[23,0,"fddfe9"],[24,0,"ffffff"]]}
```

`pixels` lists every painted pixel as `[x, y, "rrggbb"]`, in rows from the top left; a pixel it leaves out is
transparent. `stage` is the stage's name, such as `seed`, `young tree`, or `venerable`.

### `pixel-bonsai sheet`

```sh
pixel-bonsai sheet DIR [VARIETY ...] [--scale S] [--zoom N]
```

Writes a folder per variety into `DIR`: `stages.png`, `day-100M.png`, `life-13B.png`, `scales.png`, and
`frames.txt`, and prints each file's path. These are for reviewing art; their layout may change.

### `pixel-bonsai doctor`

```sh
pixel-bonsai doctor [--json]
```

Reads every source once and reports the state folder, the color mode, the garden, and for each source the files it
found, the events and tokens it counted, today's tokens, malformed lines, and any error. It changes nothing. The
report is for people; its fields may change.

## Environment

| Variable | Effect |
| --- | --- |
| `BONSAI_STATE_DIR` | The state folder, in place of the system's default. |
| `BONSAI_USAGE_FILE` | Extra event feed files. |
| `CLAUDE_CONFIG_DIR` | Another Claude Code configuration folder to read transcripts from. |
| `CODEX_HOME` | Codex's folder, by default `~/.codex`. |
| `PI_CODING_AGENT_SESSION_DIR`, `PI_CODING_AGENT_DIR` | Pi's sessions folder, or the folder that holds it. |
| `OPENCODE_DB` | OpenCode's database, absolute or relative to its data folder. |
| `XDG_DATA_HOME` | The data folder that holds OpenCode's, by default `~/.local/share`. |
| `XDG_STATE_HOME` | The state folder's parent, except on Windows. |
| `LOCALAPPDATA` | The state folder's parent on Windows. |
| `NO_COLOR`, `TERM`, `COLORTERM`, `TERM_PROGRAM` | The live pane's color mode, as the README describes. |

## State folder

| File | Contents |
| --- | --- |
| `garden.json` | The selected variety and view, each variety's lifetime and day growth, and each source's highest token count for each UTC day. The daily highs keep the lifetime total from shrinking when a harness prunes old logs. |
| `garden.json.lock` | The lock every pane takes to update the garden. |
| `garden.json.damaged` | A garden that failed to decode, set aside when a new one started. |
| `events/` | The monthly feed files and their lock. |

`garden.json` records its schema version. A build refuses a garden from a newer build and leaves it untouched.
