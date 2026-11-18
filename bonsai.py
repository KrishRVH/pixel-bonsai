"""The pixel bonsai's ages, varieties, token counting and garden. The tree is drawn by a variety (sakura.py, maple.py,
pine.py, on the kit and the ladder of stages in art.py) at an age from the day's tokens or from the lifetime tokens
that variety has grown.

  day_growth(tokens)   the day's tree's age, 0 to 0.65: (tokens / DAY) ** 0.5, quick early in the day; an autumn 2026
                       median day (about 0.23B tokens) thickens its trunk and a busy one (1B) establishes it
  life_growth(tokens)  a variety's lifetime age, 0 to 1: (tokens / MATURE) ** 0.3, quick while young and slower with
                       age; at autumn 2026's pace (about 0.4B tokens a day) it is a venerable tree in about two years
  VIEWS, VARIETIES     the two age mappings by view name, and the variety modules by name, in the order a switch
                       steps through them
  rows(px, ...)        a canvas's pixels as terminal lines of half blocks, two pixels to a cell
  garden(), accrue(total), choose(name)
                       each variety's own lifetime growth, kept beside the token store: tokens counted go to the
                       variety selected when they are counted
  Ledger               the transcripts' tokens (input, cache reads and writes, output), each API message once, read
                       incrementally: an update reads only what was appended since the last
  today()              the tokens since local midnight
  tokens()             every token counted; each day's highest count is kept, so transcripts Claude prunes for age
                       (always from past days) never shrink the lifetime total
Python 3.9, standard library.
"""
import fcntl
import glob
import json
import os
import time

import maple
import pine
import sakura
from art import H, ROWS, W  # noqa: F401  the canvas at scale 1, for callers that draw rows of it

DAY = 1e9                # tokens at which the day's tree stops growing
MATURE = 250e9           # lifetime tokens for a variety's full-grown tree
VARIETIES = {tree.NAME: tree for tree in (sakura, maple, pine)}


def day_growth(tokens):
    return 0.65 * max(0.0, min(1.0, (max(0.0, tokens) / DAY) ** 0.5))


def life_growth(tokens):
    return max(0.0, min(1.0, (max(0.0, tokens) / MATURE) ** 0.3))


VIEWS = {"today": day_growth, "lifetime": life_growth}


