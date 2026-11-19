"""Interactive terminal view for the pixel bonsai."""
import os
import re
import select
import signal
import sys
import termios
import threading
import time
import tty
import unicodedata

import art
import bonsai

COUNT_EVERY = 60
COLOR = True
GOLD, TEXT, DIM, SUBTEXT = "e5b850", "ece4d2", "7d7a73", "b2aa98"


def cells(text):
    return sum(0 if unicodedata.combining(c) or c in "‍️" else
               2 if unicodedata.east_asian_width(c) in "WF" or ord(c) >= 0x1F000 else 1 for c in text)


def clip(text, width):
    if cells(text) <= width:
        return text
    out = ""
    for c in text:
        if cells(out + c) > width - 1:
            return out + "…"
        out += c
    return out


def paint(rgb, text, bold=False, dim=False):
    if not COLOR:
        return text
    r, g, b = (int(rgb[i:i + 2], 16) for i in (0, 2, 4))
    return f"\x1b[{'1;' if bold else ''}{'2;' if dim else ''}38;2;{r};{g};{b}m{text}\x1b[0m"


def part(text, color=TEXT, bold=False, dim=False):
    return (text, color, bold, dim)


def row(parts, width, align="center"):
    """Parts clipped to width as one painted line, padded to exactly width cells."""
    out, used = "", 0
    for text, color, bold, dim in parts:
        if used >= width:
            break
        text = clip(text, width - used)
        out += paint(color, text, bold, dim)
        used += cells(text)
    left = (width - used) // 2 if align == "center" else 0 if align == "left" else \
        max(0, min(width - used, align - used // 2))  # a number centers the line on that column
    return " " * left + out + " " * (width - used - left)


def amount(n):
    """1234567 -> '1.2M'."""
    for size, unit in ((1e12, "T"), (1e9, "B"), (1e6, "M"), (1e3, "K")):
        if n >= size:
            return f"{n / size:.1f}{unit}"
    return str(int(n))


def tree_lines(count, width, height, view="today", variety=bonsai.DEFAULT, hits=None):
    """The bonsai pane's lines: the variety's tree at the view's age for this many tokens, at the largest scale the
    pane holds (art.fit), centered over its caption: variety, stage, amount and view. The caption alone when the tree
    does not fit. hits, when given, takes the caption's variety span as (line, x0, x1, ("variety",))."""
    tree = bonsai.VARIETIES[variety]
    g = bonsai.VIEWS[view](count)
    name = f"{tree.ICON} {tree.NAME}"
    for word in (" tokens", ""):  # the word goes first when the caption is too wide
        caption = [part(name, TEXT), part(" · ", DIM), part(art.stage(g), SUBTEXT),
                   part(f" · {amount(count)}{word} · ", DIM), part(view, GOLD), part(" ⇄", DIM)]
        span = sum(cells(p[0]) for p in caption)
        if span <= width:
            break
    if hits is not None and height > 0:
        left = (width - min(width, span)) // 2
        hits.append((height - 1, left, min(width, left + cells(name)), ("variety",)))
    scale = art.fit(width, height) if COLOR else 0
    if scale:
        w, h = art.size(scale)
        picture = [" " * ((width - w) // 2) + line for line in bonsai.rows(tree.pixels(g, scale), w=w, h=h)]
    else:
        picture = []
    return ([""] * height + picture + [row(caption, width)])[-height:] if height > 0 else []


def switched(view, variety, events, hits=()):
    """The (view, variety) after this input: a click on the caption's variety, v, tab or the right arrow steps to the
    next variety and the left arrow to the previous; any other click or key switches between today and lifetime; the
    wheel does nothing."""
    names, views = list(bonsai.VARIETIES), list(bonsai.VIEWS)
    for e in events:
        if e[0] == "wheel":
            continue
        if e[0] in ("left", "right") or (e[0] == "char" and e[1] in ("v", "V", "\t")) or \
                (e[0] == "click" and hit_at(hits, e[1], e[2])):
            variety = names[(names.index(variety) + (-1 if e[0] == "left" else 1)) % len(names)]
        else:
            view = views[(views.index(view) + 1) % len(views)]
    return view, variety


CSI = re.compile(rb"\x1b\[[0-9;?]*[A-Za-z~]")
MOUSE = re.compile(rb"\x1b\[<(\d+);(\d+);(\d+)([Mm])")
ARROWS = {b"\x1b[A": "up", b"\x1b[B": "down", b"\x1b[C": "right", b"\x1b[D": "left",
          b"\x1bOA": "up", b"\x1bOB": "down", b"\x1bOC": "right", b"\x1bOD": "left"}


def keys_of(data):
    """Terminal input as events: (up,) (down,) (left,) (right,) (enter,) (esc,) (char, c), (click, x, y) zero-based,
    (wheel, ±1)."""
    out, i = [], 0
    while i < len(data):
        m = MOUSE.match(data, i)
        if m:
            button = int(m[1])
            if m[4] == b"M" and button == 0:
                out.append(("click", int(m[2]) - 1, int(m[3]) - 1))
            elif button in (64, 65):
                out.append(("wheel", -1 if button == 64 else 1))
            i = m.end()
            continue
        arrow = next((k for k in ARROWS if data.startswith(k, i)), None)
        if arrow:
            out.append((ARROWS[arrow],))
            i += len(arrow)
            continue
        m = CSI.match(data, i)
        if m:  # focus reports and other unused keys
            i = m.end()
            continue
        c = data[i:i + 1]
        if c == b"\x1b":
            out.append(("esc",))
        elif c in (b"\r", b"\n"):
            out.append(("enter",))
        elif c.isascii():  # a byte of a wider character is ignored
            out.append(("char", c.decode()))
        i += 1
    return out


def hit_at(hits, x, y):
    return next((place for line, x0, x1, place in hits if line == y and x0 <= x < x1), None)


def run_tree(view_file):
    """The bonsai pane: the selected variety's day or lifetime tree as large as the pane holds it, redrawn when its
    size, variety, view or token count changes; clicks and keys switch variety and view. Tokens are counted every
    COUNT_EVERY seconds, and the garden hands the ones counted to the variety selected then, for its day and lifetime
    trees alike."""
    fd = sys.stdin.fileno()
    saved = termios.tcgetattr(fd) if os.isatty(fd) else None
    if saved:
        tty.setcbreak(fd)  # keys switch the view rather than echo over the tree
    signal.signal(signal.SIGTERM, lambda *_: sys.exit(0))
    sys.stdout.write("\x1b[?25l\x1b[?1000h\x1b[?1006h")  # hide the cursor; report clicks
    state, hits, drawn, garden_lock = {"garden": bonsai.garden()}, [], None, threading.Lock()
    try:
        view = view_file.read_text(encoding="utf-8").strip()
    except OSError:
        view = ""
    if view not in bonsai.VIEWS:
        view = "today"

    def recount():
        ledger = bonsai.HarnessUsage()  # the first update reads every transcript; later ones only what was appended
        while True:
            ledger.update()
            today, total = bonsai.today(ledger=ledger), bonsai.tokens(ledger)
            with garden_lock:  # so a switch and a recount reach the file and this pane in the same order
                state["garden"] = bonsai.accrue(total, today)
            time.sleep(COUNT_EVERY)
    threading.Thread(target=recount, daemon=True).start()
    try:
        while True:
            size, garden = os.get_terminal_size(), state["garden"]
            variety = garden["variety"]
            count = bonsai.day_count(garden, variety) if view == "today" else garden["grown"][variety]
            if drawn != (view, variety, count, size):
                drawn = (view, variety, count, size)
                hits.clear()
                lines = tree_lines(count, size.columns - 2, size.lines, view, variety, hits)
                sys.stdout.write("\x1b[H" + "\x1b[K\n".join(lines) + "\x1b[K\x1b[J")
                sys.stdout.flush()
            if not saved:
                time.sleep(1)
            elif select.select([fd], [], [], 1)[0]:
                new_view, new_variety = switched(view, variety, keys_of(os.read(fd, 256)), hits)
                if new_view != view:
                    view = new_view
                    try:
                        view_file.parent.mkdir(parents=True, exist_ok=True)
                        view_file.write_text(view + "\n", encoding="utf-8")
                    except OSError:
                        pass
                if new_variety != variety:
                    with garden_lock:
                        state["garden"] = bonsai.choose(new_variety)
    except KeyboardInterrupt:
        pass
    finally:
        sys.stdout.write("\x1b[?1006l\x1b[?1000l\x1b[0m\x1b[?25h")
        if saved:
            termios.tcsetattr(fd, termios.TCSADRAIN, saved)
