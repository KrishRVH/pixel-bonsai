"""The Japanese black pine bonsai (Pinus thunbergii) on the art kit: flat tiers of dark needle pads on a plated,
near-black trunk with rust in its cracks, silver candles standing on the pad tops, in a deep unglazed clay pot. The
interface every variety keeps:

  NAME, ICON   what the caption calls it
  SIGNATURE    colors that mark it: every frame from "young tree" on holds one at scale 0.6, no other variety draws one
  pixels(g, scale=1.0)   {(x, y): "rrggbb"} for age g in [0, 1] on a canvas of art.size(scale), the same every call
"""
import math
from collections import namedtuple

import art
from art import clamp, during, mix, noise, steps, tone, wobble

NAME, ICON = "black pine", "🌲"
NEEDLE = ("15323a", "1c4440", "26574a", "336a55", "4a8967", "7fb47c")    # shadow to lit tips
SIGNATURE = NEEDLE[1:4]
HALF = mix(NEEDLE[3], NEEDLE[4], 0.5)
TONES = (("young tree", NEEDLE[:5]), ("ramified", NEEDLE[:4] + (HALF, NEEDLE[4])),   # a pad's tones, shadow to lit:
         ("refined", NEEDLE[:4] + (HALF, NEEDLE[4], NEEDLE[5])),                    # more of them as it fills out
         ("venerable", NEEDLE[:4] + (HALF, NEEDLE[4], NEEDLE[5], "9ccf8f")))
PAD_COLORS = frozenset(color for _, tones in TONES for color in tones)
SPROUT = ("5f9a62", "8cc476", "bfe29a")                                   # seed needles and fresh growth
STEM = ("6f6a42", "9a9358", "c2b878")                                      # a young stem
OLIVE = ("5d5a3a", "857e4e", "aaa268")                                    # the same, going woody
CANDLE, TIP = ("b4c1b2", "dbe4d5"), "c6a468"                              # a candle's body, shadow to lit, and tip
BARK = ("251e1f", "342c2c", "463b39", "5d5046", "85756a")                  # fissure shadow, plates to lit edge
RUST = ("6a3a2a", "8a4c30")
OLD = ("7a5a34", "a07a45")                                                 # old needles
DEAD = ("8a8472", "bdb7a4", "e9e4d3", "fbf8ee")                            # deadwood, shadow to bleached
LICHEN = ("9fb87a", "c4d29a")
POT = ("33251f", "523c2f", "7a5c47", "a58565")                             # unglazed clay: shadow, body, lit rim
PLASTIC = ("2a303c", "3d4655", "5a6578")
SOIL = ("2b2118", "3d3022", "574429")
MOSS = ("2c5428", "3f7430", "5a963e", "7fb552")
STONE = ("4d4d55", "6e6e76", "9a98a0", "c0bdc2")
STAND = ("4a3322", "6b4a32", "8f6a4a", "b08a62")
SLATE = ("3d5a6e", "5d8199", "8fb0c4")
GRASS = ("4f7d3a", "78a84a", "a8cf72", "d7e8a0")
WIRE = ("a85d2a", "d58a46", "f0b472")
SEED = ("5c3d22", "8a6538")

SOIL_Y = 21.0                          # the soil line, the pot's top
BASE_Y, CX, FULL = 22.0, 24.0, 19.8    # the trunk's foot in the soil, its axis there and its full height
UNDER = SOIL_Y - 2.0                   # no pad reaches below this: two pixels of air over the soil
TRUNK = ((0.0, 0.0), (0.1, -0.8), (0.22, -2.0), (0.4, -0.3), (0.55, 2.0), (0.68, 2.4), (0.8, 0.8), (0.9, -0.4),
         (1.0, 0.3))
HEIGHT = (("sprout", 3.0), ("seedling", 6.0), ("sapling", 9.0), ("whip", 14.0), ("young tree", 15.5),
          ("first branches", 16.8), ("thickening trunk", 17.8), ("first styling", 18.8), ("shaped", FULL))
WIDTH = (("sprout", 0.7), ("whip", 1.2), ("young tree", 1.9), ("first branches", 2.3), ("thickening trunk", 4.0),
         ("shaped", 5.0), ("mature", 5.6), ("ancient", 6.4))
