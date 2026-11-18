"""The maple bonsai: a red Japanese maple (Acer palmatum) from a seed in a nursery pot to a venerable specimen on a
display stand. Fine, dense and fiery: tiers of palmate leaves from deep crimson through scarlet and orange to gold, on a
smooth grey trunk with a buttressed nebari, in an ivory pot. The interface every variety keeps:

  NAME, ICON   what the caption calls it
  SIGNATURE    colors that mark it: every frame from "young tree" on holds one at scale 0.6, no other variety draws one
  pixels(g, scale=1.0)   {(x, y): "rrggbb"} for age g in [0, 1] on a canvas of art.size(scale), the same every call

The tree is a trunk, the branches that carry its pads of foliage (PADS), and the ground it stands on. Growth is read
from stage names: a pad appears at its stage and grows for a while, the wire is on between two stages. Foliage is lit
on its upper left; a larger scale adds per-pixel leaf glyphs, bark streaks and finer twigs rather than bigger blocks.
"""
import math

import art
from art import LIGHT, START, clamp, during, noise, ramp, since, steps, tone, wobble

NAME, ICON = "maple", "🍁"
LEAF = ("5c0f2c", "8e1530", "b81c2e", "dc2f2a", "ee5026", "f6802c", "fab238", "fee08a")   # deep wine to pale gold
SIGNATURE = (LEAF[2], LEAF[3])
SPRING = ("2e5f2e", "4c8c38", "7fb648")                      # the green showing in young foliage
BARK = ("34303a", "524c54", "75706e", "958f88", "b5ada0")    # shadow to lit: smooth grey
TWIG = ("34262a", "54403a", "7a5e4e")
YOUNG = ("6a3430", "a04a3c", "d06a52")                       # a seedling's red stem
LICHEN, SCAR = ("c8c8b4", "9a9c88"), "241818"
POT = ("a89f88", "cfc6ac", "ebe3cc", "faf5e6")               # ivory glaze
SOIL = ("2b1f19", "3d2c22", "57402f")
MOSS = ("2f5a2a", "4f8a38", "7cb450", "a8d878")
GRASS = ("3f6e28", "6e9c38", "a8cc58")
STONE = ("4c5461", "6c7684", "909aa8")
DEADWOOD = ("a39b89", "d8cfba", "f0e8d4")
STAND = ("3a281c", "5a3e2a", "7e5a3c", "a07a52")
PLASTIC = ("2a2f3a", "454e60", "6a7690", "8a96b0")
CLAY = ("6a3e30", "8e5a44", "b4785a")
WIRE, SEED, SAMARA, LABEL = "c0773a", "6e4424", "caa468", "e4e0d0"

BASE, FULL = (24.0, 22.6), 15.0              # the trunk's foot on the soil and its full height, in units
TRUNK = ((0, 0), (0.2, 1.6), (0.4, -1.2), (0.6, -2.6), (0.8, -1.0), (1, 1.5))   # share of height, sideways offset
HEIGHT = (("sprout", 3), ("seedling", 6), ("sapling", 9), ("whip", 14), ("young tree", 14.5), ("first branches", 15),
          ("shaped", FULL))
WIDTH = (("sprout", 0.9), ("young tree", 1.7), ("first branches", 2.0), ("first wiring", 2.4),
         ("thickening trunk", 3.8), ("first styling", 4.2), ("shaped", 4.6), ("ramified", 4.7), ("mature", 5.6),
         ("old", 5.9), ("ancient", 6.6))
BEND = (("young tree", 0.25), ("first wiring", 1.0))
SPREAD = (("young tree", 0.5), ("shaped", 0.8), ("ramified", 0.9), ("venerable", 1.0))   # how many tones a pad spans
GAPS = (("refined", 0.5), ("old", 0.8))                      # sky showing through pads
MOTTLE = (("mature", 0.22), ("old", 0.3))                    # pale patches on the bark
PADS = (  # where it leaves the trunk (a share of its height), the pad's final centre, radius, and when it appears
    (1.00, 24.5, 4.0, 4.6, "young tree"), (0.58, 33.5, 11.0, 4.6, "young tree"), (0.52, 15.5, 11.4, 4.6, "young tree"),
    (0.82, 18.0, 7.4, 4.8, "first branches"), (0.78, 31.0, 7.0, 4.8, "first branches"),
    (0.30, 17.0, 14.8, 4.0, "first styling"), (0.25, 31.0, 15.2, 4.2, "first styling"),
    (0.72, 24.0, 8.6, 3.8, "shaped"),
    (0.50, 12.0, 10.6, 3.4, "layered pads"), (0.55, 36.5, 10.4, 3.4, "layered pads"),
    (0.40, 12.5, 14.4, 2.8, "layered pads"), (0.42, 36.0, 14.2, 2.8, "layered pads"))
