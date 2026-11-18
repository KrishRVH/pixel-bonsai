"""The sakura bonsai: a cherry in blossom, drawn on the art kit. The interface every variety keeps:

  NAME, ICON   what the caption calls it
  SIGNATURE    colors that mark it: every frame from "young tree" on holds one at scale 0.6, no other variety draws one
  pixels(g, scale=1.0)   {(x, y): "rrggbb"} for age g in [0, 1] on a canvas of art.size(scale), the same every call

A reddish seedling in a nursery pot grows into an informal upright cherry with a gentle S trunk and a broad crown of
blossom pads on a celadon pot, ending as a gnarled, hollow old tree under a drift of petals.
"""
import math

import art
from art import clamp, during, noise, since, steps, tone

NAME, ICON = "sakura", "🌸"
BARK = ("362130", "4c2b3c", "683d4b", "8a5360", "b07b85")        # glossy purplish red-brown, shadow to lit
GLOSS = "dcaab1"                                                 # the lit edge of the trunk, a streak of sheen
LENTICEL = ("a99a8d", "cdbdac")                                  # the pale horizontal dashes of cherry bark
STEM = (("a64f5c", "d0707b", "f0a0a8"), ("8a3f4a", "b8586a", "d87f8a"), BARK[1:4])   # a stem reddens, then browns
LEAF = ("4a3320", "6a5a2c", "7a8236", "8fa148", "a9bd6a")        # fresh bronze-green leaves, shadow to lit
SEED_LEAF = ("5f8a3c", "86b04f", "b4d46e")                       # the pale green seed leaves
BLOOM = ("6f3d68", "9a527f", "c9729b", "e895b6", "f6bcd1", "fddfe9", "ffffff")   # blossom cloud, mauve to white
CENTER, BUD = ("a8407a", "cf5f93"), "d94a86"                     # a flower's deep pink heart, a bud
POT = ("4e7d76", "6fa597", "98c8b6", "c4e6d6", "e2f6ec")         # celadon glaze, shadow to lit
SIGNATURE = (BLOOM[3], BLOOM[4], LEAF[2], LEAF[3])
SOIL = ("231710", "33221a", "4a3425", "5f4632")
SEED, TAG, WIRE = "c9a46a", "e6d9c8", ("a85d2c", "d38a45")
MOSS, STONE = ("35602a", "568c35", "7fb348", "a6d46b"), ("55555f", "7a7a86", "a0a0aa", "c4c4cc")
SHARI, STAND = ("7f7869", "a89f8c", "cfc6b2", "ece4d2"), ("3a2619", "5b3e29", "7d5a3a", "a07a52")
PLASTIC, DISH = ("2e3140", "474b60", "6a7092", "8f97b8"), ("a89c86", "d9cdb6", "f3ead8")
CX, NURSERY_Y, SOIL_Y = 24, 24.9, 22.8   # the trunk's foot, on the nursery soil and on the bonsai pot's
ANCHORS = ((0, 0), (0.8, 0.08), (0.2, 0.2), (-2.4, 0.32), (-4.2, 0.44), (-3.4, 0.56), (-0.8, 0.66), (2.4, 0.77),
           (3.4, 0.88), (1.6, 1.0))   # the finished S: sideways shift from the foot at each share of the height
HEIGHT = (("sprout", 3.2), ("seedling", 6), ("sapling", 9), ("whip", 13), ("young tree", 14.5),
          ("first branches", 15.2), ("first styling", 16), ("shaped", 16.6), ("mature", 17))
BEND = (("sapling", 0), ("young tree", 0.25), ("first wiring", 0.45), ("thickening trunk", 1))
WIDTH = (("sprout", 0.9), ("whip", 1.1), ("young tree", 1.8), ("first wiring", 2.4), ("thickening trunk", 3.2),
         ("shaped", 4.2), ("established", 4.7), ("mature", 5.3), ("ancient", 6.0), ("venerable", 6.4))
LIMB = (("first branches", 1.0), ("shaped", 2.0), ("mature", 2.3), ("ancient", 2.8))
REACH = (("first branches", 0.55), ("first styling", 0.7), ("shaped", 0.84), ("layered pads", 0.9),
         ("ramified", 0.94), ("established", 1.0), ("venerable", 1.07))