TAPER = (("sprout", 0.9), ("young tree", 0.6), ("thickening trunk", 0.42), ("shaped", 0.4))
BEND = (("young tree", 0.15), ("first wiring", 0.6), ("thickening trunk", 0.85), ("first styling", 1.0))
ROUGH = (("thickening trunk", 0.25), ("shaped", 0.5), ("mature", 0.85), ("ancient", 1.0))
REACH = (("young tree", 0.3), ("first branches", 0.45), ("thickening trunk", 0.6), ("first styling", 0.8),
         ("shaped", 1.0))
SIZE = (("young tree", 0.22), ("first branches", 0.27), ("thickening trunk", 0.34), ("first styling", 0.62),
        ("shaped", 0.98), ("layered pads", 0.99), ("ramified", 1.0), ("refined", 1.01), ("mature", 1.02),
        ("venerable", 1.04))
GAPS = (("refined", 0.3), ("mature", 0.45))

Pad = namedtuple("Pad", "inner base hw th rise born candles parent")
# inner: the distance from the trunk's axis to the pad's inner end (negative to its left), base: the underside y at the
# inner end, hw and th: its half width and the height of its dome, rise: how far its outer end stands higher, born:
# its stage, candles: the (stage, share of hw from the middle) of each candle on top, parent: the pad it grows from
# (inner and base are then from that pad's outer end)
PADS = (
    Pad(-3.6, 18.9, 6.0, 3.0, 1.0, "young tree", (("shaped", -0.8),), None),
    Pad(3.0, 12.6, 5.0, 2.6, 0.8, "young tree", (("shaped", 0.3), ("venerable", -0.5)), None),
    Pad(3.6, 18.5, 5.4, 3.0, 0.8, "first branches", (), None),
    Pad(-3.0, 12.8, 4.6, 2.6, 0.8, "first branches", (("layered pads", -0.4),), None),
    Pad(0.0, -1.0, 2.3, 2.2, 0.6, "layered pads", (("refined", 0.6),), 2),
    Pad(0.0, -1.0, 2.2, 1.8, 0.6, "layered pads", (("venerable", 0.0),), 3))
CROWN = Pad(0.0, 5.9, 4.2, 2.6, 0.0, "young tree", (("shaped", 0.0), ("venerable", -0.6)), None)
DIP = 1.4              # how far below a pad its branch leaves the trunk
JIN = (  # the pad (an index of PADS, CROWN being -1) whose outer end it grows from, angle (radians, y down), length,
         # stage
    (3, -2.2, 4.0, "old"), (1, -0.9, 3.8, "old"), (-1, -2.3, 3.4, "old"), (4, -0.7, 3.2, "venerable"))
PLATE = (2.8, 3.2)     # a bark plate's width and length in pixels, so a larger scale shows more plates


def trunk_x(share, bend):
    """The trunk's x at share of its full height: an informal S that leans back over its foot."""
    pts = art.spline(TRUNK, 6)
    for (s0, x0), (s1, x1) in zip(pts, pts[1:]):
        if s0 <= share <= s1 and s1 > s0:
            return CX + bend * (x0 + (x1 - x0) * (share - s0) / (s1 - s0))
    return CX + bend * pts[-1][1]


def trunk_at(h, bend):
    """The trunk's point h units above its foot."""
    return trunk_x(h / FULL, bend), BASE_Y - h


