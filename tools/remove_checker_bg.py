#!/usr/bin/env python3
from __future__ import annotations

import argparse
import collections
from pathlib import Path

from PIL import Image, ImageDraw


def is_background_pixel(r: int, g: int, b: int, min_brightness: int, max_channel_delta: int) -> bool:
    return min(r, g, b) >= min_brightness and max(r, g, b) - min(r, g, b) <= max_channel_delta


def estimate_background_color(image: Image.Image, min_brightness: int, max_channel_delta: int) -> tuple[int, int, int]:
    width, height = image.size
    pixels = image.load()
    samples: list[tuple[int, int, int]] = []

    def add_if_background(x: int, y: int) -> None:
        r, g, b, _ = pixels[x, y]
        if is_background_pixel(r, g, b, min_brightness, max_channel_delta):
            samples.append((r, g, b))

    for x in range(width):
        add_if_background(x, 0)
        add_if_background(x, height - 1)
    for y in range(height):
        add_if_background(0, y)
        add_if_background(width - 1, y)

    if not samples:
        return (245, 245, 245)

    count = len(samples)
    return tuple(sum(channel) // count for channel in zip(*samples))


def decontaminate_edge_pixels(image: Image.Image, background_rgb: tuple[int, int, int]) -> None:
    pixels = image.load()
    width, height = image.size
    bg_r, bg_g, bg_b = background_rgb

    for y in range(height):
        for x in range(width):
            r, g, b, a = pixels[x, y]
            if a <= 0 or a >= 255:
                continue

            alpha = a / 255.0
            new_r = round((r - bg_r * (1.0 - alpha)) / alpha)
            new_g = round((g - bg_g * (1.0 - alpha)) / alpha)
            new_b = round((b - bg_b * (1.0 - alpha)) / alpha)

            pixels[x, y] = (
                min(255, max(0, new_r)),
                min(255, max(0, new_g)),
                min(255, max(0, new_b)),
                a,
            )


def remove_residual_edge_components(
    image: Image.Image,
    min_brightness: int,
    max_channel_delta: int,
    edge_margin: int,
    min_area: int,
) -> int:
    pixels = image.load()
    width, height = image.size
    visited = [[False for _ in range(width)] for _ in range(height)]
    removed = 0

    for y in range(height):
        for x in range(width):
            if visited[y][x]:
                continue

            r, g, b, a = pixels[x, y]
            if a == 0 or not is_background_pixel(r, g, b, min_brightness, max_channel_delta):
                continue

            queue: collections.deque[tuple[int, int]] = collections.deque([(x, y)])
            visited[y][x] = True
            component: list[tuple[int, int]] = []
            min_x = max_x = x
            min_y = max_y = y

            while queue:
                cx, cy = queue.popleft()
                component.append((cx, cy))
                min_x = min(min_x, cx)
                max_x = max(max_x, cx)
                min_y = min(min_y, cy)
                max_y = max(max_y, cy)

                for dx, dy in (
                    (-1, 0),
                    (1, 0),
                    (0, -1),
                    (0, 1),
                    (-1, -1),
                    (-1, 1),
                    (1, -1),
                    (1, 1),
                ):
                    nx = cx + dx
                    ny = cy + dy
                    if not (0 <= nx < width and 0 <= ny < height) or visited[ny][nx]:
                        continue
                    nr, ng, nb, na = pixels[nx, ny]
                    if na == 0 or not is_background_pixel(nr, ng, nb, min_brightness, max_channel_delta):
                        continue
                    visited[ny][nx] = True
                    queue.append((nx, ny))

            near_edge = (
                min_x <= edge_margin
                or min_y <= edge_margin
                or max_x >= width - edge_margin - 1
                or max_y >= height - edge_margin - 1
            )
            if near_edge and len(component) >= min_area:
                for cx, cy in component:
                    cr, cg, cb, _ = pixels[cx, cy]
                    pixels[cx, cy] = (cr, cg, cb, 0)
                removed += len(component)

    return removed


def parse_polygon_points(raw_points: list[str]) -> list[tuple[int, int]]:
    points: list[tuple[int, int]] = []
    for raw_point in raw_points:
        x_text, y_text = raw_point.split(",", 1)
        points.append((int(x_text), int(y_text)))
    return points


def remove_polygon_background(
    image: Image.Image,
    polygon_points: list[tuple[int, int]],
    min_brightness: int,
    max_channel_delta: int,
) -> int:
    mask = Image.new("L", image.size, 0)
    ImageDraw.Draw(mask).polygon(polygon_points, fill=255)
    mask_pixels = mask.load()
    pixels = image.load()
    width, height = image.size
    removed = 0

    for y in range(height):
        for x in range(width):
            if mask_pixels[x, y] == 0:
                continue
            r, g, b, a = pixels[x, y]
            if a == 0 or not is_background_pixel(r, g, b, min_brightness, max_channel_delta):
                continue
            pixels[x, y] = (r, g, b, 0)
            removed += 1

    return removed


def remove_checker_background(
    input_path: Path,
    output_path: Path,
    min_brightness: int,
    max_channel_delta: int,
    erase_polygons: list[list[tuple[int, int]]] | None = None,
    relaxed_min_brightness: int | None = None,
    relaxed_max_channel_delta: int | None = None,
    cleanup_edge_residuals: bool = False,
) -> tuple[int, int]:
    image = Image.open(input_path).convert("RGBA")
    width, height = image.size
    pixels = image.load()
    background_rgb = estimate_background_color(image, min_brightness, max_channel_delta)

    visited = [[False for _ in range(width)] for _ in range(height)]
    queue: collections.deque[tuple[int, int]] = collections.deque()

    def try_enqueue(x: int, y: int) -> None:
        if visited[y][x]:
            return
        r, g, b, _ = pixels[x, y]
        if not is_background_pixel(r, g, b, min_brightness, max_channel_delta):
            return
        visited[y][x] = True
        queue.append((x, y))

    for x in range(width):
        try_enqueue(x, 0)
        try_enqueue(x, height - 1)
    for y in range(height):
        try_enqueue(0, y)
        try_enqueue(width - 1, y)

    removed = 0
    while queue:
        x, y = queue.popleft()
        r, g, b, _ = pixels[x, y]
        pixels[x, y] = (r, g, b, 0)
        removed += 1

        for dx, dy in (
            (-1, 0),
            (1, 0),
            (0, -1),
            (0, 1),
            (-1, -1),
            (-1, 1),
            (1, -1),
            (1, 1),
        ):
            nx = x + dx
            ny = y + dy
            if 0 <= nx < width and 0 <= ny < height:
                try_enqueue(nx, ny)

    if cleanup_edge_residuals:
        removed += remove_residual_edge_components(
            image,
            min_brightness=min_brightness,
            max_channel_delta=max_channel_delta,
            edge_margin=max(96, min(width, height) // 8),
            min_area=max(800, (width * height) // 2000),
        )
    for polygon_points in erase_polygons or []:
        removed += remove_polygon_background(
            image,
            polygon_points,
            min_brightness=relaxed_min_brightness or min_brightness,
            max_channel_delta=relaxed_max_channel_delta or max_channel_delta,
        )
    decontaminate_edge_pixels(image, background_rgb)

    output_path.parent.mkdir(parents=True, exist_ok=True)
    image.save(output_path)
    return removed, width * height


def collect_input_files(input_path: Path) -> list[Path]:
    if input_path.is_dir():
        return sorted(
            path
            for path in input_path.iterdir()
            if path.is_file() and path.suffix.lower() in {".png", ".jpg", ".jpeg", ".webp"}
        )
    return [input_path]


def build_output_path(input_path: Path, output: Path | None, suffix: str, input_root: Path) -> Path:
    if output is None:
        return input_path.with_name(f"{input_path.stem}{suffix}.png")
    if output.suffix:
        return output
    return output / input_path.relative_to(input_root).with_suffix(".png")


def parse_args() -> argparse.Namespace:
    parser = argparse.ArgumentParser(description="Remove light checkerboard-like backgrounds and export transparent PNG.")
    parser.add_argument("input", type=Path, help="Input image file or directory.")
    parser.add_argument("-o", "--output", type=Path, help="Output file or output directory.")
    parser.add_argument("--suffix", default="_transparent", help="Suffix for output filenames when no explicit file path is given.")
    parser.add_argument("--min-brightness", type=int, default=214, help="Minimum RGB channel value for a background candidate.")
    parser.add_argument("--max-channel-delta", type=int, default=20, help="Maximum RGB spread for a background candidate.")
    parser.add_argument("--relaxed-min-brightness", type=int, default=180, help="Relaxed minimum RGB channel value for polygon cleanup.")
    parser.add_argument("--relaxed-max-channel-delta", type=int, default=28, help="Relaxed maximum RGB spread for polygon cleanup.")
    parser.add_argument(
        "--erase-polygon",
        nargs="+",
        action="append",
        metavar="X,Y",
        help="Polygon points for extra cleanup. Example: --erase-polygon 560,40 690,0 825,120 790,185 545,185",
    )
    parser.add_argument("--cleanup-edge-residuals", action="store_true", help="Remove large neutral residual components near the image edge.")
    return parser.parse_args()


def main() -> int:
    args = parse_args()
    input_path = args.input.expanduser().resolve()
    files = collect_input_files(input_path)

    if not files:
        raise SystemExit("No supported image files found.")

    input_root = input_path if input_path.is_dir() else input_path.parent

    for file_path in files:
        output_path = build_output_path(file_path, args.output, args.suffix, input_root)
        removed, total = remove_checker_background(
            file_path,
            output_path,
            min_brightness=args.min_brightness,
            max_channel_delta=args.max_channel_delta,
            erase_polygons=[parse_polygon_points(points) for points in args.erase_polygon or []],
            relaxed_min_brightness=args.relaxed_min_brightness,
            relaxed_max_channel_delta=args.relaxed_max_channel_delta,
            cleanup_edge_residuals=args.cleanup_edge_residuals,
        )
        ratio = removed / total * 100
        print(f"{file_path.name} -> {output_path} | removed {removed}/{total} pixels ({ratio:.2f}%)")

    return 0


if __name__ == "__main__":
    raise SystemExit(main())