NODES = ((5.0, "seedling"), (8.5, "sapling"), (11.5, "whip"), (14.0, "whip"))   # leaf pairs: height on the stem, from
FALLEN = ((17.0, 21.6, 0.35, 0), (19.8, 21.9, 0.7, 1), (29.0, 21.0, 0.5, 0), (31.2, 21.8, 0.3, 2),
          (10.8, 27.0, 0.65, 1), (35.4, 27.2, 0.4, 0))   # x, y, tone and glyph of a leaf lying on the moss and stand
DRIFT = ((38.5, 9.0, 0.7, 0), (42.5, 15.0, 0.4, 1), (35.5, 20.0, 0.8, 0), (8.5, 19.0, 0.6, 1), (4.5, 10.5, 0.45, 0),
         (43.5, 22.0, 0.7, 1), (3.0, 16.0, 0.8, 0))      # the same in the air: three from "ancient", all at "venerable"
STARS = {  # palmate leaf glyphs by width in pixels, each with five lobes
    1: ("#",),
    3: ((".#.", "###", "#.#"), ("#.#", "###", ".#."), ("#.#", ".#.", "#.#"), (".#.", "###", ".#.")),
    5: ((".#.#.", ".###.", ".###.", "#.#.#", "..#.."), ("..#..", "#.#.#", ".###.", ".###.", ".#.#."),
        ("..#..", ".###.", "#####", ".###.", ".#.#."))}
BAYER = ((0, 8, 2, 10), (12, 4, 14, 6), (3, 11, 1, 9), (15, 7, 13, 5))


def leaf_color(i, j, level, green, jitter=0.25):
    """The color of a leaf pixel at tone level in [0, 1], deep wine to gold, dithered; green shows in the low tones."""
    if green and level + 0.12 * (noise(i, j, 7) - 0.5) < green * 0.34:
        return tone(SPRING, level / (0.34 * green), 0.4, noise(i, j, 8))
    return tone(LEAF, level, jitter, 0.65 * (BAYER[j % 4][i % 4] + 0.5) / 16 + 0.35 * noise(i, j, 9))


def star(c, ci, cj, size, level, green, kind, jitter=0.25, heart=0.0):
    """A palmate leaf glyph size pixels wide centered on pixel (ci, cj), lit on its upper left; heart darkens its
    middle."""
    shapes = STARS[size]
    half = size // 2
    for dj, row in enumerate(shapes[kind % len(shapes)]):
        for di, ch in enumerate(row):
            if ch == "#":
                i, j = ci + di - half, cj + dj - half
                lit = ((di - half) * LIGHT[0] + (dj - half) * LIGHT[1]) / max(half, 1)
                c.put(i, j, leaf_color(i, j, level + 0.2 * lit - heart * (lit == 0 and size > 1), green, jitter))


def leaf(c, x, y, level, green=0.0, kind=0, jitter=0.25, heart=0.0):
    """A single leaf at (x, y) in units: a pixel on a small canvas, 3 across on a normal one, 5 on a large one."""
    size = 1 if c.scale < 0.8 else 3 if c.scale < 1.6 else 5
    star(c, math.floor(x * c.scale), math.floor(y * c.scale), size, level, green, kind, jitter, heart)