def bark(c, cells, half, rough, salt=0, dead=None, start=0.0, lift=0.0):
    """Paint a stroke's trace (c.trace) as plated bark: lit plates split by fissures, rust in the cracks. half(f) is
    its half width a fraction f along it, dead(u, across) says where deadwood bares it, start the distance along the
    tree this stroke begins at. Nothing is painted below the soil."""
    cover = {}
    for (i, j), (u, f, across, shade) in cells.items():
        if (j + 0.5) / c.scale > BASE_Y:
            continue
        u += start
        col = across * half(f) * c.scale / PLATE[0]
        ci = math.floor(col)
        row = u * c.scale / PLATE[1] + noise(ci, 3, salt)
        ri = math.floor(row)
        fc, fr = col - ci, row - ri
        if dead and dead(u, across):
            c.put(i, j, DEAD[0] if noise(i, j, salt + 8) < 0.12 else tone(DEAD[:3], shade, 0.3, noise(i, j, salt + 9)))
        elif (noise(ci, 5, salt) < 0.6 * rough and fc < 1 / PLATE[0]) or (
                fr < 1 / PLATE[1] and noise(ci, ri, salt + 6) < 0.4 * rough):
            rusty = rough > 0.8 and noise(ci, ri, salt + 4) < 0.4 * rough and shade > 0.3
            c.put(i, j, RUST[int(noise(ci, ri, salt + 7) * len(RUST))] if rusty else BARK[0])
        else:
            s = 0.2 + lift + 0.65 * shade ** 1.2 + (noise(ci, ri, salt + 1) - 0.5) * 0.3 * rough
            if fc < 2 / PLATE[0] and fr < 2 / PLATE[1]:
                s += 0.15 * rough
            c.put(i, j, tone(BARK, clamp(s), 0.2, noise(i, j, salt + 3)))
        cover[(i, j)] = shade
    return cover


def bark_stroke(c, path, widths, rough, salt=0, dead=None, lift=0.0):
    """A stroke of bark along a path, widths (w0, w1, ...) apart evenly along it."""
    cover, start = {}, 0.0
    for k, (w0, w1) in enumerate(zip(widths, widths[1:])):
        lo, hi = k * (len(path) - 1) // (len(widths) - 1), (k + 1) * (len(path) - 1) // (len(widths) - 1)
        part = path[lo:hi + 1]
        half = lambda f: (w0 + (w1 - w0) * f) / 2  # noqa: E731
        cover.update(bark(c, c.trace(part, w0, w1), half, rough, salt, dead, start, lift))
        start += sum(math.dist(a, b) for a, b in zip(part, part[1:]))
    return cover


def above(c, y):
    """A test for Canvas.line that keeps the pixels centered at or above the line y (in units)."""
    return lambda i, j: (j + 0.5) / c.scale <= y


def needles(c, x, y, length, count, colors, a0, a1, salt, bias=0.0, test=None, spread=0.5):
    """A starburst of needles from (x, y), length units long, fanned from angle a0 to a1 (radians, y down), darker at
    the root and lighter at the tip; a larger scale draws more of them. They are lit toward the upper left, more so by
    bias."""
    n = max(3, round(count * (0.55 + 0.45 * c.scale)))
    for k in range(n):
        a = a0 + (a1 - a0) * (k + 0.5) / n
        reach = length * (0.75 + 0.5 * noise(k, salt))
        lit = clamp(0.5 - 0.5 * (math.cos(a) * 0.6 + math.sin(a) * 0.8)) ** 1.6 * 0.9 + bias
        shift = (noise(k, salt + 1) - 0.5) * spread
        mid = (x + math.cos(a) * reach * 0.55, y + math.sin(a) * reach * 0.55)
        c.line(x, y, *mid, tone(colors, lit - 0.1 + shift), test)
        c.line(*mid, x + math.cos(a) * reach, y + math.sin(a) * reach, tone(colors, lit + 0.12 + shift), test)


