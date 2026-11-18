"""The sakura bonsai: a cherry in blossom, drawn on the art kit. The interface every variety keeps:

  NAME, ICON   what the caption calls it
  SIGNATURE    colors that mark it: every frame from "young tree" on holds one at scale 0.6, no other variety draws one
  pixels(g, scale=1.0)   {(x, y): "rrggbb"} for age g in [0, 1] on a canvas of art.size(scale), the same every call

A reddish seedling in a nursery pot grows into an informal upright cherry with a gentle S trunk and a broad crown of
blossom pads on a celadon pot, ending as a gnarled, hollow old tree under a drift of petals. From "shaped" to "show
tree" the crown widens and rises, tiers of pads stack up, twigs, moss, a stone, bold bark and a stand arrive in turn.
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
         ("shaped", 4.2), ("layered pads", 4.9), ("ramified", 5.3), ("refined", 5.6), ("established", 6.0),
         ("mature", 6.8), ("show tree", 6.9), ("ancient", 7.4), ("venerable", 7.7))
LIMB = (("first branches", 1.0), ("shaped", 2.0), ("layered pads", 2.2), ("ramified", 2.4), ("refined", 2.6),
        ("established", 2.9), ("mature", 3.3), ("show tree", 3.5), ("ancient", 3.9))
REACH = (("first branches", 0.55), ("first styling", 0.7), ("shaped", 0.84), ("layered pads", 0.9),
         ("ramified", 0.94), ("refined", 0.97), ("established", 1.0), ("mature", 1.015), ("show tree", 1.03),
         ("venerable", 1.07))
STATURE = (("shaped", 0.9), ("layered pads", 0.92), ("ramified", 0.94), ("refined", 0.96), ("established", 0.98),
        ("mature", 0.99), ("show tree", 1.0))   # how much of its full height the crown has above the lower pads
DROOP = (("shaped", 0), ("mature", 0.5), ("venerable", 1.0))
PADK = (("first branches", 0.5), ("first wiring", 0.55), ("first styling", 0.64), ("shaped", 0.9),
        ("layered pads", 0.915), ("ramified", 0.93), ("refined", 0.94), ("established", 0.95), ("mature", 0.957),
        ("show tree", 0.965), ("venerable", 1.0))
LIMBS = (  # trunk share it leaves at, its pad's center and radii when finished, arch, stage born
    (0.24, (8.5, 16.8), (6.2, 2.9), 2.2, "first branches"),
    (0.56, (39.0, 15.3), (6.4, 3.0), 2.0, "first branches"),
    (0.66, (11.2, 8.6), (6.8, 3.2), 1.8, "first branches"),
    (0.78, (38.7, 7.2), (6.6, 3.1), 1.6, "first styling"),
    (0.28, (31.5, 19.8), (3.0, 1.6), 1.0, "shaped"),
)
TIERS = (30, 31, 32)   # the salts of the pads that cap others: the two over the upper pads and the one over the apex
LAYERS = (  # the limb whose pad it sits on, its shift (in that pad's radii) and radii (a share of them), stage born
    (2, (-0.25, -1.4), (0.88, 0.82), "layered pads"),
    (3, (0.25, -1.4), (0.88, 0.82), "layered pads"),
)
CAP = (("layered pads", 0.7), ("show tree", 1.0))   # the pad on the apex: how much of its full size it has
CROWN_Y = 18.0   # the height the crown rises from: the lower pads sit here and the top pads climb away from it
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


def stature(g):
    """How much of its full height the crown has: 1 before "shaped" and from "show tree" on."""
    return lerp(g, STATURE) if during(g, "shaped") else 1.0


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
    """A small dish planting beside the pot: a tuft of grass with three flowers on a mossy mound."""
    c.block(40.6, 25.6, 46.4, 27.6, DISH, taper=0.6, rounding=0.6)
    c.blob(43.5, 25.4, 2.9, 1.0, MOSS)
    blades = ((41.8, 21.8, -0.8), (42.8, 20.2, -0.2), (43.9, 19.4, 0.2), (45.0, 20.6, 0.9), (44.4, 22.6, 0.4))
    for x, top, lean in blades:
        c.line(x, 25.2, x + lean, top, MOSS[2 + (x > 43.5)])
    for x, y, petal in ((41.0, 21.0, BLOOM[5]), (43.9, 18.6, BLOOM[4]), (45.9, 19.8, BLOOM[5])):
        i, j = c.at(x, y)
        for di, dj in ((-1, 0), (1, 0), (0, -1), (0, 1)):
            c.put(i + di, j + dj, petal)
        c.put(i, j, CENTER[1])


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
        c.block(7.5, 27.6, 46.5, 29.1, STAND, rounding=0.2)
        c.block(10, 29.1, 13.5, 30, STAND[:2])
        c.block(40.5, 29.1, 44, 30, STAND[:2])
    c.block(11.8, 22.2, 36.2, 23.6, SOIL, rounding=0.6, jitter=1)
    pot_body(c)
    moss = steps(g, (("refined", 1.0), ("established", 1.5)))
    for k, (x, r) in enumerate(MOSS_AT if moss else ()):
        c.blob(x, 22.5, r * (0.6 + 0.6 * moss), 0.7 + 0.8 * moss, MOSS, salt=k, where=above_rim)
    if moss:
        c.blob(24, 22.7, 11.5, steps(g, (("refined", 0.95), ("established", 1.6), ("mature", 1.9))), MOSS, jitter=1,
               where=above_rim)


def nebari(c, g):
    """The roots flaring out over the soil from the trunk's foot, thicker from "mature"."""
    fat = 1 + steps(g, (("mature", 0.25),))
    c.blob(CX + 0.4, 22.7, 6.4 * fat, 1.7 * fat, BARK, lump=0.1, where=above_rim)
    for side, reach, lift in ((-1, 10.5, 0), (-1, 6.8, 1.0), (1, 10.0, 0), (1, 6.4, 1.0)):
        c.stroke([(CX + side * 1.5, 21.0 - lift), (CX + side * reach * 0.5, 22.1), (CX + side * reach, 22.8)],
                 2.8 * fat, 0.9, BARK[1:], salt=side)


