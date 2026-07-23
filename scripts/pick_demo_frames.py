#!/usr/bin/env python3
"""Extract README stills from docs/main.gif (interesting screens, not connections).

Timeline is approximate for docs/demo.tape @ 24fps after Show.
"""

from __future__ import annotations

import sys
from pathlib import Path

try:
    from PIL import Image
except ImportError:
    print("pick_demo_frames: Pillow required (pip install pillow)", file=sys.stderr)
    sys.exit(1)

# name -> (from_frac, to_frac) windows inside main.gif
# Prefer dense browse/detail/search screens over the connections splash.
STILLS = {
    "indices.png": (0.12, 0.22),
    "documents.png": (0.25, 0.36),
    "detail.png": (0.36, 0.42),
    "nodes.png": (0.50, 0.58),
    "products.png": (0.72, 0.82),
}


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


def pick_best(gif_path: Path, out_path: Path, from_frac: float, to_frac: float) -> None:
    im = Image.open(gif_path)
    n = getattr(im, "n_frames", 1)
    start = int((n - 1) * from_frac)
    end = int((n - 1) * to_frac)
    if end <= start:
        end = min(n - 1, start + 10)

    step = 1
    if end - start > 80:
        step = max(1, (end - start) // 50)

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

    im.seek(best_idx)
    frame = im.convert("RGB")
    if frame.size[0] < 800 or frame.size[1] < 400:
        raise SystemExit(f"pick_demo_frames: bad frame size {frame.size} for {out_path}")
    frame.save(out_path, format="PNG")
    print(f"  {out_path.name} ← frame {best_idx}/{n} ({frame.size[0]}x{frame.size[1]}, score {best_score:.0f})")


def main() -> int:
    root = Path(__file__).resolve().parents[1]
    gif = root / "docs/main.gif"
    if not gif.exists():
        print(f"pick_demo_frames: missing {gif}", file=sys.stderr)
        return 1
    out_dir = root / "docs"
    for name, (a, b) in STILLS.items():
        pick_best(gif, out_dir / name, a, b)
    return 0


if __name__ == "__main__":
    raise SystemExit(main())