def pad(c, x, base, hw, th, rise, look, salt, peak=2.4, point=0.0):
    """A pad of needles: a flat underside at y = base at its inner end and its outer end rise higher, a domed, scalloped
    top, and fans of lit needles along the top and at the ends, a leader of them on a pointed top (point). A larger
    scale draws more fans. Returns the pixel row of its top in each column."""
    p = noise(salt, 11) * 6.283
    out = 1 if rise >= 0 else -1                 # the side its outer end is on
    lift = lambda t: abs(rise) * (t * out + 1) / 2   # noqa: E731
    ridge = lambda t: base - lift(t) - th * (1 - abs(t) ** peak) ** 0.6 * (  # noqa: E731
        1 + 0.16 * math.sin(t * hw * 1.9 + p) + 0.1 * look.fringe * math.sin(t * hw * 3.3 + 2 * p))
    for i, j, px, py in c.scan(x - hw, base - th - 2, x + hw, base):
        t = (px - x) / hw
        if abs(t) >= 1:
            continue
        bottom = base - lift(t)
        high = ridge(t)
        if not high <= py <= min(bottom, UNDER):
            continue
        d = (py - high) / max(bottom - high, 1.0)
        c.put(i, j, tone(look.body, clamp(0.74 - 0.75 * d - 0.12 * t), 0.5, noise(i, j, salt)))
    fans = max(3, round(hw * 2 * c.scale * (1 + 0.5 * look.fringe) / 2.2))
    for k in range(fans + 2):
        t = -0.96 + 1.92 * (k + 0.15 + 0.7 * noise(k, salt + 2)) / fans if k < fans else (-0.99, 0.99)[k - fans]
        bottom = base - lift(t)
        fy = (bottom + ridge(t)) / 2 if k < fans else bottom - 0.45 * th
        a = -1.57 + (0.7 + 0.5 * abs(t) ** 2) * t if k < fans else (-3.0, -0.14)[k - fans] + 0.2 * out * (k - fans)
        length = (1.0 + 0.6 * noise(k, salt + 3)) * (1 + 0.4 * look.fringe * (abs(t) > 0.6))
        if point and abs(t) < 0.3:
            a, length = -1.57 + t, length + point
        needles(c, x + t * hw, fy, length, 4, look.tips, a - 0.6, a + 0.6, salt + k, bias=look.glow,
                test=above(c, min(bottom, UNDER)))
    for n in range(round(hw * c.scale * 0.8) * (c.scale >= 1.3)):
        t = (noise(n, salt + 40) - 0.5) * 1.7
        bottom = base - lift(t)
        fy = bottom - (bottom - ridge(t)) * (0.35 + 0.35 * noise(n, salt + 41))
        a = -1.57 + 0.7 * t
        needles(c, x + t * hw, fy, 1.3, 2, look.tips, a - 0.4, a + 0.4, salt + 60 + n, bias=look.glow - 0.1,
                test=above(c, min(bottom, UNDER)))
    for n in range(round(look.gaps * hw * c.scale / 2)):
        i, j = c.at(x + (noise(n, salt + 20) - 0.5) * 1.3 * hw, base - th * (0.25 + 0.3 * noise(n, salt + 21)))
        for hole in ((i, j), (i + 1, j)):
            if c.px.get(hole) in PAD_COLORS:
                del c.px[hole]
    tops = {}
    for i in range(math.floor((x - hw) * c.scale), math.ceil((x + hw) * c.scale)):
        rows = [j for j in range(max(0, math.floor((base - th - 3) * c.scale)), c.h)
                if c.px.get((i, j)) in PAD_COLORS and (j + 0.5) / c.scale <= base]
        if rows:
            tops[i] = min(rows)
    return tops


def candle(c, i, j):
    """A candle, a new silvery shoot, standing on the pad top at pixel (i, j) (its foot just above it): pale silver
    with a tan tip."""
    height, wide = max(2, round(2 * c.scale)), 1 + (c.scale >= 1.75)
    for k in range(height):
        for w in range(wide):
            c.put(i + w, j - 1 - k, TIP if k == height - 1 else CANDLE[1 - w] if wide > 1 else CANDLE[1])


def tuft(c, x, y, r, look, salt):
    """A young pine's foliage: a ball of needles r units long, dark in its heart and lit at the tips."""
    y = min(y, UNDER - 1.0)
    c.blob(x, y, r * 0.3, r * 0.28, look.tones[:2], jitter=0.5, salt=salt)
    needles(c, x, y, r * 1.1, round(2.6 * r), look.tones[1:5], -3.5, 2.8, salt, bias=0.1, test=above(c, UNDER))


def roots(c, g, look):
    """The trunk's foot over the soil: a flare from thickening trunk, nebari spreading from established."""
    if during(g, "thickening trunk"):
        c.blob(CX, BASE_Y - 0.4, look.base * 0.75, 1.3, BARK[1:], jitter=0.3, where=lambda x, y: y < BASE_Y)
    if during(g, "established"):
        for side in (-1, 1):
            for k in (0, 1):
                reach = look.base * (0.6 + 0.25 * k) + 2.5 + 2.4 * k
                path = [(CX + side * look.base * 0.25, BASE_Y - 1.4), (CX + side * look.base * 0.5, BASE_Y - 0.6),
                        (CX + side * reach, SOIL_Y + 1.4)]
                bark_stroke(c, path, (1.3 + 0.2 * look.base / 5, 0.5), look.rough, salt=9 + k)