def stone(c):
    """A big mossy stone on the soil in front of the tree's left side, a crack and moss on its top."""
    def inside(x, y):
        return ((x - 15.2) / 4.6) ** 2 + ((y - 21.2) / 3.7) ** 2 < 1 and y < 23.3

    c.blob(15.2, 21.2, 4.6, 3.7, STONE, lump=0.12, salt=5, where=above_rim)
    c.line(14.2, 20.4, 15.6, 22.0, STONE[0], test=lambda i, j: (i, j) in c.px)
    c.blob(14.4, 18.3, 4.0, 1.6, MOSS, lump=0.2, salt=6, where=inside)
    c.blob(19.4, 22.9, 1.5, 0.9, MOSS, salt=7, where=above_rim)


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
    lit = clamp(0.8 - (cx - CX) / 60 - (CROWN_Y - (CROWN_Y - cy) / stature(g) - 8) / 30)
    window = BLOOM[max(0, round(lit * (5.4 - count))):][:count] if count else LEAF
    buds = steps(g, (("young tree", 0.01), ("first branches", 0.03), ("first styling", 0.06), ("shaped", 0.02)))
    out, holes = (1 if cx > CX else -1), []
    if during(g, "refined") and rx > 4.5 and k not in TIERS:
        holes = [(cx - out * rx * 0.25, cy + ry * 0.55, 2.2, 1.5), (cx + out * rx * 0.5, cy + ry * 0.45, 1.8, 1.3)]
    sharp = since(g, "refined", 0.03)
    behind = dict(c.px) if holes else {}
    cover = puffs(c, cx, cy, rx, ry, window, k, round(steps(g, (("ramified", 1), ("mature", 2)))))
    window, leaves = rich(c, window), rich(c, LEAF)
    shades = {}
    for (i, j), shade in cover.items():
        x, y = (i + 0.5) / c.scale, (j + 0.5) / c.scale
        if any(((x - hx) / hrx) ** 2 + ((y - hy) / hry) ** 2 < 1 for hx, hy, hrx, hry in holes):
            c.px.pop((i, j), None)   # puffs painted it: let what is behind show
            if (i, j) in behind:
                c.px[(i, j)] = behind[(i, j)]
            continue
        shade = 0.7 * shade + 0.15 * (1 + (x - cx) / rx * art.LIGHT[0] + (y - cy) / ry * art.LIGHT[1])
        shade += 0.2 * (soft(x, y, k, 2.4) - 0.5) - (0.3 + 0.2 * sharp) * max(0, (y - cy) / ry - 0.1)
        shade = clamp(0.5 + (1.4 + 0.5 * sharp) * (shade - 0.5) + 0.12 * (k in TIERS))
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
    fan = round(steps(g, (("ramified", 3), ("mature", 2))) * max(1.0, c.scale ** 0.5)) if rx > 4.5 else 0
    for m in range(fan):
        t = (m + 0.5) / fan - 0.5
        x, y = cx + t * rx, cy + ry * (0.7 - 0.5 * t * t)
        end = (x + t * 1.5, y + 1.8 + 1.4 * noise(k, m))
        c.stroke([(cx + t * rx * 0.5, y - 0.6), (cx + t * rx * 0.8, y + 0.5), end], 1.0, 0.5, BARK[2:])


