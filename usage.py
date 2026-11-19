"""Discover local harness logs and normalize usage into (event id, timestamp, token delta)."""
import argparse
import glob
import json
import os
import sqlite3
from contextlib import closing
from datetime import datetime, timezone
from pathlib import Path


def directory(variable, default):
    return Path(os.environ.get(variable) or default).expanduser()


def state_dir():
    return directory("XDG_STATE_HOME", "~/.local/state") / "qd"


def files(source):
    home = Path.home()
    if source == "claude":
        return glob.iglob(str(home / ".claude*/projects/**/*.jsonl"), recursive=True)
    if source == "codex":
        root = directory("CODEX_HOME", "~/.codex")
        return (p for folder in ("sessions", "archived_sessions")
                for p in (root / folder).rglob("rollout-*.jsonl"))
    if source == "antigravity":
        return home.glob(".gemini/antigravity*/brain/*/.system_generated/logs/transcript.jsonl")
    if source == "pi":
        # Pi and Oh My Pi share a session format and directory overrides.
        agent = directory("PI_CODING_AGENT_DIR", "~/.pi/agent")
        custom = directory("PI_CODING_AGENT_SESSION_DIR", agent / "sessions")
        roots = dict.fromkeys((home / ".pi/agent/sessions", home / ".omp/agent/sessions", custom))
        return (p for root in roots for p in root.rglob("*.jsonl"))
    paths = [state_dir() / "bonsai-events.jsonl"]
    if os.environ.get("BONSAI_USAGE_FILE"):
        paths.append(Path(os.environ["BONSAI_USAGE_FILE"]).expanduser())
    return paths


def count(value):
    return value if type(value) is int and value >= 0 else 0


def utc_time(stamp):
    """Native sessions use RFC 3339 or Unix milliseconds. Return UTC for day accounting."""
    if type(stamp) is int:
        return datetime.fromtimestamp(stamp / 1000, timezone.utc)
    if not isinstance(stamp, str):
        raise ValueError("missing timestamp")
    dt = datetime.fromisoformat(stamp.replace("Z", "+00:00"))
    if dt.tzinfo is None:
        raise ValueError("timestamp needs a time zone")
    return dt.astimezone(timezone.utc)


def claude(entry, path, context):
    message = entry.get("message")
    if entry.get("type") != "assistant" or not isinstance(message, dict) or not isinstance(message.get("usage"), dict):
        return
    if message.get("id"):
        keys = ("input_tokens", "cache_creation_input_tokens", "cache_read_input_tokens", "output_tokens")
        return (message["id"], entry.get("requestId")), entry.get("timestamp"), sum(count(message["usage"].get(k)) for k in keys)


def codex(entry, path, context):
    payload = entry.get("payload")
    if entry.get("type") != "event_msg" or not isinstance(payload, dict) or payload.get("type") != "token_count":
        return
    info = payload.get("info")
    if not isinstance(info, dict):
        return
    total = count((info.get("total_token_usage") or {}).get("total_tokens"))
    if not total or total == context.get("total"):
        return  # rate-limit refreshes repeat the same token counters
    last = count((info.get("last_token_usage") or {}).get("total_tokens"))
    if not last:
        return
    context["total"] = total
    stamp = entry.get("timestamp")
    # A fork can copy its parent's events into a different rollout.
    return (stamp, total, last), stamp, last


def antigravity(entry, path, context):
    step = entry.get("step_index")
    if entry.get("source") != "MODEL" or entry.get("status") != "DONE" or type(step) is not int:
        return
    keys = ("input_tokens", "output_tokens", "cache_read_tokens", "cache_creation_input_tokens")
    tokens = count(entry.get("total_tokens")) or sum(count(entry.get(k)) for k in keys)
    conversation = Path(path).parents[2].name
    return (conversation, step), entry.get("created_at"), tokens