DROOP = (("shaped", 0), ("mature", 0.5), ("venerable", 1.0))
PADK = (("first branches", 0.5), ("first wiring", 0.55), ("first styling", 0.64), ("shaped", 0.9),
        ("venerable", 1.0))
LIMBS = (  # trunk share it leaves at, its pad's center and radii when finished, arch, stage born
    (0.24, (8.5, 16.8), (6.2, 2.9), 2.2, "first branches"),
    (0.56, (39.0, 15.3), (6.4, 3.0), 2.0, "first branches"),
    (0.66, (11.2, 8.6), (6.8, 3.2), 1.8, "first branches"),
    (0.78, (38.7, 7.2), (6.6, 3.1), 1.6, "first styling"),
    (0.28, (31.5, 19.8), (3.0, 1.6), 1.0, "shaped"),
)
LAYERS = (  # the limb whose pad it sits on, its shift (in that pad's radii) and radii (a share of them), stage born
    (2, (-0.3, -1.1), (0.62, 0.58), "layered pads"),
    (3, (0.3, -1.1), (0.62, 0.58), "layered pads"),
)
MOSS_AT = ((13.5, 2.2), (18.5, 1.6), (28.5, 1.8), (31.5, 1.6), (34.5, 2.2))   # moss patches: center x, radius
CASCADE = ((8.0, 18.6), (4.6, 22.0), (3.0, 1.6))   # twig from, pad center, radii: a pad hanging in full bloom
KNOTS = ((0.12, -1, "old"), (0.34, 1, "old"), (0.46, -1, "ancient"), (0.07, 1, "ancient"))   # share, side, born


def lerp(g, table):
    """A value that moves linearly through the (stage name, value) pairs of table, each reached at its stage's start."""
    pts = [(art.START[name], v) for name, v in table]
    if g <= pts[0][0]:
        return pts[0][1]
    for (a, va), (b, vb) in zip(pts, pts[1:]):
        if g <= b:
            return va + (vb - va) * (g - a) / (b - a)
    return pts[-1][1]


def soft(x, y, salt, cell):
    """Smooth fixed noise in [0, 1) at a point in units: blotches about cell units across."""
    fx, fy = x / cell, y / cell
    ix, iy = math.floor(fx), math.floor(fy)
    tx, ty = fx - ix, fy - iy
    tx, ty = tx * tx * (3 - 2 * tx), ty * ty * (3 - 2 * ty)
    top = noise(ix, iy, salt) * (1 - tx) + noise(ix + 1, iy, salt) * tx
    low = noise(ix, iy + 1, salt) * (1 - tx) + noise(ix + 1, iy + 1, salt) * tx
    return top * (1 - ty) + low * ty


def trunk_path(g, foot):
    top, bend = lerp(g, HEIGHT), lerp(g, BEND)
    return art.spline([(foot[0] + dx * bend, foot[1] - top * f) for dx, f in ANCHORS], 5)


FINAL = trunk_path(1, (CX, SOIL_Y))   # the finished trunk, which the branch plan is drawn against


def rich(c, tones):
    """At larger scales a finer run of tones, with a blend between each neighbouring pair."""
    if c.scale < 1.5:
        return tones
    return (tones[0],) + tuple(t for a, b in zip(tones, tones[1:]) for t in (art.mix(a, b, 0.5), b))


def glaze(c, cover, xl, xr, y0, y1, tones, salt):
    """Repaint a pot's cover as glaze: lit at the left, shaded to the right and at the foot, with a soft sheen."""
    tones = rich(c, tones)
    for (i, j), _ in cover.items():
        x, y = (i + 0.5) / c.scale, (j + 0.5) / c.scale
        shade = 0.95 - 0.8 * (x - xl) / (xr - xl) - 0.25 * (y - y0) / (y1 - y0)
        shade += 0.25 * max(0, 1 - abs(x - xl - 0.22 * (xr - xl)) / 2.5) * (1 - (y - y0) / (y1 - y0))
        c.put(i, j, tone(tones, clamp(shade), 0.18, ((i % 2) + 2 * (j % 2) + 0.5) / 4))


def wire(c, path, w0, w1):
    """Copper wire wound round a limb: thin diagonal turns."""
    c.coil(path, w0, w1, lambda across, shade: WIRE[1] if across < 0 else WIRE[0], pitch=3.2, twist=0.4, duty=0.32,
           grow=0.6)


