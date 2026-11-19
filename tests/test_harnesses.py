"""Local usage sources and the public, harness-independent CLI."""
import json
import os
from pathlib import Path
import sqlite3
import subprocess
import sys
import tempfile
import unittest
from unittest.mock import patch

import bonsai

ROOT = Path(__file__).resolve().parents[1]
STAMP = "2026-11-10T12:00:00Z"


class Harnesses(unittest.TestCase):
    def setUp(self):
        temp = tempfile.TemporaryDirectory()
        self.addCleanup(temp.cleanup)
        self.home = Path(temp.name)
        env = patch.dict(os.environ, {"HOME": temp.name}, clear=True)
        env.start()
        self.addCleanup(env.stop)

    def append(self, path, data):
        path = self.home / path
        path.parent.mkdir(parents=True, exist_ok=True)
        with path.open("a") as f:
            f.write(json.dumps(data) + "\n")

    def cli(self, *args, check=True):
        return subprocess.run([sys.executable, str(ROOT), *args], cwd=self.home,
                              capture_output=True, text=True, check=check)

    def test_custom_harness_retries_scoping_timestamps_and_incremental_input(self):
        args = ("record", "--source", "my-tool", "--id", "turn42", "--tokens", "71",
                "--timestamp", "2026-11-11T00:30:00+02:00")
        self.cli(*args)
        self.cli(*args)
        os.environ["BONSAI_USAGE_FILE"] = str(self.home / "extra.jsonl")
        self.append("extra.jsonl", {"source": "second-tool", "id": "turn42", "tokens": 3, "timestamp": STAMP})
        self.append("extra.jsonl", {"source": "my-tool", "id": "missing-count", "tokens": True, "timestamp": STAMP})
        book = bonsai.Ledger("events").update()
        self.assertEqual(book.days(), {"2026-11-10": 74})
        self.append("extra.jsonl", {"source": "my-tool", "id": "next", "tokens": 7, "timestamp": STAMP})
        self.assertEqual(bonsai.tokens(book.update()), 81)
        self.assertNotEqual(self.cli(*args, "--tokens", "-1", check=False).returncode, 0)
        self.assertEqual(bonsai.tokens(), 81)

    def test_codex_rollouts_count_usage_once_across_refresh_fork_and_archive(self):
        os.environ["CODEX_HOME"] = str(self.home / "codex")
        path = "codex/sessions/2026/11/10/rollout-session.jsonl"
        def token(stamp, total, last):
            return {"type": "event_msg", "timestamp": stamp, "payload": {"type": "token_count", "info": {
                "total_token_usage": {"total_tokens": total}, "last_token_usage": {"total_tokens": last}}}}
        first = token(STAMP, 100, 100)
        self.append(path, first)
        self.append(path, token("2026-11-10T12:01:00Z", 100, 100))
        self.append("codex/archived_sessions/rollout-fork.jsonl", first)
        book = bonsai.Ledger("codex").update()
        self.assertEqual(book.days(), {"2026-11-10": 100})
        self.append(path, token("2026-11-11T12:02:00Z", 108, 8))
        self.assertEqual(book.update().days(), {"2026-11-10": 100, "2026-11-11": 8})

    def test_antigravity_pi_and_oh_my_pi_sessions(self):
        path = ".gemini/antigravity-cli/brain/conversation/.system_generated/logs/transcript.jsonl"
        for status in ("ACTIVE", "DONE", "DONE"):
            self.append(path, {"step_index": 3, "source": "MODEL", "status": status, "created_at": STAMP,
                               "input_tokens": 21, "cache_read_tokens": 9, "output_tokens": 3})
        pi = {"type": "message", "id": "entry", "timestamp": STAMP, "message": {"role": "assistant",
              "usage": {"totalTokens": 100, "cacheRead": 98}}}
        self.append(".pi/agent/sessions/project/a.jsonl", pi)
        self.append(".pi/agent/sessions/project/fork.jsonl", pi)
        self.append(".omp/agent/sessions/project/a.jsonl",
                    dict(pi, id="other", message={"role": "assistant", "usage": {"input": 2, "cacheWrite": 4}}))
        self.assertEqual(bonsai.tokens(), 139)

    def test_native_usage_is_additive_after_old_claude_transcripts_are_pruned(self):
        Path(bonsai.store_path()).parent.mkdir(parents=True)
        bonsai.save(bonsai.store_path(), {"days": {"2026-11-10": 70}})
        bonsai.choose("maple")
        self.cli("record", "--source", "any-harness", "--id", "a", "--tokens", "12", "--timestamp", STAMP)
        books = bonsai.HarnessUsage().update()
        self.assertEqual(bonsai.tokens(books), 82)
        self.assertEqual(bonsai.accrue(82)["grown"], {"sakura": 70, "maple": 12, "black pine": 0})
        for book in books.ledgers:
            book.update()
        self.assertEqual(bonsai.tokens(books), 82)
        self.assertEqual(bonsai.stored(), 82)

    def test_opencode_reads_completed_messages_in_both_database_formats(self):
        dbfile = self.home / "custom.sqlite"
        os.environ["OPENCODE_DB"] = str(dbfile)
        tokens = {"input": 2, "output": 3, "reasoning": 4, "cache": {"read": 5, "write": 6}}
        data = {"time": {"completed": 1794312000000}, "role": "assistant", "tokens": tokens}
        with sqlite3.connect(dbfile) as db:
            db.execute("create table message (id text, time_updated integer, data text)")
            db.execute("insert into message values ('old', 1, ?)", (json.dumps(data),))
            db.execute("create table session_message (id text, type text, time_updated integer, data text)")
            db.execute("insert into session_message values ('new', 'assistant', 10, ?)", (json.dumps(data),))
            db.execute("insert into session_message values ('next', 'assistant', 10, ?)",
                       (json.dumps(dict(data, time={})),))
        book = bonsai.Ledger("opencode").update()
        self.assertEqual(sum(book.days().values()), 40)
        with sqlite3.connect(dbfile) as db:
            db.execute("update session_message set data=?, time_updated=11 where id='next'", (json.dumps(data),))
        self.assertEqual(sum(book.update().days().values()), 60)
        self.assertEqual(sum(book.update().days().values()), 60)

    def test_render_can_run_inside_any_host_without_a_terminal_or_garden(self):
        args = ("render", "--growth", "0.7", "--variety", "maple")
        result = json.loads(self.cli(*args, "--format", "json").stdout)
        self.assertEqual((result["width"], result["height"]), (48, 30))
        self.assertTrue(all(0 <= x < 48 and 0 <= y < 30 and len(rgb) == 6 for x, y, rgb in result["pixels"]))
        self.assertIn("▀", self.cli(*args).stdout)
        self.assertFalse(Path(bonsai.store_path()).parent.exists())
