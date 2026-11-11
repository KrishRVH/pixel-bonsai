"""A pixel bonsai, in its own pane under the nav column, that grows with the captain's Claude token usage.

  growth(tokens)  0..1: (tokens / MATURE) ** 0.4, quick while young and slower with age; at the fleet's autumn 2026
                  pace (about 0.3B tokens a day) it reaches full size in about two years
  pixels(g)       {(x, y): color} for the tree at growth g; it is one tree at every size, so later growth only adds
  rows(px, size)  the pixels as terminal lines of half blocks, two pixels to a cell, each pixel size times larger
  tokens()        every token in the Claude transcripts (input, cache reads and writes, output), each API message
                  once; each day's highest count is kept, so transcripts Claude prunes for age (always from past
                  days) never shrink the tree
Python 3.9, standard library.
"""
import glob
import json
import math
import os
import random

W, H = 34, 30            # the canvas in pixels: 34 cells wide, 15 rows tall
ROWS = H // 2
MATURE = 250e9           # tokens for a full-grown tree
SEED = 5                 # which tree: the same one at every size
TRUNK_TOP = 0.5          # growth at which the trunk reaches its full height
BASE_Y, APEX_Y = 24, 3   # where the trunk meets the soil, and its full height

BARK = ("4a3222", "6b4a32", "8f6a4a")              # shadow, bark, lit edge
LEAF = ("24502e", "3a7442", "5c9f5c", "9fd38a")    # shadow, deep, mid, lit
BLOSSOM = "f2b8c9"
POT = ("263d5a", "35587e", "5f86b3")               # shadow, glaze, rim light
SOIL, MOSS, STONE, DEADWOOD = "3b2a1e", "6f9a3f", "8c8a84", "d9d2c3"
STAGES = ((0.08, "sprout"), (0.18, "seedling"), (0.3, "sapling"), (0.45, "young tree"), (0.6, "first shaping"),
          (0.75, "taking form"), (0.9, "mature"), (2, "ancient"))


def growth(tokens):
    return max(0.0, min(1.0, (max(0.0, tokens) / MATURE) ** 0.4))


def stage(g):
    return next(name for top, name in STAGES if g < top)


def ramp(g, start, span):
    """0 before start, rising to 1 at start + span."""
    return max(0.0, min(1.0, (g - start) / span))


def noise(x, y, salt=0):
    """A fixed pseudo-random value in [0, 1) for a pixel, so texture and blossoms stay put as the tree grows."""
    return ((x * 73856093) ^ (y * 19349663) ^ (salt * 83492791)) % 1000 / 1000


def trunk_x(y):
    """The trunk's line: an informal upright S that leans back over the pot toward its apex."""
    f = (BASE_Y - y) / (BASE_Y - APEX_Y)
    return round(17 + 3.4 * math.sin(math.pi * 1.45 * f + 0.35) * (1 - 0.35 * f) - 1.2)


def plan():
    """The grown tree as data: each main branch with its birth, side, attach height, length, rise and pad, and the
    side shoots that come later."""
    rng = random.Random(SEED)
    side = rng.choice((-1, 1))
    branches = []
    for i in range(7):
        f = 0.26 + i * 0.1 + rng.uniform(-0.02, 0.02)          # height on the trunk, as a share of its full height
        born = TRUNK_TOP * f ** 1.25 + 0.04 + i * 0.015        # just after the trunk passes it
        length = round(9.5 - i * 0.85 + rng.uniform(-0.8, 0.8))
        rise = 0.12 + i * 0.07 + rng.uniform(-0.04, 0.04)
        pad = 5.6 - i * 0.4 + rng.uniform(-0.3, 0.3)
        shoots = [(rng.uniform(0.45, 0.75), min(0.93, born + 0.28 + rng.uniform(0, 0.3)), rng.randint(2, 4),
                   rng.uniform(1.8, 2.8)) for _ in range(rng.randint(1, 2) if i < 5 else 1)]
        branches.append({"y": round(BASE_Y - f * (BASE_Y - APEX_Y)), "born": born, "side": side, "length": length,
                         "rise": rise, "pad": pad, "shoots": shoots})
        side = -side
    return branches


