"""The sakura bonsai, a placeholder on the art kit (the scaffold's shared geometry in its colors) until the
variety's own art replaces it. The interface every variety keeps:

  NAME, ICON   what the caption calls it
  SIGNATURE    colors that mark it: every frame from "young tree" on holds one at scale 0.6, no other variety draws one
  pixels(g, scale=1.0)   {(x, y): "rrggbb"} for age g in [0, 1] on a canvas of art.size(scale), the same every call
"""
import math

import art
from art import during, since, steps

NAME, ICON = "sakura", "🌸"
BARK = ("2e1d26", "4f303a", "7a5058")          # shadow, body, lit
LEAF = ("5a4a2a", "7b6b38", "9c9248", "c0b86a")  # shadow to lit
BLOOM = ("e59ab6", "f7cbdb", "fff0f5", "c97a9c")
POT = ("4f7f78", "86b8ac", "c3e6d8")         # shadow, glaze, lit
SIGNATURE = (LEAF[1], LEAF[2])
SOIL, SEED, WIRE = ("3b2a1e",), "b08a52", "c0773a"
MOSS, STONE = ("3f6a2e", "6f9a3f", "98c25a"), ("5f5d58", "86837c", "aba79e")
DEADWOOD, STAND, PLASTIC = ("a39b89", "dcd4c4"), ("4a3322", "6b4a32", "8f6a4a"), ("1f232b", "2f3541", "4a5262")
CX, BASE_Y, FULL = 24, 22.5, 19.0   # the trunk's foot on the soil, and its full height, in units
HEIGHT = (("sprout", 3.5), ("seedling", 6), ("sapling", 9), ("whip", 14), ("young tree", 15), ("first branches", 16),
          ("thickening trunk", 17), ("first styling", 18), ("shaped", 19))
WIDTH = (("sprout", 1.0), ("young tree", 1.8), ("thickening trunk", 3.4), ("mature", 4.2), ("ancient", 5.4))
BRANCHES = (  # height on the trunk as a share of FULL, side, length, rise, the stage it appears at, pad radius
    (0.28, -1, 10, 0.25, "first branches", 3.6), (0.44, 1, 10, 0.3, "first branches", 3.6),
    (0.60, -1, 8, 0.35, "first branches", 3.2), (0.74, 1, 6, 0.4, "first branches", 2.8),
    (0.36, 1, 12, 0.15, "shaped", 3.8), (0.52, -1, 12, 0.2, "shaped", 3.8),
    (0.66, 1, 9, 0.3, "layered pads", 3.2), (0.82, -1, 5, 0.45, "layered pads", 2.6))


def trunk_at(h, bend):
    """The trunk's point h units above the soil: an informal S that leans back over its foot."""
    f = h / FULL
    return CX + bend * (math.sin(math.pi * 1.45 * f + 0.35) * (1 - 0.35 * f) - 0.34), BASE_Y - h


def pot(c, g):
    if during(g, "seed", "thickening trunk"):
        c.block(19, 23, 29, 28, PLASTIC, taper=1.5)
        c.block(18.5, 23, 29.5, 24, PLASTIC[1:])
        c.block(19.5, 22, 28.5, 23, SOIL)
    else:
        c.block(12, 24, 36, 27.5, POT, taper=1.5)
        c.block(11, 23, 37, 24.5, POT[1:])
        for x in (14, 32):
            c.block(x, 27.5, x + 2.5, 28, POT[:1])
        c.block(12, 22, 36, 23, SOIL)
    if during(g, "seed", "sprout"):
        c.blob(24, 22.3, 1.2, 0.8, (SEED,))
        c.line(21.5, 22.5, 22.8, 22.5, SOIL[0])
    if during(g, "refined"):
        c.scatter((12, 22, 36, 23), MOSS, 0.35 + 0.45 * since(g, "established", 0.004))
    if during(g, "established"):
        c.blob(15.5, 22, 2.6, 1.5, STONE, where=lambda x, y: y < 23)
        c.blob(30, 22.2, 4, 1.1, MOSS, where=lambda x, y: y < 23)
    if during(g, "show tree"):
        c.block(9, 28, 45, 30, STAND)
        c.block(39, 25, 44, 28, POT, taper=1)
        for k, x in enumerate((40.5, 41.5, 42.5)):
            c.stroke([(x, 25), (x + k - 1, 22 - k % 2)], 0.8, 0.4, MOSS, salt=k)


