#!/usr/bin/env bash
# Generate desktop/build/appicon.png from media/logo.svg (nui wordmark).
# Requires: python3 (stdlib), npx (for @resvg/resvg-js-cli).
set -euo pipefail

ROOT="$(cd "$(dirname "$0")/.." && pwd)"
SRC="$ROOT/media/logo.svg"
OUT="$ROOT/desktop/build/appicon.png"
# Portable mktemp: GNU (Linux CI) requires XXXXXX; macOS BSD accepts this form too.
TMP_DIR="$(mktemp -d "${TMPDIR:-/tmp}/nui-icon.XXXXXX")"
TMP_SVG="$TMP_DIR/icon.svg"
TMP_PNG="$TMP_DIR/icon.png"
trap 'rm -rf "$TMP_DIR"' EXIT

if [[ ! -f "$SRC" ]]; then
  echo "error: missing $SRC" >&2
  exit 1
fi

mkdir -p "$ROOT/desktop/build"
rm -f "$ROOT/desktop/build"/_debug-logo-*.png "$ROOT/desktop/build"/_debug-n-*.png

# media/logo.svg once used a viewBox that hugged the ink (esp. bottoms ~1.7u),
# which clipped antialiasing and made stems look shaved. Render with an explicit
# padded box, keep a soft AA margin when cropping, then geometrically center.
python3 - "$SRC" "$TMP_SVG" <<'PY'
import re
import sys

src_path, dst_path = sys.argv[1], sys.argv[2]
raw = open(src_path, encoding="utf-8").read()

body = re.sub(r"<\?xml[^>]*\?>", "", raw)
body = re.sub(r"<style>[\s\S]*?</style>", "", body)
body = re.sub(r"</?svg[^>]*>", "", body)
body = body.replace('class="wordmark"', 'fill="#e8e8e8"')

# Matches media/logo.svg — generous pad around measured ink
# (104.7,5.1)–(864.4,428.3).
VIEWBOX = "60 -40 850 520"

out = f"""<svg xmlns="http://www.w3.org/2000/svg" width="2048" height="2048" viewBox="0 0 2048 2048">
  <rect width="2048" height="2048" fill="#0a0a0a"/>
  <svg x="0" y="0" width="2048" height="2048" viewBox="{VIEWBOX}" preserveAspectRatio="xMidYMid meet">
{body}
  </svg>
</svg>
"""
open(dst_path, "w", encoding="utf-8").write(out)
PY

npx --yes @resvg/resvg-js-cli --fit-width 2048 "$TMP_SVG" "$TMP_PNG" >/dev/null

python3 - "$TMP_PNG" "$OUT" <<'PY'
import struct
import sys
import zlib

src, dst = sys.argv[1], sys.argv[2]
SIZE = 1024
MARGIN = int(SIZE * 0.10)
AA_PAD = 12  # keep soft fringe outside thresholded ink


def read_png_rgba(path: str) -> tuple[int, int, bytearray]:
    with open(path, "rb") as f:
        assert f.read(8) == b"\x89PNG\r\n\x1a\n", "not a PNG"
        width = height = None
        color_type = None
        idat = bytearray()
        while True:
            hdr = f.read(4)
            if not hdr:
                break
            (length,) = struct.unpack(">I", hdr)
            ctype = f.read(4)
            data = f.read(length)
            f.read(4)
            if ctype == b"IHDR":
                width, height, bit_depth, color_type, *_ = struct.unpack(">IIBBBBB", data)
                if bit_depth != 8 or color_type not in (2, 6):
                    raise SystemExit(f"unsupported PNG: bit={bit_depth} color={color_type}")
            elif ctype == b"IDAT":
                idat.extend(data)
            elif ctype == b"IEND":
                break
    assert width and height and color_type is not None
    raw = bytearray(zlib.decompress(bytes(idat)))
    bpp = 4 if color_type == 6 else 3
    stride = width * bpp
    rows: list[bytearray] = []
    i = 0

    def paeth(a: int, b: int, c: int) -> int:
        p = a + b - c
        pa, pb, pc = abs(p - a), abs(p - b), abs(p - c)
        if pa <= pb and pa <= pc:
            return a
        if pb <= pc:
            return b
        return c

    for _ in range(height):
        filt = raw[i]
        i += 1
        row = bytearray(raw[i : i + stride])
        i += stride
        prev = rows[-1] if rows else None
        if filt == 1:
            for x in range(bpp, stride):
                row[x] = (row[x] + row[x - bpp]) & 255
        elif filt == 2:
            for x in range(stride):
                row[x] = (row[x] + prev[x]) & 255
        elif filt == 3:
            for x in range(stride):
                left = row[x - bpp] if x >= bpp else 0
                up = prev[x] if prev is not None else 0
                row[x] = (row[x] + ((left + up) >> 1)) & 255
        elif filt == 4:
            for x in range(stride):
                left = row[x - bpp] if x >= bpp else 0
                up = prev[x] if prev is not None else 0
                ul = prev[x - bpp] if prev is not None and x >= bpp else 0
                row[x] = (row[x] + paeth(left, up, ul)) & 255
        elif filt != 0:
            raise SystemExit(f"unsupported PNG filter {filt}")
        rows.append(row)

    out = bytearray(width * height * 4)
    o = 0
    for row in rows:
        if bpp == 4:
            out[o : o + stride] = row
            o += stride
        else:
            for x in range(width):
                r, g, b = row[x * 3 : x * 3 + 3]
                out[o : o + 4] = bytes((r, g, b, 255))
                o += 4
    return width, height, out