def wire(c, path, w0, w1):
    """Copper wire wound round a stroke: thin diagonal stripes, lit on the upper left, none below the soil."""
    c.coil(path, w0, w1, lambda across, shade: WIRE[2] if shade > 0.6 else WIRE[1] if shade > 0.3 else WIRE[0],
           pitch=2.6, twist=0.4, duty=0.28, grow=0.6, test=lambda i, j: (j + 0.5) / c.scale < BASE_Y)


def deadwood(c, g, placed):
    """Bleached jin, stubs of dead branches, from old."""
    for k, (which, angle, length, born) in enumerate(JIN):
        if during(g, born):
            p = placed[which]
            side = 1 if p.x > p.ax else -1
            x, y = p.x + side * (p.hw + 0.4), p.base - p.th * 0.4
            end = (x + math.cos(angle) * length, y + math.sin(angle) * length)
            mid = (x + math.cos(angle) * length * 0.5 + 0.3, y + math.sin(angle) * length * 0.5 - 0.3)
            cover = c.stroke(art.spline([(x, y), mid, end], 3), 1.0, 0.2, DEAD, jitter=0.3, salt=k)
            c.scatter(cover, DEAD[0], 0.12, salt=3)


def shari(u, across):
    """Where the trunk's deadwood vein runs: a strip up its lit side that narrows to a point at both ends."""
    narrow = clamp(min(u - 1.0, 9.5 - u) / 2.5)
    return abs(across + 0.35 + 0.15 * wobble(u * 0.5)) < 0.4 * narrow


def drift(c, count, colors, salt):
    """Needles falling through the air, in the sky around the tree."""
    for n in range(round(count * (0.5 + 0.5 * c.scale))):
        x, y = 2 + 44 * noise(n, salt), 4 + 17 * noise(n, salt + 1)
        i, j = c.at(x, y)
        if (i, j) not in c.px:
            reach = 0.8 / c.scale * (1 + 0.5 * noise(n, salt + 2))
            c.line(x, y, x + reach, y + 1.6 / c.scale, colors[n % len(colors)])


def accent(c):
    """A little slate pot of grass beside the pot, on the stand."""
    c.block(38.6, 26.4, 43.6, 28.4, SLATE, taper=0.8, rounding=0.3)
    for k, (dx, lean, h) in enumerate(((-1.4, -0.7, 3.6), (-0.5, -0.2, 4.6), (0.5, 0.25, 4.0), (1.4, 0.7, 3.2))):
        x = 41.1 + dx
        c.stroke(art.spline([(x, 26.8), (x + lean * h * 0.4, 26.8 - h * 0.65), (x + lean * h, 26.8 - h)], 3), 0.9, 0.3,
                 GRASS, salt=k)


def clay(c, x0, y0, x1, y1, taper=0.0, under=0.0, lit=0.0, salt=0):
    """A box of unglazed clay: its left edge lit, its right side in shadow, a darker band of height under below its top
    edge or a lit line of height lit along it, a grain that shows more at a larger scale."""
    for i, j, x, y in c.scan(x0, y0, x1, y1):
        inset = taper * (y - y0) / (y1 - y0)
        left, right = x0 + inset, x1 - inset
        if not (y0 <= y < y1 and left <= x < right):
            continue
        s = 0.55 + (noise(i, j, salt) - 0.5) * 0.18
        if x - left < 1.0:
            s += 0.4
        elif x - left < 2.0:
            s += 0.12
        if right - x < 2.2:
            s -= 0.45
        if y - y0 < under:
            s -= 0.22
        if y - y0 < lit:
            s += 0.35
        if y1 - y < 0.8:
            s -= 0.18
        c.put(i, j, tone(POT, clamp(s), 0.1, noise(i, j, salt + 1)))


def above_rim(x, y):
    """A clip for the moss and stone on the soil: they stay above the pot's rim."""
    return y < 22.6


