# Harnesses

The bonsai reads the usage these harnesses already record, several at once, whenever their files are present.
`pixel-bonsai doctor` shows which ones it found and what each counted.

| Harness | Usage source |
| --- | --- |
| Claude Code | Assistant messages in the transcripts under `~/.claude*/projects` and `$CLAUDE_CONFIG_DIR/projects` |
| Codex | Token-count events in the active and archived rollouts under `$CODEX_HOME`, by default `~/.codex` |
| GitHub Copilot CLI 1.0.94 or later | Usage records in the session journals under `$COPILOT_HOME/session-state`, by default `~/.copilot/session-state` |
| Antigravity | Completed model steps in the conversation transcripts under `~/.gemini/antigravity*/brain` |
| Pi and Oh My Pi | Assistant messages in the sessions under `~/.pi/agent/sessions`, `~/.omp/agent/sessions`, and `$PI_CODING_AGENT_SESSION_DIR` or `$PI_CODING_AGENT_DIR/sessions` |
| OpenCode | Completed assistant messages in its SQLite databases: `$OPENCODE_DB`, or else `opencode.db` and each release channel's `opencode-*.db` in `$XDG_DATA_HOME/opencode`, by default `~/.local/share/opencode` |
| Kilo Code 7 or later | The same, from `$KILO_DB`, or else `kilo.db`, `kilo-*.db`, and `opencode-*.db` in `$XDG_DATA_HOME/kilo` |
| Qwen Code 0.19 or later | The monthly usage ledgers under `~/.qwen/usage`, `$QWEN_HOME/usage`, and `$QWEN_RUNTIME_DIR/usage` |
| Kimi Code | Usage records in each agent's journal under `$KIMI_CODE_HOME/sessions`, by default `~/.kimi-code/sessions` |
| Goose | The usage ledger in `sessions/sessions.db` in its data folder: `$GOOSE_PATH_ROOT/data`, or else `$XDG_DATA_HOME/goose` (by default `~/.local/share/goose`) or, on Windows, `%APPDATA%\Block\goose\data` |

## Any other harness

Any other harness records one event for each completed model call:

```sh
pixel-bonsai record --source my-harness --id session42/turn7 --tokens 1250
```

Give the call's own tokens, and the same ID when a call is retried; IDs are scoped to the source. A hook or a remote
worker can instead append JSON lines to a file and point `BONSAI_USAGE_FILE` at it. The
[integration contracts](integrations.md) define both.

## Cursor

Cursor keeps no usage log the bonsai can read, but its `stop` hook receives each agent turn's tokens. Cursor's hooks
reference leaves the token fields out, but Cursor Agent 2026.11.19 sends them. `pixel-bonsai doctor` counts the
turns a hook records under `events`.

Save this script as `~/.cursor/hooks/pixel-bonsai.sh`, make it executable, and register it in `~/.cursor/hooks.json`.
It needs [jq](https://jqlang.org/).

```sh
#!/bin/sh
# Cursor passes the finished turn as JSON on standard input. Its input tokens include the cache reads and writes.
jq -r '"\(.conversation_id)/\(.generation_id) \((.input_tokens // 0) + (.output_tokens // 0))"' | {
  read -r id tokens
  pixel-bonsai record --source cursor --id "$id" --tokens "$tokens" >/dev/null 2>&1
}
echo '{}'
```

```json
{"version": 1, "hooks": {"stop": [{"command": "./hooks/pixel-bonsai.sh", "timeout": 10}]}}
```

User hooks run from `~/.cursor`, so the command's path is relative to it. On Windows, save the PowerShell form as
`~/.cursor/hooks/pixel-bonsai.ps1` and make the command
`powershell -NoProfile -ExecutionPolicy Bypass -File ./hooks/pixel-bonsai.ps1`:

```powershell
$turn = [Console]::In.ReadToEnd() | ConvertFrom-Json
$tokens = [int64]$turn.input_tokens + [int64]$turn.output_tokens
pixel-bonsai record --source cursor --id "$($turn.conversation_id)/$($turn.generation_id)" --tokens $tokens *> $null
'{}'
```
