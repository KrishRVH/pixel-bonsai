"""The pixel bonsai's ages, varieties, token counting and garden. The tree is drawn by a variety (sakura.py, maple.py,
pine.py, on the kit and the ladder of stages in art.py) at an age from the day's tokens or from the lifetime tokens
that variety has grown.

  day_growth(tokens)   the day's tree's age, 0 to 1: (tokens / DAY) ** 0.47, quick early in the day and never capped
                       short of the ladder's end: 100M is a young tree, an autumn 2026 median day (about 0.25B)
                       thickens its trunk, a busy 1B is refined, the 2.1B record is old, about 2.6B reaches venerable
  life_growth(tokens)  a variety's lifetime age, 0 to 1: (tokens / MATURE) ** 0.3, quick while young and slower with
                       age; at autumn 2026's pace (about 0.4B tokens a day) it is a venerable tree in about two years
  VIEWS, VARIETIES     the two age mappings by view name, and the variety modules by name, in the order a switch
                       steps through them
  rows(px, ...)        a canvas's pixels as terminal lines of half blocks, two pixels to a cell
  garden(), accrue(total, today), choose(name), day_count(garden, name)
                       each variety's own lifetime and day growth, kept beside the token store: tokens counted go to
                       the variety selected when they are counted, for its day's tree and its lifetime tree alike
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

import art
import maple
import pine
import sakura

DAY = 3e9                # tokens in a day for a fully venerable day's tree
MATURE = 250e9           # lifetime tokens for a variety's full-grown tree
VARIETIES = {tree.NAME: tree for tree in (sakura, maple, pine)}
DEFAULT = "sakura"       # the variety the old tree continues as, and the one a garden falls back to


def day_growth(tokens):
    return art.clamp((max(0.0, tokens) / DAY) ** 0.47)


def life_growth(tokens):
    return art.clamp((max(0.0, tokens) / MATURE) ** 0.3)


VIEWS = {"today": day_growth, "lifetime": life_growth}


def rows(px, rgb=None, size=1, w=art.W, h=art.H):
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


def save(path, data):
    """Write data to path as JSON atomically: to a temporary file beside it, then a rename over it."""
    os.makedirs(os.path.dirname(path), exist_ok=True)
    with open(path + ".tmp", "w", encoding="utf-8") as f:
        json.dump(data, f, sort_keys=True)
    os.replace(path + ".tmp", path)


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
        save(path, {"days": kept})
    except OSError:
        pass
    return sum(kept.values())


# ---- the garden: each variety's own lifetime and day growth ------------------------------------------------------

def no_day():
    """A day record for no day yet: the first accrual with the day's tokens starts the real one."""
    return {"date": "", "counted": 0, "grown": {name: 0 for name in VARIETIES}}


def seeded(total):
    """A new garden: sakura continues the old tree, grown to total, and every other variety starts from seed."""
    return {"variety": DEFAULT, "grown": {name: total if name == DEFAULT else 0 for name in VARIETIES},
            "counted": total, "day": no_day()}


def local_date(now=None):
    return time.strftime("%Y-%m-%d", time.localtime(time.time() if now is None else now))


def day_count(data, name, now=None):
    """The tokens this variety has grown today: its share of the day record when that record is today's, else 0 (a
    record from yesterday until the first accrual after midnight replaces it)."""
    day = data.get("day") or {}
    return day.get("grown", {}).get(name, 0) if day.get("date") == local_date(now) else 0


def number(value):
    if isinstance(value, bool) or not isinstance(value, (int, float)):
        raise ValueError(value)
    return max(0, value)


def garden(total=None):
    """The garden as stored: the selected variety, the lifetime tokens grown by each, the lifetime total already
    handed out ("counted"), and the day record (its local date, the day's tokens already handed out and each
    variety's share). Absent or unreadable, it is seeded from total (the tokens counted so far, stored() when None);
    a garden from before day records, or with a damaged one, keeps its lifetime growth and gets no_day()."""
    try:
        with open(garden_path(), encoding="utf-8") as f:
            data = json.load(f)
        name = data["variety"] if data["variety"] in VARIETIES else DEFAULT
        out = {"variety": name, "grown": {n: number(data["grown"].get(n, 0)) for n in VARIETIES},
               "counted": number(data["counted"])}
    except (OSError, ValueError, KeyError, TypeError, AttributeError):
        return seeded(stored() if total is None else total)
    try:
        day = data["day"]
        if not isinstance(day["date"], str):
            raise ValueError(day["date"])
        out["day"] = {"date": day["date"], "counted": number(day["counted"]),
                      "grown": {n: number(day["grown"].get(n, 0)) for n in VARIETIES}}
    except (ValueError, KeyError, TypeError, AttributeError):
        out["day"] = no_day()
    return out


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
                save(path, data)
    except OSError:
        pass
    return data if data is not None else garden(total)


def accrue(total, today=None, now=None):
    """Hand the selected variety the lifetime tokens counted since the last call: total less what was already
    handed out. The same total twice adds nothing, and a smaller one is ignored. With today (the tokens since local
    midnight), the day's tokens are handed out the same way to the selected variety's day tree; the first call on a
    new local date (or on a garden without a day record) starts the day afresh and gives the selected variety all of
    today, since nothing recorded which variety was selected earlier in that day, so a minute that crosses midnight
    lands on the right day."""
    def add(data):
        new = max(0, total - data["counted"])
        data["grown"][data["variety"]] += new
        data["counted"] = max(data["counted"], total)
        changed = new > 0
        if today is not None:
            day, date = data["day"], local_date(now)
            if day["date"] != date:
                data["day"] = day = no_day()
                day["date"], changed = date, True
            fresh = max(0, today - day["counted"])
            day["grown"][data["variety"]] += fresh
            day["counted"] = max(day["counted"], today)
            changed = changed or fresh > 0
        return changed
    return update(add, total)


def choose(name):
    """Select a variety. It hands out nothing: tokens counted later go to it."""
    def pick(data):
        if name not in VARIETIES or data["variety"] == name:
            return False
        data["variety"] = name
        return True
    return update(pick)