def sprigs(c, g, k, cx, cy, rx, ry):
    """From "ramified" until "established" twig tips with a bud on them, poking out of a pad's top and ends."""
    if not during(g, "ramified", "established") or rx < 4.5:
        return
    for m, t in enumerate((-0.75, -0.25, 0.3, 0.8)):
        x, y = cx + t * rx, cy - ry * (0.78 - 0.4 * t * t)
        lean = (noise(k, 60 + m) - 0.5) * 2.4
        tip = (x + lean, y - 1.4 - 1.0 * noise(k, 64 + m))
        if tip[1] < 1.5:
            continue
        c.line(x, y + 0.4, *tip, BARK[3], test=lambda i, j: (i, j) not in c.px)
        c.put(*c.at(*tip), BLOOM[3])


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
    """From "mature" weeping chains of blossom hanging from a pad's underside, longer on a show tree."""
    n = round(steps(g, (("mature", 3), ("show tree", 4)))) if rx > 4.5 and cy > 14 else 0
    for m in range(n):
        t = (m + 0.5) / n - 0.5
        x = cx + t * rx * 1.5
        y = cy + ry * (0.7 - 0.5 * t * t * 2.25)
        drop = (2.6 + 1.8 * noise(k, 80 + m)) * (1 + steps(g, (("show tree", 0.3),)))
        sway = 1 if noise(k, 70 + m) < 0.5 else -1
        c.line(x, y - 0.3, x + sway * 0.7, y + drop, BLOOM[2], test=lambda i, j: (i, j) not in c.px)
        c.put(*c.at(x + sway * 0.7, y + drop), BLOOM[4 + m % 2])


def bark(c, g, trunk, width, cover):
    """Gloss, lenticels and knots on the trunk, a hollow and from "ancient" a vein of shari."""
    if during(g, "thickening trunk"):
        c.scatter(cover, BARK[4], 0.22, salt=8, test=lambda i, j, s: s > 0.78)
    if during(g, "refined"):
        c.scatter(cover, GLOSS, 1.0, salt=15, test=lambda i, j, s: s > 0.9)
    if during(g, "mature"):
        row = 2 + round(c.scale)
        bold = during(g, "mature", "old")
        for color, low, high in ((LENTICEL[0], 0.25, 0.55), (LENTICEL[1], 0.5, 0.92)):
            c.scatter(cover, color, 0.65 if bold else 0.3, salt=5, run=3 if bold else 2,
                      test=lambda i, j, s: j % row == 0 and low < s < high)
    for share, side, born in KNOTS:
        if during(g, born):
            x, y = art.along(trunk, share)
            w = width * (1 - 0.58 * share)
            c.blob(x + side * w * 0.42, y, 1.3, 1.7, BARK, lump=0.2, salt=int(share * 99))
    if during(g, "old"):
        x, y = art.along(trunk, 0.44)
        x -= 0.4 * width * (1 - 0.58 * 0.44)
        k = width / 5.8
        c.stroke([(x + 0.5, y + 1.2), (x - 2.6 * k, y - 0.2 * k), (x - 5.6 * k, y - 3.4 * k)], 3.0 * k, 0.5, SHARI,
                 salt=7)
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
    drift = steps(g, (("ancient", 0.04), ("venerable", 0.055)))

    def free(i, j, shade):
        return (i, j) not in c.px and (i + 1, j) not in c.px

    c.scatter((0, 1, 48, 27), BLOOM[3:6], drift, salt=9, test=free)
    c.scatter((0, 1, 48, 27), BLOOM[4:6], drift * 0.4, salt=10, run=2, test=free)
    if during(g, "venerable"):
        c.scatter((11.8, 22.0, 36.2, 23.4), BLOOM[4:6], 0.3, salt=11, test=lambda i, j, s: (i, j) in c.px)
        c.scatter((10.8, 23.2, 37.2, 24.0), BLOOM[5:], 0.12, salt=12, test=lambda i, j, s: (i, j) in c.px)
        c.scatter((8, 27.6, 40, 28.2), BLOOM[4:6], 0.2, salt=13, test=lambda i, j, s: (i, j) in c.px)


