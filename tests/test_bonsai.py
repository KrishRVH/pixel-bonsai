"""Rendering, local token accounting, terminal controls, and contact sheets."""
import contextlib
import fcntl
import io
import json
import os
import re
import struct
import subprocess
import sys
import tempfile
import threading
import time
import unittest
import zlib

HERE = os.path.dirname(os.path.realpath(__file__))
sys.path.insert(0, os.path.join(HERE, ".."))
import art
import bonsai
import pane
import sheet

pane.COLOR = False


def usage(message, when, tokens, request="r"):
    """A transcript line for an API message, stamped at noon UTC on a day or at a full UTC stamp."""
    stamp = when if "T" in when else f"{when}T12:00:00Z"
    return json.dumps({"type": "assistant", "requestId": request, "timestamp": stamp,
                       "message": {"id": message, "usage": {"input_tokens": tokens, "output_tokens": 0}}}) + "\n"


@contextlib.contextmanager
def transcripts_in(home):
    """Point the token counts at a home of fake transcripts and a store of its own."""
    folder = os.path.join(home, ".claude", "projects", "p")
    os.makedirs(os.path.join(folder, "sid", "subagents"))
    saved = {k: os.environ.get(k) for k in ("HOME", "XDG_STATE_HOME")}
    os.environ.update(HOME=home, XDG_STATE_HOME=os.path.join(home, "state"))
    try:
        yield folder
    finally:
        for k, v in saved.items():
            if v is None:
                os.environ.pop(k, None)
            else:
                os.environ[k] = v


@contextlib.contextmanager
def state_in(folder):
    """Point the bonsai's stores at a folder of their own."""
    saved = os.environ.get("XDG_STATE_HOME")
    os.environ["XDG_STATE_HOME"] = folder
    try:
        yield
    finally:
        if saved is None:
            os.environ.pop("XDG_STATE_HOME", None)
        else:
            os.environ["XDG_STATE_HOME"] = saved


def differs(a, b):
    """How many pixels two frames disagree on, a pixel set in only one counting."""
    return sum(1 for p in a.keys() | b.keys() if a.get(p) != b.get(p))


def at_start(name):
    """The age a frame of the stage called name is judged at: just past its start, where its new feature has arrived."""
    return art.START[name] + art.PLUS


MIN_CHANGE = 10   # pixels a stage's frame must differ by from the stage before's, at scale 1
SCALES = (0.6, 1, 2)


class Bonsai(unittest.TestCase):
    def test_half_blocks_stack_two_pixels_to_a_cell(self):
        rgb = lambda c, bg: f"<{'b' if bg else 'f'}{c}>"  # noqa: E731
        lines = bonsai.rows({(0, 0): "a", (0, 1): "b", (1, 0): "a", (2, 1): "c", (3, 0): "d", (3, 1): "d"}, rgb)
        self.assertEqual(len(lines), art.ROWS)
        self.assertTrue(lines[0].startswith("<fa><bb>▀\x1b[0m<fa>▀\x1b[0m<fc>▄\x1b[0m<fd>█\x1b[0m "))
        self.assertEqual(lines[1], " " * art.W)
        big = bonsai.rows({(0, 0): "a", (1, 1): "b"}, rgb, size=2)
        self.assertEqual(len(big), 2 * art.ROWS)
        self.assertEqual(big[0], "<fa>█\x1b[0m" * 2 + " " * (2 * art.W - 2), "a pixel fills two by two")
        self.assertEqual(big[1], "  " + "<fb>█\x1b[0m" * 2 + " " * (2 * art.W - 4))
        self.assertEqual(big[2], " " * (2 * art.W))
        self.assertEqual(bonsai.rows({(0, 0): "a", (2, 3): "b"}, rgb, w=3, h=4),
                         ["<fa>▀\x1b[0m  ", "  <fb>▄\x1b[0m"], "another canvas")

    def test_tokens_count_each_message_once_and_never_shrink(self):
        with tempfile.TemporaryDirectory() as home, transcripts_in(home) as folder:
            first = os.path.join(folder, "a.jsonl")
            with open(first, "w") as f:
                f.write(usage("m1", "2026-11-10", 100) + usage("m1", "2026-11-10", 100) + "not json\n")
            with open(os.path.join(folder, "resumed.jsonl"), "w") as f:
                f.write(usage("m1", "2026-11-10", 100) + usage("m2", "2026-11-11", 5))
            subagent = os.path.join(folder, "sid", "subagents", "agent-x.jsonl")
            with open(subagent, "w") as f:
                f.write(usage("m3", "2026-11-10", 7))
            self.assertEqual(bonsai.stored(), 0)
            self.assertEqual(bonsai.tokens(), 112, "a message repeated in a line or a resumed copy counts once")
            os.remove(first)
            os.remove(subagent)
            self.assertEqual(bonsai.tokens(), 112, "transcripts pruned for age leave their day's count")
            with open(os.path.join(folder, "b.jsonl"), "w") as f:
                f.write(usage("m4", "2026-11-11", 30))
            self.assertEqual(bonsai.tokens(), 142)
            self.assertEqual(bonsai.stored(), 142)

    def test_two_panes_merge_daily_highs_under_the_same_lock(self):
        with tempfile.TemporaryDirectory() as folder, state_in(folder):
            ledger = bonsai.Ledger()
            ledger.add(usage("m", "2026-11-11", 7).encode())
            os.makedirs(os.path.dirname(bonsai.store_path()))
            with open(bonsai.store_path() + ".lock", "a") as lock:
                fcntl.flock(lock, fcntl.LOCK_EX)
                worker = threading.Thread(target=bonsai.tokens, args=(ledger,))
                worker.start()
                worker.join(0.2)
                self.assertTrue(worker.is_alive())
                bonsai.save(bonsai.store_path(), {"days": {"2026-11-10": 12}})
            worker.join(5)
            self.assertFalse(worker.is_alive())
            self.assertEqual(bonsai.stored(), 19)

    def test_the_day_counts_from_local_midnight(self):
        now = time.time()
        midnight = time.mktime(time.localtime(now)[:3] + (0, 0, 0, 0, 0, -1))
        stamp = lambda t: time.strftime("%Y-%m-%dT%H:%M:%S.000Z", time.gmtime(t))  # noqa: E731
        with tempfile.TemporaryDirectory() as home, transcripts_in(home) as folder:
            with open(os.path.join(folder, "a.jsonl"), "w") as f:
                f.write(usage("m1", stamp(midnight - 60), 50) + usage("m2", stamp(now), 7) +
                        usage("m2", stamp(now), 7))
            with open(os.path.join(folder, "b.jsonl"), "w") as f:
                f.write(usage("m2", stamp(now), 7) + usage("m3", stamp(midnight), 3))
            self.assertEqual(bonsai.today(now), 10, "yesterday's message is left out")

    def test_a_ledger_reads_only_what_was_appended(self):
        with tempfile.TemporaryDirectory() as home, transcripts_in(home) as folder:
            path = os.path.join(folder, "a.jsonl")
            with open(path, "w") as f:
                f.write(usage("m1", "2026-11-10", 100) + usage("m2", "2026-11-10", 5)[:20])  # m2 still being written
            ledger = bonsai.Ledger().update()
            self.assertEqual(ledger.days(), {"2026-11-10": 100}, "a half-written line waits")
            with open(path, "a") as f:
                f.write(usage("m2", "2026-11-10", 5)[20:] + usage("m1", "2026-11-10", 100))
            self.assertEqual(ledger.update().days(), {"2026-11-10": 105}, "the rest of the line, and m1 again once")
            read = ledger.read[path][1]
            self.assertEqual(read, os.path.getsize(path))
            os.remove(path)
            with open(path, "w") as f:
                f.write(usage("m3", "2026-11-11", 7))
            self.assertEqual(ledger.update().days(), {"2026-11-10": 105, "2026-11-11": 7},
                             "a replaced transcript is read again; the pruned one keeps its count")
            self.assertEqual(bonsai.tokens(ledger), 112)


