from __future__ import annotations

from pathlib import Path

from PIL import Image, ImageDraw


ROOT = Path(__file__).resolve().parent.parent
OUT_DIR = ROOT / "assets" / "resource" / "image"


def draw_sakura(draw: ImageDraw.ImageDraw, center: tuple[float, float], petal_r: float, color):
    cx, cy = center
    offsets = [
        (0, -petal_r * 1.35),
        (petal_r * 1.18, -petal_r * 0.35),
        (petal_r * 0.72, petal_r * 0.95),
        (-petal_r * 0.72, petal_r * 0.95),
        (-petal_r * 1.18, -petal_r * 0.35),
    ]
    for dx, dy in offsets:
        x0 = cx + dx - petal_r
        y0 = cy + dy - petal_r * 1.02
        x1 = cx + dx + petal_r
        y1 = cy + dy + petal_r * 1.08
        draw.ellipse((x0, y0, x1, y1), fill=color)

    draw.ellipse(
        (
            cx - petal_r * 0.48,
            cy - petal_r * 0.48,
            cx + petal_r * 0.48,
            cy + petal_r * 0.48,
        ),
        fill=(0, 0, 0, 0),
    )


def main():
    size = 512
    img = Image.new("RGBA", (size, size), (0, 0, 0, 0))
    draw = ImageDraw.Draw(img)

    icon_color = (245, 245, 245, 255)

    frame = [
        (142, 130),
        (382, 174),
        (338, 404),
        (98, 360),
    ]
    frame_width = 40

    for start, end in zip(frame, frame[1:] + frame[:1]):
        draw.line((start, end), fill=icon_color, width=frame_width, joint="curve")

    draw.line((244, 78, 142, 130), fill=icon_color, width=24)
    draw.line((244, 78, 382, 174), fill=icon_color, width=24)

    ring_box = (220, 54, 268, 102)
    draw.ellipse(ring_box, outline=icon_color, width=14)

    blossom = Image.new("RGBA", (size, size), (0, 0, 0, 0))
    blossom_draw = ImageDraw.Draw(blossom)
    draw_sakura(blossom_draw, (244, 74), 30, icon_color)
    blossom = blossom.rotate(-12, resample=Image.Resampling.BICUBIC, center=(244, 74))
    img.alpha_composite(blossom)

    out_path = OUT_DIR / "app_icon_tiny.png"
    img.save(out_path)
    print(out_path)


if __name__ == "__main__":
    main()