def nursery(c, g):
    """The plastic nursery pot with its label, and the seed on the bare soil."""
    c.block(19, 24.3, 29, 25.2, SOIL, rounding=0.5, jitter=1)
    c.block(20.6, 25.6, 27.4, 29.6, PLASTIC, taper=0.1)
    c.block(18.4, 24.8, 29.6, 26.0, PLASTIC[1:], rounding=0.3)
    if c.scale >= 1.5:
        for x in (22, 24, 26):
            c.line(x, 26.4, x - 0.3, 29.3, PLASTIC[0])
    c.line(20, 23.2, 20.4, 24.6, STAND[2])
    c.block(19.2, 22.2, 21.4, 23.2, (TAG,))
    if during(g, "seed", "sprout"):
        c.blob(24, 24.7, 1.1, 0.65, (SEED,))
        c.line(25.4, 24.9, 27.4, 25.1, SOIL[0])


def accent(c):
    """A small dish planting beside the pot: a tuft of grass with two flowers on a mossy mound."""
    c.block(41, 25.8, 46, 27.6, DISH, taper=0.6, rounding=0.6)
    c.blob(43.5, 25.6, 2.2, 0.9, MOSS)
    for x, top, lean in ((42.0, 22.6, -0.8), (43.0, 21.8, -0.2), (44.0, 22.2, 0.3), (45.0, 22.8, 0.9)):
        c.line(x, 25.2, x + lean, top, MOSS[2 + (x > 43.5)])
    c.line(42.3, 24.0, 41.6, 21.4, MOSS[1])
    c.put(*c.at(41.6, 21.0), BLOOM[5])
    c.put(*c.at(45.4, 21.6), BLOOM[4])


def pot_body(c):
    """The pot's feet, body and rim: drawn again over the trunk's foot, which goes into the soil behind the lip."""
    for x in (15, 30):
        c.block(x, 27.2, x + 3, 28.0, POT[:2])
    body = c.block(12.6, 24.2, 35.4, 27.4, POT, taper=1.6, rounding=0.55)
    glaze(c, body, 12.6, 35.4, 24.2, 27.4, POT[:4], 3)
    rim = c.block(10.8, 23.2, 37.2, 24.8, POT[1:], rounding=0.7)
    glaze(c, rim, 10.8, 37.2, 23.2, 24.8, POT[1:], 4)


def above_rim(x, y):
    """A clip for what sits on the soil: moss, a stone or roots stay above the pot's rim."""
    return y < 23.3


def bonsai_pot(c, g):
    """The display stand and its accent from "show tree", the soil, moss and a stone, and the pot."""
    if during(g, "show tree"):
        c.block(7.5, 27.6, 46.5, 28.9, STAND, rounding=0.2)
        c.block(10, 28.9, 13.5, 30, STAND[:2])
        c.block(40.5, 28.9, 44, 30, STAND[:2])
        accent(c)
    c.block(11.8, 22.2, 36.2, 23.6, SOIL, rounding=0.6, jitter=1)
    pot_body(c)
    moss = steps(g, (("refined", 0.4), ("established", 1.0)))
    for k, (x, r) in enumerate(MOSS_AT if moss else ()):
        c.blob(x, 22.5, r * (0.5 + 0.7 * moss), 0.6 + 0.9 * moss, MOSS, salt=k, where=above_rim)
    if during(g, "established"):
        c.blob(24, 22.7, 12, 0.7, MOSS, jitter=1, where=above_rim)
        c.blob(14.2, 21.4, 4.0, 2.6, STONE, lump=0.15, where=above_rim)
        c.blob(13.4, 19.6, 2.2, 0.8, MOSS[1:])


def nebari(c):
    """The roots flaring out over the soil from the trunk's foot."""
    c.blob(CX + 0.4, 22.6, 5.0, 1.5, BARK, lump=0.1, where=above_rim)
    for side, reach, lift in ((-1, 7.5, 0), (-1, 4.8, 0.8), (1, 7.0, 0), (1, 4.5, 0.8)):
        c.stroke([(CX + side * 1.2, 21.2 - lift), (CX + side * reach * 0.5, 22.2), (CX + side * reach, 22.6)], 1.8, 0.7,
                 BARK[1:])


