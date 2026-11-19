"""Launch the bonsai from a checkout or a Python zipapp."""
import argparse
import json
import sys
from pathlib import Path

import art
import bonsai


def render(argv):
    parser = argparse.ArgumentParser(prog="pixel-bonsai render", description="Render one tree at a fixed age.")
    parser.add_argument("--variety", choices=list(bonsai.VARIETIES), default=bonsai.DEFAULT)
    parser.add_argument("--growth", required=True, type=float, help="0 (seed) through 1 (venerable)")
    parser.add_argument("--scale", type=float, default=1.0, help="canvas scale, from 0.5 through 3")
    parser.add_argument("--format", choices=("ansi", "json"), default="ansi")
    args = parser.parse_args(argv)
    if not 0 <= args.growth <= 1 or not art.MIN_SCALE <= args.scale <= art.MAX_SCALE:
        parser.error("growth must be between 0 and 1 and scale between 0.5 and 3")
    width, height = art.size(args.scale)
    pixels = bonsai.VARIETIES[args.variety].pixels(args.growth, args.scale)
    if args.format == "json":
        print(json.dumps({"width": width, "height": height,
                          "pixels": [[x, y, rgb] for (x, y), rgb in pixels.items()]}))
    else:
        print("\n".join(bonsai.rows(pixels, w=width, h=height)))


def main():
    if sys.argv[1:2] == ["render"]:
        return render(sys.argv[2:])
    if sys.argv[1:2] == ["record"]:
        from usage import record
        return record(sys.argv[2:])
    if sys.argv[1:2] == ["sheet"]:
        import sheet
        return sheet.main(sys.argv[2:])
    parser = argparse.ArgumentParser(description="Grow a pixel bonsai in the terminal with your agent usage.")
    parser.epilog = "Use render for one frame, record for harness usage, or sheet for PNG and ANSI contact sheets."
    parser.add_argument("--view-file", type=Path, default=Path(bonsai.store_path()).with_name("bonsai-view"),
                        help="where to remember the daily or lifetime view")
    args = parser.parse_args()
    if not sys.stdout.isatty():
        parser.exit(1, "The live bonsai needs a terminal. Use render to print one frame.\n")
    from pane import run_tree
    run_tree(args.view_file)


if __name__ == "__main__":
    sys.exit(main())