def crown(g, trunk):
    """The branches and pads of the tree at g: (paths, pads), each path with its widths and each pad with its salt, so
    a pad keeps its blossoms as others join it."""
    limb_w, reach, droop, padk = lerp(g, LIMB), lerp(g, REACH), lerp(g, DROOP), lerp(g, PADK)
    height = stature(g)
    set_ = since(g, "first styling", 0.05)
    paths, made, pads = [], {}, []
    for k, (share, (px, py), (rx, ry), rise, born) in enumerate(LIMBS):
        if not during(g, born):
            continue
        at = round(share * (len(trunk) - 1))
        (ax, ay), (fx, fy) = trunk[at], FINAL[at]
        dx, dy = (px - fx) * reach, (py - fy) * set_ - abs(px - fx) * 0.55 * reach * (1 - set_)
        dy = CROWN_Y - (CROWN_Y - (ay + dy)) * height - ay
        end = (ax + dx, ay + dy + droop + ry * 0.3 * padk)
        arch = [(ax + 0.3 * dx, ay + 0.3 * dy - rise * 0.6), (ax + 0.65 * dx, ay + 0.65 * dy - rise)]
        paths.append((art.spline([(ax, ay)] + arch + [end], 4), limb_w, limb_w * 0.4))
        made[k] = (ax + dx, ay + dy + droop, rx * padk, ry * padk)
        pads.append((10 + k,) + made[k])
    for k, (limb, (ox, oy), (sx, sy), born) in enumerate(LAYERS):
        if limb in made and during(g, born):
            x, y, rx, ry = made[limb]
            pads.append((30 + k, x + ox * rx, y + oy * ry, rx * sx, ry * sy))
    if during(g, "young tree", "first branches"):
        for share, side in ((0.45, -1), (0.65, 1)):
            ax, ay = art.along(trunk, share)
            pads.append((20 + side, ax + side * 2.6, ay - 1.6, 1.8, 1.0))
            paths.append(([(ax, ay), (ax + side * 1.5, ay - 0.8), (ax + side * 2.6, ay - 1.2)], 0.8, 0.5))
    tx, ty = trunk[-1]
    top = CROWN_Y - (CROWN_Y - ty + 1.0) * height
    pads.append((40, tx, top, 5.9 * padk, 3.4 * padk))
    if during(g, "layered pads"):
        cap = lerp(g, CAP)
        pads.append((32, tx - 0.6, top - 2.5 * cap * padk, 4.2 * cap * padk, 2.0 * cap * padk))
    if during(g, "venerable"):
        (fx, fy), (px, py), (rx, ry) = CASCADE
        paths.append(([(fx, fy), (fx + (px - fx) * 0.4, fy + (py - fy) * 0.7), (px, py)], 0.8, 0.5))
        pads.append((41, px, py, rx, ry))
    pads.sort(key=lambda p: (p[0] in TIERS, p[2]))   # the tiers are drawn last, over the pads they cap
    if not during(g, "shaped"):   # before "shaped" a pad's blossoms follow its place in the crown, as they always did
        pads = [(10 + n,) + p[1:] for n, p in enumerate(pads)]
    return paths, pads


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
            nebari(c, g)
        elif during(g, "thickening trunk"):
            c.blob(CX + 0.3, 22.9, 2.8, 0.9, BARK, lump=0.1, where=above_rim)
    branchlets(c, g, paths)
    for p in pads:
        twigs(c, g, *p)
    for p in pads:
        pad(c, g, *p)
    for p in pads:
        sprigs(c, g, *p)
        fringe(c, g, *p)
    if during(g, "established"):
        stone(c)
    if during(g, "show tree"):
        accent(c)
    if during(g, "ancient"):
        petals(c, g)
    return c.px