def leaf(c, x, y, side, size, tones, salt=0):
    """A pointed leaf leaving a stem at (x, y), up and out to one side."""
    c.polygon([(x, y), (x + side * size * 0.4, y - size * 0.5), (x + side * size * 0.9, y - size * 0.65),
               (x + side * size * 0.6, y - size * 0.05)], tones, salt=salt)


def stem_plant(c, g, foot):
    """The sprout, seedling, sapling and whip: a thin stem with seed leaves, then true leaves."""
    top = lerp(g, HEIGHT)
    x0, y0 = foot[0] + 0.5, foot[1]
    hook = during(g, "sprout", "seedling")
    lean = 1.1 if hook else 0.3
    spine = art.spline([(x0, y0), (x0, y0 - top * 0.5), (x0 + 0.3, y0 - top * 0.8), (x0 + lean, y0 - top)], 4)
    c.stroke(spine, 0.8, 0.8, STEM[round(steps(g, (("sapling", 1), ("whip", 2))))], jitter=0.3)
    x, y = spine[-1]
    if hook:
        for side in (-1, 1):
            leaf(c, x, y + 0.5, side, 3.2, SEED_LEAF, side)
            c.line(x0 + side * 1.2, y0, x0 + side * 2.4, y0, SOIL[3])
    elif during(g, "seedling", "sapling"):
        for side in (-1, 1):
            c.blob(x + side * 1.8, y + top * 0.5, 1.5, 0.75, SEED_LEAF, salt=side)
            leaf(c, x, y + 0.6, side, 2.8, LEAF[1:], side)
    else:
        heights = (0.42, 0.58, 0.74, 0.9) if during(g, "sapling", "whip") else (0.3, 0.42, 0.54, 0.66, 0.78, 0.9)
        for k, h in enumerate(heights):
            lx, ly = art.along(spine, h)
            leaf(c, lx, ly, 1 if k % 2 else -1, 3.0 - 0.4 * (k == len(heights) - 1), LEAF[1:], k)
        if not during(g, "sapling", "whip"):
            c.put(*c.at(x, y - 0.4), BUD)


def puffs(c, cx, cy, rx, ry, tones, k, extra=0):
    """The cover of a pad: a row of overlapping domes over an underside that curves up at its ends."""
    def under(x, y):
        return y < cy + ry * (0.7 - 0.5 * ((x - cx) / rx) ** 2)
    cover = c.blob(cx, cy + ry * 0.1, rx, ry * 0.7, tones, lump=0.06, salt=k, where=under)
    n = 3 + int(rx / 2.8) + extra
    for m in range(n):
        t = (m + 0.5) / n - 0.5
        size = 1 - 0.9 * abs(t)
        x = cx + t * 2 * rx * 0.8 + (noise(k, m) - 0.5) * 1.2
        w, h = rx * (0.3 + 0.12 * noise(k, m + 9)) * (0.8 + 0.4 * size), ry * (0.75 + 0.3 * size)
        cover.update(c.blob(x, cy - ry * 0.18 * size + (noise(k, m + 5) - 0.5) * ry * 0.3, w, h, tones, lump=0.1,
                            salt=k + m, where=under))
    return cover