PLAN = plan()


def pot(px, g):
    """The pot, repotted wider as the tree grows, with its soil mound, moss and a stone."""
    width = 13 if g < 0.25 else 19 if g < 0.6 else 25
    x0 = (W - width) // 2
    for x in range(x0, x0 + width):                      # rim
        px[(x, H - 5)] = POT[2] if x < x0 + width // 3 else POT[1]
    for y in range(H - 4, H - 1):                        # body, tapering
        inset = 1 + (y == H - 2)
        for x in range(x0 + inset, x0 + width - inset):
            px[(x, y)] = POT[0] if x >= x0 + width - inset - 2 or y == H - 2 else POT[1]
    for x in (x0 + 2, x0 + 3, x0 + width - 4, x0 + width - 3) if width > 13 else (x0 + 3, x0 + width - 4):
        px[(x, H - 1)] = POT[0]                          # feet
    for x in range(x0 + 2, x0 + width - 2):              # soil mound
        px[(x, BASE_Y)] = MOSS if noise(x, BASE_Y, 3) < 0.7 * ramp(g, 0.3, 0.3) else SOIL
    if g >= 0.5:
        px[(x0 + 3, BASE_Y)] = px[(x0 + 4, BASE_Y)] = STONE


def pad(px, cx, cy, rx, g):
    """A cloud of foliage: a flattened ellipse with a lumpy top, lit from above, in more shades as the tree ages."""
    if rx < 0.5:
        return
    ry = max(1.0, rx * 0.5 + 0.4)
    tones = 2 + (g >= 0.35) + (g >= 0.6)
    bloom = 0.14 * ramp(g, 0.8, 0.2)
    for y in range(int(cy - ry - 1), int(cy + ry) + 1):
        for x in range(int(cx - rx), int(cx + rx) + 1):
            lump = 0.35 * noise(x, 0, 7) if y < cy else 0
            if ((x - cx) / rx) ** 2 + ((y - cy) / (ry + lump)) ** 2 > 1 or not (0 <= x < W and 0 <= y < H):
                continue
            v = (y - cy) / ry + (noise(x, y, 11) - 0.5) * 0.6  # -1 top .. 1 bottom, dithered
            shade = LEAF[3] if v < -0.55 else LEAF[2] if v < 0.05 else LEAF[1] if v < 0.6 else LEAF[0]
            if tones < 4 and shade == LEAF[3]:
                shade = LEAF[2]
            if tones < 3 and shade == LEAF[0]:
                shade = LEAF[1]
            px[(x, y)] = BLOSSOM if v < 0.1 and noise(x, y, 13) < bloom else shade


