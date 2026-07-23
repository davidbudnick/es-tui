#!/usr/bin/env python3
"""Extract README stills from VHS demo GIFs.

main.png  → connections (main) page near the start of main.gif
metrics.png → densest late frame of metrics.gif
"""

from __future__ import annotations

import sys
from pathlib import Path

try:
    from PIL import Image
except ImportError:
    print("pick_demo_frames: Pillow required (pip install pillow)", file=sys.stderr)
    sys.exit(1)


def frame_score(im: Image.Image) -> float:
    im = im.convert("RGB")
    w, h = im.size
    step_x = max(1, w // 240)
    step_y = max(1, h // 135)
    sat = 0.0
    nonbg = 0.0
    for y in range(0, h, step_y):
        for x in range(0, w, step_x):
            r, g, b = im.getpixel((x, y))
            if abs(r - 40) + abs(g - 42) + abs(b - 54) > 35:
                nonbg += 1
            if max(r, g, b) - min(r, g, b) > 40:
                sat += 1
    return nonbg * 1.2 + sat * 2.5


def save_frame(im: Image.Image, idx: int, out_path: Path, label: str) -> None:
    im.seek(idx)
    frame = im.convert("RGB")
    if frame.size[0] < 800 or frame.size[1] < 400:
        raise SystemExit(f"pick_demo_frames: bad frame size {frame.size} for {out_path}")
    frame.save(out_path, format="PNG")
    n = getattr(im, "n_frames", 1)
    print(f"  {out_path} ← {label} frame {idx}/{n} ({frame.size[0]}x{frame.size[1]})")


def pick_connections(gif_path: Path, out_path: Path) -> None:
    """Main page = first stable connections screen after the app is up."""
    im = Image.open(gif_path)
    n = getattr(im, "n_frames", 1)
    # Connections hold is the first ~4–6s after Show (~100–150 frames @24fps).
    # Skip the first few frames in case the terminal is still painting.
    start = min(8, max(0, n // 50))
    end = min(n - 1, max(start + 10, int(n * 0.12)))

    best_idx = start
    best_score = -1.0
    for i in range(start, end + 1):
        im.seek(i)
        score = frame_score(im)
        # Prefer colorful frames (ELASTIC logo pink/yellow/teal/blue).
        if score > best_score:
            best_score = score
            best_idx = i
    save_frame(im, best_idx, out_path, gif_path.name + " (connections/main)")


def pick_best(gif_path: Path, out_path: Path, from_frac: float, to_frac: float) -> None:
    im = Image.open(gif_path)
    n = getattr(im, "n_frames", 1)
    start = int((n - 1) * from_frac)
    end = int((n - 1) * to_frac)
    if end <= start:
        end = n - 1

    step = 1
    if end - start > 120:
        step = max(1, (end - start) // 80)

    best_idx = start
    best_score = -1.0
    for i in range(start, end + 1, step):
        im.seek(i)
        score = frame_score(im)
        if score > best_score:
            best_score = score
            best_idx = i

    lo = max(start, best_idx - step)
    hi = min(end, best_idx + step)
    for i in range(lo, hi + 1):
        im.seek(i)
        score = frame_score(im)
        if score > best_score:
            best_score = score
            best_idx = i

    save_frame(im, best_idx, out_path, gif_path.name)


def main() -> int:
    root = Path(__file__).resolve().parents[1]
    # README main still MUST be the connections / main page.
    pick_connections(root / "docs/main.gif", root / "docs/main.png")
    pick_best(root / "docs/metrics.gif", root / "docs/metrics.png", 0.45, 0.98)
    return 0


if __name__ == "__main__":
    raise SystemExit(main())