def flowers(c, shades, k):
    """Five-dot blossoms, a deep pink center and four petals, one in most cells of a 4 pixel grid over a pad."""
    for ci in range(min(i for i, _ in shades) // 4, max(i for i, _ in shades) // 4 + 1):
        for cj in range(min(j for _, j in shades) // 4, max(j for _, j in shades) // 4 + 1):
            i, j = ci * 4 + int(noise(ci, cj, k) * 4), cj * 4 + int(noise(ci, cj, k + 1) * 4)
            shade = shades.get((i, j))
            if shade is None or noise(ci, cj, k + 2) > 0.92:
                continue
            petal = BLOOM[4 + (shade > 0.5) + (shade > 0.75)]
            for di, dj in ((-1, 0), (1, 0), (0, -1), (0, 1)):
                c.put(i + di, j + dj, petal)
            c.put(i, j, CENTER[shade > 0.45])


def pad(c, g, k, cx, cy, rx, ry):
    """A crown pad: a leafy clump that becomes a cloud of blossom, with leaves showing through."""
    flush = steps(g, (("shaped", 0.92), ("ramified", 0.94), ("refined", 0.95), ("mature", 0.96), ("venerable", 0.98)))
    count = round(steps(g, (("shaped", 4), ("ramified", 5), ("refined", 6))))
    lit = clamp(0.8 - (cx - CX) / 60 - (cy - 8) / 30)
    window = BLOOM[max(0, round(lit * (5.4 - count))):][:count] if count else LEAF
    buds = steps(g, (("young tree", 0.01), ("first branches", 0.03), ("first styling", 0.06), ("shaped", 0.02)))
    out, holes = (1 if cx > CX else -1), []
    if during(g, "refined") and rx > 4.5:
        holes = [(cx - out * rx * 0.2, cy + ry * 0.65), (cx + out * rx * 0.5, cy + ry * 0.55)]
    sharp = since(g, "refined", 0.03)
    behind = dict(c.px) if holes else {}
    cover = puffs(c, cx, cy, rx, ry, window, k, round(steps(g, (("ramified", 1), ("mature", 2)))))
    window, leaves = rich(c, window), rich(c, LEAF)
    shades = {}
    for (i, j), shade in cover.items():
        x, y = (i + 0.5) / c.scale, (j + 0.5) / c.scale
        if any(((x - hx) / 1.7) ** 2 + ((y - hy) / 1.5) ** 2 < 1 for hx, hy in holes):
            c.px.pop((i, j), None)   # puffs painted it: let what is behind show
            if (i, j) in behind:
                c.px[(i, j)] = behind[(i, j)]
            continue
        shade = 0.7 * shade + 0.15 * (1 + (x - cx) / rx * art.LIGHT[0] + (y - cy) / ry * art.LIGHT[1])
        shade += 0.2 * (soft(x, y, k, 2.4) - 0.5) - (0.3 + 0.2 * sharp) * max(0, (y - cy) / ry - 0.1)
        shade = clamp(0.5 + (1.4 + 0.5 * sharp) * (shade - 0.5))
        shades[(i, j)] = shade
        n, m = noise(i, j, k), noise(i, j, k + 31)
        if m < (1 - flush) * 1.4 * (1 - shade) or not flush:
            color = tone(leaves, 0.3 + 0.6 * shade, 0.6, n)
            if c.scale >= 1.5 and (i + 1, j) in cover:
                c.put(i + 1, j, color)
        else:
            color = tone(window, shade, 0.35, n)
            petals = (0.16 if c.scale < 1.5 else 0.08) * (1 + 0.5 * during(g, "venerable"))
            if noise(i, j, k + 17) < petals * shade * shade * (0.3 + 1.4 * soft(x, y, k + 3, 1.8)):
                color = BLOOM[6] if shade > 0.6 else BLOOM[5]
            elif noise(i, j, k + 19) < 0.035 * (1 + during(g, "venerable")) and 0.2 < shade < 0.8:
                color = CENTER[shade < 0.4]
        if m > 1 - buds:
            color = BUD
        c.put(i, j, color)
    if flush and c.scale >= 1.5:
        flowers(c, shades, k)
    halo = steps(g, (("shaped", 0.1), ("refined", 0.16), ("venerable", 0.32)))
    for i, j in list(shades) if halo else ():
        for p in ((i, j - 1), (i - 1, j), (i + 1, j)):
            if p not in shades and p not in c.px and noise(*p, k + 23) < halo:
                c.put(*p, BLOOM[3 + int(noise(*p, k + 24) * 3)])


def twigs(c, g, k, cx, cy, rx, ry):
    """From "ramified" a fan of fine twigs under a pad, drawn before it."""
    fan = round(steps(g, (("ramified", 2),)) * max(1.0, c.scale ** 0.5)) if rx > 4.5 else 0
    for m in range(fan):
        t = (m + 0.5) / fan - 0.5
        x, y = cx + t * rx, cy + ry * (0.7 - 0.5 * t * t)
        c.stroke([(cx + t * rx * 0.5, y - 0.6), (cx + t * rx * 0.8, y + 0.5), (x + t, y + 1.6 + noise(k, m))], 0.9, 0.4,
                 BARK[2:])


def branchlets(c, g, paths):
    """From "ramified" short forked twigs along each limb, reaching into the gaps between the pads."""
    if not during(g, "ramified"):
        return
    for k, (path, w0, _) in enumerate(paths):
        for m, at in enumerate((0.55, 0.75)):
            x, y = art.along(path, at)
            up = 1 if (k + m) % 2 else -1
            lean = 1 if noise(k, 90 + m) < 0.5 else -1
            ex, ey = x + lean * (1.4 + noise(k, 91 + m)), y + up * (2.4 + 1.2 * noise(k, 92 + m))
            c.stroke([(x, y), (x + lean * 0.4, y + up * 1.4), (ex, ey)], 0.9, 0.5, BARK[2:])
            c.stroke([(x + lean * 0.4, y + up * 1.6), (ex + lean * 1.6, ey - up * 0.2)], 0.7, 0.4, BARK[2:])


def fringe(c, g, k, cx, cy, rx, ry):
    """From "mature" weeping tips hanging from a big pad's outer rim, each ending in a blossom."""
    side = 1 if cx > CX else -1
    for m in range(2 if rx > 4.5 and cy > 14 and during(g, "mature") else 0):
        x = cx + side * rx * (0.55 + 0.25 * m)
        y = cy + ry * (0.7 - 0.5 * ((x - cx) / rx) ** 2)
        drop = 2.0 + 1.0 * noise(k, 80 + m)
        c.stroke([(x, y - 0.4), (x + side * 0.5, y + drop * 0.5), (x + side * 0.3, y + drop)], 0.6, 0.3, BARK[2:])
        c.put(*c.at(x + side * 0.3, y + drop), BLOOM[4 + m % 2])


def bark(c, g, trunk, width, cover):
    """Gloss, lenticels and knots on the trunk, a hollow and from "ancient" a vein of shari."""
    if during(g, "thickening trunk"):
        c.scatter(cover, BARK[4], 0.22, salt=8, test=lambda i, j, s: s > 0.78)
    if during(g, "refined"):
        c.scatter(cover, GLOSS, 1.0, salt=15, test=lambda i, j, s: s > 0.9)
    if during(g, "mature"):
        row = 2 + round(c.scale)
        for color, low, high in ((LENTICEL[0], 0.3, 0.55), (LENTICEL[1], 0.55, 0.9)):
            c.scatter(cover, color, 0.3, salt=5, run=2, test=lambda i, j, s: j % row == 0 and low < s < high)
    for share, side, born in KNOTS:
        if during(g, born):
            x, y = art.along(trunk, share)
            w = width * (1 - 0.58 * share)
            c.blob(x + side * w * 0.42, y, 1.3, 1.7, BARK, lump=0.2, salt=int(share * 99))
    if during(g, "old"):
        x, y = art.along(trunk, 0.44)
        x -= 0.4 * width * (1 - 0.58 * 0.44)
        c.stroke([(x + 0.5, y + 1.2), (x - 2.6, y - 0.2), (x - 5.6, y - 3.4)], 3.0, 0.5, SHARI, salt=7)
        fissures = steps(g, (("old", 0.03), ("ancient", 0.06)))
        c.scatter(cover, BARK[0], fissures, salt=14, test=lambda i, j, s: 0.2 < s < 0.8)
        at = round(0.2 * (len(trunk) - 1))
        x, y = trunk[at]
        w = width * (1 - 0.58 * 0.2)
        c.blob(x - 0.15 * w, y, 0.2 * w + 0.7, 2.3, SHARI[1:], lump=0.1, salt=3)
        c.blob(x - 0.1 * w + 0.2, y + 0.2, 0.2 * w, 1.8, ("120a0e", BARK[0]), salt=4)
        if during(g, "ancient"):
            vein = [(px - 0.22 * w, py) for px, py in trunk[at - 2:round(0.46 * (len(trunk) - 1))]]
            c.stroke(vein, 1.4, 0.6, SHARI, salt=6)


def petals(c, g):
    """Petals in the air from "ancient", and from "venerable" fallen on the moss, the pot and the stand."""
    drift = steps(g, (("ancient", 0.02), ("venerable", 0.035)))

    def free(i, j, shade):
        return (i, j) not in c.px and (i + 1, j) not in c.px

    c.scatter((1, 2, 47, 23), BLOOM[3:6], drift, salt=9, test=free)
    c.scatter((1, 2, 47, 23), BLOOM[4:6], drift * 0.4, salt=10, run=2, test=free)
    if during(g, "venerable"):
        c.scatter((11.8, 22.0, 36.2, 23.4), BLOOM[4:6], 0.3, salt=11, test=lambda i, j, s: (i, j) in c.px)
        c.scatter((10.8, 23.2, 37.2, 24.0), BLOOM[5:], 0.12, salt=12, test=lambda i, j, s: (i, j) in c.px)
        c.scatter((8, 27.6, 40, 28.2), BLOOM[4:6], 0.2, salt=13, test=lambda i, j, s: (i, j) in c.px)


def crown(g, trunk):
    """The branches and pads of the tree at g: (paths, pads), each path with its widths."""
    limb_w, reach, droop, padk = lerp(g, LIMB), lerp(g, REACH), lerp(g, DROOP), lerp(g, PADK)
    set_ = since(g, "first styling", 0.05)
    paths, made = [], {}
    for k, (share, (px, py), (rx, ry), rise, born) in enumerate(LIMBS):
        if not during(g, born):
            continue
        at = round(share * (len(trunk) - 1))
        (ax, ay), (fx, fy) = trunk[at], FINAL[at]
        dx, dy = (px - fx) * reach, (py - fy) * set_ - abs(px - fx) * 0.55 * reach * (1 - set_)
        end = (ax + dx, ay + dy + droop + ry * 0.3 * padk)
        arch = [(ax + 0.3 * dx, ay + 0.3 * dy - rise * 0.6), (ax + 0.65 * dx, ay + 0.65 * dy - rise)]
        path = art.spline([(ax, ay)] + arch + [end], 4)
        paths.append((path, limb_w, limb_w * 0.4))
        made[k] = (ax + dx, ay + dy + droop, rx * padk, ry * padk)
    pads = list(made.values())
    for k, (ox, oy), (sx, sy), born in LAYERS:
        if k in made and during(g, born):
            x, y, rx, ry = made[k]
            pads.append((x + ox * rx, y + oy * ry, rx * sx, ry * sy))
    if during(g, "young tree", "first branches"):
        for share, side in ((0.45, -1), (0.65, 1)):
            ax, ay = art.along(trunk, share)
            pads.append((ax + side * 2.6, ay - 1.6, 1.8, 1.0))
            paths.append(([(ax, ay), (ax + side * 1.5, ay - 0.8), (ax + side * 2.6, ay - 1.2)], 0.8, 0.5))
    tx, ty = trunk[-1]
    pads.append((tx, ty - 1.0, 5.9 * padk, 3.4 * padk))
    if during(g, "venerable"):
        (fx, fy), (px, py), (rx, ry) = CASCADE
        paths.append(([(fx, fy), (fx + (px - fx) * 0.4, fy + (py - fy) * 0.7), (px, py)], 0.8, 0.5))
        pads.append((px, py, rx, ry))
    return paths, sorted(pads, key=lambda p: p[1])


def pixels(g, scale=1.0):
    """The tree at age g in [0, 1]."""
    c = art.Canvas(scale)
    young = not during(g, "thickening trunk")
    foot = (CX, NURSERY_Y if young else SOIL_Y)
    if young:
        nursery(c, g)
    else:
        bonsai_pot(c, g)
    if not during(g, "sprout"):
        return c.px
    if not during(g, "young tree"):
        stem_plant(c, g, foot)
        return c.px
    trunk, width = trunk_path(g, foot), lerp(g, WIDTH)
    paths, pads = crown(g, trunk)
    wood = rich(c, BARK)
    for k, (path, w0, w1) in enumerate(paths):
        c.stroke(path, w0, w1, rich(c, BARK[:4]), salt=k)
    cover = c.stroke(trunk, width, width * 0.42, wood)
    if during(g, "first wiring", "thickening trunk"):
        for path, w0, w1 in [(trunk, width, width * 0.42)] + paths:
            wire(c, path, w0, w1)
    bark(c, g, trunk, width, cover)
    if not young:
        pot_body(c)
        if during(g, "established"):
            nebari(c)
        elif during(g, "thickening trunk"):
            c.blob(CX + 0.3, 22.9, 2.8, 0.9, BARK, lump=0.1, where=above_rim)
    branchlets(c, g, paths)
    for k, p in enumerate(pads):
        twigs(c, g, k + 10, *p)
    for k, p in enumerate(pads):
        pad(c, g, k + 10, *p)
    for k, p in enumerate(pads):
        fringe(c, g, k + 10, *p)
    if during(g, "ancient"):
        petals(c, g)
    return c.px
