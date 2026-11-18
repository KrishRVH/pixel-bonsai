"""The bonsai's ladder of growth stages and the drawing kit every variety draws with. It imports no variety.

A tree is a function of its age g in [0, 1] and a scale: pixels(g, scale) in a variety module. This module gives them

  STAGES, stage(g)        the 20 named stages: (start age, name), and the name of the stage g is in; index(g) is its
                          position, ramp(g, start, span) a plain 0 to 1 ramp in age
  since(g, name, span)    0 before the stage called name starts, rising to 1 over span of age: key features off stage
                          names with it, so the ladder and the drawings cannot drift apart
  steps(g, table, span)   a value that steps to each (stage name, value) of a table as that stage starts
  during(g, first, last)  True from the start of stage first until the start of stage last (the end when None), for
                          a feature that is only there a while: wire, the nursery pot
  noise, wobble           fixed pseudo-random values for a pixel or a position, never random state
  Canvas(scale)           a drawing: geometry is in base units (W by H, 48 by 30, x right, y down, light from the
                          upper left) times scale, and drawing hands back covers ({(i, j): shade}) to detail further
    stroke, coil, trace   a tapered stroke along a path (trunk, branch, twig, root), wire wound round one, and the
                          low-level geometry of a stroke with its distance along and across
    blob, cloud           a shaded, dithered ellipse and a lumpy mass of them: foliage, moss, stones
    block, polygon        a shaded box with taper and rounding, and a shaded polygon: pots, stands, stones, deadwood
    line, scatter         one-pixel lines (needles) and per-pixel detail on a cover or in a box: blossoms, bark
                          flecks, petals; placed per output pixel, so a larger scale shows more of them
  size(scale), fit(w, rows)   the canvas in pixels for a scale, and the scale a pane of that size holds
  tone, mix, spline       a color from tones by shade, a blend of two colors, a smooth path through points
Colors are "rrggbb" strings and tones run from shadow to lit; build a long run of them with mix. Every shape takes
light, the unit vector toward the light, LIGHT (upper left) unless a drawing needs another.
Python 3.9, standard library.
"""
import math

W, H = 48, 30            # the base canvas in pixel units: 48 cells wide, 15 rows of half blocks tall
ROWS = H // 2
MIN_SCALE, MAX_SCALE = 0.5, 3.0
LIGHT = (-0.6, -0.8)     # the unit vector toward the light, upper left

STAGES = ((0.000, "seed"), (0.015, "sprout"), (0.040, "seedling"), (0.070, "sapling"), (0.100, "whip"),
          (0.140, "young tree"), (0.190, "first branches"), (0.240, "first wiring"), (0.290, "thickening trunk"),
          (0.340, "first styling"), (0.390, "shaped"), (0.450, "layered pads"), (0.510, "ramified"),
          (0.570, "refined"), (0.630, "established"), (0.690, "mature"), (0.750, "show tree"), (0.810, "old"),
          (0.870, "ancient"), (0.940, "venerable"))
START = {name: start for start, name in STAGES}


def index(g):
    """The position of g's stage in STAGES."""
    return sum(g >= start for start, _ in STAGES[1:])


def stage(g):
    return STAGES[index(g)][1]


def ramp(g, start, span):
    """0 before start, rising to 1 at start + span."""
    return max(0.0, min(1.0, (g - start) / span))


def since(g, name, span=0.05):
    """0 before the stage called name starts, rising to 1 over span of age."""
    return ramp(g, START[name], span)


def steps(g, table, span=0.004):
    """A value that is 0 until the first stage of table, ((stage name, value), ...), and moves to each value in turn
    over span of age from that stage's start. Frames are judged within 0.005 of a stage's start, so keep span below
    that for a feature that must show at once."""
    v = 0.0
    for name, to in table:
        v += (to - v) * since(g, name, span)
    return v


def during(g, first, last=None):
    """Whether g is from the start of stage first until the start of stage last (for ever when last is None)."""
    return g >= START[first] and (last is None or g < START[last])


def noise(x, y, salt=0):
    """A fixed pseudo-random value in [0, 1) for a pixel, so texture and blossoms stay put as the tree grows."""
    h = (int(x) * 374761393 + int(y) * 668265263 + int(salt) * 2246822519) & 0xFFFFFFFF
    h = ((h ^ (h >> 13)) * 1274126177) & 0xFFFFFFFF
    return ((h ^ (h >> 16)) & 0xFFFF) / 65536


