"""Launch the bonsai from a checkout or a Python zipapp."""
import argparse
import sys
from pathlib import Path

import bonsai
from pane import run_tree


def main():
    if sys.argv[1:2] == ["sheet"]:
        import sheet
        return sheet.main(sys.argv[2:])
    parser = argparse.ArgumentParser(description="Grow a pixel bonsai in the terminal with your Claude Code usage.")
    parser.epilog = "Use the sheet subcommand to export PNG and ANSI frames."
    parser.add_argument("--view-file", type=Path, default=Path(bonsai.store_path()).with_name("bonsai-view"),
                        help="where to remember the daily or lifetime view")
    args = parser.parse_args()
    if not sys.stdout.isatty():
        parser.exit(1, "The live bonsai needs a terminal. Use the sheet subcommand to export PNG and ANSI frames.\n")
    run_tree(args.view_file)


if __name__ == "__main__":
    sys.exit(main())