def lump(c, cx, cy, rx, ry, level, green, spread, holes, salt):
    """A rounded clump of leaves lit on its upper left, level its mean tone; returns {(i, j): level}."""
    cover = {}
    depth = 0.2 * clamp(c.scale - 1)          # on a larger canvas the leaf glyphs are lit over a darker mass
    jitter = 0.25 - 0.1 * clamp(c.scale - 1)
    for i, j, x, y in c.scan(cx - rx, cy - ry, cx + rx, cy + ry):
        nx, ny = (x - cx) / rx, (y - cy) / ry
        d = nx * nx + ny * ny
        if d > 1 or (d > 0.6 and noise(i, j, salt) < (d - 0.6) * 1.2):
            continue
        if any(((x - hx) / hr) ** 2 + ((y - hy) / (0.7 * hr)) ** 2 < 1 for hx, hy, hr in holes):
            continue
        cover[(i, j)] = lv = clamp(level + spread * 0.32 * (nx * LIGHT[0] + ny * LIGHT[1]) - 0.1 * d - depth)
        c.put(i, j, leaf_color(i, j, lv, green, jitter))
    return cover


def pad(c, cx, cy, r, level, green, spread, gaps, salt):
    """A pad of foliage r wide: a ring of leaf clumps round a heart, darker below, with sky showing through gaps of
    it; returns {(i, j): level}."""
    rx, ry = r, r * 0.58
    holes = [(cx + rx * (noise(salt, 20 + k) - 0.5) * 1.3, cy + ry * (noise(salt, 30 + k) - 0.3) * 0.8,
              0.8 + 0.4 * noise(salt, 40 + k)) for k in range(round(gaps * 4))]
    clumps = [(0.0, 0.1 * ry, 0.8 * rx, 0.75 * ry)]
    for k in range(8):
        a = 6.283 * (k + 0.6 * noise(salt, k)) / 8
        clumps.append((0.66 * rx * math.cos(a), 0.62 * ry * math.sin(a), 0.42 * rx, 0.55 * ry))
    cover = {}
    for k, (ox, oy, lx, ly) in enumerate(sorted(clumps, key=lambda p: -p[1])):
        lv = level - 0.34 * oy / ry - 0.12 * ox / rx
        cover.update(lump(c, cx + ox, cy + oy, lx, ly, lv, green, spread, holes, salt + k))
    return cover


def shade_behind(c, cx, cy, rx, ry, drop):
    """Cast a pad's shadow on the foliage already painted behind its upper rim: those leaves drop tones."""
    for i, j, x, y in c.scan(cx - rx - 1.5, cy - ry - 1.5, cx + rx + 1.5, cy):
        nx, ny = (x - cx) / (rx + 1.2), (y - cy + 0.4) / (ry + 1.2)
        color = c.px.get((i, j))
        if nx * nx + ny * ny < 1 and color in LEAF:
            c.px[(i, j)] = LEAF[max(0, LEAF.index(color) - drop)]


def fringe(c, cover, green, salt):
    """Leaves hanging from a pad's underside."""
    drop = max(1, round(0.9 * c.scale))
    for (i, j), lv in list(cover.items()):
        if (i, j + 1) not in cover and noise(i, j, salt) < 0.2:
            for k in range(1, 2 + int(noise(i, j, salt + 1) * drop)):
                c.put(i, j + k, leaf_color(i, j + k, lv - 0.15, green))