def wobble(t, salt=0):
    """A smooth fixed wander in [-1, 1] along t, for a trunk's irregularity or an uneven edge."""
    a, b = noise(salt, 1) * 6.283, noise(salt, 2) * 6.283
    return 0.6 * math.sin(t * 1.7 + a) + 0.4 * math.sin(t * 3.9 + b)


def size(scale):
    """The canvas in pixels at a scale: round(W * scale) wide, round(H * scale) high rounded up to even."""
    h = round(H * scale)
    return round(W * scale), h + h % 2


def fit(width, rows):
    """The scale a pane width cells wide and rows tall (its last row is the caption) holds: as large as fits,
    floored to an eighth and capped at 3; 0 below 0.5, where the caption shows alone."""
    scale = math.floor(min(width / W, 2 * (rows - 1) / H, MAX_SCALE) * 8 + 1e-9) / 8
    return scale if scale >= MIN_SCALE else 0.0


def mix(a, b, t):
    """The color a fraction t of the way from a to b."""
    ca, cb = (tuple(int(c[i:i + 2], 16) for i in (0, 2, 4)) for c in (a, b))
    return "%02x%02x%02x" % tuple(round(p + (q - p) * t) for p, q in zip(ca, cb))


def tone(tones, s, jitter=0.0, n=0.5):
    """The color of shade s in [0, 1] (0 shadow, 1 lit) from tones. n, a noise value in [0, 1), moves s by up to
    jitter / 2 either way, so the edge between two tones breaks up into dither."""
    return tones[max(0, min(len(tones) - 1, int((s + (n - 0.5) * jitter) * len(tones))))]


def spline(points, steps=6):
    """A smooth path through points (Catmull-Rom), steps segments between each pair."""
    pts = [points[0]] + list(points) + [points[-1]]
    out = []
    for p0, p1, p2, p3 in zip(pts, pts[1:], pts[2:], pts[3:]):
        for k in range(steps):
            t = k / steps
            out.append(tuple(0.5 * (2 * b + (c - a) * t + (2 * a - 5 * b + 4 * c - d) * t * t +
                                    (3 * b - a - 3 * c + d) * t ** 3) for a, b, c, d in zip(p0, p1, p2, p3)))
    return out + [tuple(points[-1])]


def clamp(v):
    return max(0.0, min(1.0, v))


