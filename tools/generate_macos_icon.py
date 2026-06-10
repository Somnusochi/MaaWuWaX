#!/usr/bin/env python3
from __future__ import annotations

import argparse
import shutil
import subprocess
import tempfile
from pathlib import Path

from PIL import Image


ICON_SIZES = (16, 32, 64, 128, 256, 512, 1024)


def build_iconset(source_path: Path, iconset_dir: Path) -> None:
    image = Image.open(source_path).convert("RGBA")
    for size in ICON_SIZES:
        resized = image.resize((size, size), Image.LANCZOS)
        resized.save(iconset_dir / f"icon_{size}x{size}.png")
        if size < 1024:
            retina_size = size * 2
            retina = image.resize((retina_size, retina_size), Image.LANCZOS)
            retina.save(iconset_dir / f"icon_{size}x{size}@2x.png")


def generate_icns(source_path: Path, output_path: Path) -> None:
    iconutil = shutil.which("iconutil")
    if iconutil is None:
        raise SystemExit("iconutil not found. This script must run on macOS.")

    output_path.parent.mkdir(parents=True, exist_ok=True)
    with tempfile.TemporaryDirectory() as temp_dir:
        iconset_dir = Path(temp_dir) / "AppIcon.iconset"
        iconset_dir.mkdir(parents=True, exist_ok=True)
        build_iconset(source_path, iconset_dir)

        subprocess.run(
            [iconutil, "-c", "icns", str(iconset_dir), "-o", str(output_path)],
            check=True,
        )


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