def ground(c, g):
    """The pot, soil and everything on the ground."""
    nursery = during(g, "seed", "thickening trunk")
    if nursery:
        c.block(19.6, SOIL_Y, 28.4, 22.4, SOIL, jitter=0.5)
        c.block(19.4, 23.6, 28.6, 28.0, PLASTIC, taper=1.3)
        c.block(18.8, 22.4, 29.2, 23.6, PLASTIC[1:])
    else:
        c.block(14.3, SOIL_Y, 33.7, 22.6, SOIL, jitter=0.5)
        clay(c, 13.2, 22.6, 34.8, 23.6, lit=0.9, salt=1)
        clay(c, 14.0, 23.6, 34.0, 27.4, taper=0.6, under=0.9, salt=2)
        for x in (15.4, 30.6):
            clay(c, x, 27.4, x + 3.0, 28.4, salt=3)
    if during(g, "seed", "sprout"):
        c.blob(23.4, 21.4, 1.5, 0.9, SEED)
        c.line(24.4, 20.9, 27.0, 20.1, "c9b890")
        c.line(20.5, 22.0, 22.4, 21.6, SOIL[0])
        c.line(25.5, 22.0, 27.5, 21.7, SOIL[0])
    if during(g, "refined", "established"):
        c.scatter((14.3, SOIL_Y, 33.7, 22.6), MOSS, 0.4)
    if during(g, "established"):
        for k in range(8):
            c.blob(15 + 2.8 * k, 21.7, 2.4, 1.0, MOSS, lump=0.2, salt=k, where=above_rim)
        c.blob(17.5, 21.5, 2.8, 1.3, STONE, where=above_rim)
    if during(g, "venerable"):
        c.scatter((14.3, SOIL_Y, 33.7, 22.4), OLD, 0.12, salt=5, run=2)
    if during(g, "show tree"):
        c.block(7.5, 28.4, 44.5, 29.2, STAND)
        c.block(9, 29.2, 12, 30, STAND[:2])
        c.block(40, 29.2, 43, 30, STAND[:2])
        accent(c)


def fascicle(c, x, y, angle, length, spread=0.8):
    """A small dark tuft of needles, a V of three with lit tips, from the node (x, y) leaning toward angle."""
    for da in (-spread, 0.0, spread):
        ex, ey = x + math.cos(angle + da) * length, y + math.sin(angle + da) * length
        c.line(x, y, ex, ey, NEEDLE[2])
        c.put(*c.at(ex, ey), NEEDLE[4])


def stem(c, g, look):
    """From the sprout to the whip: a thin stem; a whorl of seed needles on it, then sparse dark needle fascicles
    along it."""
    path = [(CX + 0.4 * wobble(k * 0.5, 3) * look.top / 14, BASE_Y - look.top * k / 12) for k in range(13)]
    c.stroke(path, look.base, look.base * 0.8, OLIVE if during(g, "whip") else STEM)
    x, y = path[-1]
    if during(g, "sprout", "sapling"):
        needles(c, x, y, 3.4 + 0.8 * during(g, "seedling"), 6, SPROUT, -3.2, 0.05, 1, spread=0.1)
        c.put(*c.at(x, y), SEED[1])
    if during(g, "sapling"):
        count = max(2, round(steps(g, (("sapling", 2), ("whip", 4)))))
        for k in range(count):
            fx, fy = art.along(path, 0.25 + 0.5 * k / (count - 1))
            fascicle(c, fx, fy, -1.57 + (1.2 if k % 2 else -1.2), 2.3, 0.55)
        fascicle(c, x, y, -1.57, 2.8, 0.9)
    if during(g, "whip"):
        candle(c, *c.at(x, y - 1.0))


Placed = namedtuple("Placed", "ax ay x base hw th rise spec")   # a pad where it is: its branch starts at (ax, ay)


def place(g, look):
    """Where each pad is at this age."""
    k = look.top / FULL
    out = {}
    for n, spec in enumerate(PADS + (CROWN,)):
        if not during(g, spec.born):
            continue
        hw, th = max(spec.hw * look.size, 0.9), max(spec.th * look.size ** 0.5, 1.0)
        if spec.parent is not None:
            near = out[spec.parent]
            side = 1 if near.x > near.ax else -1
            ax, ay = near.x + side * near.hw * 0.8, near.base - 0.5
            x, base = ax + side * (spec.inner + hw), ay + 0.4 + spec.base
        elif spec is CROWN:
            ax, ay = trunk_at(look.top, look.bend)
            x, base = ax + spec.inner, BASE_Y - (BASE_Y - spec.base) * k
        else:
            h = (BASE_Y - spec.base) * k
            tx = trunk_x(h / FULL, look.bend)
            side = 1 if spec.inner > 0 else -1
            x, base = tx + spec.inner * look.reach + side * hw, BASE_Y - h
            ax, ay = trunk_at(max(h - DIP * k, 0.5), look.bend)
        out[n] = Placed(ax, ay, x, base, hw, th, spec.rise * (1 if x >= ax else -1), spec)
    return list(out.values())