def pixels(g, scale=1.0):
    """The tree at age g in [0, 1]."""
    c = art.Canvas(scale)
    pot(c, g)
    top = steps(g, HEIGHT)
    if g < art.START["sprout"]:
        return c.px
    bend = steps(g, (("sapling", 0.5), ("young tree", 0.5), ("first wiring", 3.0)))
    wood = BARK if g >= art.START["young tree"] else MOSS
    width = steps(g, WIDTH)
    pads, paths = [], []
    if during(g, "sprout", "seedling"):
        stem = art.spline([(CX, BASE_Y), (CX, 20.5), (CX + 0.6, 19.3), (CX + 1.8, 19.0)], 4)
        c.stroke(stem, 1.0, 0.8, wood)
        c.blob(CX - 0.5, 19.3, 2.0, 0.9, LEAF[1:3])
        c.blob(CX + 4, 19.8, 2.0, 0.9, LEAF[1:3], salt=1)
        return c.px
    path = art.spline([trunk_at(top * k / 5, bend) for k in range(6)], 4)
    paths.append((path, width, width * 0.5))
    crown = steps(g, (("young tree", 3.2), ("first styling", 4.2), ("layered pads", 4.8)))
    if crown < 3.0:                       # leaves along the stem until the crown is there
        for k in range(round(steps(g, (("seedling", 2), ("sapling", 1), ("whip", 2))))):
            x, y = trunk_at(top * (0.35 + 0.6 * k / 4), bend)
            c.blob(x + (-1) ** k * 1.8, y - 0.3, 2.0, 0.9, LEAF[1:3], salt=k)
    clump = steps(g, (("first branches", 0.7), ("first styling", 1.0)))
    for share, side, length, rise, born, radius in BRANCHES:
        if g < art.START[born]:
            continue
        grown = 0.6 + 0.4 * since(g, born, 0.08)
        (x, y), reach = trunk_at(share * FULL, bend), length * grown
        branch = art.spline([(x, y), (x + side * 0.3 * reach, y + 0.6),
                             (x + side * 0.7 * reach, y - 0.3 * rise * reach),
                             (x + side * reach, y - rise * reach - 1)], 4)
        paths.append((branch, 0.9, 0.5))
        pads.append((branch[-1][0] + side, branch[-1][1] - 1, radius * grown * clump))
        if during(g, "layered pads"):
            sx, sy = branch[len(branch) * 11 // 20]
            sub = [(sx, sy), (sx + side * 2, sy - 2), (sx + side * 3.5, sy - 3.2)]
            paths.append((sub, 0.6, 0.4))
            pads.append((sub[-1][0], sub[-1][1] - 1, 2.2 * grown))
    if during(g, "established"):          # nebari over the soil
        for side in (-1, 1):
            c.stroke([(CX, BASE_Y), (CX + side * 4, BASE_Y + 0.3), (CX + side * 8, BASE_Y + 0.6)], 2.0, 0.6, BARK)
    elif during(g, "thickening trunk"):   # the first root flare
        c.blob(CX, 22.2, 3.4, 1.4, BARK, where=lambda x, y: y < 23)
    covers = [c.stroke(p, w0, w1, wood, salt=k) for k, (p, w0, w1) in enumerate(paths)]
    if during(g, "first wiring", "thickening trunk"):
        for p, w0, w1 in paths:
            c.coil(p, w0 + 0.5, w1 + 0.5, WIRE)
    if during(g, "mature"):
        c.scatter(covers[0], BARK[0], 0.3, salt=5, test=lambda i, j, s: s < 0.7)
    if during(g, "ramified"):
        for k, (x, y, r) in enumerate(pads):
            for a in (-2.3, -1.5, -0.7):
                c.stroke([(x, y), (x + math.cos(a) * r * 1.7, y + math.sin(a) * r * 1.5)], 0.7, 0.3, BARK, salt=k)
    if during(g, "old"):
        for h, side in ((0.5, -1), (0.8, 1)):
            x, y = trunk_at(h * top, bend)
            c.stroke([(x, y), (x + side * 2.5, y - 3.5)], 1.6, 0.6, DEADWOOD)
    if during(g, "ancient"):
        c.stroke([trunk_at(0.5 * top * k / 3, bend) for k in range(4)], 1.2, 0.8, DEADWOOD)
    if crown:
        x, y = trunk_at(top, bend)
        pads.append((x, y - 1.2, crown))
    tones = LEAF if during(g, "refined") else LEAF[:3] if during(g, "ramified") else LEAF[1:3]
    for k, (x, y, r) in enumerate(sorted(pads, key=lambda p: -p[1])):
        cover = c.cloud(x, y, r, r * 0.62, tones, gaps=0.07 if during(g, "refined") else 0, salt=k)
        bloom = steps(g, (("shaped", 0.35), ("ramified", 0.45), ("venerable", 0.7)))
        if bloom:
            c.scatter(cover, BLOOM, bloom, salt=50 + k, test=lambda i, j, s: s > 0.35)
    if during(g, "ancient"):
        c.scatter((4, 3, 44, 22), BLOOM, 0.012, salt=9, test=lambda i, j, s: (i, j) not in c.px)
    if during(g, "venerable"):
        c.scatter((12, 22, 36, 23), BLOOM, 0.3, salt=11)
    return c.px