def rows(px, rgb=None, size=1, w=W, h=H):
    """Half-block lines for a w by h canvas: each cell shows two stacked pixels, the upper as the glyph's color, the
    lower as its background (or the glyph's color when only the lower is set), each pixel drawn size by size.
    rgb(hex, background) gives an escape code."""
    rgb = rgb or (lambda c, bg: "\x1b[%d;2;%d;%d;%dm" % ((48 if bg else 38,) + tuple(
        int(c[i:i + 2], 16) for i in (0, 2, 4))))
    out = []
    for r in range(h // 2 * size):
        line = ""
        for x in range(w * size):
            top, low = px.get((x // size, 2 * r // size)), px.get((x // size, (2 * r + 1) // size))
            if top and low:
                line += rgb(top, False) + rgb(low, True) + "▀\x1b[0m" if top != low else rgb(top, False) + "█\x1b[0m"
            elif top or low:
                line += rgb(top or low, False) + ("▀" if top else "▄") + "\x1b[0m"
            else:
                line += " "
        out.append(line)
    return out


def store_path():
    return os.path.join(os.environ.get("XDG_STATE_HOME") or os.path.expanduser("~/.local/state"), "qd",
                        "bonsai.json")


def garden_path():
    return os.path.join(os.path.dirname(store_path()), "bonsai-varieties.json")


def stored():
    """The tokens counted so far, from the store alone."""
    try:
        with open(store_path(), encoding="utf-8") as f:
            return sum(json.load(f).get("days", {}).values())
    except (OSError, ValueError, AttributeError):
        return 0


def transcripts():
    return glob.glob(os.path.expanduser("~/.claude*/projects/**/*.jsonl"), recursive=True)


BLOCK = 1 << 20           # bytes read at a time: larger blocks raise the peak, not the speed
USAGE_KEYS = ("input_tokens", "cache_creation_input_tokens", "cache_read_input_tokens", "output_tokens")


class Ledger:
    """Every API message's tokens in the transcripts, each message once (a message repeated in a line or in a resumed
    copy counts once), by UTC minute. Transcripts only grow, so each update reads just the bytes appended since the
    last one: the bonsai pane keeps one ledger for its life, and a recount costs what was written since, not the
    gigabytes of history. A transcript Claude prunes keeps its count here; a replaced one is read again, which the
    message keys make harmless."""

    def __init__(self):
        self.read = {}       # path -> (inode, bytes read up to the last whole line)
        self.seen = set()    # hash of (message id, request id), which holds a few MB for years of messages
        self.minutes = {}    # "YYYY-MM-DDTHH:MM" (UTC) -> tokens

    def update(self, paths=None):
        paths = transcripts() if paths is None else paths
        for path in paths:
            try:
                st = os.stat(path)
                inode, at = self.read.get(path, (st.st_ino, 0))
                if inode != st.st_ino or st.st_size < at:
                    at = 0  # replaced or rewritten: read it again
                with open(path, "rb") as f:
                    f.seek(at)
                    rest = b""
                    while at + len(rest) < st.st_size:  # in blocks, so a transcript of hundreds of MB stays small
                        block = rest + f.read(min(BLOCK, st.st_size - at - len(rest)))
                        end = block.rfind(b"\n") + 1  # a line still being written waits for the next update
                        if not end and len(block) == len(rest):
                            break
                        for line in block[:end].split(b"\n"):
                            self.add(line)
                        at, rest = at + end, block[end:]
            except OSError:
                continue
            self.read[path] = (st.st_ino, at)
        return self

    def add(self, line):
        if b'"usage"' not in line or b'"assistant"' not in line:
            return
        try:
            entry = json.loads(line)
        except ValueError:
            return
        message = entry.get("message") or {}
        used = message.get("usage")
        key = hash((message.get("id"), entry.get("requestId")))
        if not message.get("id") or key in self.seen or not isinstance(used, dict):
            return
        self.seen.add(key)
        minute = (entry.get("timestamp") or "")[:16]
        self.minutes[minute] = self.minutes.get(minute, 0) + sum(used.get(k) or 0 for k in USAGE_KEYS)

    def since(self, stamp):
        """Tokens from this UTC stamp ("YYYY-MM-DDTHH:MM:SS", on a whole minute) on."""
        return sum(n for minute, n in self.minutes.items() if minute >= stamp[:16])

    def days(self):
        out = {}
        for minute, n in self.minutes.items():
            out[minute[:10]] = out.get(minute[:10], 0) + n
        return out


def midnight_utc(now=None):
    """Local midnight today as the UTC stamp transcripts use."""
    midnight = time.mktime(time.localtime(time.time() if now is None else now)[:3] + (0, 0, 0, 0, 0, -1))
    return time.strftime("%Y-%m-%dT%H:%M:%S", time.gmtime(midnight))


def today(now=None, ledger=None):
    """The tokens since local midnight."""
    return (ledger or Ledger().update()).since(midnight_utc(now))


def tokens(ledger=None):
    """Every token counted, with each UTC day's highest count kept in the store, so transcripts Claude prunes for age
    never shrink the lifetime tree; the total."""
    days = (ledger or Ledger().update()).days()
    path = store_path()
    try:
        with open(path, encoding="utf-8") as f:
            kept = json.load(f).get("days", {})
    except (OSError, ValueError, AttributeError):
        kept = {}
    for day, count in days.items():
        kept[day] = max(kept.get(day, 0), count)
    try:
        os.makedirs(os.path.dirname(path), exist_ok=True)
        with open(path + ".tmp", "w", encoding="utf-8") as f:
            json.dump({"days": kept}, f, sort_keys=True)
        os.replace(path + ".tmp", path)
    except OSError:
        pass
    return sum(kept.values())


# ---- the garden: each variety's own lifetime growth --------------------------------------------------------------

def seeded(total):
    """A new garden: sakura continues the old tree, grown to total, and every other variety starts from seed."""
    first = next(iter(VARIETIES))
    return {"variety": first, "grown": {name: total if name == first else 0 for name in VARIETIES}, "counted": total}


def number(value):
    if isinstance(value, bool) or not isinstance(value, (int, float)):
        raise ValueError(value)
    return max(0, value)


def garden(total=None):
    """The garden as stored: the selected variety, the lifetime tokens grown by each, and the lifetime total already
    handed out ("counted"). Absent or unreadable, it is seeded from total (the tokens counted so far, stored() when
    None)."""
    try:
        with open(garden_path(), encoding="utf-8") as f:
            data = json.load(f)
        name = data["variety"] if data["variety"] in VARIETIES else next(iter(VARIETIES))
        return {"variety": name, "grown": {n: number(data["grown"].get(n, 0)) for n in VARIETIES},
                "counted": number(data["counted"])}
    except (OSError, ValueError, KeyError, TypeError, AttributeError):
        return seeded(stored() if total is None else total)


def update(change, total=None):
    """Apply change(garden) under the garden's lock and save it atomically when it reports a change (or the file is
    new); the garden as it is after. A state folder that cannot be written leaves the garden as it was read."""
    data = None
    try:
        path = garden_path()
        os.makedirs(os.path.dirname(path), exist_ok=True)
        with open(path + ".lock", "w") as lock:
            fcntl.flock(lock, fcntl.LOCK_EX)  # a sibling file, since the JSON itself is replaced
            data = garden(total)
            if change(data) or not os.path.exists(path):
                with open(path + ".tmp", "w", encoding="utf-8") as f:
                    json.dump(data, f, sort_keys=True)
                os.replace(path + ".tmp", path)
    except OSError:
        pass
    return data if data is not None else garden(total)


def accrue(total):
    """Hand the selected variety the lifetime tokens counted since the last call: total less what was already
    handed out. The same total twice adds nothing, and a smaller one is ignored."""
    def add(data):
        new = max(0, total - data["counted"])
        data["grown"][data["variety"]] += new
        data["counted"] = max(data["counted"], total)
        return new > 0
    return update(add, total)


def choose(name):
    """Select a variety. It hands out nothing: tokens counted later go to it."""
    def pick(data):
        if name not in VARIETIES or data["variety"] == name:
            return False
        data["variety"] = name
        return True
    return update(pick)
