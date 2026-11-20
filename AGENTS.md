# Agent Guide

Read `CONTEXT.md` first. Then read the docs that own the change before changing architecture or domain language:
`README.md` for behavior, `docs/integrations.md` for the contracts other programs rely on.

## Commands

`mise run` is the entry point for project workflows; tasks wrap the toolchain. Run `mise tasks` and read the
definitions of the tasks you use. Extend an existing task before you add one.

- `mise install` installs the pinned tools. When an install needs network, run it and report that.
- `mise run standards` applies the available autofixes.
- `mise run standards:check` is the gate; run it before handoff. The project has no CI: the gate runs locally.
- `mise run release vX.Y.Z` publishes a release from `main`. Run it only when the owner asks.

Run `git`, `rg`, `tokei`, and standalone scripts directly. Use native commands for focused diagnosis and the mise
gate for final verification. Use `mise exec -- <command>` only when a command needs a pinned tool or environment and no
task covers it.

## Project map

| Path | Owns |
| --- | --- |
| `main.go` | The process: signals, environment, and exit code. |
| `internal/cli` | The commands and their flags, behind `cli.App`, which holds everything the process provides. |
| `art` | The drawing kit and the ladder of stages. Public, so programs can draw trees. |
| `bonsai` | The varieties in switch order and the curves from tokens to age. |
| `bonsai/<variety>` | One variety's drawing. |
| `internal/usage` | Reading usage: one file per harness, the JSONL tailer, the event feed, and the ledgers. |
| `internal/garden` | The saved state and the rules for handing tokens to varieties. |
| `internal/pane` | The live view: layout, input, and the run loop. |
| `internal/term` | The terminal on each system: raw input, size, and reads that time out. |
| `internal/ansi` | Color modes and half-block output. |
| `internal/state` | The state folder on each system, file locks, and atomic writes. |
| `internal/sheet` | Contact sheets. |

## Drawing rules

A tree is a pure function of its age and scale. The same inputs give the same pixels on every call, so a frame never
flickers. Builds for different processors can differ in a rare edge pixel, because the Go compiler may fuse a multiply
and an add on processors such as arm64.

- Take texture from `art.Noise` with a salt, never from random state.
- Never range over a map in drawing code; Go randomizes the order. Keep pixels in a slice or an `art.Cover`.
- Round with `math.RoundToEven`, as the kit does.
- Key features to stages with `art.From`, `art.During`, `art.Since`, `art.Steps`, and `art.Lerp`, not raw ages.
- A shape option left at zero takes the kit's default. Pass `art.NoJitter` for no dither.
- Work in canvas units, 48 by 30, and let the canvas turn them into pixels at its scale.

## Change the art

1. Edit the variety's package under `bonsai/`.
2. Look at the result: `go run . sheet artifacts/sheets <variety>` writes PNG contact sheets of every stage, the
   reference day and lifetime trees, and three scales.
3. Run `go test ./bonsai/...`. The conformance tests in `bonsai/bonsai_test.go` hold every variety to the same
   bar: each stage shows something new, pixels stay on the canvas and repeat exactly, a larger scale draws more
   detail, the tree grows with age, varieties stay distinct, and each shows its signature color small.

## Add a variety

1. Create `bonsai/<name>/`, with a `Variety()` that returns an `art.Variety`: a slug ID, a display name, an icon, the
   signature colors, and the draw function. Start from the variety closest to the new one.
2. Add it to `bonsai.Varieties()` at its place in the switch order.
3. Run the conformance tests and look at the sheets.

## Add a harness

1. Add `internal/usage/<harness>.go`. For JSONL logs, write a function that finds the files and a `lineParser` that
   turns a line into an `Event`; the tailer reads only new lines. For other stores, implement `reader`, as
   `opencode.go` does. Choose the event key so that a retried, resumed, or forked call keeps its key.
2. Add it to `usage.New`. The order there is the order `doctor` reports.
3. Add a fixture test to `internal/usage/usage_test.go`, and a seed line to `FuzzParsers`.
4. Add it to the harness table in `README.md` and its environment variables to `docs/integrations.md`.

A harness that does not keep local logs reports through the event feed instead; it needs no code here.