def leaves(c, cover, green, salt):
    """Leaf glyphs over a pad and just past its rim, thicker where it is lit: only a canvas larger than 1 has them."""
    size = 3 if c.scale < 2.6 else 5
    sx = size + 1 - size // 5
    cells = dict(cover)
    for (i, j), lv in cover.items():
        for p in ((i + 1, j), (i - 1, j), (i, j - 1)):
            cells.setdefault(p, lv - 0.1)
    rows, cols = [j for _, j in cells], [i for i, _ in cells]
    for row, j in enumerate(range(min(rows), max(rows) + 1, size)):
        for i in range(min(cols) + row % 2 * (sx // 2), max(cols) + 1, sx):
            ci, cj = i + round(noise(i, j, salt) * 2 - 1), j + round(noise(i, j, salt + 1) * 2 - 1)
            if (ci, cj) not in cells or noise(i, j, salt + 2) > cells[(ci, cj)] * 1.6 + 0.1:
                continue
            lv = cells[(ci, cj)] + 0.26 + 0.36 * (noise(i, j, salt + 3) - 0.5)
            star(c, ci, cj, size, lv, green, int(noise(i, j, salt + 4) ** 2 * 3), 0.1)


def twigs(r, salt):
    """The ends of fine twigs fanning up and out from a pad's heart past its rim, relative to the heart."""
    out = []
    for k in range(5):
        a = -3.14 + 0.78 * k + 0.2 * noise(salt, k)
        far = r * (1.05 + 0.2 * noise(salt, k + 9)) + 1.2
        out.append((math.cos(a) * far, math.sin(a) * far * 0.62))
    return out


def bark(c, path, w0, w1, mottle=0.0, salt=0):
    """A trunk, root or limb of smooth grey bark: shaded as a cylinder lit on the left, with faint streaks along it
    and, as it ages, pale mottling on its lit side. Returns the stroke's cover."""
    cover = {}
    for (i, j), (u, f, across, shade) in c.trace(path, w0, w1).items():
        s = shade - 0.16 * (noise(round(across * w0 / 1.6), int(u / 5), salt + 4) < 0.2)
        color = tone(BARK, s, 0.25, noise(i, j, salt))
        if mottle and shade > 0.4 and noise(i // 2, j // 2, salt + 5) < mottle:
            color = LICHEN[noise(i, j, salt + 6) < 0.4]
        elif noise(i, j, salt + 7) < 0.05 * mottle:
            color = BARK[0]
        c.put(i, j, color)
        cover[(i, j)] = shade
    return cover


def trunk_path(top, bend):
    """The trunk's centre line from its foot up to height top, leaning out by bend of its full curve."""
    return art.spline([(BASE[0] + bend * dx, BASE[1] - top * s) for s, dx in TRUNK], 5)


def along(path, share):
    """The point a share of the way along a path."""
    return path[round(share * (len(path) - 1))]


def bowl(c, cx, y0, y1, hw0, hw1, curve, tones, band=None):
    """A round-sided vessel: half width falling from hw0 at y0 to hw1 at y1 along a curve, shaded as a cylinder, with
    an incised band between two heights."""
    for i, j, x, y in c.scan(cx - hw0 - 1, y0, cx + hw0 + 1, y1):
        t = clamp((y - y0) / (y1 - y0))
        nx = (x - cx) / (hw1 + (hw0 - hw1) * (1 - t ** curve) ** 0.5)
        if abs(nx) <= 1:
            s = clamp(0.62 - 0.5 * nx - 0.05 * t - 0.16 * bool(band and band[0] <= y < band[1]))
            c.put(i, j, tone(tones, s, 0.04, noise(i, j)))


def nursery(c):
    """A plastic nursery pot: a lipped tub tapering to its foot, with ribs down its sides."""
    c.block(17.4, 23.0, 30.6, 24.2, PLASTIC[2:], jitter=0.1)
    c.block(18.4, 24.2, 29.6, 28.2, PLASTIC[1:2], taper=1.1)
    c.line(19.2, 24.6, 19.9, 27.8, PLASTIC[2])
    c.line(28.3, 24.8, 27.8, 27.8, PLASTIC[0])
    c.line(22.0, 24.6, 22.2, 27.8, PLASTIC[0])
    c.line(25.7, 24.6, 25.5, 27.8, PLASTIC[0])
    c.line(21.0, 28.0, 27.0, 28.0, PLASTIC[0])


def pot(c, g):
    """The nursery pot with its label and the samara in it, then from "thickening trunk" the ivory bonsai pot."""
    if during(g, "seed", "thickening trunk"):
        c.block(18.4, 22.4, 29.6, 23.4, SOIL, jitter=0.4)
        nursery(c)
        c.scatter((18.4, 22.4, 29.6, 23.2), SOIL[2], 0.2, salt=2)
        c.line(28.6, 22.4, 28.6, 20.4, LICHEN[0])
        c.block(27.7, 19.2, 29.5, 20.6, (LABEL,))
    else:
        c.block(16.2, 22.2, 31.8, 23.5, SOIL, rounding=0.4, jitter=0.4)
        bowl(c, 24, 23.2, 27.6, 8.4, 5.4, 1.8, POT, (25.3, 25.8))
        c.block(14.9, 23.2, 33.1, 24.3, POT, rounding=0.5)
        c.block(14.9, 24.3, 33.1, 24.8, POT[:1])
        c.block(18.4, 27.4, 29.6, 28.2, POT[:2], rounding=0.3)
    if during(g, "seed", "sprout"):
        c.blob(23.2, 22.1, 0.7, 0.6, (SEED,))
        c.stroke([(23.6, 21.9), (26.4, 20.7)], 0.9, 0.5, (SAMARA,))
        c.stroke([(23.6, 22.1), (26.2, 22.4)], 0.9, 0.5, (SAMARA,))


def ground(c, g):
    """What lies on the soil: moss, a stone with roots clasping it, the nebari, then the stand and its accent, and
    the fallen leaves."""
    if during(g, "refined"):
        c.scatter((16.2, 21.6, 31.8, 23.2), MOSS, 0.3 + 0.5 * since(g, "established", 0.004), salt=3)
    if during(g, "established"):
        for x, rx in ((17.5, 2.6), (21, 2.0), (28, 2.4), (31, 3.0)):
            c.blob(x, 22.4, rx, 1.2, MOSS, jitter=0.6, salt=int(x), where=lambda x, y: y < 23.3)
        c.blob(29.8, 21.6, 3.4, 1.8, STONE, jitter=0.4, where=lambda x, y: y < 23.3)
        for side, reach in ((-1, 7.5), (-1, 4.5), (1, 3.5)):
            bark(c, [(BASE[0], 21.6), (BASE[0] + side * reach * 0.5, 22.0), (BASE[0] + side * reach, 22.7)],
                 1.9, 0.5, 0.0, int(reach))
        bark(c, [(25.4, 21.4), (27.2, 20.9), (29.8, 20.7), (32.2, 22.3)], 1.9, 0.7, 0.0, 9)
    elif during(g, "thickening trunk"):
        c.blob(BASE[0], 22.0, 3.2, 1.3, BARK[1:4], where=lambda x, y: y < 23.2)
    if during(g, "show tree"):
        c.block(8, 28.2, 43, 29.2, STAND[1:], jitter=0.2)
        c.block(9, 29.2, 42, 30, STAND[:2])
        c.block(36.6, 26.2, 41.4, 28.2, CLAY, taper=0.8)
        for k, (dx, dy) in enumerate(((-2.6, -3.4), (-1.4, -4.4), (0.2, -4.8), (1.6, -4.0), (2.8, -2.6))):
            c.stroke([(39, 26.4), (39 + dx * 0.4, 26.4 + dy * 0.7), (39 + dx, 26.4 + dy)], 0.9, 0.3, GRASS, salt=k)
    if during(g, "venerable"):
        for x, y, level, kind in FALLEN:
            leaf(c, x, y, level, 0, kind)


def shoot(c):
    """The hooked shoot of "sprout" with its two seed leaves."""
    c.stroke([(24, 22.6), (24, 21.0), (24.5, 19.7), (25.8, 19.1), (26.9, 19.8), (27.1, 21.0)], 0.9, 0.9, YOUNG)
    c.stroke([(27.1, 20.8), (26.0, 22.0)], 1.0, 0.5, SPRING)
    c.stroke([(27.1, 20.8), (28.4, 21.9)], 1.0, 0.5, SPRING)


def stem(c, g, path, top):
    """A seedling's seed leaves and staggered pairs of true leaves along its stem, before it has a crown."""
    for h, born in NODES:
        if g >= START[born] and h <= top + 0.5:
            x, y = along(path, min(h, top) / top)
            for side, rise in ((-1, 0), (1, 1.3)):
                c.line(x + side * 0.7, y - rise, x + side * 1.6, y - rise - 0.4, YOUNG[1])
                leaf(c, x + side * 2.6, y - rise - 0.6, 0.38, 1.0, 1 - (side > 0), 0.1, 0.25)
    if during(g, "sprout", "whip"):
        x, y = along(path, 2.4 / top)
        for side in (-1, 1):
            c.stroke([(x, y), (x + side * 2.8, y - 0.5)], 1.0, 0.6, SPRING)


def limbs(c, g, path, width):
    """Draw the branches (and from "ramified" the twigs inside the pads); returns the branch paths and the pads they
    carry, each as (x, y, radius, index) at its present growth."""
    pads, branches = [], []
    for k, (at, x, y, r, born) in enumerate(PADS):
        if g < START[born]:
            continue
        ax, ay = along(path, at)
        grown = ramp(g, START[born], 0.12)
        ex, ey = ax + (x - ax) * (0.35 + 0.65 * grown), ay + (y - ay) * (0.6 + 0.4 * grown)
        limb = art.spline([(ax, ay), (ax + 0.6 * (ex - ax), ay + 0.3 * (ey - ay)), (ex, ey)], 4)
        branches.append(limb)
        c.stroke(limb, max(0.9, width * 0.3), 0.5, TWIG, salt=k)
        r *= 0.5 + 0.5 * grown
        if during(g, "ramified"):
            for tx, ty in twigs(r, k):
                c.stroke([(ex, ey), (ex + 0.8 * tx, ey + 0.8 * ty)], 0.6, 0.3, TWIG, salt=k)
        pads.append((ex, ey, r, k))
    return branches, pads


def wire(c, g, path, width, branches):
    """The copper wire coiled round the trunk and branches while they are first being bent."""
    if during(g, "first wiring", "thickening trunk"):
        c.coil(path, width + 0.5, width * 0.4 + 0.5, WIRE)
        for limb in branches:
            c.coil(limb, 1.4, 1.0, WIRE)


def trunk(c, g, path, width):
    """The trunk with its first root flare, the hollow of "old" and the deadwood vein of "ancient"."""
    taper = max(width * 0.4, 0.9)
    if g < START["young tree"]:
        c.stroke(path, width, taper, YOUNG)
        return
    mottle = steps(g, MOTTLE)
    bark(c, path, width, taper, mottle)
    if g >= START["thickening trunk"]:
        bark(c, path[:7], width * 1.3, width * 1.02, mottle, 3)
    if during(g, "old"):
        x, y = along(path, 0.26)
        c.blob(x - 0.3, y, width * 0.2, width * 0.34, BARK[2:], jitter=0.2)
        c.blob(x - 0.1, y + 0.2, width * 0.14, width * 0.27, (SCAR,))
    if during(g, "ancient"):
        vein = [(along(path, s)[0] - width * 0.2 * (1 - s) + 0.5 * wobble(s * 4), along(path, s)[1])
                for s in (0.04, 0.12, 0.2, 0.3, 0.42, 0.52)]
        c.stroke(vein, width * 0.17, width * 0.09, DEADWOOD, salt=3)


def foliage(c, g, pads):
    """Paint the pads from the top down, so each lower pad's lit crown overlaps the dark underside above it; from
    "ramified" the crown's rim ends in twigs and tufts, drawn under what is already painted."""
    green = 1 - ramp(g, START["first branches"], START["shaped"] - START["first branches"])
    spread, gaps = steps(g, SPREAD, 0.03), steps(g, GAPS)
    for x, y, r, k in sorted(pads, key=lambda p: p[1]):
        level = 0.40 + 0.04 * (g >= START["venerable"]) - 0.1 * (x - 24) / 20 - 0.08 * (y - 10) / 10
        shade_behind(c, x, y, r, r * 0.58, 1 + (g >= START["refined"]))
        cover = pad(c, x, y, r, level, green, spread, gaps, k * 11)
        fringe(c, cover, green, k * 5)
        if c.scale >= 1.4:
            leaves(c, cover, green, k * 3)
    if during(g, "ramified"):
        rim = art.Canvas(c.scale)
        for x, y, r, k in pads:
            for tx, ty in twigs(r, k):
                if c.at(x + tx, y + ty) not in c.px:
                    rim.stroke([(x + 0.7 * tx, y + 0.7 * ty), (x + tx, y + ty)], 0.6, 0.3, TWIG, salt=k)
                    leaf(rim, x + tx, y + ty - 0.3, 0.55, green, k)
        for p, color in rim.px.items():
            c.px.setdefault(p, color)


def pixels(g, scale=1.0):
    """The tree at age g in [0, 1]."""
    c = art.Canvas(scale)
    if g < START["seedling"]:
        pot(c, g)
        if g >= START["sprout"]:
            shoot(c)
        return c.px
    top, width = steps(g, HEIGHT), steps(g, WIDTH)
    path = trunk_path(top, steps(g, BEND))
    branches, pads = limbs(c, g, path, width)
    trunk(c, g, path, width)
    wire(c, g, path, width, branches)
    pot(c, g)
    ground(c, g)
    if g < START["young tree"]:
        stem(c, g, path, top)
    foliage(c, g, pads)
    if during(g, "ancient"):
        for x, y, level, kind in DRIFT[:3 + 4 * (g >= START["venerable"])]:
            leaf(c, x, y, level, 0, kind)
    return c.px
