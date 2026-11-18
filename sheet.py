#!/usr/bin/env python3
"""The bonsai's contact sheet, for reviewing a variety without a terminal pane.

  sheet.py OUTDIR [variety ...] [--scale S] [--zoom N]

writes, for each variety named (all of them when none is), a folder in OUTDIR holding

  stages.png    every stage at its start age plus a little, one frame each, labelled, in a grid (at --scale, default 1)
  day-100M.png  the day's tree at 100M tokens, and life-13B.png the lifetime tree at 13B tokens
  scales.png    the 13B frame at scales 0.6, 1 and 2
  frames.txt    the same frames as the pane draws them: ANSI half-block rows from bonsai.rows

Each pixel of a frame is --zoom (default 6) PNG pixels square, on the terminal's #1e1e2e. Python 3.9, standard library.
"""
import argparse
import os
import struct
import sys
import zlib

import art
import bonsai

BACKGROUND, LABEL = "1e1e2e", "b2aa98"
PLUS = 0.005             # a stage's frame is this far past its start, where its feature has just arrived
COLUMNS = 5
GAP = 8
FONT = """
a .#. #.# ### #.# #.#|b ##. #.# ##. #.# ##.|c .## #.. #.. #.. .##|d ##. #.# #.# #.# ##.|e ### #.. ##. #.. ###
f ### #.. ##. #.. #..|g .## #.. #.# #.# .##|h #.# #.# ### #.# #.#|i ### .#. .#. .#. ###|j ..# ..# ..# #.# .#.
k #.# #.# ##. #.# #.#|l #.. #.. #.. #.. ###|m #.# ### ### #.# #.#|n ### #.# #.# #.# #.#|o .#. #.# #.# #.# .#.
p ##. #.# ##. #.. #..|q .#. #.# #.# ##. .##|r ##. #.# ##. #.# #.#|s .## #.. .#. ..# ##.|t ### .#. .#. .#. .#.
u #.# #.# #.# #.# ###|v #.# #.# #.# #.# .#.|w #.# #.# ### ### #.#|x #.# #.# .#. #.# #.#|y #.# #.# .#. .#. .#.
z ### ..# .#. #.. ###|0 ### #.# #.# #.# ###|1 .#. ##. .#. .#. ###|2 ##. ..# .#. #.. ###|3 ##. ..# .#. ..# ##.
4 #.# #.# ### ..# ..#|5 ### #.. ##. ..# ##.|6 .## #.. ### #.# ###|7 ### ..# .#. .#. .#.|8 ### #.# ### #.# ###
9 ### #.# ### ..# ##.|. ... ... ... ... .#.|- ... ... ### ... ...|: ... .#. ... .#. ...|= ... ### ... ### ...
"""
GLYPHS = {entry[0]: entry[2:].split() for entry in FONT.replace("\n", "|").split("|") if entry}


class Image:
    """An RGB picture, filled with the terminal's background."""

    def __init__(self, w, h):
        self.w, self.h = w, h
        self.rows = [bytearray(bytes.fromhex(BACKGROUND) * w) for _ in range(h)]

    def rect(self, x, y, w, h, color):
        pixel = bytes.fromhex(color) * w
        for j in range(max(0, y), min(self.h, y + h)):
            lo = max(0, x)
            self.rows[j][3 * lo:3 * min(self.w, x + w)] = pixel[:3 * (min(self.w, x + w) - lo)]

    def frame(self, px, x, y, zoom):
        """A canvas's pixels, each zoom by zoom, with its top left at (x, y)."""
        for (i, j), color in px.items():
            self.rect(x + i * zoom, y + j * zoom, zoom, zoom, color)

    def text(self, x, y, words, zoom=2, color=LABEL):
        for k, char in enumerate(words.lower()):
            for j, row in enumerate(GLYPHS.get(char, ())):
                for i, bit in enumerate(row):
                    if bit == "#":
                        self.rect(x + (4 * k + i) * zoom, y + j * zoom, zoom, zoom, color)

    def save(self, path):
        def chunk(tag, body):
            return struct.pack(">I", len(body)) + tag + body + struct.pack(">I", zlib.crc32(tag + body))
        raw = b"".join(b"\x00" + bytes(row) for row in self.rows)
        with open(path, "wb") as f:
            f.write(b"\x89PNG\r\n\x1a\n" + chunk(b"IHDR", struct.pack(">IIBBBBB", self.w, self.h, 8, 2, 0, 0, 0)) +
                    chunk(b"IDAT", zlib.compress(raw, 6)) + chunk(b"IEND", b""))