## Leave It Current

The repository holds current contracts: the code, the tests that defend it, and the docs that explain it. Git holds
history. Apart from the documented contracts in `docs/integrations.md`, the project makes no compatibility promise:
no shims, aliases, deprecations, dual code paths, format migrations, or version-numbered names.

- Supersede rather than accumulate. Rename in place, replace a format, and delete the old reader with its fixtures
  and tests in the same change.
- A change that makes something obsolete removes it: functions and exports nothing calls, config keys nothing reads,
  tests of retired behavior, and docs that describe it.
- Shipped code has no test-only branches, alternate rules, or test-only diagnostic surfaces. A seam that production
  also uses, such as `cli.App` or `pane.Terminal`, is fine.
- Proof lives in the gate. Back each lasting claim with a test in `standards:check`. A one-off diagnostic leaves with
  the question it answered, and its result goes in the commit message.
- Keep task notes in `.scratch/` and run output in `artifacts/`. Both stay untracked. Do not commit briefs, plans,
  handoffs, research, reports, or dated audits.
- `mise run hygiene` enforces the mechanical part and prints the tree's size by category. Explain growth that the
  change does not account for.

## Principles

- Complexity is the enemy; see [grugbrain.dev](https://grugbrain.dev/). Prefer boring, obvious code with local state,
  explicit side effects, and direct data flow. Small duplication beats premature indirection.
- Build deep modules with small interfaces. Change a feature's behavior, state, and tests together.
- Respect Chesterton fences. Understand why code exists before deleting or replacing it.
- Name things for what they do in the domain, in `CONTEXT.md` vocabulary.
- Comments explain a current invariant, a non-obvious algorithm, a trust boundary, or a public interface.
- Optimize only what a measurement shows matters. `mise run go:bench` measures drawing and reading usage.

## Go

- Prefer concrete types and direct calls. Define small interfaces where a consumer needs substitution.
- Pass dependencies explicitly. Keep I/O at boundaries; avoid mutable package state and hidden initialization.
- Take `context.Context` first for cancellable operations. Every goroutine needs an owner, a stopping condition, and a
  way for its owner to wait for it.
- Return errors with useful operation context, wrapped for `errors.Is` and `errors.As`, and handle each once.
- Keep resource cleanup next to acquisition. Check errors from writes and closes when they can affect correctness.
- The restricted dialect in `README.md` forbids range-over-function iterators and generic methods.
- Read `.golangci.yml` for the enforced diagnostics. Fix the cause before adding a specific, reasoned `nolint`.
- Test observable behavior with table-driven cases where inputs are peers. Prefer real boundaries, such as temporary
  folders, and narrow fakes. Fuzz parsers of untrusted input.
- Code for one system goes in a file with a build constraint, as in `internal/term` and `internal/state`. Cross-compile
  for Windows and macOS after touching one: `GOOS=windows go vet ./...` and `GOOS=darwin go vet ./...`.

## Documentation

- Docs explain current behavior and design constraints. Update the owning page when the contract changes. State each
  policy once and link to it elsewhere.
- Write in the present tense. Do not date text or narrate history.
- Follow the [Google developer documentation style guide](https://developers.google.com/style).

## Testing

- A test earns its place by defending behavior, an invariant, a failure mode, or a trust boundary that types and the
  gate do not already enforce.
- A meaningful assertion fails for a concrete production defect and stays valid through behavior-preserving
  refactors. For a bug, reproduce it with a failing test before the fix when practical.
- Test counts and coverage percentages are not goals.

## Git

- Commit to `main`; the project uses no branches or pull requests.
- Keep generated and local-only files out of commits.
- Follow [Conventional Commits 1.0.0](https://www.conventionalcommits.org/en/v1.0.0/#specification). Keep the subject
  under 72 characters, imperative, lowercase, and without ending punctuation. Add a body only when it explains the
  problem, the reason, or a material consequence.

## Handoff

- Get an independent read-only agent review of the actual diff for non-trivial behavior and architecture changes.
  Verify material findings before acting on them.
- Report the checked revision, behavior proved, commands run and their results, skipped checks and why, remaining
  findings, and tree growth the change does not explain.