def write_png_rgba(path: str, width: int, height: int, rgba: bytes) -> None:
    def chunk(tag: bytes, data: bytes) -> bytes:
        return (
            struct.pack(">I", len(data))
            + tag
            + data
            + struct.pack(">I", zlib.crc32(tag + data) & 0xFFFFFFFF)
        )

    raw = bytearray()
    stride = width * 4
    for y in range(height):
        raw.append(0)
        raw.extend(rgba[y * stride : (y + 1) * stride])
    ihdr = struct.pack(">IIBBBBB", width, height, 8, 6, 0, 0, 0)
    open(path, "wb").write(
        b"\x89PNG\r\n\x1a\n"
        + chunk(b"IHDR", ihdr)
        + chunk(b"IDAT", zlib.compress(bytes(raw), 9))
        + chunk(b"IEND", b"")
    )


def bilinear_resize(src: bytearray, sw: int, sh: int, dw: int, dh: int) -> bytearray:
    out = bytearray(dw * dh * 4)
    for y in range(dh):
        sy = (y + 0.5) * sh / dh - 0.5
        y0 = max(0, min(sh - 1, int(sy)))
        y1 = min(y0 + 1, sh - 1)
        fy = sy - int(sy)
        for x in range(dw):
            sx = (x + 0.5) * sw / dw - 0.5
            x0 = max(0, min(sw - 1, int(sx)))
            x1 = min(x0 + 1, sw - 1)
            fx = sx - int(sx)
            di = (y * dw + x) * 4
            for c in range(4):
                v00 = src[(y0 * sw + x0) * 4 + c]
                v10 = src[(y0 * sw + x1) * 4 + c]
                v01 = src[(y1 * sw + x0) * 4 + c]
                v11 = src[(y1 * sw + x1) * 4 + c]
                v0 = v00 * (1 - fx) + v10 * fx
                v1 = v01 * (1 - fx) + v11 * fx
                out[di + c] = int(v0 * (1 - fy) + v1 * fy + 0.5)
    return out


def is_ink(r: int, g: int, b: int, a: int) -> bool:
    return a > 20 and (r + g + b) / 3 > 40


w, h, px = read_png_rgba(src)
xs: list[int] = []
ys: list[int] = []
for y in range(h):
    for x in range(w):
        i = (y * w + x) * 4
        if is_ink(px[i], px[i + 1], px[i + 2], px[i + 3]):
            xs.append(x)
            ys.append(y)
if not xs:
    raise SystemExit("logo render produced no visible pixels")

min_x, max_x = min(xs), max(xs)
min_y, max_y = min(ys), max(ys)

if min_x <= 2 or min_y <= 2 or max_x >= w - 3 or max_y >= h - 3:
    raise SystemExit(
        f"ink touches raster edge (bbox {min_x},{min_y}-{max_x},{max_y} in {w}x{h}); viewBox too tight"
    )

min_x = max(0, min_x - AA_PAD)
min_y = max(0, min_y - AA_PAD)
max_x = min(w - 1, max_x + AA_PAD)
max_y = min(h - 1, max_y + AA_PAD)
cw, ch = max_x - min_x + 1, max_y - min_y + 1

cropped = bytearray(cw * ch * 4)
for y in range(ch):
    src_i = ((min_y + y) * w + min_x) * 4
    dst_i = y * cw * 4
    cropped[dst_i : dst_i + cw * 4] = px[src_i : src_i + cw * 4]

max_side = SIZE - 2 * MARGIN
scale = min(max_side / cw, max_side / ch)
nw = max(1, int(round(cw * scale)))
nh = max(1, int(round(ch * scale)))
logo = bilinear_resize(cropped, cw, ch, nw, nh)

# Nudge up for optical centering (sparkles inflate the bbox above the letter bodies).
x = (SIZE - nw) // 2
y = (SIZE - nh) // 2 - int(SIZE * 0.06)

canvas = bytearray(SIZE * SIZE * 4)
for i in range(0, len(canvas), 4):
    canvas[i : i + 4] = bytes((10, 10, 10, 255))

bg = 10
for ly in range(nh):
    for lx in range(nw):
        si = (ly * nw + lx) * 4
        r, g, b, a = logo[si], logo[si + 1], logo[si + 2], logo[si + 3]
        # Skip near-background samples from the AA pad region.
        if a < 8 or (r + g + b) / 3 < bg + 8:
            continue
        di = ((y + ly) * SIZE + (x + lx)) * 4
        src_a = a / 255.0
        for c, v in enumerate((r, g, b)):
            canvas[di + c] = int(v * src_a + canvas[di + c] * (1 - src_a) + 0.5)
        canvas[di + 3] = 255

write_png_rgba(dst, SIZE, SIZE, bytes(canvas))

# Guard: five equal stems at mid-height (left stem not clipped).
mid = nh // 2
runs: list[int] = []
start = None
for lx in range(nw):
    i = (mid * nw + lx) * 4
    on = logo[i + 3] > 20 and sum(logo[i : i + 3]) / 3 > 40
    if on and start is None:
        start = lx
    elif not on and start is not None:
        runs.append(lx - start)
        start = None
if start is not None:
    runs.append(nw - start)
if len(runs) < 5:
    raise SystemExit(f"expected 5 stems at mid-height, got {runs}")
left, *rest = runs[:5]
if left < min(rest) * 0.85:
    raise SystemExit(f"left stem thinner than others: {runs[:5]}")

print(
    f"wrote {dst} logo={nw}x{nh} at ({x},{y}) "
    f"margins L={x} R={SIZE - x - nw} T={y} B={SIZE - y - nh} stems={runs[:5]}"
)
PY