def pixels(g):
    """The bonsai at growth g in [0, 1]."""
    px = {}
    pot(px, g)
    height = (BASE_Y - APEX_Y) * max(0.06, ramp(g, 0, TRUNK_TOP)) ** 0.8
    top = BASE_Y - max(1, round(height))
    base = 1 + 3 * ramp(g, 0.15, 0.7)
    pads, wood = [], {}
    for y in range(BASE_Y, top - 1, -1):                 # trunk, thick at the soil and tapering
        width = max(1, round(base * (1 - (BASE_Y - y) / max(height, 1) * 0.75)))
        x0 = trunk_x(y) - (width - 1) // 2
        for i in range(width):
            texture = g >= 0.6 and 0 < i < width - 1 and noise(x0 + i, y, 5) < 0.3
            wood[(x0 + i, y)] = BARK[2] if i == 0 and width > 1 else BARK[0] if (i == width - 1 and width > 2
                                                                                 ) or texture else BARK[1]
    flare = round(2 * ramp(g, 0.55, 0.3))                # nebari: roots spreading over the soil
    for dy, spread in ((0, flare), (1, flare - 1)):
        y = BASE_Y - dy
        xs = [x for (x, yy) in wood if yy == y]
        for i in range(1, max(0, spread) + 1):
            wood[(min(xs) - i, y)] = BARK[1]
            wood[(max(xs) + i, y)] = BARK[0]
    jin = round(BASE_Y - 0.62 * (BASE_Y - APEX_Y))       # a weathered deadwood spike, from 0.7
    for k in range(1, 1 + round(3 * ramp(g, 0.7, 0.1))):
        wood[(trunk_x(jin) - PLAN[4]["side"] * k, jin - k)] = DEADWOOD
    for b in PLAN:
        reach = round(b["length"] * ramp(g, b["born"], 0.16))
        if reach < 1 or b["y"] < top:
            continue
        x0 = trunk_x(b["y"])
        path = [(x0 + b["side"] * k, b["y"] - round(b["rise"] * k - 0.9 * math.sin(math.pi * k / b["length"])))
                for k in range(1, reach + 1)]  # dipping, then lifting to its pad
        for k, (x, y) in enumerate(path):
            wood[(x, y)] = BARK[1]
            if g >= 0.6 and k < len(path) * 0.4:
                wood[(x, y + 1)] = BARK[0]
        tip = path[-1]
        pads.append((tip[0] + b["side"], tip[1] - 1, 1.2 + (b["pad"] - 1.2) * ramp(g, b["born"] + 0.05, 0.3)))
        for at, born, length, size in b["shoots"]:
            grown = round(length * ramp(g, born, 0.1))
            if grown < 1 or len(path) < 3:
                continue
            sx, sy = path[int(len(path) * at)]
            shoot = [(sx + b["side"] * (k // 2 + 1), sy - k) for k in range(1, grown + 1)]
            for p in shoot:
                wood[p] = BARK[1]
            pads.append((shoot[-1][0], shoot[-1][1] - 1, 1 + (size - 1) * ramp(g, born + 0.04, 0.2)))
    pads.append((trunk_x(top), top - 1, 1 + 3.2 * ramp(g, 0.0, 0.65)))  # the apex, first as a sprout's leaves
    px.update((p, c) for p, c in wood.items() if 0 <= p[0] < W and 0 <= p[1] < H)
    for cx, cy, rx in sorted(pads, key=lambda p: -p[1]):  # lower pads first, so higher ones sit in front
        pad(px, cx, cy, rx, g)
    return px


def rows(px, rgb=None, size=1):
    """Half-block lines: each cell shows two stacked pixels, the upper as the glyph's color, the lower as its
    background (or the glyph's color when only the lower is set), each pixel drawn size by size.
    rgb(hex, background) gives an escape code."""
    rgb = rgb or (lambda c, bg: "\x1b[%d;2;%d;%d;%dm" % ((48 if bg else 38,) + tuple(
        int(c[i:i + 2], 16) for i in (0, 2, 4))))
    out = []
    for r in range(ROWS * size):
        line = ""
        for x in range(W * size):
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


def stored():
    """The tokens counted so far, from the store alone."""
    try:
        with open(store_path(), encoding="utf-8") as f:
            return sum(json.load(f).get("days", {}).values())
    except (OSError, ValueError, AttributeError):
        return 0


def tokens():
    """Count every transcript again, keep each day's highest count in the store, and return the total."""
    days, seen = {}, set()
    for path in glob.glob(os.path.expanduser("~/.claude*/projects/**/*.jsonl"), recursive=True):
        try:
            with open(path, "rb") as f:
                for line in f:
                    if b'"usage"' not in line or b'"assistant"' not in line:
                        continue
                    try:
                        entry = json.loads(line)
                    except ValueError:
                        continue
                    message = entry.get("message") or {}
                    key = (message.get("id"), entry.get("requestId"))
                    if not key[0] or key in seen or not isinstance(message.get("usage"), dict):
                        continue
                    seen.add(key)
                    used = message["usage"]
                    day = (entry.get("timestamp") or "")[:10]
                    days[day] = days.get(day, 0) + sum(used.get(k) or 0 for k in (
                        "input_tokens", "cache_creation_input_tokens", "cache_read_input_tokens", "output_tokens"))
        except OSError:
            continue
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