def stage_frames(scale):
    return [(f"{k:02d} {name}", start + PLUS, scale) for k, (start, name) in enumerate(art.STAGES)]


def view_frames():
    return [("100m day", bonsai.day_growth(100e6), 1), ("13b lifetime", bonsai.life_growth(13e9), 1)]


def scale_frames():
    return [(f"13b at {scale}", bonsai.life_growth(13e9), scale) for scale in (0.6, 1, 2)]


def grid(tree, frames, zoom, columns=COLUMNS):
    """The frames, each labelled above it, in rows of columns cells."""
    drawn = [(label, tree.pixels(g, scale), art.size(scale)) for label, g, scale in frames]
    cell_w = max(w for _, _, (w, _) in drawn) * zoom + GAP
    cell_h = max(h for _, _, (_, h) in drawn) * zoom + 12 + GAP
    columns = min(columns, len(drawn))
    image = Image(cell_w * columns + GAP, cell_h * -(-len(drawn) // columns) + GAP)
    for n, (label, px, _) in enumerate(drawn):
        x, y = GAP + n % columns * cell_w, GAP + n // columns * cell_h
        image.text(x, y, label)
        image.frame(px, x, y + 12, zoom)
    return image


def ansi(tree, name, frames):
    """The frames as the pane draws them: a heading and the half-block rows of each."""
    out = []
    for label, g, scale in frames:
        w, h = art.size(scale)
        out += [f"== {name} {label} (age {g:.3f}, scale {scale}) =="]
        out += bonsai.rows(tree.pixels(g, scale), w=w, h=h) + [""]
    return "\n".join(out) + "\n"


def variety(word):
    """The variety a name, a folder-style name or a module name points at: sakura, black-pine, pine."""
    word = word.lower().replace("-", " ").replace("_", " ")
    for name, tree in bonsai.VARIETIES.items():
        if word in (name, tree.__name__):
            return tree
    raise SystemExit(f"sheet.py: no variety {word!r}; the varieties are {', '.join(bonsai.VARIETIES)}")


def main(argv):
    ap = argparse.ArgumentParser(description="Render the bonsai varieties as PNG sheets and ANSI frames.")
    ap.add_argument("outdir")
    ap.add_argument("varieties", nargs="*")
    ap.add_argument("--scale", type=float, default=1.0, help="canvas scale of the stage grid")
    ap.add_argument("--zoom", type=int, default=6, help="PNG pixels per bonsai pixel")
    args = ap.parse_args(argv)
    trees = [variety(word) for word in args.varieties] or list(bonsai.VARIETIES.values())
    written = []
    for tree in trees:
        folder = os.path.join(args.outdir, tree.NAME.replace(" ", "-"))
        os.makedirs(folder, exist_ok=True)
        views = view_frames()
        for name, frames in (("stages", stage_frames(args.scale)), ("day-100M", views[:1]), ("life-13B", views[1:]),
                             ("scales", scale_frames())):
            grid(tree, frames, args.zoom).save(os.path.join(folder, name + ".png"))
            written.append(os.path.join(folder, name + ".png"))
        with open(os.path.join(folder, "frames.txt"), "w", encoding="utf-8") as f:
            f.write(ansi(tree, tree.NAME, stage_frames(1) + views + scale_frames()))
        written.append(os.path.join(folder, "frames.txt"))
    print("\n".join(written))
    return 0


if __name__ == "__main__":
    sys.exit(main(sys.argv[1:]))