class Ladder(unittest.TestCase):
    """The stages every variety grows through, and the ages the day's and the lifetime tokens reach."""

    def test_the_ladder_is_about_twenty_distinct_stages_in_order(self):
        starts, names = [s for s, _ in art.STAGES], [n for _, n in art.STAGES]
        self.assertTrue(15 <= len(art.STAGES) <= 25, len(art.STAGES))
        self.assertEqual(starts[0], 0)
        self.assertEqual(starts, sorted(set(starts)), "starts strictly increase")
        self.assertEqual(len(set(names)), len(names), "names are unique")

    def test_a_stage_begins_at_its_start_age(self):
        for k, (start, name) in enumerate(art.STAGES):
            self.assertEqual(art.stage(start), name)
            self.assertEqual(art.stage(start + 1e-6), name)
            if k:
                self.assertEqual(art.stage(start - 1e-6), art.STAGES[k - 1][1], name)
        self.assertEqual(art.stage(0), "seed")
        self.assertEqual(art.stage(1), art.STAGES[-1][1])

    def test_each_view_lands_on_its_own_stage(self):
        day = lambda tokens: art.stage(bonsai.day_growth(tokens))  # noqa: E731
        landed = [day(100e6), day(232e6), art.stage(bonsai.life_growth(13e9)), day(1e9)]
        self.assertEqual(landed, ["first branches", "thickening trunk", "shaped", "refined"])
        self.assertEqual(len(set(landed)), 4, "100M day, median day, 13B lifetime and 1B day: four stages")
        self.assertGreaterEqual(art.index(bonsai.life_growth(13e9)), art.index(art.START["shaped"]), "13B is shaped")

    def test_the_day_tree_keeps_growing_through_every_stage(self):
        self.assertEqual(bonsai.day_growth(0), 0)
        seen = {art.stage(bonsai.day_growth(1e4 * 1.05 ** k)) for k in range(259)}  # 10K up to 3B
        self.assertEqual(seen, {name for _, name in art.STAGES}, "a day reaches every stage by 3B tokens")
        self.assertEqual(art.stage(bonsai.day_growth(2.7e9)), "venerable")
        heavy = [bonsai.day_growth(t) for t in (1e9, 1.5e9, 2e9, 2.5e9)]
        self.assertEqual(heavy, sorted(set(heavy)), "a heavier day is always an older tree")
        self.assertNotEqual(art.stage(bonsai.day_growth(1e9)), art.stage(bonsai.life_growth(60e9)))
        steps = [bonsai.day_growth(t) for t in (1e7, 1e8, 3e8, 6e8)]
        self.assertEqual(steps, sorted(steps))
        self.assertGreater((steps[1] - steps[0]) / 9e7, (steps[3] - steps[2]) / 3e8, "early, a token grows it most")
        self.assertEqual(bonsai.life_growth(0), 0)
        self.assertEqual(bonsai.life_growth(bonsai.MATURE), 1)
        self.assertEqual(bonsai.life_growth(50 * bonsai.MATURE), 1)
        self.assertLess(bonsai.life_growth(0.4e9 * 365), 1, "a year at autumn 2026's pace leaves it growing")
        self.assertEqual(list(bonsai.VIEWS), ["today", "lifetime"])

    def test_the_pane_scale_is_as_large_as_fits_in_eighths_and_capped(self):
        self.assertEqual(art.fit(54, 20), 1.125, "the nav column's width")
        self.assertEqual(art.fit(100, 50), 2.0)
        self.assertEqual(art.fit(400, 200), 3.0, "capped at 3")
        self.assertEqual(art.fit(20, 40), 0, "too narrow: the caption alone")
        self.assertEqual(art.fit(54, 8), 0, "too short")
        self.assertEqual(art.fit(54, 9), 0.5)
        for width, rows in ((54, 20), (100, 50), (30, 12), (80, 18)):
            w, h = art.size(art.fit(width, rows))
            self.assertLessEqual(w, width)
            self.assertLessEqual(h // 2, rows - 1, "the canvas and its caption fit the pane")
        self.assertEqual(art.size(1), (48, 30))
        self.assertEqual(art.size(0.6), (29, 18))
        self.assertEqual(art.size(0.58), (28, 18), "the height rounds up to even: two pixels to a cell")


class Kit(unittest.TestCase):
    """art.py's primitives, which every variety draws with."""
    TONES = ("111111", "222222", "333333")

    def test_noise_is_fixed_in_range_and_well_spread(self):
        values = [art.noise(x, y, 7) for x in range(-20, 20) for y in range(-20, 20)]
        self.assertEqual(values, [art.noise(x, y, 7) for x in range(-20, 20) for y in range(-20, 20)])
        self.assertTrue(all(0 <= v < 1 for v in values))
        self.assertAlmostEqual(sum(values) / len(values), 0.5, delta=0.05)
        self.assertNotEqual(art.noise(3, 4, 1), art.noise(3, 4, 2), "a salt makes another pattern")
        self.assertTrue(all(-1 <= art.wobble(t / 3, 4) <= 1 for t in range(100)))

    def test_since_during_and_steps_follow_the_stage_names(self):
        start = art.START["first wiring"]
        self.assertEqual(art.since(start - 0.01, "first wiring", 0.05), 0)
        self.assertAlmostEqual(art.since(start + 0.025, "first wiring", 0.05), 0.5)
        self.assertEqual(art.since(start + 0.2, "first wiring", 0.05), 1)
        ends = art.START["thickening trunk"]
        window = [art.during(g, "first wiring", "thickening trunk") for g in (start - 0.001, start, ends - 0.001, ends)]
        self.assertEqual(window, [False, True, True, False])
        self.assertTrue(art.during(1, "show tree"))
        table = (("sprout", 3), ("whip", 14))
        self.assertEqual([art.steps(g, table) for g in (0, 0.02, 0.09, 0.2)], [0, 3, 3, 14])
        self.assertAlmostEqual(art.steps(art.START["whip"] + 0.002, table), 8.5, msg="moves over the span")
        with self.assertRaises(KeyError):
            art.since(0.5, "no such stage")

    def test_colors_blend_and_tones_run_shadow_to_lit(self):
        self.assertEqual(art.mix("000000", "ffffff", 0), "000000")
        self.assertEqual(art.mix("000000", "ffffff", 1), "ffffff")
        self.assertEqual(art.mix("ff0000", "0000ff", 0.5), "800080")
        self.assertEqual([art.tone(self.TONES, s) for s in (0, 0.4, 0.99, 1)], ["111111", "222222", "333333", "333333"])
        self.assertEqual(art.tone(self.TONES, 0.6, 0.0, 0.99), "222222", "no jitter, no dither")
        self.assertEqual(art.tone(self.TONES, 0.6, 0.4, 0.5), "222222")
        self.assertEqual(art.tone(self.TONES, 0.6, 0.4, 0.99), "333333", "noise dithers the edge between tones")

    def test_a_spline_runs_through_its_points(self):
        pts = [(0, 0), (10, 5), (20, 0), (30, 10)]
        path = art.spline(pts, 5)
        self.assertEqual((path[0], path[-1]), (pts[0], pts[-1]))
        for p in pts[1:-1]:
            self.assertIn(p, [tuple(round(c, 9) for c in q) for q in path])
        self.assertEqual(art.spline([(3, 4)]), [(3, 4)])

    def test_a_stroke_tapers_and_the_canvas_never_gets_a_pixel_outside_it(self):
        c = art.Canvas(1)
        cover = c.stroke([(24, 2), (24, 28)], 8, 1, self.TONES)
        rows = {j: sorted(i for i, jj in cover if jj == j) for j in range(30)}
        self.assertGreater(len(rows[4]), 6)
        self.assertLessEqual(len(rows[26]), 2)
        self.assertEqual([len(rows[j]) >= len(rows[j + 4]) for j in range(2, 22, 4)], [True] * 5, "narrower going down")
        self.assertTrue(all(rows[j] for j in range(2, 28)), "unbroken")
        far = art.Canvas(1)
        far.stroke([(-20, -20), (80, 80)], 6, 6, self.TONES)
        far.blob(100, 100, 30, 30, self.TONES)
        self.assertTrue(far.px and all(0 <= i < 48 and 0 <= j < 30 for i, j in far.px))

    def test_a_stroke_is_lit_on_its_upper_left_edge(self):
        c = art.Canvas(2)
        c.stroke([(24, 3), (24, 27)], 6, 6, self.TONES)
        j = 30
        row = sorted(i for i, jj in c.px if jj == j)
        self.assertEqual((c.px[(row[0], j)], c.px[(row[-1], j)]), ("333333", "111111"), "lit left, shadow right")

    def test_a_thin_stroke_is_unbroken_at_any_angle(self):
        for scale in (0.6, 1, 2):
            c = art.Canvas(scale)
            cover = c.stroke([(4, 4), (40, 22)], 0.1, 0.1, self.TONES)
            cols = sorted({i for i, _ in cover})
            self.assertEqual(cols, list(range(cols[0], cols[-1] + 1)), scale)
            self.assertGreater(len(cols), 30 * scale)
            self.assertLess(len(cover), 2.2 * len(cols), "about a pixel wide")

    def test_a_blob_fills_its_ellipse_with_light_from_the_upper_left(self):
        c = art.Canvas(1)
        cover = c.blob(24, 15, 10, 6, self.TONES, jitter=0)
        self.assertAlmostEqual(len(cover), 3.1416 * 10 * 6, delta=12)
        self.assertTrue(all(((i + 0.5 - 24) / 10) ** 2 + ((j + 0.5 - 15) / 6) ** 2 <= 1 for i, j in cover))
        self.assertEqual(c.px[(17, 12)], "333333", "lit upper left")
        self.assertEqual(c.px[(31, 18)], "111111", "shadow lower right")

    def test_a_blob_can_be_lumpy_holed_and_clipped(self):
        plain = art.Canvas(1).blob(24, 15, 10, 6, self.TONES)
        self.assertNotEqual(set(art.Canvas(1).blob(24, 15, 10, 6, self.TONES, lump=0.3)), set(plain))
        holed = art.Canvas(1).blob(24, 15, 10, 6, self.TONES, gaps=0.3)
        self.assertLess(len(holed), 0.9 * len(plain))
        self.assertGreater(len(holed), 0.6 * len(plain))
        top = art.Canvas(1).blob(24, 15, 10, 6, self.TONES, where=lambda x, y: y < 15)
        self.assertTrue(top and all(j < 15 for _, j in top))

    def test_a_block_a_polygon_and_a_line_stay_in_their_shapes(self):
        c = art.Canvas(1)
        box = c.block(10, 20, 30, 26, self.TONES, taper=3)
        self.assertEqual({j for _, j in box}, set(range(20, 26)))
        self.assertGreater(sum(1 for _, j in box if j == 20), sum(1 for _, j in box if j == 25), "tapered foot")
        tri = art.Canvas(1).polygon([(5, 5), (25, 5), (5, 25)], self.TONES)
        self.assertAlmostEqual(len(tri), 200, delta=15)
        self.assertTrue(all(i + j <= 30 for i, j in tri), "inside the triangle")
        line = art.Canvas(1).line(2, 3, 12, 8, "aabbcc")
        self.assertEqual((line[0], line[-1]), ((2, 3), (12, 8)))
        self.assertEqual(len(line), 11, "a pixel a step of its longer side")
        clipped = art.Canvas(1)
        kept = clipped.line(2, 3, 12, 3, "aabbcc", test=lambda i, j: i < 6)
        self.assertEqual(kept, [(i, 3) for i in range(2, 6)])
        self.assertEqual(set(clipped.px), set(kept), "a vetoed pixel is not drawn")

    def test_scatter_marks_only_its_cover_and_a_box_gains_pixels_with_scale(self):
        c = art.Canvas(1)
        cover = c.block(10, 10, 20, 14, ("111111",))
        marked = c.scatter(cover, "ffffff", 1.0)
        self.assertEqual(set(marked), set(cover))
        self.assertEqual(c.scatter(cover, "ffffff", 0.0), [])
        some = art.Canvas(1).scatter(cover, "ffffff", 0.5, salt=2)
        self.assertTrue(0.3 * len(cover) < len(some) < 0.7 * len(cover))
        dashed = art.Canvas(1).scatter(cover, "ffffff", 0.5, salt=2, run=3)
        self.assertTrue(set(dashed) <= set(cover), "a dash stays inside the cover")
        self.assertGreater(len(dashed), len(some))
        counts = [len(art.Canvas(s).scatter((5, 5, 25, 15), "ffffff", 0.3)) for s in (1, 2)]
        self.assertGreater(counts[1], 3 * counts[0], "finer detail at a larger scale")
        vetoed = art.Canvas(1).scatter(cover, "ffffff", 1.0, test=lambda i, j, s: i < 15)
        self.assertTrue(vetoed and all(i < 15 for i, _ in vetoed))

    def test_wire_winds_round_a_stroke_in_stripes(self):
        c = art.Canvas(2)
        c.stroke([(24, 3), (24, 27)], 5, 5, self.TONES)
        wound = c.coil([(24, 3), (24, 27)], 5, 5, "c0773a", pitch=3)
        stroke = c.trace([(24, 3), (24, 27)], 5, 5)
        self.assertTrue(0.2 * len(stroke) < len(wound) < 0.7 * len(stroke), "stripes with gaps, not a solid wrap")
        self.assertTrue(all(c.px[p] == "c0773a" for p in wound))

    def test_wire_can_take_a_color_per_side_a_veto_and_extra_width(self):
        path = [(24, 3), (24, 27)]
        c = art.Canvas(2)
        wound = c.coil(path, 5, 5, lambda across, shade: "aa0000" if across < 0 else "0000aa", test=lambda i, j: j < 30)
        self.assertTrue(wound and all(j < 30 for _, j in wound), "a vetoed pixel is not wound")
        self.assertEqual({c.px[p] for p in wound}, {"aa0000", "0000aa"}, "a color for each side")
        plain = art.Canvas(2).coil(path, 5, 5, "c0773a")
        self.assertGreater(len(art.Canvas(2).coil(path, 5, 5, "c0773a", grow=2)), len(plain), "grow widens it")
        self.assertNotEqual(art.Canvas(2).coil(path, 5, 5, "c0773a", duty=0.8), plain, "duty is the stripe's share")


class Varieties(unittest.TestCase):
    """What every variety module must do, however it draws: the tests the art is held to."""

    def test_the_registry_lists_the_three_in_switch_order(self):
        self.assertEqual(list(bonsai.VARIETIES), ["sakura", "maple", "black pine"])
        for name, tree in bonsai.VARIETIES.items():
            self.assertEqual(tree.NAME, name)
            self.assertTrue(tree.ICON and tree.SIGNATURE)

    def test_each_stage_shows_something_new(self):
        for name, tree in bonsai.VARIETIES.items():
            for k in range(1, len(art.STAGES)):
                stage, before = art.STAGES[k][1], art.STAGES[k - 1][1]
                with self.subTest(variety=name, stage=stage):
                    change = differs(tree.pixels(at_start(stage)), tree.pixels(at_start(before)))
                    self.assertGreaterEqual(change, MIN_CHANGE, f"{stage} looks like {before}")

    def test_every_pixel_is_on_the_canvas_and_the_same_every_time(self):
        for name, tree in bonsai.VARIETIES.items():
            for scale in SCALES:
                w, h = art.size(scale)
                for g in (0, 0.02, 0.1, 0.2, 0.31, 0.412, 0.55, 0.7, 0.85, 1):
                    with self.subTest(variety=name, scale=scale, g=g):
                        px = tree.pixels(g, scale)
                        self.assertTrue(px)
                        self.assertTrue(all(0 <= x < w and 0 <= y < h for x, y in px), "inside the canvas")
                        self.assertTrue(all(re.fullmatch("[0-9a-f]{6}", c) for c in px.values()))
                        tree.pixels(g / 2, scale)  # another tree in between leaves no trace
                        self.assertEqual(px, tree.pixels(g, scale), "the same tree every call")
            self.assertEqual(tree.pixels(0.5), tree.pixels(0.5, 1.0), "the scale defaults to 1")

    def test_a_larger_scale_draws_more_and_richer(self):
        for name, tree in bonsai.VARIETIES.items():
            for g in (bonsai.life_growth(13e9), 0.7, 1):
                with self.subTest(variety=name, g=g):
                    small, one, big = (tree.pixels(g, s) for s in SCALES)
                    self.assertLess(len(small), len(one))
                    self.assertGreater(len(big), 2.5 * len(one), "more pixels, not just bigger ones")
                    self.assertGreaterEqual(len(set(big.values())), len(set(one.values())), "no fewer colors")

    def test_a_tree_grows_and_gains_detail_with_age(self):
        for name, tree in bonsai.VARIETIES.items():
            with self.subTest(variety=name):
                sizes = [len(tree.pixels(g)) for g in (0.02, 0.07, 0.14, 0.24, 0.34, 0.45, 0.63, 0.81, 0.94)]
                self.assertEqual(sizes, sorted(sizes))
                young, old = tree.pixels(0.15), tree.pixels(1)
                self.assertGreater(len(set(old.values())), len(set(young.values())), "an old tree has more tones")
                self.assertLess(min(y for _, y in old), min(y for _, y in young), "and stands taller")

    def test_the_varieties_are_distinct_at_the_same_age(self):
        trees = list(bonsai.VARIETIES.items())
        for g in (0.2, bonsai.life_growth(13e9), 0.7, 1):
            for k, (a, one) in enumerate(trees):
                for b, two in trees[k + 1:]:
                    with self.subTest(a=a, b=b, g=g):
                        px, other = one.pixels(g), two.pixels(g)
                        self.assertGreaterEqual(differs(px, other), 50, "different drawings")
                        self.assertGreaterEqual(len(set(px.values()) - set(other.values())), 3, "different palettes")
                        self.assertFalse(set(one.SIGNATURE) & set(other.values()), "each keeps its signature color")

    def test_each_is_recognisable_small_from_a_young_tree_on(self):
        for name, tree in bonsai.VARIETIES.items():
            for k in range(int((1 - at_start("young tree")) / 0.03) + 1):
                g = at_start("young tree") + 0.03 * k
                with self.subTest(variety=name, g=g):
                    self.assertTrue(set(tree.SIGNATURE) & set(tree.pixels(g, 0.6).values()), "signature color at 0.6")


class Garden(unittest.TestCase):
    """Each variety's own lifetime growth, kept in a store beside the token store."""

    def written(self, days):
        os.makedirs(os.path.dirname(bonsai.store_path()), exist_ok=True)
        with open(bonsai.store_path(), "w") as f:
            json.dump({"days": days}, f)

    def test_a_new_garden_continues_the_old_tree_as_sakura(self):
        with tempfile.TemporaryDirectory() as folder, state_in(folder):
            self.written({"2026-11-10": 12_000_000_000, "2026-11-11": 1_000_000_000})
            self.assertEqual(bonsai.garden(), {"variety": "sakura", "counted": 13_000_000_000,
                                               "grown": {"sakura": 13_000_000_000, "maple": 0, "black pine": 0},
                                               "day": bonsai.no_day()})
            self.assertEqual(bonsai.accrue(14_000_000_000)["grown"]["sakura"], 14_000_000_000)

    def test_accrue_gives_the_selected_variety_only_what_is_new(self):
        with tempfile.TemporaryDirectory() as folder, state_in(folder):
            self.written({"2026-11-10": 100})
            self.assertEqual(bonsai.accrue(150)["grown"], {"sakura": 150, "maple": 0, "black pine": 0})
            bonsai.choose("maple")
            after = bonsai.accrue(180)
            self.assertEqual(after["grown"], {"sakura": 150, "maple": 30, "black pine": 0})
            self.assertEqual((after["variety"], after["counted"]), ("maple", 180))
            bonsai.choose("black pine")
            self.assertEqual(bonsai.accrue(200)["grown"], {"sakura": 150, "maple": 30, "black pine": 20})

    def test_the_same_total_twice_adds_nothing_and_a_smaller_one_is_ignored(self):
        with tempfile.TemporaryDirectory() as folder, state_in(folder):
            self.written({"2026-11-10": 100})
            bonsai.choose("maple")
            first = bonsai.accrue(180)
            self.assertEqual(bonsai.accrue(180), first)
            self.assertEqual(bonsai.accrue(120), first, "a shrinking total hands nothing out")
            self.assertEqual(bonsai.accrue(200)["grown"]["maple"], first["grown"]["maple"] + 20,
                             "and the next growth counts from the high-water mark")

    def test_choosing_hands_nothing_out(self):
        with tempfile.TemporaryDirectory() as folder, state_in(folder):
            self.written({"2026-11-10": 100})
            before = bonsai.accrue(100)
            after = bonsai.choose("maple")
            self.assertEqual((after["grown"], after["counted"]), (before["grown"], before["counted"]))
            self.assertEqual(after["variety"], "maple")
            self.assertEqual(bonsai.choose("no such tree")["variety"], "maple", "an unknown name is ignored")

    def test_the_garden_survives_a_restart(self):
        with tempfile.TemporaryDirectory() as folder, state_in(folder):
            self.written({"2026-11-10": 100})
            bonsai.choose("black pine")
            bonsai.accrue(160)
            code = "import json, bonsai; print(json.dumps(bonsai.garden()))"
            fresh = subprocess.run([sys.executable, "-c", code], capture_output=True, text=True, timeout=30,
                                   cwd=os.path.join(HERE, ".."), env=dict(os.environ))
            self.assertEqual(json.loads(fresh.stdout), {"variety": "black pine", "counted": 160,
                                                        "grown": {"sakura": 100, "maple": 0, "black pine": 60},
                                                        "day": bonsai.no_day()})

    def test_an_accrual_waits_for_the_garden_lock(self):
        with tempfile.TemporaryDirectory() as folder, state_in(folder):
            self.written({"2026-11-10": 100})
            bonsai.accrue(100)
            with open(bonsai.garden_path() + ".lock", "w") as lock:
                fcntl.flock(lock, fcntl.LOCK_EX)
                worker = threading.Thread(target=bonsai.accrue, args=(130,))
                worker.start()
                worker.join(0.5)
                self.assertTrue(worker.is_alive(), "held while another has the lock")
                self.assertEqual(bonsai.garden()["grown"]["sakura"], 100)
            worker.join(10)
            self.assertFalse(worker.is_alive())
            self.assertEqual(bonsai.garden()["grown"]["sakura"], 130)

    def test_the_day_tree_grows_only_while_its_variety_is_selected(self):
        noon = time.mktime((2026, 11, 19, 12, 0, 0, 0, 0, -1))
        with tempfile.TemporaryDirectory() as folder, state_in(folder):
            self.written({"2026-11-10": 100})
            bonsai.choose("maple")
            first = bonsai.accrue(150, 40, noon)
            self.assertEqual(first["day"]["grown"], {"sakura": 0, "maple": 40, "black pine": 0},
                             "a new day gives the selected variety all of today so far")
            bonsai.choose("black pine")
            after = bonsai.accrue(180, 70, noon + 60)
            self.assertEqual(after["day"]["grown"], {"sakura": 0, "maple": 40, "black pine": 30})
            self.assertEqual(after["grown"], {"sakura": 100, "maple": 50, "black pine": 30})
            self.assertEqual(bonsai.accrue(180, 60, noon + 120)["day"]["grown"]["black pine"], 30,
                             "a smaller day count hands nothing out")
            self.assertEqual([bonsai.day_count(after, n, noon) for n in ("sakura", "maple", "black pine")],
                             [0, 40, 30], "each view of today shows only that variety's share")

    def test_the_day_tree_starts_afresh_at_local_midnight(self):
        late = time.mktime((2026, 11, 19, 23, 59, 0, 0, 0, -1))
        with tempfile.TemporaryDirectory() as folder, state_in(folder):
            self.written({"2026-11-10": 100})
            bonsai.choose("maple")
            yesterday = bonsai.accrue(200, 90, late)
            self.assertEqual(bonsai.day_count(yesterday, "maple", late + 120), 0,
                             "yesterday's record shows nothing once the date has turned")
            today = bonsai.accrue(230, 5, late + 120)
            self.assertEqual(today["day"], {"date": bonsai.local_date(late + 120), "counted": 5,
                                            "grown": {"sakura": 0, "maple": 5, "black pine": 0}})
            self.assertEqual(today["grown"]["maple"], 130, "the lifetime tree keeps every token")

    def test_a_garden_from_before_day_records_keeps_its_lifetime_growth(self):
        with tempfile.TemporaryDirectory() as folder, state_in(folder):
            os.makedirs(os.path.dirname(bonsai.garden_path()))
            with open(bonsai.garden_path(), "w") as f:
                json.dump({"variety": "maple", "counted": 90, "grown": {"sakura": 60, "maple": 30}}, f)
            old = bonsai.garden()
            self.assertEqual((old["grown"]["maple"], old["day"]), (30, bonsai.no_day()))
            self.assertEqual(bonsai.accrue(100, 12)["day"]["grown"], {"sakura": 0, "maple": 12, "black pine": 0})
            with open(bonsai.garden_path()) as f:
                data = json.load(f)
            data["day"] = {"date": 7}
            with open(bonsai.garden_path(), "w") as f:
                json.dump(data, f)
            damaged = bonsai.garden()
            self.assertEqual((damaged["grown"]["maple"], damaged["day"]), (40, bonsai.no_day()))

    def test_a_damaged_or_unwritable_store_does_not_stop_the_pane(self):
        with tempfile.TemporaryDirectory() as folder, state_in(folder):
            os.makedirs(os.path.dirname(bonsai.garden_path()))
            with open(bonsai.garden_path(), "w") as f:
                f.write("{not json")
            self.assertEqual(bonsai.garden(500)["grown"], {"sakura": 500, "maple": 0, "black pine": 0})
        with tempfile.NamedTemporaryFile() as blocker, state_in(blocker.name):  # a file where the folder should be
            self.assertEqual(bonsai.accrue(700)["grown"]["sakura"], 700)
            self.assertIn("maple", bonsai.choose("maple")["grown"])


def plain(line):
    return re.sub(r"\x1b\[[0-9;]*m", "", line)


class Pane(unittest.TestCase):
    """The bonsai pane's caption, its drawing at the pane's size and its controls."""

    def lines(self, *args, **kwargs):
        pane.COLOR = True
        try:
            return pane.tree_lines(*args, **kwargs)
        finally:
            pane.COLOR = False

    def test_the_caption_names_variety_stage_amount_and_view(self):
        self.assertEqual(pane.tree_lines(13e9, 60, 20, "lifetime", "sakura")[-1].strip(),
                         "🌸 sakura · shaped · 13.0B tokens · lifetime ⇄")
        self.assertEqual(pane.tree_lines(100e6, 60, 20, "today", "maple")[-1].strip(),
                         "🍁 maple · first branches · 100.0M tokens · today ⇄")
        self.assertIn("🌲 black pine · seed · 0 tokens", pane.tree_lines(0, 60, 20, "today", "black pine")[-1])

    def test_a_narrow_caption_drops_the_word_tokens_first(self):
        caption = lambda width: pane.tree_lines(13e9, width, 20, "lifetime")[-1]  # noqa: E731
        self.assertEqual(caption(45).strip(), "🌸 sakura · shaped · 13.0B · lifetime ⇄")
        self.assertEqual(pane.cells(caption(20)), 20, "clipped to the width")
        self.assertIn("sakura", caption(20))

    def test_the_variety_in_the_caption_is_a_click_target(self):
        hits = []
        line = pane.tree_lines(13e9, 60, 20, "lifetime", "sakura", hits)[-1]
        x0 = pane.cells(line[:line.index("🌸")])
        self.assertEqual(hits, [(19, x0, x0 + pane.cells("🌸 sakura"), ("variety",))])
        self.assertEqual(pane.hit_at(hits, x0, 19), ("variety",))
        self.assertIsNone(pane.hit_at(hits, x0 + pane.cells("🌸 sakura"), 19), "the words after it are not")
        self.assertEqual(pane.tree_lines(13e9, 60, 0, "lifetime", "sakura", hits=[]), [])

    def test_a_bigger_pane_draws_a_bigger_canvas(self):
        small, big = self.lines(688e6, 54, 20), self.lines(688e6, 100, 50)
        drawn = lambda lines: [plain(l) for l in lines[:-1] if l]  # noqa: E731
        self.assertEqual((len(small), len(big)), (20, 50))
        self.assertEqual((len(drawn(small)), len(drawn(big))), (17, 30), "rows of half blocks: scale 1.125, then 2")
        self.assertLessEqual(max(len(l) for l in drawn(small)), 54)
        self.assertGreater(max(len(l) for l in drawn(big)), 90)
        self.assertEqual(len(drawn(self.lines(688e6, 54, 9))), 8, "the smallest tree, half scale")

    def test_the_tree_is_the_selected_variety_at_the_views_age(self):
        trees = {v: self.lines(13e9, 60, 24, "lifetime", v)[:-1] for v in bonsai.VARIETIES}
        self.assertEqual(len({tuple(t) for t in trees.values()}), 3, "three different pictures")
        day, life = self.lines(100e6, 60, 24, "today")[:-1], self.lines(100e6, 60, 24, "lifetime")[:-1]
        self.assertNotEqual(day, life, "the same tokens are a younger tree in the day's view")
        w, h = art.size(art.fit(60, 24))
        expected = bonsai.rows(bonsai.VARIETIES["maple"].pixels(bonsai.life_growth(13e9), art.fit(60, 24)), w=w, h=h)
        self.assertEqual(self.lines(13e9, 60, 24, "lifetime", "maple")[-1 - len(expected):-1],
                         [" " * ((60 - w) // 2) + l for l in expected])

    def test_too_small_a_pane_shows_the_caption_alone_and_no_color_no_pixels(self):
        self.assertEqual(self.lines(688e6, 54, 3)[:2], ["", ""], "too short for the tree")
        self.assertEqual(self.lines(688e6, 20, 40)[-2], "", "too narrow for the tree")
        self.assertEqual(pane.tree_lines(688e6, 54, 20)[:-1], [""] * 19, "no color, no pixels")
        self.assertEqual(pane.tree_lines(688e6, 54, 0), [])

    def test_every_variety_and_view_draws_at_any_pane_size(self):
        for view in bonsai.VIEWS:
            for variety in bonsai.VARIETIES:
                for tokens in (0, 1e6, 3e8, 5e9, 13e9, 4e11):
                    for width, height in ((54, 18), (80, 30), (130, 70)):
                        lines = self.lines(tokens, width, height, view, variety)
                        self.assertEqual(len(lines), height, (view, variety, tokens))
                        self.assertTrue(all(pane.cells(plain(l)) <= width for l in lines))

    def test_arrows_are_keys(self):
        self.assertEqual(pane.keys_of(b"\x1b[C\x1b[D\x1bOC\x1bOD"), [("right",), ("left",), ("right",), ("left",)])
        self.assertEqual(pane.keys_of(b"\x1b[A\x1b[C\t"), [("up",), ("right",), ("char", "\t")])

    def test_the_caption_click_v_tab_and_the_arrows_step_through_the_varieties(self):
        hits = [(19, 4, 15, ("variety",))]
        step = lambda variety, *events: pane.switched("today", variety, list(events), hits)  # noqa: E731
        self.assertEqual(step("sakura", ("char", "v")), ("today", "maple"))
        self.assertEqual(step("maple", ("char", "\t")), ("today", "black pine"))
        self.assertEqual(step("black pine", ("right",)), ("today", "sakura"), "round the ring")
        self.assertEqual(step("sakura", ("left",)), ("today", "black pine"))
        self.assertEqual(step("maple", ("left",)), ("today", "sakura"))
        self.assertEqual(step("sakura", ("click", 4, 19)), ("today", "maple"))
        self.assertEqual(step("sakura", ("char", "v"), ("char", "v"), ("left",)), ("today", "maple"), "each in turn")

    def test_any_other_click_or_key_switches_today_and_lifetime_and_the_wheel_nothing(self):
        hits = [(19, 4, 15, ("variety",))]
        for event in (("click", 3, 4), ("click", 15, 19), ("click", 4, 18), ("char", " "), ("char", "x"), ("up",),
                      ("down",), ("enter",), ("esc",)):
            self.assertEqual(pane.switched("today", "maple", [event], hits), ("lifetime", "maple"), event)
        self.assertEqual(pane.switched("lifetime", "maple", [("char", " ")], hits), ("today", "maple"))
        for wheel in (("wheel", 1), ("wheel", -1)):
            self.assertEqual(pane.switched("today", "maple", [wheel], hits), ("today", "maple"), wheel)
        self.assertEqual(pane.switched("today", "maple", [], hits), ("today", "maple"))
        self.assertEqual(pane.switched("today", "maple", [("char", "v"), ("char", " ")], hits),
                         ("lifetime", "black pine"), "both, in turn")


class Sheet(unittest.TestCase):
    """sheet.py, the reviewer's PNG and ANSI frames."""

    def decode(self, path):
        with open(path, "rb") as f:
            data = f.read()
        self.assertEqual(data[:8], b"\x89PNG\r\n\x1a\n")
        chunks, at = {}, 8
        while at < len(data):
            (n,), tag = struct.unpack(">I", data[at:at + 4]), data[at + 4:at + 8]
            body = data[at + 8:at + 8 + n]
            self.assertEqual(struct.unpack(">I", data[at + 8 + n:at + 12 + n])[0], zlib.crc32(tag + body), tag)
            chunks[tag] = chunks.get(tag, b"") + body
            at += 12 + n
        w, h = struct.unpack(">II", chunks[b"IHDR"][:8])
        raw = zlib.decompress(chunks[b"IDAT"])
        self.assertEqual(len(raw), h * (1 + 3 * w))
        return w, h, raw

    def test_it_writes_a_labelled_png_and_ansi_frames_for_each_variety(self):
        with tempfile.TemporaryDirectory() as out:
            shown = io.StringIO()
            with contextlib.redirect_stdout(shown):
                self.assertEqual(sheet.main([out, "black-pine", "maple"]), 0)
            self.assertEqual(len(shown.getvalue().split()), 10, "it names the ten files it wrote")
            self.assertTrue(all(os.path.exists(p) for p in shown.getvalue().split()))
            self.assertEqual(sorted(os.listdir(out)), ["black-pine", "maple"])
            folder = os.path.join(out, "black-pine")
            self.assertEqual(sorted(os.listdir(folder)),
                             ["day-100M.png", "frames.txt", "life-13B.png", "scales.png", "stages.png"])
            w, h, raw = self.decode(os.path.join(folder, "stages.png"))
            self.assertGreater(w, 5 * 48 * 6)
            self.assertGreater(h, 4 * 30 * 6)
            pine = bonsai.VARIETIES["black pine"]
            self.assertIn(bytes.fromhex(pine.SIGNATURE[0]), raw, "the tree's colors are in it")
            self.assertEqual(raw[1:4], bytes.fromhex("1e1e2e"), "on the terminal's background")
            w, h, _ = self.decode(os.path.join(folder, "day-100M.png"))
            self.assertGreater(w, 48 * 6)
            with open(os.path.join(folder, "frames.txt"), encoding="utf-8") as f:
                text = f.read()
            for _, name in art.STAGES:
                self.assertIn(f" {name} (age ", text)
            self.assertEqual(text.count("== black pine "), len(art.STAGES) + 2 + 3)
            self.assertIn("13b at 0.6", text)
            self.assertIn("\x1b[38;2;", text)
            self.assertIn("▀", text)

    def test_an_unknown_variety_is_refused(self):
        with tempfile.TemporaryDirectory() as out, self.assertRaises(SystemExit):
            sheet.main([out, "baobab"])


if __name__ == "__main__":
    unittest.main()