def pi(entry, path, context):
    message = entry.get("message")
    if entry.get("type") != "message" or not isinstance(message, dict) or message.get("role") != "assistant":
        return
    used = message.get("usage")
    if not isinstance(used, dict):
        return
    keys = ("input", "output", "cacheRead", "cacheWrite")
    tokens = count(used.get("totalTokens")) or sum(count(used.get(k)) for k in keys)
    stamp = entry.get("timestamp") or message.get("timestamp")
    if message.get("responseId") or entry.get("id"):
        return message.get("responseId") or (entry["id"], stamp), stamp, tokens


def event(entry, path, context):
    if all(isinstance(entry.get(k), str) and entry[k].strip() for k in ("source", "id")):
        return (entry["source"], entry["id"]), entry.get("timestamp"), count(entry.get("tokens"))


# Check an inexpensive marker before decoding potentially large transcript lines.
READERS = {"claude": (b'"usage"', claude), "codex": (b'"token_count"', codex),
           "antigravity": (b'"step_index"', antigravity), "pi": (b'"usage"', pi),
           "events": (b'"tokens"', event)}


def record(argv):
    parser = argparse.ArgumentParser(prog="pixel-bonsai record", description="Record one harness usage event.")
    parser.add_argument("--source", required=True, help="stable name of your harness")
    parser.add_argument("--id", required=True, help="unique event ID within that source; reuse on retry")
    parser.add_argument("--tokens", required=True, type=int, help="token delta for this event")
    parser.add_argument("--timestamp", help="event time with UTC offset (default: now)")
    args = parser.parse_args(argv)
    if not args.source.strip() or not args.id.strip() or args.tokens < 0:
        parser.error("provide a nonblank source and id and a nonnegative token count")
    try:
        stamp = utc_time(args.timestamp) if args.timestamp else datetime.now(timezone.utc)
    except ValueError as exc:
        parser.error(str(exc))
    entry = dict(source=args.source, id=args.id, timestamp=stamp.isoformat(), tokens=args.tokens)
    folder = state_dir()
    folder.mkdir(parents=True, exist_ok=True)
    import fcntl
    with (folder / "bonsai-events.jsonl").open("a", encoding="utf-8") as stream:
        fcntl.flock(stream, fcntl.LOCK_EX)
        stream.write(json.dumps(entry) + "\n")


class OpenCode:
    """Read completed assistant messages from both generations of the local OpenCode database."""

    def __init__(self):
        self.watermarks = {}
        self.identity = None

    def update(self, credit):
        base = directory("XDG_DATA_HOME", "~/.local/share") / "opencode"
        path = base / (os.environ.get("OPENCODE_DB") or "opencode.db")
        try:
            stat = path.stat()
            identity = (path, stat.st_dev, stat.st_ino)
            if self.identity != identity:
                self.watermarks.clear()
                self.identity = identity
            with closing(sqlite3.connect(path.absolute().as_uri() + "?mode=ro", uri=True)) as db:
                tables = {name for name, in db.execute("SELECT name FROM sqlite_master WHERE type='table'")}
                for table in ("session_message", "message"):
                    if table in tables:
                        self.read(db, table, credit)
        except (OSError, sqlite3.Error):
            return  # absent, busy or incompatible databases can be retried on the next poll

    def read(self, db, table, credit):
        role = "type" if table == "session_message" else "json_extract(data, '$.role')"
        sql = f"""SELECT id, time_updated, json_extract(data, '$.time.completed'), json_extract(data, '$.tokens')
                  FROM {table} WHERE time_updated >= ? AND json_valid(data) AND {role} = 'assistant'"""
        watermark = self.watermarks.get(table, 0)
        for ident, updated, stamp, raw in db.execute(sql, (watermark,)):
            watermark = max(watermark, updated)
            if not stamp or not isinstance(raw, str):
                continue
            try:
                used = json.loads(raw)
            except ValueError:
                continue
            if not isinstance(used, dict):
                continue
            cache = used.get("cache") or {}
            if not isinstance(cache, dict):
                continue
            total = count(used.get("total")) or (sum(count(used.get(k)) for k in ("input", "output", "reasoning"))
                                                  + count(cache.get("read")) + count(cache.get("write")))
            credit(ident, stamp, total)
        self.watermarks[table] = watermark