class Canvas:
    """A drawing at a scale. Shapes take base units (x right, y down); the pixels land in px, {(i, j): color}, on a
    canvas of size(scale), never outside it. Later drawing paints over earlier. Each shape returns its cover,
    {(i, j): shade} with shade in [0, 1] (1 lit), for scatter and for tests of where detail goes."""

    def __init__(self, scale=1.0):
        self.scale = scale
        self.w, self.h = size(scale)
        self.px = {}

    def put(self, i, j, color):
        """Set the pixel (i, j) if it is on the canvas."""
        if 0 <= i < self.w and 0 <= j < self.h:
            self.px[(i, j)] = color

    def at(self, x, y):
        """The pixel holding the point (x, y) in units."""
        return math.floor(x * self.scale), math.floor(y * self.scale)

    def scan(self, x0, y0, x1, y1):
        """(i, j, x, y) for the pixels overlapping a box in units, x and y being the pixel's center in units."""
        s = self.scale
        for j in range(max(0, math.floor(y0 * s)), min(self.h, math.ceil(y1 * s))):
            for i in range(max(0, math.floor(x0 * s)), min(self.w, math.ceil(x1 * s))):
                yield i, j, (i + 0.5) / s, (j + 0.5) / s

    def trace(self, path, w0, w1=None, light=LIGHT):
        """The pixels of a stroke along a path (a list of (x, y) in units, see spline) w0 wide at its start and
        w1 at its end, as {(i, j): (u, f, across, shade)}: u the distance along it in units, f that as a fraction
        of its length, across -1 to 1 from one edge to the other, shade as in a cover. It is at least a pixel wide."""
        w1 = w0 if w1 is None else w1
        segs, total = [], 0.0
        for (ax, ay), (bx, by) in zip(path, path[1:]):
            n = math.hypot(bx - ax, by - ay)
            if n:
                segs.append((ax, ay, bx - ax, by - ay, n, total))
                total += n
        if not segs:
            (ax, ay), total = path[0], 1e-6
            segs = [(ax, ay, 1e-6, 0.0, 1e-6, 0.0)]
        floor = 0.6 / self.scale
        reach = max(w0, w1) / 2 + floor
        best = {}
        for ax, ay, dx, dy, n, u0 in segs:
            for i, j, x, y in self.scan(min(ax, ax + dx) - reach, min(ay, ay + dy) - reach,
                                        max(ax, ax + dx) + reach, max(ay, ay + dy) + reach):
                t = clamp(((x - ax) * dx + (y - ay) * dy) / (n * n))
                f = (u0 + t * n) / total
                half = max((w0 + (w1 - w0) * f) / 2, floor)
                ox, oy = (x - ax - t * dx) / half, (y - ay - t * dy) / half
                d = ox * ox + oy * oy
                if d <= 1 and ((i, j) not in best or d < best[(i, j)][0]):
                    best[(i, j)] = (d, u0 + t * n, f, (ox * -dy + oy * dx) / n,
                                    clamp(0.5 + 0.5 * (ox * light[0] + oy * light[1])))
        return {p: v[1:] for p, v in best.items()}

    def stroke(self, path, w0, w1=None, tones=(), jitter=0.25, salt=0, light=LIGHT):
        """Paint a tapered stroke (trunk, branch, twig, root) in tones, lit on its upper left edge; returns its
        cover. With no tones it only measures."""
        cover = {}
        for (i, j), (_, _, _, shade) in self.trace(path, w0, w1, light).items():
            cover[(i, j)] = shade
            if tones:
                self.put(i, j, tone(tones, shade, jitter, noise(i, j, salt)))
        return cover

    def coil(self, path, w0, w1, color, pitch=3.0):
        """Wire wound round a stroke of those widths, a diagonal stripe every pitch units along it; returns the
        pixels it set."""
        out = []
        for (i, j), (u, _, across, _) in self.trace(path, w0, w1).items():
            if (u / pitch + 0.35 * across) % 1 < 0.45:
                self.put(i, j, color)
                out.append((i, j))
        return out

    def blob(self, cx, cy, rx, ry, tones, jitter=0.5, lump=0.0, gaps=0.0, salt=0, where=None, light=LIGHT):
        """A filled ellipse of radii rx, ry, shaded lit on its upper left with a dithered edge between tones. lump
        (0 to about 0.4) makes its outline uneven, gaps (a share) leaves sky showing through its inside, and
        where(x, y) in units can clip it. Returns its cover."""
        p, q = noise(salt, 3) * 6.283, noise(salt, 4) * 6.283
        reach = 1 + lump
        cover = {}
        for i, j, x, y in self.scan(cx - rx * reach, cy - ry * reach, cx + rx * reach, cy + ry * reach):
            nx, ny = (x - cx) / rx, (y - cy) / ry
            edge = 1.0
            if lump:
                a = math.atan2(ny, nx)
                edge += lump * (0.6 * math.sin(3 * a + p) + 0.4 * math.sin(5 * a + q))
            r2 = nx * nx + ny * ny
            if r2 > edge * edge or (where and not where(x, y)):
                continue
            if gaps and r2 < 0.55 and noise(i, j, salt + 101) < gaps:
                continue
            shade = clamp(0.5 + 0.5 * (nx * light[0] + ny * light[1]))
            cover[(i, j)] = shade
            self.put(i, j, tone(tones, shade, jitter, noise(i, j, salt)))
        return cover

    def cloud(self, cx, cy, rx, ry, tones, lumps=4, jitter=0.5, gaps=0.0, salt=0, where=None, light=LIGHT):
        """A mass of foliage: a blob with lumps blobs round its rim, the upper ones in front. Returns its cover."""
        cover = self.blob(cx, cy, rx, ry, tones, jitter, 0.12, gaps, salt, where, light)
        rim = sorted((cx + math.cos(a) * rx * 0.6, cy + math.sin(a) * ry * 0.55) for a in (
            6.283 * (k + noise(salt, 5 + k)) / lumps for k in range(lumps)))
        for k, (x, y) in enumerate(sorted(rim, key=lambda p: -p[1])):
            cover.update(self.blob(x, y, rx * 0.55, ry * 0.6, tones, jitter, 0.1, gaps, salt + 7 + k, where, light))
        return cover

    def block(self, x0, y0, x1, y1, tones, taper=0.0, rounding=0.0, jitter=0.0, salt=0, light=LIGHT):
        """A shaded box, lit on its left and top: taper narrows its foot by that many units each side, rounding
        (0 box, 1 ellipse) rounds its corners. Returns its cover."""
        cover = {}
        for i, j, x, y in self.scan(x0, y0, x1, y1):
            if not (y0 <= y < y1):
                continue
            inset = taper * (y - y0) / (y1 - y0)
            left, right = x0 + inset, x1 - inset
            if not (left <= x < right):
                continue
            nx, ny = 2 * (x - left) / (right - left) - 1, 2 * (y - y0) / (y1 - y0) - 1
            if rounding and abs(nx) ** (2 + 8 * (1 - rounding)) + abs(ny) ** (2 + 8 * (1 - rounding)) > 1:
                continue
            shade = clamp(0.5 + 0.5 * (nx * light[0] + ny * light[1]))
            cover[(i, j)] = shade
            self.put(i, j, tone(tones, shade, jitter, noise(i, j, salt)))
        return cover

    def polygon(self, points, tones, jitter=0.25, salt=0, light=LIGHT):
        """A filled polygon through points in units (a pad's flat underside, a jin, a stone), shaded lit toward
        light across its extent. Returns its cover."""
        points = list(points)
        xs, ys = [p[0] for p in points], [p[1] for p in points]
        cx, cy = (min(xs) + max(xs)) / 2, (min(ys) + max(ys)) / 2
        hw, hh = (max(xs) - min(xs)) / 2 or 1, (max(ys) - min(ys)) / 2 or 1
        cover = {}
        for i, j, x, y in self.scan(min(xs), min(ys), max(xs), max(ys)):
            inside = False
            for (ax, ay), (bx, by) in zip(points, points[1:] + points[:1]):
                if (ay > y) != (by > y) and x < ax + (y - ay) * (bx - ax) / (by - ay):
                    inside = not inside
            if inside:
                shade = clamp(0.5 + 0.5 * ((x - cx) / hw * light[0] + (y - cy) / hh * light[1]))
                cover[(i, j)] = shade
                self.put(i, j, tone(tones, shade, jitter, noise(i, j, salt)))
        return cover

    def line(self, x0, y0, x1, y1, color):
        """A one-pixel line between two points in units (a needle, a crack); returns its pixels."""
        (i0, j0), (i1, j1) = self.at(x0, y0), self.at(x1, y1)
        steps = max(abs(i1 - i0), abs(j1 - j0), 1)
        out = [(round(i0 + (i1 - i0) * k / steps), round(j0 + (j1 - j0) * k / steps)) for k in range(steps + 1)]
        for i, j in out:
            self.put(i, j, color)
        return out

    def scatter(self, where, colors, density, salt=0, run=1, test=None):
        """Per-pixel detail. Each pixel of where, a cover or a box (x0, y0, x1, y1) in units, takes a color from
        colors with probability density, as a horizontal dash of run pixels (kept inside a cover). test(i, j, shade)
        vetoes a pixel, the shade being the cover's, or 0.5 in a box. Returns the pixels it set."""
        colors = (colors,) if isinstance(colors, str) else tuple(colors)
        box = isinstance(where, tuple)
        cells = where
        if box:
            x0, y0, x1, y1 = where
            cells = {(i, j): 0.5 for i, j, x, y in self.scan(*where) if x0 <= x < x1 and y0 <= y < y1}
        out = []
        for (i, j), shade in cells.items():
            if noise(i, j, salt) >= density or (test and not test(i, j, shade)):
                continue
            color = colors[int(noise(i, j, salt + 1) * len(colors))]
            for k in range(run):
                if box or (i + k, j) in cells:
                    self.put(i + k, j, color)
                    out.append((i + k, j))
        return out
