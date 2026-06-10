#!/usr/bin/env python3
from __future__ import annotations

import argparse
from pathlib import Path

from PIL import Image


ICNS_SIZES = (
    (16, 16),
    (32, 32),
    (64, 64),
    (128, 128),
    (256, 256),
    (512, 512),
    (1024, 1024),
)


def generate_icns(source_path: Path, output_path: Path) -> None:
    image = Image.open(source_path).convert("RGBA")
    output_path.parent.mkdir(parents=True, exist_ok=True)
    image.save(output_path, format="ICNS", sizes=ICNS_SIZES)


def parse_args() -> argparse.Namespace:
    parser = argparse.ArgumentParser(description="Generate a macOS .icns file from a source PNG.")
    parser.add_argument("source", type=Path, help="Source PNG with alpha channel.")
    parser.add_argument("output", type=Path, help="Output .icns file path.")
    return parser.parse_args()


def main() -> int:
    args = parse_args()
    generate_icns(args.source.expanduser().resolve(), args.output.expanduser().resolve())
    print(args.output.expanduser().resolve())
    return 0


if __name__ == "__main__":
    raise SystemExit(main())