class Look:
    """The tree's features at age g, each keyed to a stage name."""

    def __init__(self, g):
        self.top = steps(g, HEIGHT)
        self.base = steps(g, WIDTH)
        self.taper = steps(g, TAPER)
        self.widths = [self.base * (1 - (1 - self.taper) * f ** 0.8) for f in (0, 1 / 3, 2 / 3, 1)]
        self.bend = steps(g, BEND)
        self.rough = steps(g, ROUGH)
        self.reach = steps(g, REACH)
        self.size = steps(g, SIZE)
        self.gaps = steps(g, GAPS)
        self.fringe = steps(g, (("ramified", 0.5), ("refined", 0.8), ("mature", 1.0)))
        self.tones = ([TONES[0][1]] + [tones for name, tones in TONES if during(g, name)])[-1]
        self.glow = -0.12 + steps(g, (("venerable", 0.06),))   # how lit the needle tips are
        self.body, self.tips = self.tones[1:5], self.tones[2:]


def tree(c, g, look):
    """From the young tree on: trunk, branches, foliage and what grows on them."""
    path = [trunk_at(look.top * k / 18, look.bend) for k in range(19)]
    placed = place(g, look)
    young = not during(g, "first styling")
    branches = []
    for k, p in enumerate(placed):
        if p.spec is CROWN or p.spec.parent is not None:
            continue
        side = 1 if p.x > p.ax else -1
        if young:
            end = (p.x, p.base - 0.3 * p.hw)
        else:
            end = (p.x - side * p.hw * 0.7, p.base - 0.8)
        mid = ((p.ax + end[0]) / 2, max(p.ay, end[1]) + 0.2)
        branch = art.spline([(p.ax, p.ay), mid, end], 4)
        c.stroke(branch, 0.18 * look.base + 0.4, 0.6, BARK[1:], salt=k)
        branches.append(branch)
    roots(c, g, look)
    shaft = lambda u, across: during(g, "ancient") and shari(u, across)  # noqa: E731
    trunk = bark_stroke(c, path, look.widths, look.rough, dead=shaft, lift=0.15 * (1 - look.rough))
    if during(g, "first wiring", "thickening trunk"):
        wire(c, path, look.base, look.base * look.taper)
        for branch in branches:
            wire(c, branch, 0.9, 0.6)
    if during(g, "ancient"):
        c.scatter(trunk, LICHEN, 0.05, salt=4, test=lambda i, j, s: s > 0.3 and (i, j) in c.px)
    for k, p in enumerate(placed):
        if young:
            tx, ty = (p.x, p.base - 0.3 * p.hw) if p.spec is not CROWN else (p.ax, p.ay - 0.5)
            tuft(c, tx, ty, p.hw if p.spec is not CROWN else max(p.hw, 3.0), look, 10 * k)
            continue
        crown = p.spec is CROWN
        tops = pad(c, p.x, p.base, p.hw, p.th, p.rise, look, 10 * k, 1.5 if crown else 2.4, 0.8 if crown else 0.0)
        for born, share in p.spec.candles:
            if during(g, born):
                i = math.floor((p.x + share * p.hw) * c.scale)
                if i in tops:
                    candle(c, i, tops[i])
    deadwood(c, g, placed)
    if during(g, "ancient"):
        drift(c, 7, OLD, 21)


def pixels(g, scale=1.0):
    """The tree at age g in [0, 1]."""
    c = art.Canvas(scale)
    look = Look(g)
    ground(c, g)
    if during(g, "young tree"):
        tree(c, g, look)
    elif during(g, "sprout"):
        stem(c, g, look)
    return c.px
