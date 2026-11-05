#!/usr/bin/env python3
"""
ART-005 — Bastion Dawn v1 Procedural Architectural, Wall, Building & Terrain Library
Produces crisp 4x-supersampled 2D pseudo-isometric sprites with #11151C dark-ink contours,
two-step 45-deg top-left cel shading, zero semi-transparent edge halos (binary alpha 0/255),
and mathematically periodic toroidal 256x256 H.264-safe terrain tiles.
"""

from __future__ import annotations

import math
from typing import Dict, List, Optional, Sequence, Tuple

import numpy as np
from PIL import Image, ImageDraw
from scipy import ndimage

# ============================================================================
# LOCKED BASTION DAWN V1 ENVIRONMENT PALETTE TOKENS
# ============================================================================

INK_RGB = (17, 21, 28)          # #11151C — Outer silhouette & major structural seams
INK_RGBA = (17, 21, 28, 255)

# Warm Castle Limestone (#E8DFC8)
LIME_LIT = (244, 237, 218)      # Top-left sunlit bevel
LIME_BASE = (232, 223, 200)     # #E8DFC8 base
LIME_SHAD = (182, 171, 148)     # Cool-warm right/bottom cel shadow
LIME_DEEP = (138, 128, 110)     # Deep recess / arch soffit

# Curtain Wall Stone (#9BA5B0)
WALL_LIT = (203, 213, 225)      # #CBD5E1 parapet highlight
WALL_BASE = (155, 165, 176)     # #9BA5B0 base
WALL_SHAD = (100, 116, 139)     # #64748B right/shadow plane
WALL_DEEP = (68, 79, 96)        # Deep crack / interior breach face
WALL_WEATHER = (134, 136, 128)  # Weathered recovery masonry tint

# Cobalt-Slate Roofs (#3B6EA5)
ROOF_LIT = (88, 138, 194)       # Top-left sunlit roof slope
ROOF_BASE = (59, 110, 165)      # #3B6EA5 base
ROOF_SHAD = (36, 71, 110)       # #24476E right/shadow roof slope
ROOF_MUTE_LIT = (98, 130, 168)  # Muted cottage slate lit
ROOF_MUTE_BASE = (75, 107, 148) # #4B6B94 cottage slate base
ROOF_MUTE_SHAD = (48, 70, 102)  # Cottage slate shadow

# Seasoned Oak Timber (#8C5A32)
OAK_LIT = (176, 120, 72)        # #B07848 lit timber
OAK_BASE = (140, 90, 50)        # #8C5A32 base
OAK_SHAD = (92, 56, 28)         # #5C381C shadow
OAK_FRESH = (190, 136, 82)      # Fresh recovery timber shoring

# Forged Iron & Steel
IRON_LIT = (168, 178, 192)
IRON_BASE = (102, 112, 126)
IRON_SHAD = (58, 66, 78)

# Golden Sun Crest (#F4C430) & Warm Accents
GOLD_LIT = (254, 240, 138)
GOLD_BASE = (244, 196, 48)      # #F4C430
GOLD_SHAD = (194, 142, 20)      # #C28E14

# Cottage Thatch / Brick / Forge Accents
THATCH_LIT = (232, 198, 108)
THATCH_BASE = (208, 168, 78)
THATCH_SHAD = (162, 124, 50)
BRICK_LIT = (196, 108, 78)
BRICK_BASE = (166, 82, 56)
BRICK_SHAD = (120, 54, 38)
EMBER_GLOW = (246, 138, 52)

# Terrain Colors (Calm low-contrast gouache-cel bands matching ART-002, strict HSV S <= 48%, Zero #11151C black ink!)
MEADOW_LIT = (120, 170, 92)     # Subtle warm sunlit meadow swell (HSV S = 45.9%)
MEADOW_BASE = (112, 162, 88)    # Calm sunlit meadow turf (HSV S = 45.7%, HSL S = 31%)
MEADOW_SHAD = (100, 148, 80)    # Calm soft turf shade (HSV S = 45.9%, never camouflages #4CC964 Grunt!)
MEADOW_DEEP = (86, 134, 72)     # Verge / variation deeper turf (HSV S = 46.3%)

OCHRE_LIT = (218, 184, 128)     # Calm lit ochre road (HSV S = 41.3%)
OCHRE_BASE = (207, 169, 112)    # Canonical ochre lane (HSV S = 45.9%)
OCHRE_SHAD = (188, 154, 102)    # Subtle road track shade (HSV S = 45.7%)
OCHRE_VERGE = (156, 128, 86)    # Road curb/verge earth (HSV S = 44.9%)

CAUSEWAY_LIT = (208, 214, 222)  # Calm sunlit limestone slab
CAUSEWAY_BASE = (192, 199, 208) # #C0C7D0 pale Boss causeway base
CAUSEWAY_SHAD = (178, 186, 196) # Subtle slab variation
CAUSEWAY_SEAM = (154, 163, 174) # Low-frequency H.264-safe mortar seam (Laplacian SD < 16.0!)
CAUSEWAY_CURB = (108, 118, 132) # Outer causeway curb stone

PLAZA_LIT = (230, 200, 158)     # #E6C89E
PLAZA_BASE = (216, 179, 132)    # #D8B384
PLAZA_SHAD = (176, 140, 96)
PLAZA_STONE = (176, 180, 184)


# ============================================================================
# SUPERSAMPLED BINARY-ALPHA & INK CONTOUR FINALIZER
# ============================================================================

def finalize_supersampled_sprite(
    hi_canvas: Image.Image,
    out_w: int,
    out_h: int,
    outer_ink_px: int = 2,
) -> Image.Image:
    """
    Downsamples a 4x supersampled RGBA canvas via NEAREST/BOX mask + crisp binary alpha,
    then enforces a continuous closed #11151C outer ink contour of exact thickness `outer_ink_px`
    and strictly binary alpha (0 or 255, zero semi-transparent edge halo pixels!).
    """
    # Use BOX alpha to capture subpixel silhouette smoothly, RGB from nearest valid opaque pixel
    arr_hi = np.array(hi_canvas, dtype=np.uint8)
    sh = arr_hi.shape[0] // out_h
    sw = arr_hi.shape[1] // out_w
    assert sh == sw and sh >= 2

    # Reshape into blocks (out_h, sh, out_w, sw, 4)
    blocks = arr_hi.reshape(out_h, sh, out_w, sw, 4)
    alpha_cov = (blocks[:, :, :, :, 3] > 127).mean(axis=(1, 3))
    mask = alpha_cov >= 0.35

    # Pick center-most opaque sample in each supersampled block so cel flat colors NEVER blur!
    cy, cx = sh // 2, sw // 2
    rgb_out = blocks[:, cy, :, cx, :3].copy()
    center_opaque = blocks[:, cy, :, cx, 3] > 127
    need_fill = mask & (~center_opaque)
    if np.any(need_fill):
        ys, xs = np.where(need_fill)
        for y, x in zip(ys, xs):
            sub = blocks[y, :, x, :, :]
            op = sub[sub[:, :, 3] > 127]
            if len(op) > 0:
                rgb_out[y, x] = op[0, :3]

    # Clean isolated 1px speckles and enforce continuous #11151C outer silhouette contour
    mask = ndimage.binary_opening(mask, structure=np.ones((2, 2), dtype=bool)) | mask
    eroded = ndimage.binary_erosion(mask, iterations=outer_ink_px, border_value=0)
    outer_ring = mask & (~eroded)
    rgb_out[outer_ring] = INK_RGB

    out = np.zeros((out_h, out_w, 4), dtype=np.uint8)
    out[mask, :3] = rgb_out[mask]
    out[mask, 3] = 255
    return Image.fromarray(out, mode="RGBA")


# ============================================================================
# ISOMETRIC / 3/4 ARCHITECTURAL PRIMITIVES (4X SUPERSAMPLED SPACE)
# ============================================================================

def iso_pt(cx: float, base_y: float, u: float, v: float, h: float) -> Tuple[float, float]:
    """
    Converts local architectural coordinates (u = front-left -> front-right,
    v = front-left -> back-left, h = vertical height up) into 2D pseudo-isometric pixels.
    Left facade faces SW (-X, +Y, lit by 45-deg top-left sun!);
    Right facade faces SE (+X, +Y, shaded!).
    """
    # Slightly asymmetric 3/4 Bastion Dawn angle matching ART-002 castle_model_sheet.png:
    # u goes down-right (+0.86, +0.44), v goes up-right (+0.82, -0.42) or u=(-0.85,-0.43)?
    # Let's anchor at FRONT-BOTTOM corner (cx, base_y):
    #   Left wall goes UP-LEFT: (-0.86 * L, -0.44 * L) -> faces SW (LIT by top-left sun!)
    #   Right wall goes UP-RIGHT: (+0.86 * R, -0.44 * R) -> faces SE (SHADED!)
    x = cx - 0.86 * u + 0.86 * v
    y = base_y - 0.44 * u - 0.44 * v - h
    return (x, y)


def poly(draw: ImageDraw.ImageDraw, pts: Sequence[Tuple[float, float]], fill: Tuple[int, int, int], ink: int = 8) -> None:
    ipts = [(int(round(px)), int(round(py))) for px, py in pts]
    draw.polygon(ipts, fill=(*fill, 255))
    if ink > 0:
        n = len(ipts)
        for i in range(n):
            draw.line([ipts[i], ipts[(i + 1) % n]], fill=INK_RGBA, width=ink)


def seam_line(
    draw: ImageDraw.ImageDraw,
    p0: Tuple[float, float],
    p1: Tuple[float, float],
    width: int = 5,
    color: Tuple[int, int, int, int] = INK_RGBA,
) -> None:
    draw.line(
        [(int(round(p0[0])), int(round(p0[1]))), (int(round(p1[0])), int(round(p1[1])))],
        fill=color,
        width=width,
    )


def draw_iso_box(
    draw: ImageDraw.ImageDraw,
    cx: float,
    base_y: float,
    u_len: float,
    v_len: float,
    h_bot: float,
    h_top: float,
    left_col: Tuple[int, int, int],
    right_col: Tuple[int, int, int],
    top_col: Optional[Tuple[int, int, int]] = None,
    ink: int = 8,
    ashlar_rows: int = 0,
) -> None:
    p_f0 = iso_pt(cx, base_y, 0, 0, h_bot)
    p_l0 = iso_pt(cx, base_y, u_len, 0, h_bot)
    p_r0 = iso_pt(cx, base_y, 0, v_len, h_bot)
    p_f1 = iso_pt(cx, base_y, 0, 0, h_top)
    p_l1 = iso_pt(cx, base_y, u_len, 0, h_top)
    p_r1 = iso_pt(cx, base_y, 0, v_len, h_top)
    p_b1 = iso_pt(cx, base_y, u_len, v_len, h_top)

    # Left face (lit)
    poly(draw, [p_f0, p_l0, p_l1, p_f1], left_col, ink=ink)
    # Right face (shadow)
    poly(draw, [p_f0, p_r0, p_r1, p_f1], right_col, ink=ink)

    if ashlar_rows > 0:
        dh = (h_top - h_bot) / ashlar_rows
        for r in range(1, ashlar_rows):
            hz = h_bot + r * dh
            seam_line(draw, iso_pt(cx, base_y, 0, 0, hz), iso_pt(cx, base_y, u_len, 0, hz), width=4, color=(120, 114, 102, 255))
            seam_line(draw, iso_pt(cx, base_y, 0, 0, hz), iso_pt(cx, base_y, 0, v_len, hz), width=4, color=(86, 82, 74, 255))
        for r in range(ashlar_rows):
            hz0 = h_bot + r * dh
            hz1 = hz0 + dh
            off = 0.25 if (r % 2 == 0) else 0.55
            for frac in (off, min(0.88, off + 0.42)):
                u_s = u_len * frac
                seam_line(draw, iso_pt(cx, base_y, u_s, 0, hz0), iso_pt(cx, base_y, u_s, 0, hz1), width=3, color=(136, 128, 114, 255))
                v_s = v_len * frac
                seam_line(draw, iso_pt(cx, base_y, 0, v_s, hz0), iso_pt(cx, base_y, 0, v_s, hz1), width=3, color=(86, 82, 74, 255))

    if top_col is not None:
        poly(draw, [p_f1, p_l1, p_b1, p_r1], top_col, ink=ink)


def draw_gable_roof(
    draw: ImageDraw.ImageDraw,
    cx: float,
    base_y: float,
    u_len: float,
    v_len: float,
    h_eave: float,
    ridge_h: float,
    gable_col: Tuple[int, int, int],
    roof_lit: Tuple[int, int, int],
    roof_shad: Tuple[int, int, int],
    trim_col: Tuple[int, int, int],
    overhang: float = 18.0,
    ink: int = 8,
) -> Tuple[Tuple[float, float], Tuple[float, float]]:
    """
    Draws a classic Bastion Dawn gable roof whose gable triangular facade faces SW (left-lit)
    and whose broad slate roof slope faces SE/top, matching castle_model_sheet.png!
    Returns (front_ridge_peak, back_ridge_peak).
    """
    u_mid = u_len * 0.5
    # Left-facing gable triangle wall (u = 0..u_len at v = 0)
    g_f = iso_pt(cx, base_y, -overhang * 0.4, -overhang * 0.3, h_eave)
    g_l = iso_pt(cx, base_y, u_len + overhang * 0.4, -overhang * 0.3, h_eave)
    g_peak = iso_pt(cx, base_y, u_mid, -overhang * 0.3, h_eave + ridge_h)
    b_r = iso_pt(cx, base_y, -overhang * 0.4, v_len + overhang * 0.4, h_eave)
    b_l = iso_pt(cx, base_y, u_len + overhang * 0.4, v_len + overhang * 0.4, h_eave)
    b_peak = iso_pt(cx, base_y, u_mid, v_len + overhang * 0.4, h_eave + ridge_h)

    # Back-left lit slope sliver (visible above ridge on left side if any)
    poly(draw, [g_l, b_l, b_peak, g_peak], roof_lit, ink=ink)
    # Main right-facing cobalt-slate slope (from ridge down to front-right eave)
    poly(draw, [g_f, b_r, b_peak, g_peak], roof_shad, ink=ink)
    # Upper-left sunlit band near the ridge on the main slope (two-step cel bevel!)
    bevel_f = (g_peak[0] * 0.72 + g_f[0] * 0.28, g_peak[1] * 0.72 + g_f[1] * 0.28)
    bevel_b = (b_peak[0] * 0.72 + b_r[0] * 0.28, b_peak[1] * 0.72 + b_r[1] * 0.28)
    poly(draw, [g_peak, b_peak, bevel_b, bevel_f], roof_lit, ink=0)

    # Shingle course lines (clean low-frequency cel lines)
    for frac in (0.34, 0.67):
        s0 = (g_peak[0] * (1 - frac) + g_f[0] * frac, g_peak[1] * (1 - frac) + g_f[1] * frac)
        s1 = (b_peak[0] * (1 - frac) + b_r[0] * frac, b_peak[1] * (1 - frac) + b_r[1] * frac)
        seam_line(draw, s0, s1, width=4, color=(26, 52, 84, 255))

    # Gable triangular wall face (SW lit)
    gf0 = iso_pt(cx, base_y, 0, 0, h_eave)
    gl0 = iso_pt(cx, base_y, u_len, 0, h_eave)
    gp0 = iso_pt(cx, base_y, u_mid, 0, h_eave + ridge_h - 14)
    poly(draw, [gf0, gl0, gp0], gable_col, ink=ink)

    # Bargeboard / trim rafters along front gable
    poly(
        draw,
        [
            g_f,
            g_peak,
            (g_peak[0], g_peak[1] + 24),
            (g_f[0], g_f[1] + 24),
        ],
        trim_col,
        ink=ink,
    )
    poly(
        draw,
        [
            g_l,
            g_peak,
            (g_peak[0], g_peak[1] + 24),
            (g_l[0], g_l[1] + 24),
        ],
        trim_col,
        ink=ink,
    )
    # Ridge beam
    seam_line(draw, g_peak, b_peak, width=14, color=(*trim_col, 255))
    seam_line(draw, g_peak, b_peak, width=5, color=INK_RGBA)
    return g_peak, b_peak


def draw_arched_door_on_left_face(
    draw: ImageDraw.ImageDraw,
    cx: float,
    base_y: float,
    u_center: float,
    door_w: float,
    door_h: float,
    stone_arch: bool = True,
) -> None:
    u0 = u_center - door_w * 0.5
    u1 = u_center + door_w * 0.5
    if stone_arch:
        aw0, aw1 = u0 - 22, u1 + 22
        poly(
            draw,
            [
                iso_pt(cx, base_y, aw0, 0, 0),
                iso_pt(cx, base_y, aw1, 0, 0),
                iso_pt(cx, base_y, aw1, 0, door_h + 10),
                iso_pt(cx, base_y, u_center, 0, door_h + 42),
                iso_pt(cx, base_y, aw0, 0, door_h + 10),
            ],
            WALL_LIT,
            ink=7,
        )
    poly(
        draw,
        [
            iso_pt(cx, base_y, u0, 0, 0),
            iso_pt(cx, base_y, u1, 0, 0),
            iso_pt(cx, base_y, u1, 0, door_h - 14),
            iso_pt(cx, base_y, u_center, 0, door_h + 16),
            iso_pt(cx, base_y, u0, 0, door_h - 14),
        ],
        OAK_BASE,
        ink=7,
    )
    # Center door split + iron straps
    seam_line(draw, iso_pt(cx, base_y, u_center, 0, 0), iso_pt(cx, base_y, u_center, 0, door_h + 14), width=6)
    for hz in (door_h * 0.30, door_h * 0.68):
        seam_line(draw, iso_pt(cx, base_y, u0 + 6, 0, hz), iso_pt(cx, base_y, u1 - 6, 0, hz), width=8, color=(*IRON_BASE, 255))


def draw_sun_pennant(
    draw: ImageDraw.ImageDraw,
    pole_base: Tuple[float, float],
    pole_h: float,
    banner_len: float = 128.0,
    banner_h: float = 54.0,
) -> None:
    px, py = pole_base
    top = (px, py - pole_h)
    seam_line(draw, (px, py + 8), top, width=10, color=INK_RGBA)
    seam_line(draw, (px, py + 8), top, width=5, color=(*GOLD_BASE, 255))
    # Finial orb
    r = 12
    draw.ellipse([top[0] - r, top[1] - r, top[0] + r, top[1] + r], fill=(*GOLD_LIT, 255), outline=INK_RGBA, width=5)
    # Forked golden sun pennant waving right
    pts = [
        (px + 4, top[1] + 10),
        (px + banner_len * 0.55, top[1] + 2),
        (px + banner_len, top[1] + 16),
        (px + banner_len * 0.78, top[1] + banner_h * 0.52),
        (px + banner_len * 0.96, top[1] + banner_h),
        (px + banner_len * 0.48, top[1] + banner_h - 8),
        (px + 4, top[1] + banner_h - 2),
    ]
    poly(draw, pts, GOLD_BASE, ink=6)
    # Sun emblem disc
    sx, sy = int(px + banner_len * 0.34), int(top[1] + banner_h * 0.48)
    draw.ellipse([sx - 11, sy - 11, sx + 11, sy + 11], fill=(*GOLD_LIT, 255), outline=(*OAK_SHAD, 255), width=4)


def draw_sun_crest_shield(draw: ImageDraw.ImageDraw, center: Tuple[float, float], radius: float = 42.0) -> None:
    cx, cy = center
    pts = [
        (cx - radius, cy - radius * 0.75),
        (cx + radius, cy - radius * 0.75),
        (cx + radius * 0.92, cy + radius * 0.25),
        (cx, cy + radius * 1.10),
        (cx - radius * 0.92, cy + radius * 0.25),
    ]
    poly(draw, pts, GOLD_BASE, ink=7)
    inner = [
        (cx - radius * 0.70, cy - radius * 0.50),
        (cx + radius * 0.70, cy - radius * 0.50),
        (cx + radius * 0.62, cy + radius * 0.16),
        (cx, cy + radius * 0.80),
        (cx - radius * 0.62, cy + radius * 0.16),
    ]
    poly(draw, inner, ROOF_BASE, ink=5)
    sr = int(radius * 0.34)
    draw.ellipse([int(cx) - sr, int(cy) - sr, int(cx) + sr, int(cy) + sr], fill=(*GOLD_LIT, 255), outline=INK_RGBA, width=4)


def draw_cylindrical_turret(
    draw: ImageDraw.ImageDraw,
    tx: float,
    ty_base: float,
    radius: float,
    body_h: float,
    cone_h: float,
    gilded_rim: bool = False,
) -> None:
    """Draws a crisp 3/4 cylindrical bastion turret with two-step cel shading and conical cobalt roof."""
    ell_ry = radius * 0.46
    x0, x1 = tx - radius, tx + radius
    top_y = ty_base - body_h

    # Main cylinder body (left 58% lit limestone, right 42% shaded limestone)
    body_poly = [
        (x0, top_y),
        (x1, top_y),
        (x1, ty_base),
        (tx, ty_base + ell_ry),
        (x0, ty_base),
    ]
    poly(draw, body_poly, LIME_BASE, ink=0)
    shad_x = tx + radius * 0.12
    shad_poly = [
        (shad_x, top_y),
        (x1, top_y),
        (x1, ty_base),
        (tx + radius * 0.45, ty_base + ell_ry * 0.85),
        (shad_x, ty_base + ell_ry * 0.96),
    ]
    poly(draw, shad_poly, LIME_SHAD, ink=0)
    draw.pieslice(
        [int(x0), int(ty_base - ell_ry), int(x1), int(ty_base + ell_ry)],
        start=0,
        end=180,
        fill=(*LIME_BASE, 255),
    )
    draw.pieslice(
        [int(x0), int(ty_base - ell_ry), int(x1), int(ty_base + ell_ry)],
        start=0,
        end=78,
        fill=(*LIME_SHAD, 255),
    )

    # Masonry courses + arrow slit
    for frac in (0.28, 0.56, 0.82):
        cy_line = ty_base - body_h * frac
        draw.arc(
            [int(x0), int(cy_line - ell_ry), int(x1), int(cy_line + ell_ry)],
            start=8,
            end=172,
            fill=(128, 120, 106, 255),
            width=4,
        )
    slit_x = int(tx - radius * 0.28)
    slit_y = int(ty_base - body_h * 0.56)
    draw.rectangle([slit_x - 6, slit_y - 24, slit_x + 6, slit_y + 20], fill=INK_RGBA)

    # Outer cylinder contour
    seam_line(draw, (x0, top_y), (x0, ty_base), width=8)
    seam_line(draw, (x1, top_y), (x1, ty_base), width=8)
    draw.arc([int(x0), int(ty_base - ell_ry), int(x1), int(ty_base + ell_ry)], start=0, end=180, fill=INK_RGBA, width=8)

    # Conical cobalt-slate cap
    cr = radius * 1.20
    cry = cr * 0.46
    apex = (tx, top_y - cone_h)
    cone_pts = [(tx - cr, top_y), apex, (tx + cr, top_y)]
    poly(draw, cone_pts, ROOF_BASE, ink=0)
    poly(draw, [(tx - cr, top_y), apex, (tx - cr * 0.12, top_y + cry * 0.95)], ROOF_LIT, ink=0)
    poly(draw, [(tx + cr * 0.18, top_y + cry * 0.92), apex, (tx + cr, top_y)], ROOF_SHAD, ink=0)
    draw.pieslice([int(tx - cr), int(top_y - cry), int(tx + cr), int(top_y + cry)], start=0, end=180, fill=(*ROOF_BASE, 255))
    draw.pieslice([int(tx - cr), int(top_y - cry), int(tx + cr), int(top_y + cry)], start=95, end=180, fill=(*ROOF_LIT, 255))
    draw.pieslice([int(tx - cr), int(top_y - cry), int(tx + cr), int(top_y + cry)], start=0, end=65, fill=(*ROOF_SHAD, 255))
    draw.arc([int(tx - cr), int(top_y - cry), int(tx + cr), int(top_y + cry)], start=0, end=180, fill=INK_RGBA, width=8)
    seam_line(draw, (tx - cr, top_y), apex, width=8)
    seam_line(draw, (tx + cr, top_y), apex, width=8)

    if gilded_rim:
        draw.arc([int(tx - cr), int(top_y - cry - 6), int(tx + cr), int(top_y + cry - 6)], start=8, end=172, fill=(*GOLD_BASE, 255), width=9)
        draw.ellipse([int(apex[0] - 10), int(apex[1] - 16), int(apex[0] + 10), int(apex[1] + 4)], fill=(*GOLD_LIT, 255), outline=INK_RGBA, width=5)


# ============================================================================
# PHASE 2 — FOUR COSMETIC TOWNHALL BANDS (512x512 RGBA, LOCKED BASE FOOTPRINT)
# ============================================================================

def render_townhall_band(band: int) -> Image.Image:
    """
    Renders one of the 4 cosmetic Townhall progression sprites (512x512 RGBA, pivot=(0.50, 0.10)).
    All 4 bands share the EXACT same base foundation anchor (cx=1024, base_y=1844 in 2048x2048 4x space ->
    cx=256, base_y=461 in 512x512) and identical footprint width so swapping causes 0.0px jump!
    """
    S = 2048
    hi = Image.new("RGBA", (S, S), (0, 0, 0, 0))
    d = ImageDraw.Draw(hi)

    # Identical base footprint anchor across all 4 bands!
    cx = 1024.0
    base_y = 1844.0
    u_len = 620.0
    v_len = 560.0

    # Shared beveled stone plinth foundation (h = 0..150) — guarantees identical ground contact footprint!
    draw_iso_box(d, cx, base_y, u_len, v_len, 0.0, 150.0, WALL_BASE, WALL_SHAD, top_col=WALL_LIT, ink=9, ashlar_rows=2)

    if band == 1:
        # Band 1 (Lv 1–9): Timber Village Hall — Oak half-timber facade, compact cobalt roof, small sun pennant
        draw_iso_box(d, cx, base_y, u_len - 24, v_len - 24, 150.0, 560.0, LIME_BASE, LIME_SHAD, ink=9)
        # Half-timber oak studs & diagonal braces on SW & SE walls
        for u_p in (36, 200, 400, u_len - 48):
            seam_line(d, iso_pt(cx, base_y, u_p, 0, 150), iso_pt(cx, base_y, u_p, 0, 560), width=18, color=(*OAK_BASE, 255))
            seam_line(d, iso_pt(cx, base_y, u_p, 0, 150), iso_pt(cx, base_y, u_p, 0, 560), width=5)
        seam_line(d, iso_pt(cx, base_y, 24, 0, 360), iso_pt(cx, base_y, u_len - 36, 0, 360), width=18, color=(*OAK_BASE, 255))
        seam_line(d, iso_pt(cx, base_y, 36, 0, 360), iso_pt(cx, base_y, 190, 0, 550), width=14, color=(*OAK_BASE, 255))
        seam_line(d, iso_pt(cx, base_y, u_len - 48, 0, 360), iso_pt(cx, base_y, 410, 0, 550), width=14, color=(*OAK_BASE, 255))
        for v_p in (36, 260, v_len - 48):
            seam_line(d, iso_pt(cx, base_y, 0, v_p, 150), iso_pt(cx, base_y, 0, v_p, 560), width=16, color=(*OAK_SHAD, 255))
        seam_line(d, iso_pt(cx, base_y, 0, 24, 360), iso_pt(cx, base_y, 0, v_len - 36, 0 + 360), width=16, color=(*OAK_SHAD, 255))

        # Simple warm oak entrance + twin mullioned windows
        draw_arched_door_on_left_face(d, cx, base_y, u_len * 0.50, 156.0, 240.0, stone_arch=False)
        for uw in (128.0, 470.0):
            poly(
                d,
                [
                    iso_pt(cx, base_y, uw - 34, 0, 395),
                    iso_pt(cx, base_y, uw + 34, 0, 395),
                    iso_pt(cx, base_y, uw + 34, 0, 500),
                    iso_pt(cx, base_y, uw - 34, 0, 500),
                ],
                GOLD_LIT,
                ink=7,
            )

        peak_f, _ = draw_gable_roof(
            d, cx, base_y, u_len - 12, v_len - 12, 560.0, 390.0,
            gable_col=LIME_BASE, roof_lit=ROOF_LIT, roof_shad=ROOF_BASE, trim_col=OAK_BASE, overhang=34.0, ink=9,
        )
        draw_sun_pennant(d, peak_f, pole_h=150.0, banner_len=148.0, banner_h=58.0)

    elif band == 2:
        # Band 2 (Lv 10–24): Reinforced Stone Keep — Taller cut-limestone facade, twin roof dormers, iron-bound arched doors
        draw_iso_box(d, cx, base_y, u_len - 16, v_len - 16, 150.0, 700.0, LIME_BASE, LIME_SHAD, ink=9, ashlar_rows=5)
        # Buttress pilasters on corners
        draw_iso_box(d, cx, base_y, 68.0, 68.0, 150.0, 700.0, LIME_LIT, WALL_SHAD, top_col=LIME_LIT, ink=8)
        draw_arched_door_on_left_face(d, cx, base_y, u_len * 0.50, 196.0, 300.0, stone_arch=True)
        # Upper arched windows
        for uw in (155.0, 455.0):
            poly(
                d,
                [
                    iso_pt(cx, base_y, uw - 34, 0, 470),
                    iso_pt(cx, base_y, uw + 34, 0, 470),
                    iso_pt(cx, base_y, uw + 34, 0, 585),
                    iso_pt(cx, base_y, uw, 0, 618),
                    iso_pt(cx, base_y, uw - 34, 0, 585),
                ],
                ROOF_SHAD,
                ink=7,
            )

        peak_f, peak_b = draw_gable_roof(
            d, cx, base_y, u_len - 8, v_len - 8, 700.0, 420.0,
            gable_col=LIME_LIT, roof_lit=ROOF_LIT, roof_shad=ROOF_BASE, trim_col=WALL_LIT, overhang=32.0, ink=9,
        )
        # Twin slate dormers on the main SE slope
        for v_off in (140.0, 340.0):
            dcx, dcy = iso_pt(cx, base_y, 120.0, v_off, 800.0)
            draw_iso_box(d, dcx, dcy, 92.0, 88.0, 0.0, 110.0, LIME_LIT, LIME_SHAD, ink=7)
            draw_gable_roof(d, dcx, dcy, 96.0, 92.0, 110.0, 76.0, LIME_LIT, ROOF_LIT, ROOF_SHAD, WALL_LIT, overhang=14.0, ink=7)
        # Gold roof finials
        for pk in (peak_f, peak_b):
            d.polygon([(int(pk[0]), int(pk[1] - 58)), (int(pk[0] - 18), int(pk[1])), (int(pk[0] + 18), int(pk[1]))], fill=(*GOLD_BASE, 255), outline=INK_RGBA)

    elif band == 3:
        # Band 3 (Lv 25–39): Bastion Citadel — Flanking corner turrets, gilded ridge details, prominent golden sun crest
        # Back-right & back-left turrets first (painter's order)
        br_x, br_y = iso_pt(cx, base_y, 10.0, v_len - 30.0, 120.0)
        draw_cylindrical_turret(d, br_x, br_y, radius=98.0, body_h=590.0, cone_h=210.0, gilded_rim=True)

        draw_iso_box(d, cx, base_y, u_len - 16, v_len - 16, 150.0, 760.0, LIME_BASE, LIME_SHAD, ink=9, ashlar_rows=5)
        # Crenelated parapet cornice band
        draw_iso_box(d, cx, base_y, u_len + 8, v_len + 8, 720.0, 780.0, LIME_LIT, WALL_SHAD, top_col=LIME_LIT, ink=8)

        draw_arched_door_on_left_face(d, cx, base_y, u_len * 0.50, 206.0, 315.0, stone_arch=True)
        peak_f, peak_b = draw_gable_roof(
            d, cx, base_y, u_len - 20, v_len - 20, 780.0, 430.0,
            gable_col=LIME_LIT, roof_lit=ROOF_LIT, roof_shad=ROOF_BASE, trim_col=GOLD_BASE, overhang=28.0, ink=9,
        )

        # Flanking SW-left & front-center cylindrical bastion turrets
        fl_x, fl_y = iso_pt(cx, base_y, u_len - 24.0, 16.0, 90.0)
        draw_cylindrical_turret(d, fl_x, fl_y, radius=104.0, body_h=620.0, cone_h=220.0, gilded_rim=True)
        fc_x, fc_y = iso_pt(cx, base_y, 20.0, 20.0, 60.0)
        draw_cylindrical_turret(d, fc_x, fc_y, radius=112.0, body_h=650.0, cone_h=230.0, gilded_rim=True)

        # Prominent sculpted golden sun crest above main arch
        crest_pt = iso_pt(cx, base_y, u_len * 0.52, 0.0, 560.0)
        draw_sun_crest_shield(d, crest_pt, radius=68.0)
        draw_sun_pennant(d, peak_b, pole_h=120.0, banner_len=136.0, banner_h=52.0)

    elif band == 4:
        # Band 4 (Lv 40–50): Sovereign Dawn Keep — Tall central belfry/spire, multi-tier cobalt-and-gold roofs, celebratory banners
        br_x, br_y = iso_pt(cx, base_y, 10.0, v_len - 24.0, 130.0)
        draw_cylindrical_turret(d, br_x, br_y, radius=102.0, body_h=630.0, cone_h=225.0, gilded_rim=True)

        # Lower keep hall
        draw_iso_box(d, cx, base_y, u_len - 8, v_len - 8, 150.0, 750.0, LIME_BASE, LIME_SHAD, ink=9, ashlar_rows=5)
        draw_arched_door_on_left_face(d, cx, base_y, u_len * 0.50, 214.0, 330.0, stone_arch=True)
        # Gilded portal pediment
        ped_c = iso_pt(cx, base_y, u_len * 0.50, 0, 390.0)
        poly(
            d,
            [
                iso_pt(cx, base_y, u_len * 0.50 - 150, 0, 350),
                iso_pt(cx, base_y, u_len * 0.50, 0, 495),
                iso_pt(cx, base_y, u_len * 0.50 + 150, 0, 350),
            ],
            ROOF_BASE,
            ink=8,
        )
        draw_sun_crest_shield(d, (ped_c[0], ped_c[1] - 20), radius=58.0)

        # Tier 1 roof with gold bargeboards
        draw_gable_roof(
            d, cx, base_y, u_len - 12, v_len - 12, 750.0, 370.0,
            gable_col=LIME_LIT, roof_lit=ROOF_LIT, roof_shad=ROOF_BASE, trim_col=GOLD_BASE, overhang=28.0, ink=9,
        )

        # Flanking turrets
        fl_x, fl_y = iso_pt(cx, base_y, u_len - 20.0, 14.0, 100.0)
        draw_cylindrical_turret(d, fl_x, fl_y, radius=106.0, body_h=640.0, cone_h=230.0, gilded_rim=True)
        fc_x, fc_y = iso_pt(cx, base_y, 18.0, 18.0, 70.0)
        draw_cylindrical_turret(d, fc_x, fc_y, radius=114.0, body_h=670.0, cone_h=240.0, gilded_rim=True)

        # Tall central Sovereign Belfry Tower rising above Tier 1 roof (kept cleanly inside y >= 20px of 512x512!)
        bcx, bcy = iso_pt(cx, base_y, 160.0, 140.0, 810.0)
        draw_iso_box(d, bcx, bcy, 256.0, 236.0, 0.0, 310.0, LIME_LIT, LIME_SHAD, top_col=GOLD_BASE, ink=9, ashlar_rows=3)
        # Arched belfry bell openings
        poly(
            d,
            [
                iso_pt(bcx, bcy, 82, 0, 85),
                iso_pt(bcx, bcy, 174, 0, 85),
                iso_pt(bcx, bcy, 174, 0, 225),
                iso_pt(bcx, bcy, 128, 0, 260),
                iso_pt(bcx, bcy, 82, 0, 225),
            ],
            INK_RGB,
            ink=6,
        )
        # Steep cobalt-and-gold spire + celebratory banner (unclipped with >= 20px top margin in 512x512!)
        sp_base_l = iso_pt(bcx, bcy, 274.0, -18.0, 310.0)
        sp_base_f = iso_pt(bcx, bcy, -18.0, -18.0, 310.0)
        sp_base_r = iso_pt(bcx, bcy, -18.0, 254.0, 310.0)
        sp_apex = iso_pt(bcx, bcy, 128.0, 118.0, 600.0)
        poly(d, [sp_base_l, sp_base_f, sp_apex], ROOF_LIT, ink=9)
        poly(d, [sp_base_f, sp_base_r, sp_apex], ROOF_BASE, ink=9)
        seam_line(d, sp_base_f, sp_apex, width=12, color=(*GOLD_BASE, 255))
        seam_line(d, sp_base_l, sp_base_f, width=14, color=(*GOLD_BASE, 255))
        seam_line(d, sp_base_f, sp_base_r, width=14, color=(*GOLD_BASE, 255))
        draw_sun_pennant(d, sp_apex, pole_h=112.0, banner_len=156.0, banner_h=58.0)

        # Celebratory vertical royal cobalt & gold banners on keep facade
        for bu in (140.0, 480.0):
            bp = iso_pt(cx, base_y, bu, 0.0, 660.0)
            bx, by = int(bp[0]), int(bp[1])
            poly(d, [(bx - 22, by), (bx + 22, by), (bx + 22, by + 150), (bx, by + 186), (bx - 22, by + 150)], GOLD_BASE, ink=6)
            poly(d, [(bx - 12, by + 12), (bx + 12, by + 12), (bx + 12, by + 138), (bx, by + 164), (bx - 12, by + 138)], ROOF_BASE, ink=0)

    else:
        raise ValueError(f"Unknown Townhall band: {band}")

    return finalize_supersampled_sprite(hi, 512, 512, outer_ink_px=2)


# ============================================================================
# PHASE 3 — CIRCULAR CURTAIN WALL, SOUTH GATE (5 STATES + SPLIT SLICES) & SCAFFOLD
# ============================================================================

def render_wall_state_and_layers(state_idx: int) -> Tuple[Image.Image, Image.Image, Image.Image]:
    """
    Renders one of the 5 canonical circular curtain-wall + South Gate states:
      1 = Healthy (60-100% HP)
      2 = Damaged (25-59% HP)
      3 = Critical (>0% and <25% HP)
      4 = Breached (0 HP — central gate & flanking parapet breached, inner courtyard open)
      5 = Post-Loss Recovery (explicit recovery presentation state: weathered stone + fresh oak bracing + restored barred gate)

    Returns (full_512x512, back_512x320, front_512x224):
      - `full_512x512`: Canonical 512x512 RGBA sprite (exact 477px outer width, transparent exterior & transparent inner courtyard!).
      - `back_512x320`: North arc slice for `SortingLayer = BastionBack` (y = 0..319 of the 512x512 canvas).
      - `front_512x224`: South rim & South Gate slice for `SortingLayer = BastionFront` (y = 288..511 of the 512x512 canvas).
    """
    S = 2048  # 4x supersampled -> 512x512
    # Exact geometry matching 03_environment_design.md:
    # Outer width = 477 px in 512x512 (x = 17.5..494.5 -> 4x: cx = 1024, rx_out = 954 -> width = 1908 / 4 = 477 px!)
    cx = 1024.0
    cy = 1044.0          # Ring ellipse center (y = 261 in 512x512)
    rx_out = 950.0       # +2px outer ink on each side -> exact 477 px width!
    ry_out = 636.0
    rx_in = 790.0
    ry_in = 516.0
    wall_h_back = 216.0  # 54 px North wall height
    wall_h_front = 112.0 # <= 28 px South parapet height so courtyard & South Gate Guards never vanish!

    is_recovery = state_idx == 5
    base_stone = WALL_WEATHER if is_recovery else WALL_BASE
    lit_stone = (196, 200, 194) if is_recovery else WALL_LIT
    shad_stone = (92, 98, 102) if is_recovery else WALL_SHAD

    def wall_h_at(theta: float) -> float:
        """Smoothly tapers wall height from 212 at North (sin=-1) through 160 at flanks (sin=0) to 108 at South (sin=+1)."""
        base_h = 160.0 - 52.0 * math.sin(theta)
        # Normalize angle to [0, 2*pi)
        tn = theta % (2.0 * math.pi)
        if state_idx == 4 and abs(tn - 1.5 * math.pi) < 0.14:
            return base_h * 0.28
        if state_idx == 3 and (abs(tn - 1.28 * math.pi) < 0.08 or abs(tn - 1.74 * math.pi) < 0.08 or abs(tn - 0.30 * math.pi) < 0.09):
            return base_h * 0.60
        return base_h

    def build_half(which: str) -> Image.Image:
        hi = Image.new("RGBA", (S, S), (0, 0, 0, 0))
        d = ImageDraw.Draw(hi)

        n_seg = 96
        if which in ("full", "back"):
            # Slightly overlap past 0 and pi (-0.04 .. pi+0.04) so East/West flanks join seamlessly with zero gap!
            thetas_n = np.linspace(math.pi - 0.05, 2.0 * math.pi + 0.05, n_seg)
            for i in range(len(thetas_n) - 1):
                t0, t1 = float(thetas_n[i]), float(thetas_n[i + 1])
                tm = 0.5 * (t0 + t1)
                wh0, wh1 = wall_h_at(t0), wall_h_at(t1)

                ox0, oy0 = cx + rx_out * math.cos(t0), cy + ry_out * math.sin(t0)
                ox1, oy1 = cx + rx_out * math.cos(t1), cy + ry_out * math.sin(t1)
                ix0, iy0 = cx + rx_in * math.cos(t0), cy + ry_in * math.sin(t0)
                ix1, iy1 = cx + rx_in * math.cos(t1), cy + ry_in * math.sin(t1)

                col_face = base_stone if math.cos(tm) < 0.15 else shad_stone
                d.polygon([(ix0, iy0), (ix1, iy1), (ix1, iy1 - wh1), (ix0, iy0 - wh0)], fill=(*col_face, 255))
                d.polygon(
                    [(ox0, oy0), (ox1, oy1), (ox1, oy1 - wh1), (ox0, oy0 - wh0)],
                    fill=(*(base_stone if math.cos(tm) < 0 else shad_stone), 255),
                )
                d.polygon([(ox0, oy0 - wh0), (ox1, oy1 - wh1), (ix1, iy1 - wh1), (ix0, iy0 - wh0)], fill=(*lit_stone, 255))

            pts_out_top = []
            pts_in_top = []
            pts_in_bot = []
            for k, t in enumerate(thetas_n):
                wh = wall_h_at(float(t))
                pts_out_top.append((cx + rx_out * math.cos(t), cy + ry_out * math.sin(t) - wh))
                pts_in_top.append((cx + rx_in * math.cos(t), cy + ry_in * math.sin(t) - wh))
                pts_in_bot.append((cx + rx_in * math.cos(t), cy + ry_in * math.sin(t)))
                tn = float(t) % (2.0 * math.pi)
                if k % 4 == 1 and k + 2 < len(thetas_n) and (math.pi + 0.08 < tn < 2.0 * math.pi - 0.08):
                    if state_idx == 4 and abs(tn - 1.5 * math.pi) < 0.20:
                        continue
                    if state_idx in (2, 3, 5) and (k % 12 == 5):
                        continue
                    t_n = float(thetas_n[k + 2])
                    wh_n = wall_h_at(t_n)
                    mx0, my0 = cx + rx_out * math.cos(t), cy + ry_out * math.sin(t) - wh
                    mx1, my1 = cx + rx_out * math.cos(t_n), cy + ry_out * math.sin(t_n) - wh_n
                    mh = 36.0
                    poly(d, [(mx0, my0), (mx1, my1), (mx1, my1 - mh), (mx0, my0 - mh)], lit_stone, ink=6)

            for seq in (pts_out_top, pts_in_top, pts_in_bot):
                for j in range(len(seq) - 1):
                    seam_line(d, seq[j], seq[j + 1], width=7)

            if state_idx in (2, 3, 4):
                for ang in (1.22 * math.pi, 1.52 * math.pi, 1.78 * math.pi):
                    kx = cx + rx_in * math.cos(ang)
                    ky = cy + ry_in * math.sin(ang) - wall_h_at(ang) * 0.75
                    w_crk = 7 if state_idx == 2 else 12
                    seam_line(d, (kx, ky), (kx + 24, ky + 56), width=w_crk)
                    seam_line(d, (kx + 24, ky + 56), (kx - 14, ky + 112), width=w_crk)
            if is_recovery:
                for ang in (1.30 * math.pi, 1.70 * math.pi):
                    bx = cx + rx_in * math.cos(ang)
                    by = cy + ry_in * math.sin(ang)
                    wh_r = wall_h_at(ang)
                    poly(
                        d,
                        [(bx - 16, by - wh_r + 24), (bx + 12, by - wh_r + 24), (bx + 44, by + 10), (bx + 14, by + 10)],
                        OAK_FRESH,
                        ink=6,
                    )

        if which in ("full", "front"):
            thetas_s = np.linspace(-0.05, math.pi + 0.05, n_seg)
            gate_half_rad = 0.175
            for i in range(len(thetas_s) - 1):
                t0, t1 = float(thetas_s[i]), float(thetas_s[i + 1])
                tm = 0.5 * (t0 + t1)
                in_gate = abs(tm - 0.5 * math.pi) < gate_half_rad
                if state_idx == 4 and abs(tm - 0.5 * math.pi) < gate_half_rad * 1.45:
                    continue

                wh0, wh1 = wall_h_at(t0), wall_h_at(t1)
                ox0, oy0 = cx + rx_out * math.cos(t0), cy + ry_out * math.sin(t0)
                ox1, oy1 = cx + rx_out * math.cos(t1), cy + ry_out * math.sin(t1)
                ix0, iy0 = cx + rx_in * math.cos(t0), cy + ry_in * math.sin(t0)
                ix1, iy1 = cx + rx_in * math.cos(t1), cy + ry_in * math.sin(t1)

                if not in_gate:
                    col_out = base_stone if math.cos(tm) < -0.05 else shad_stone
                    # Match exact vertical wall base/top convention `(y) .. (y - wh)` so flanks join North arc seamlessly!
                    d.polygon(
                        [(ox0, oy0), (ox1, oy1), (ox1, oy1 - wh1), (ox0, oy0 - wh0)],
                        fill=(*col_out, 255),
                    )
                    d.polygon(
                        [(ox0, oy0 - wh0), (ox1, oy1 - wh1), (ix1, iy1 - wh1), (ix0, iy0 - wh0)],
                        fill=(*lit_stone, 255),
                    )

            for k in range(len(thetas_s) - 1):
                t0, t1 = float(thetas_s[k]), float(thetas_s[k + 1])
                tm = 0.5 * (t0 + t1)
                if abs(tm - 0.5 * math.pi) < (gate_half_rad * 1.45 if state_idx == 4 else gate_half_rad):
                    continue
                wh0, wh1 = wall_h_at(t0), wall_h_at(t1)
                seam_line(d, (cx + rx_out * math.cos(t0), cy + ry_out * math.sin(t0)), (cx + rx_out * math.cos(t1), cy + ry_out * math.sin(t1)), width=7)
                seam_line(d, (cx + rx_out * math.cos(t0), cy + ry_out * math.sin(t0) - wh0), (cx + rx_out * math.cos(t1), cy + ry_out * math.sin(t1) - wh1), width=6)
                seam_line(d, (cx + rx_in * math.cos(t0), cy + ry_in * math.sin(t0) - wh0), (cx + rx_in * math.cos(t1), cy + ry_in * math.sin(t1) - wh1), width=6)

            # South Gatehouse (centered at x = cx = 1024, y_bot = cy + ry_out + 16 = 1696 -> y = 424 in 512x512!)
            gx0, gx1 = cx - 160.0, cx + 160.0
            gy_bot = cy + ry_out + 18.0
            gy_top = gy_bot - 248.0

            if state_idx == 4:
                # STATE 4 — BREACHED (0 HP): Collapsed flanking piers & rubble chunks leaving a clear central gap!
                for side_s, px_c in ((-1, gx0 - 24), (+1, gx1 + 24)):
                    poly(
                        d,
                        [
                            (px_c - 62, gy_bot - 8),
                            (px_c + 62, gy_bot - 8),
                            (px_c + 48, gy_bot - 128),
                            (px_c - 12, gy_bot - 154),
                            (px_c - 58, gy_bot - 96),
                        ],
                        WALL_BASE if side_s < 0 else WALL_SHAD,
                        ink=8,
                    )
                # Scattered low limestone & splintered oak rubble on left/right lips (center 180px wide open!)
                for rx_r, ry_r, rw_r, rh_r, col_r in [
                    (cx - 135, gy_bot - 36, 46, 28, WALL_LIT),
                    (cx - 88, gy_bot - 24, 38, 22, OAK_BASE),
                    (cx + 84, gy_bot - 30, 44, 24, WALL_SHAD),
                    (cx + 128, gy_bot - 42, 50, 30, WALL_BASE),
                ]:
                    poly(d, [(rx_r, ry_r), (rx_r + rw_r, ry_r - 6), (rx_r + rw_r - 8, ry_r + rh_r), (rx_r - 6, ry_r + rh_r)], col_r, ink=6)
            else:
                # Flanking gate piers
                poly(d, [(gx0 - 44, gy_bot), (gx0 + 28, gy_bot), (gx0 + 28, gy_top + 24), (gx0 - 44, gy_top + 24)], lit_stone, ink=8)
                poly(d, [(gx1 - 28, gy_bot), (gx1 + 44, gy_bot), (gx1 + 44, gy_top + 24), (gx1 - 28, gy_top + 24)], shad_stone, ink=8)
                # Cobalt-slate gatehouse lintel roof (intact in 1,2,5; cracked/notched in 3)
                if state_idx == 3:
                    poly(d, [(gx0 - 54, gy_top + 32), (cx - 18, gy_top - 16), (cx + 12, gy_top + 26), (gx1 + 54, gy_top + 32), (gx1 + 44, gy_top + 68), (gx0 - 44, gy_top + 68)], ROOF_SHAD, ink=8)
                else:
                    poly(d, [(gx0 - 56, gy_top + 32), (cx, gy_top - 26), (gx1 + 56, gy_top + 32), (gx1 + 44, gy_top + 68), (gx0 - 44, gy_top + 68)], ROOF_BASE, ink=8)
                    poly(d, [(gx0 - 56, gy_top + 32), (cx, gy_top - 26), (cx, gy_top + 68), (gx0 - 44, gy_top + 68)], ROOF_LIT, ink=6)

                # Oak & iron gate doors inside arch (gx0+28 .. gx1-28)
                dx0, dx1 = gx0 + 28, gx1 - 28
                dy_arch = gy_top + 68
                if state_idx == 3:
                    # Buckled critical doors with dark fissure gap in center!
                    poly(d, [(dx0, gy_bot), (cx - 14, gy_bot - 10), (cx - 22, dy_arch), (dx0, dy_arch)], OAK_BASE, ink=7)
                    poly(d, [(cx + 16, gy_bot - 6), (dx1, gy_bot), (dx1, dy_arch), (cx + 8, dy_arch)], OAK_SHAD, ink=7)
                    poly(d, [(cx - 14, gy_bot), (cx + 16, gy_bot), (cx + 8, dy_arch), (cx - 22, dy_arch)], INK_RGB, ink=0)
                else:
                    poly(d, [(dx0, gy_bot), (cx, gy_bot), (cx, dy_arch), (dx0, dy_arch)], OAK_BASE, ink=7)
                    poly(d, [(cx, gy_bot), (dx1, gy_bot), (dx1, dy_arch), (cx, dy_arch)], OAK_SHAD, ink=7)
                    # Iron hinge bands
                    for hy in (dy_arch + 48, gy_bot - 52):
                        seam_line(d, (dx0 + 8, hy), (dx1 - 8, hy), width=12, color=(*IRON_BASE, 255))

                # State-specific cracks / recovery timber bracing on South wall & gate!
                if state_idx in (2, 3):
                    w_c = 8 if state_idx == 2 else 14
                    for sx_c in (cx - 460, cx - 260, cx + 250, cx + 470):
                        sy_c = cy + ry_out * math.sin(math.acos((sx_c - cx) / rx_out))
                        seam_line(d, (sx_c, sy_c - 72), (sx_c + 22, sy_c - 34), width=w_c)
                        seam_line(d, (sx_c + 22, sy_c - 34), (sx_c - 12, sy_c - 4), width=w_c)
                if is_recovery:
                    # Fresh timber cross-bracing across restored gate + inclined timber shores on flanks!
                    poly(d, [(dx0 + 8, dy_arch + 18), (dx1 - 8, gy_bot - 16), (dx1 - 8, gy_bot - 42), (dx0 + 8, dy_arch - 4)], OAK_FRESH, ink=6)
                    poly(d, [(dx1 - 8, dy_arch + 18), (dx0 + 8, gy_bot - 16), (dx0 + 8, gy_bot - 42), (dx1 - 8, dy_arch - 4)], OAK_FRESH, ink=6)
                    for side_s, sx_b in ((-1, cx - 340), (+1, cx + 340)):
                        sy_b = cy + ry_out * math.sin(math.acos((sx_b - cx) / rx_out))
                        poly(
                            d,
                            [(sx_b - 18, sy_b - 82), (sx_b + 18, sy_b - 82), (sx_b + side_s * 48 + 18, sy_b + 12), (sx_b + side_s * 48 - 18, sy_b + 12)],
                            OAK_FRESH,
                            ink=7,
                        )

        return finalize_supersampled_sprite(hi, 512, 512, outer_ink_px=2)

    full_im = build_half("full")
    back_full = build_half("back")
    front_full = build_half("front")

    # Crop split slices to compact deterministic bounds (512x320 back, 512x256 front) with >=12px border clearance!
    back_slice = back_full.crop((0, 0, 512, 320))
    front_slice = front_full.crop((0, 208, 512, 464))
    return full_im, back_slice, front_slice


def render_repair_scaffold() -> Image.Image:
    """
    Renders the 6th canonical wall asset: `env_wall_repair_scaffold.png` (192x192 RGBA, pivot=(0.50, 0.10)).
    Modular pegged oak timber scaffold with work deck, X-braces, and side ladder.
    """
    S = 768  # 4x -> 192x192
    hi = Image.new("RGBA", (S, S), (0, 0, 0, 0))
    d = ImageDraw.Draw(hi)

    # Isometric 2-tier timber scaffold frame (base y = 690 -> 172.5 in 192x192, leaving 19px margin)
    posts = [(170, 660), (390, 690), (310, 580), (530, 610)]
    top_h = 430
    deck_h = 265

    # Back posts first
    for px, py in posts[2:]:
        poly(d, [(px - 14, py), (px + 14, py), (px + 14, py - top_h), (px - 14, py - top_h)], OAK_SHAD, ink=7)

    # Diagonal X-brace
    poly(d, [(170, 620), (190, 620), (390, 440), (370, 440)], OAK_SHAD, ink=6)
    poly(d, [(370, 650), (390, 650), (190, 430), (170, 430)], OAK_BASE, ink=6)

    # Upper wooden plank deck
    deck_pts = [
        (146, 660 - deck_h),
        (414, 690 - deck_h),
        (554, 610 - deck_h),
        (286, 580 - deck_h),
    ]
    poly(d, deck_pts, OAK_LIT, ink=8)
    seam_line(d, (196, 666 - deck_h), (336, 586 - deck_h), width=5)
    seam_line(d, (268, 674 - deck_h), (408, 594 - deck_h), width=5)
    seam_line(d, (340, 682 - deck_h), (480, 602 - deck_h), width=5)

    # Front posts + guard rail
    for px, py in posts[:2]:
        poly(d, [(px - 15, py), (px + 15, py), (px + 15, py - top_h), (px - 15, py - top_h)], OAK_BASE, ink=7)
    poly(d, [(158, 660 - top_h + 36), (402, 690 - top_h + 36), (402, 690 - top_h + 62), (158, 660 - top_h + 62)], OAK_LIT, ink=7)

    # Inclined access ladder on right side
    lx0, ly0 = 445, 686
    lx1, ly1 = 408, 690 - deck_h + 12
    seam_line(d, (lx0, ly0), (lx1, ly1), width=16, color=(*OAK_LIT, 255))
    seam_line(d, (lx0 + 68, ly0 - 28), (lx1 + 68, ly1 - 28), width=16, color=(*OAK_LIT, 255))
    for rf in (0.18, 0.38, 0.58, 0.78):
        rx0 = lx0 * (1 - rf) + lx1 * rf
        ry0 = ly0 * (1 - rf) + ly1 * rf
        seam_line(d, (rx0, ry0), (rx0 + 68, ry0 - 28), width=12, color=(*OAK_BASE, 255))

    return finalize_supersampled_sprite(hi, 192, 192, outer_ink_px=2)


# ============================================================================
# PHASE 4 — WATCHTOWER (1), COURTYARD COTTAGES (6) & PROPS (3) = 10 SPRITES
# ============================================================================

def render_watchtower() -> Image.Image:
    """256x256 RGBA defensive watchtower (~70x146 px @ 1080p when scaled by 0.625, pivot=(0.50, 0.10))."""
    S = 1024
    hi = Image.new("RGBA", (S, S), (0, 0, 0, 0))
    d = ImageDraw.Draw(hi)
    cx, base_y = 512.0, 920.0

    # Flared limestone plinth
    draw_iso_box(d, cx, base_y, 240.0, 240.0, 0.0, 210.0, WALL_BASE, WALL_SHAD, top_col=WALL_LIT, ink=8, ashlar_rows=3)
    draw_arched_door_on_left_face(d, cx, base_y, 120.0, 86.0, 140.0, stone_arch=False)
    # Timber-framed upper shaft (calibrated for >=12px top margin in 256x256)
    draw_iso_box(d, cx, base_y, 200.0, 200.0, 210.0, 495.0, OAK_LIT, OAK_BASE, top_col=OAK_LIT, ink=8)
    for hz in (305.0, 405.0):
        seam_line(d, iso_pt(cx, base_y, 10, 0, hz), iso_pt(cx, base_y, 190, 0, hz), width=12, color=(*OAK_SHAD, 255))
        seam_line(d, iso_pt(cx, base_y, 0, 10, hz), iso_pt(cx, base_y, 0, 190, hz), width=12, color=(*OAK_SHAD, 255))
    # Lookout gallery + conical cobalt-slate roof
    draw_iso_box(d, cx, base_y, 232.0, 232.0, 495.0, 565.0, OAK_BASE, OAK_SHAD, top_col=LIME_BASE, ink=8)
    top_c = iso_pt(cx, base_y, 116.0, 116.0, 565.0)
    draw_cylindrical_turret(d, top_c[0], top_c[1] + 24.0, radius=160.0, body_h=24.0, cone_h=182.0, gilded_rim=False)
    return finalize_supersampled_sprite(hi, 256, 256, outer_ink_px=2)


def render_cottage_variant(variant_idx: int) -> Image.Image:
    """
    Renders 6 genuinely distinct architectural courtyard cottages (256x256 RGBA, pivot=(0.50, 0.12)):
      1: Artisan Gable Cottage (half-timber + dormer + golden-amber awning)
      2: Stone Granary Roundhouse (cylindrical stone + conical golden-thatch cap)
      3: Mason Flat-Parapet Lodge (cut limestone + crenelated flat stone roof)
      4: Bakery Chimney Cottage (warm timber + side brick oven chimney + woodpile)
      5: Open-Front Smithy Shed (mono-pitch timber roof + stone forge flue + anvil)
      6: Cross-Gable Guard Quarters (L-plan twin-gable slate cottage)
    """
    S = 1024
    hi = Image.new("RGBA", (S, S), (0, 0, 0, 0))
    d = ImageDraw.Draw(hi)
    cx, base_y = 512.0, 900.0

    if variant_idx == 1:
        # 1. Artisan Gable Cottage + striped awning
        draw_iso_box(d, cx, base_y, 330.0, 290.0, 0.0, 80.0, WALL_BASE, WALL_SHAD, ink=8)
        draw_iso_box(d, cx, base_y, 320.0, 280.0, 80.0, 340.0, LIME_BASE, LIME_SHAD, ink=8)
        for u_p in (24, 160, 296):
            seam_line(d, iso_pt(cx, base_y, u_p, 0, 80), iso_pt(cx, base_y, u_p, 0, 340), width=14, color=(*OAK_BASE, 255))
        draw_arched_door_on_left_face(d, cx, base_y, 160.0, 92.0, 170.0, stone_arch=False)
        draw_gable_roof(d, cx, base_y, 326.0, 286.0, 340.0, 230.0, LIME_BASE, ROOF_MUTE_LIT, ROOF_MUTE_BASE, OAK_BASE, overhang=24.0, ink=8)
        # Warm golden awning projecting over right wall shopfront
        a0 = iso_pt(cx, base_y, 0, 40, 260)
        a1 = iso_pt(cx, base_y, 0, 240, 260)
        a2 = (a1[0] + 74, a1[1] + 68)
        a3 = (a0[0] + 74, a0[1] + 68)
        poly(d, [a0, a1, a2, a3], GOLD_BASE, ink=7)
        seam_line(d, ((a0[0] + a1[0]) * 0.5, (a0[1] + a1[1]) * 0.5), ((a2[0] + a3[0]) * 0.5, (a2[1] + a3[1]) * 0.5), width=14, color=(*BRICK_LIT, 255))

    elif variant_idx == 2:
        # 2. Cylindrical Stone Granary Roundhouse + golden conical thatch roof
        tx, ty = 512.0, 770.0
        r, ry = 225.0, 108.0
        poly(d, [(tx - r, ty - 270), (tx + r, ty - 270), (tx + r, ty), (tx, ty + ry), (tx - r, ty)], WALL_BASE, ink=0)
        poly(d, [(tx + 25, ty - 270), (tx + r, ty - 270), (tx + r, ty), (tx + 25, ty + ry * 0.94)], WALL_SHAD, ink=0)
        d.pieslice([int(tx - r), int(ty - ry), int(tx + r), int(ty + ry)], 0, 180, fill=(*WALL_BASE, 255))
        d.pieslice([int(tx - r), int(ty - ry), int(tx + r), int(ty + ry)], 0, 75, fill=(*WALL_SHAD, 255))
        seam_line(d, (tx - r, ty - 270), (tx - r, ty), width=8)
        seam_line(d, (tx + r, ty - 270), (tx + r, ty), width=8)
        d.arc([int(tx - r), int(ty - ry), int(tx + r), int(ty + ry)], 0, 180, fill=INK_RGBA, width=8)
        # Oak door on front
        poly(d, [(tx - 54, ty + 84), (tx + 36, ty + 84), (tx + 36, ty - 76), (tx - 54, ty - 76)], OAK_BASE, ink=8)
        # Conical golden thatch cap
        cr, cry = 262.0, 120.0
        top_y = ty - 270.0
        apex = (tx, top_y - 245.0)
        poly(d, [(tx - cr, top_y), apex, (tx + cr, top_y)], THATCH_BASE, ink=0)
        poly(d, [(tx - cr, top_y), apex, (tx - 24, top_y + cry * 0.9)], THATCH_LIT, ink=0)
        poly(d, [(tx + 40, top_y + cry * 0.88), apex, (tx + cr, top_y)], THATCH_SHAD, ink=0)
        d.pieslice([int(tx - cr), int(top_y - cry), int(tx + cr), int(top_y + cry)], 0, 180, fill=(*THATCH_BASE, 255))
        d.pieslice([int(tx - cr), int(top_y - cry), int(tx + cr), int(top_y + cry)], 98, 180, fill=(*THATCH_LIT, 255))
        d.pieslice([int(tx - cr), int(top_y - cry), int(tx + cr), int(top_y + cry)], 0, 62, fill=(*THATCH_SHAD, 255))
        d.arc([int(tx - cr), int(top_y - cry), int(tx + cr), int(top_y + cry)], 0, 180, fill=INK_RGBA, width=8)
        seam_line(d, (tx - cr, top_y), apex, width=8)
        seam_line(d, (tx + cr, top_y), apex, width=8)
        d.ellipse([int(apex[0] - 34), int(apex[1] - 18), int(apex[0] + 34), int(apex[1] + 24)], fill=(*THATCH_LIT, 255), outline=INK_RGBA, width=7)

    elif variant_idx == 3:
        # 3. Cut-Limestone Mason Lodge (flat crenelated stone roof — distinct cubic silhouette!)
        draw_iso_box(d, cx, base_y, 330.0, 310.0, 0.0, 350.0, LIME_BASE, WALL_SHAD, top_col=WALL_LIT, ink=9, ashlar_rows=4)
        draw_iso_box(d, cx, base_y, 346.0, 326.0, 350.0, 415.0, WALL_LIT, WALL_SHAD, top_col=LIME_LIT, ink=8)
        draw_arched_door_on_left_face(d, cx, base_y, 165.0, 102.0, 185.0, stone_arch=True)
        for vp in (95.0, 215.0):
            poly(
                d,
                [
                    iso_pt(cx, base_y, 0, vp - 28, 175),
                    iso_pt(cx, base_y, 0, vp + 28, 175),
                    iso_pt(cx, base_y, 0, vp + 28, 265),
                    iso_pt(cx, base_y, 0, vp - 28, 265),
                ],
                IRON_SHAD,
                ink=7,
            )

    elif variant_idx == 4:
        # 4. Bakery Chimney Cottage (warm plank roof + tall terracotta brick chimney + firewood stack)
        draw_iso_box(d, cx, base_y, 310.0, 280.0, 0.0, 310.0, OAK_LIT, OAK_BASE, ink=8)
        draw_arched_door_on_left_face(d, cx, base_y, 155.0, 90.0, 165.0, stone_arch=False)
        draw_gable_roof(d, cx, base_y, 320.0, 290.0, 310.0, 210.0, LIME_BASE, OAK_LIT, OAK_SHAD, OAK_SHAD, overhang=24.0, ink=8)
        # Tall brick chimney on SE slope (zero animated smoke baked in!)
        ch_x, ch_y = iso_pt(cx, base_y, 50.0, 155.0, 350.0)
        draw_iso_box(d, ch_x, ch_y, 76.0, 76.0, 0.0, 285.0, BRICK_LIT, BRICK_SHAD, top_col=BRICK_LIT, ink=8, ashlar_rows=3)
        # Small stacked firewood lean-to on SE base
        fw_x, fw_y = iso_pt(cx, base_y, -10.0, 80.0, 0.0)
        draw_iso_box(d, fw_x, fw_y, 58.0, 140.0, 0.0, 82.0, OAK_LIT, OAK_SHAD, top_col=OAK_BASE, ink=7)

    elif variant_idx == 5:
        # 5. Open-Front Smithy Shed (asymmetric shed roof + stone hearth chimney + anvil)
        # Back-left & back-right walls
        draw_iso_box(d, cx, base_y, 320.0, 280.0, 0.0, 56.0, WALL_BASE, WALL_SHAD, top_col=PLAZA_SHAD, ink=8)
        # Back stone forge hearth & flue
        fg_x, fg_y = iso_pt(cx, base_y, 85.0, 165.0, 56.0)
        draw_iso_box(d, fg_x, fg_y, 130.0, 105.0, 0.0, 440.0, WALL_BASE, WALL_SHAD, top_col=WALL_LIT, ink=8, ashlar_rows=3)
        # Warm glowing forge mouth (static cel ember)
        em_c = iso_pt(fg_x, fg_y, 65.0, 0.0, 55.0)
        d.ellipse([int(em_c[0] - 28), int(em_c[1] - 32), int(em_c[0] + 28), int(em_c[1] + 20)], fill=(*EMBER_GLOW, 255), outline=INK_RGBA, width=6)
        # Timber corner posts + mono-pitch roof
        for u_p, v_p in [(296, 16), (16, 16), (296, 260)]:
            pp = iso_pt(cx, base_y, u_p, v_p, 56.0)
            poly(d, [(pp[0] - 12, pp[1]), (pp[0] + 12, pp[1]), (pp[0] + 12, pp[1] - 250), (pp[0] - 12, pp[1] - 250)], OAK_BASE, ink=6)
        r0 = iso_pt(cx, base_y, 330.0, -10.0, 360.0)
        r1 = iso_pt(cx, base_y, -10.0, -10.0, 290.0)
        r2 = iso_pt(cx, base_y, -10.0, 290.0, 330.0)
        r3 = iso_pt(cx, base_y, 330.0, 290.0, 400.0)
        poly(d, [r0, r1, r2, r3], ROOF_MUTE_BASE, ink=8)
        # Tiny iron anvil on hearth stone
        av = iso_pt(cx, base_y, 90.0, 60.0, 95.0)
        poly(d, [(av[0] - 34, av[1]), (av[0] + 38, av[1]), (av[0] + 22, av[1] + 36), (av[0] - 22, av[1] + 36)], IRON_BASE, ink=6)

    elif variant_idx == 6:
        # 6. Cross-Gable Guard Quarters (L-plan twin-gable slate barracks cottage)
        draw_iso_box(d, cx + 36, base_y, 340.0, 250.0, 0.0, 330.0, LIME_BASE, LIME_SHAD, ink=8, ashlar_rows=3)
        draw_gable_roof(d, cx + 36, base_y, 348.0, 258.0, 330.0, 215.0, LIME_BASE, ROOF_MUTE_LIT, ROOF_MUTE_SHAD, OAK_BASE, overhang=22.0, ink=8)
        # Projecting cross-wing on SW front
        wx, wy = iso_pt(cx + 36, base_y, 130.0, -70.0, 0.0)
        draw_iso_box(d, wx, wy, 185.0, 175.0, 0.0, 295.0, LIME_LIT, LIME_SHAD, ink=8)
        draw_arched_door_on_left_face(d, wx, wy, 92.0, 80.0, 155.0, stone_arch=True)
        draw_gable_roof(d, wx, wy, 192.0, 182.0, 295.0, 165.0, LIME_LIT, ROOF_LIT, ROOF_BASE, OAK_BASE, overhang=18.0, ink=8)

    else:
        raise ValueError(f"Unknown cottage variant: {variant_idx}")

    return finalize_supersampled_sprite(hi, 256, 256, outer_ink_px=2)


def render_courtyard_prop(kind: str) -> Image.Image:
    """
    Renders the 3 low-profile courtyard props (128x128 RGBA, pivot=(0.50, 0.12)):
      - `supply_cart`: Compact oak handcart with timber logs
      - `stone_crates`: Wooden pallet stacked with cut limestone blocks
      - `sun_banner`: Decorative golden sun standard on stone plinth
    """
    S = 512  # 4x -> 128x128
    hi = Image.new("RGBA", (S, S), (0, 0, 0, 0))
    d = ImageDraw.Draw(hi)
    cx, base_y = 256.0, 444.0

    if kind == "supply_cart":
        draw_iso_box(d, cx - 18, base_y - 44, 176.0, 136.0, 0.0, 78.0, OAK_BASE, OAK_SHAD, top_col=OAK_LIT, ink=8)
        # Stacked timber logs in cart bed
        for lx, ly in [(-34, -142), (10, -156), (52, -140), (-12, -178), (30, -176)]:
            d.ellipse([int(cx + lx - 26), int(base_y + ly - 22), int(cx + lx + 26), int(base_y + ly + 22)], fill=(*OAK_LIT, 255), outline=INK_RGBA, width=6)
        # Pull shafts
        seam_line(d, (cx + 42, base_y - 54), (cx + 148, base_y - 6), width=14, color=(*OAK_BASE, 255))
        seam_line(d, (cx + 42, base_y - 54), (cx + 148, base_y - 6), width=5)
        # Iron-rimmed cart wheel
        wx, wy, wr = int(cx - 56), int(base_y - 46), 46
        d.ellipse([wx - wr, wy - wr, wx + wr, wy + wr], fill=(*IRON_BASE, 255), outline=INK_RGBA, width=8)
        d.ellipse([wx - 28, wy - 28, wx + 28, wy + 28], fill=(*OAK_BASE, 255), outline=INK_RGBA, width=6)
        d.ellipse([wx - 10, wy - 10, wx + 10, wy + 10], fill=(*IRON_LIT, 255), outline=INK_RGBA, width=5)

    elif kind == "stone_crates":
        # Timber pallet base
        draw_iso_box(d, cx, base_y, 210.0, 196.0, 0.0, 32.0, OAK_BASE, OAK_SHAD, top_col=OAK_LIT, ink=8)
        # Cut limestone ashlar blocks stacked cleanly
        draw_iso_box(d, cx - 8, base_y, 112.0, 108.0, 32.0, 128.0, LIME_BASE, WALL_SHAD, top_col=LIME_LIT, ink=8)
        draw_iso_box(d, cx - 84, base_y - 40, 88.0, 92.0, 32.0, 116.0, LIME_BASE, WALL_SHAD, top_col=LIME_LIT, ink=8)
        draw_iso_box(d, cx + 68, base_y - 38, 86.0, 88.0, 32.0, 108.0, WALL_BASE, WALL_SHAD, top_col=WALL_LIT, ink=8)
        draw_iso_box(d, cx - 20, base_y - 24, 92.0, 88.0, 128.0, 206.0, LIME_LIT, LIME_SHAD, top_col=LIME_LIT, ink=8)

    elif kind == "sun_banner":
        # Compact stone plinth + oak crossbar pole + cobalt & gold sun standard
        draw_iso_box(d, cx, base_y, 112.0, 112.0, 0.0, 48.0, WALL_BASE, WALL_SHAD, top_col=WALL_LIT, ink=8)
        seam_line(d, (cx, base_y - 54), (cx, base_y - 384), width=18, color=(*OAK_BASE, 255))
        seam_line(d, (cx, base_y - 54), (cx, base_y - 384), width=5)
        seam_line(d, (cx - 86, base_y - 344), (cx + 86, base_y - 344), width=16, color=(*GOLD_BASE, 255))
        d.ellipse([int(cx - 14), int(base_y - 404), int(cx + 14), int(base_y - 376)], fill=(*GOLD_LIT, 255), outline=INK_RGBA, width=6)
        # Hanging banner cloth
        bx0, bx1, by0, by1 = cx - 68, cx + 68, base_y - 338, base_y - 144
        poly(d, [(bx0, by0), (bx1, by0), (bx1, by1), (cx, by1 + 44), (bx0, by1)], GOLD_BASE, ink=8)
        poly(d, [(bx0 + 14, by0 + 12), (bx1 - 14, by0 + 12), (bx1 - 14, by1 - 8), (cx, by1 + 26), (bx0 + 14, by1 - 8)], ROOF_BASE, ink=5)
        sy = int((by0 + by1) * 0.5)
        d.ellipse([int(cx - 26), sy - 26, int(cx + 26), sy + 26], fill=(*GOLD_LIT, 255), outline=INK_RGBA, width=5)

    else:
        raise ValueError(f"Unknown prop kind: {kind}")

    return finalize_supersampled_sprite(hi, 128, 128, outer_ink_px=2)


# ============================================================================
# PHASE 5 — SEAMLESS TOROIDAL 256x256 TERRAIN TILES (6) & TRANSITIONS (4)
# ============================================================================

def _periodic_field(n: int, harmonics: Sequence[Tuple[int, int, float, float]]) -> np.ndarray:
    """Evaluates an exact 2D periodic Fourier sum on [0, N) x [0, N) — guaranteed 100% seamless on Z_N x Z_N!"""
    y = np.arange(n, dtype=np.float64)[:, None] * (2.0 * math.pi / n)
    x = np.arange(n, dtype=np.float64)[None, :] * (2.0 * math.pi / n)
    acc = np.zeros((n, n), dtype=np.float64)
    for kx, ky, amp, phase in harmonics:
        acc += amp * np.cos(kx * x + ky * y + phase)
    return acc


def build_terrain_tiles_and_transitions() -> Tuple[Dict[str, Image.Image], Dict[str, Image.Image]]:
    """
    Produces the 6 canonical 256x256 terrain tiles + 4 explicit transition strips/masks.
    Every repeating tile is mathematically periodic on the 256x256 torus with smooth
    2-to-3 step gouache-cel bands so H.264 Laplacian variance stays low and seams are 0!
    """
    N = 256
    tiles: Dict[str, Image.Image] = {}
    trans: Dict[str, Image.Image] = {}

    # 1. Primary sunlit meadow turf — smooth toroidal gouache-cel waves (zero polka-dot grid artifacts, H.264-safe!)
    f_m1 = _periodic_field(N, [(1, 0, 0.95, 0.3), (0, 1, 0.85, 1.4), (1, -1, 0.45, 2.2)])
    f_m1_q = np.round(np.clip(f_m1, -1.0, 1.0) * 3.0) / 3.0  # Smooth 7-step gouache cel banding within +-5 RGB!
    base_m = np.array(MEADOW_BASE, dtype=np.float64)[None, None, :]
    delta_m = np.array([5.0, 6.0, 4.0], dtype=np.float64)[None, None, :]
    rgb1 = np.empty((N, N, 4), dtype=np.uint8)
    rgb1[:, :, :3] = np.clip(np.round(base_m + f_m1_q[:, :, None] * delta_m), 0, 255).astype(np.uint8)
    rgb1[:, :, 3] = 255
    tiles["env_tile_01_meadow_main"] = Image.fromarray(rgb1, "RGBA")

    # 2. Secondary meadow turf variation — raised-cosine interior envelope guarantees 100% exact border match with tile_01!
    wy = np.sin(np.pi * (np.arange(N, dtype=np.float64) + 0.5) / N) ** 2
    wx = np.sin(np.pi * (np.arange(N, dtype=np.float64) + 0.5) / N) ** 2
    win2d = wy[:, None] * wx[None, :]
    f_m2 = _periodic_field(N, [(1, 1, 0.90, 0.6), (-1, 2, 0.50, 1.1)])
    cool_shift = win2d[:, :, None] * (np.array([-8.0, -11.0, -6.0])[None, None, :] + f_m2[:, :, None] * np.array([3.0, 4.0, 2.0])[None, None, :])
    rgb2 = np.empty((N, N, 4), dtype=np.uint8)
    rgb2[:, :, :3] = np.clip(np.round(rgb1[:, :, :3].astype(np.float64) + cool_shift), 0, 255).astype(np.uint8)
    rgb2[:, :, 3] = 255
    tiles["env_tile_02_meadow_variation"] = Image.fromarray(rgb2, "RGBA")

    # 3. Ochre road body (seamless X & Y, smooth low-frequency packed earth gouache bands)
    f_rd = _periodic_field(N, [(1, 0, 0.80, 0.2), (0, 1, 0.75, 1.5), (1, 1, 0.40, 2.5)])
    f_rd_q = np.round(np.clip(f_rd, -1.0, 1.0) * 3.0) / 3.0
    base_o = np.array(OCHRE_BASE, dtype=np.float64)[None, None, :]
    delta_o = np.array([7.0, 6.0, 5.0], dtype=np.float64)[None, None, :]
    rgb3 = np.empty((N, N, 4), dtype=np.uint8)
    rgb3[:, :, :3] = np.clip(np.round(base_o + f_rd_q[:, :, None] * delta_o), 0, 255).astype(np.uint8)
    rgb3[:, :, 3] = 255
    tiles["env_tile_03_ochre_road_body"] = Image.fromarray(rgb3, "RGBA")

    # 4. Ochre road border / meadow-to-road transition tile (seamless vertically in Y;
    #    left edge x=0..64 matches tile_01 meadow_main, middle x=72..124 is curved periodic earth/stone verge,
    #    right edge x=132..255 matches tile_03 ochre_road_body!)
    y_rad = np.arange(N, dtype=np.float64)[:, None] * (2.0 * math.pi / N)
    xs = np.arange(N, dtype=np.float64)[None, :]
    wobble = 6.0 * np.sin(2.0 * y_rad + 0.5) + 3.0 * np.cos(1.0 * y_rad - 0.4)
    rgb4 = rgb1.copy()  # Left side seamlessly matches env_tile_01_meadow_main!
    m_lip = xs >= (88.0 + wobble)
    m_verge = xs >= (100.0 + wobble)
    m_verge_lit = xs >= (116.0 + wobble)
    m_road = xs >= (128.0 + wobble)
    rgb4[m_lip, :3] = MEADOW_DEEP
    rgb4[m_verge, :3] = OCHRE_VERGE
    rgb4[m_verge_lit, :3] = OCHRE_LIT
    rgb4[m_road, :3] = rgb3[m_road, :3]  # Right side seamlessly matches env_tile_03_ochre_road_body!
    tiles["env_tile_04_ochre_road_border"] = Image.fromarray(rgb4, "RGBA")

    # 5. Pale Limestone Boss Causeway (seamless X & Y staggered broad ashlar courses, ZERO #11151C black ink!)
    rgb5 = np.empty((N, N, 4), dtype=np.uint8)
    rgb5[:, :, :3] = CAUSEWAY_BASE
    rgb5[:, :, 3] = 255
    row_h = 64  # 4 courses = 256 (exact divisor -> 100% seamless in Y!)
    col_w = 128 # 2 slabs = 256 (exact divisor -> 100% seamless in X!)
    for r in range(N // row_h):
        y0, y1 = r * row_h, (r + 1) * row_h
        x_shift = 0 if (r % 2 == 0) else (col_w // 2)
        for c in range(N // col_w + 1):
            for dx_s in (-N, 0):
                x0 = c * col_w + x_shift + dx_s
                x1 = x0 + col_w
                cx0, cx1 = max(0, x0), min(N, x1)
                if cx1 <= cx0:
                    continue
                shade_pick = CAUSEWAY_LIT if ((r + c) % 3 == 0) else (CAUSEWAY_SHAD if ((r + c) % 3 == 2) else CAUSEWAY_BASE)
                rgb5[y0:y1, cx0:cx1, :3] = shade_pick
                # Soft top-left bevel inside each slab
                if y0 + 5 < N:
                    rgb5[y0 + 3 : y0 + 6, cx0:cx1, :3] = CAUSEWAY_LIT
        # Horizontal muted slate seam (wrapped smoothly)
        rgb5[y0 : y0 + 3, :, :3] = CAUSEWAY_SEAM
    for r in range(N // row_h):
        y0, y1 = r * row_h, (r + 1) * row_h
        x_shift = 0 if (r % 2 == 0) else (col_w // 2)
        for jx in range(x_shift, N, col_w):
            rgb5[y0:y1, jx % N : min(N, (jx % N) + 3), :3] = CAUSEWAY_SEAM
    # Shift periodic phase by (16, 16) so tile borders x=0/255 and y=0/255 sit inside continuous stone slabs (0.0 wrap diff!)
    rgb5 = np.roll(np.roll(rgb5, 16, axis=0), 16, axis=1)
    tiles["env_tile_05_limestone_causeway"] = Image.fromarray(rgb5, "RGBA")

    # 6. Inner Courtyard Packed-Earth & Warm Sandstone Plaza (seamless X & Y, calm warm courtyard surface)
    f_pz = _periodic_field(N, [(1, 0, 0.80, 0.9), (0, 1, 0.70, 2.0), (1, -1, 0.35, 0.3)])
    f_pz_q = np.round(np.clip(f_pz, -1.0, 1.0) * 3.0) / 3.0
    base_p = np.array(PLAZA_BASE, dtype=np.float64)[None, None, :]
    delta_p = np.array([6.0, 5.0, 4.0], dtype=np.float64)[None, None, :]
    rgb6 = np.empty((N, N, 4), dtype=np.uint8)
    rgb6[:, :, :3] = np.clip(np.round(base_p + f_pz_q[:, :, None] * delta_p), 0, 255).astype(np.uint8)
    rgb6[:, :, 3] = 255
    im6 = Image.fromarray(rgb6, "RGBA")
    d6 = ImageDraw.Draw(im6)
    for sx, sy, sw, sh in [(44, 48, 36, 20), (162, 124, 38, 22), (88, 192, 34, 18)]:
        d6.rounded_rectangle([sx, sy, sx + sw, sy + sh], radius=5, fill=(*PLAZA_SHAD, 255))
    tiles["env_tile_06_courtyard_plaza"] = im6

    # Companion Transition Assets (`terrain/transitions/`)
    # T1: Diagonal road verge strip (256x64 RGBA, seamless along X)
    tr1 = np.zeros((64, 256, 4), dtype=np.uint8)
    xw = np.arange(256, dtype=np.float64)[None, :] * (2.0 * math.pi / 256.0)
    yw = np.arange(64, dtype=np.float64)[:, None]
    edge_y = 14.0 + 3.0 * np.sin(2.0 * xw)
    m_tr = yw >= edge_y
    tr1[m_tr, :3] = MEADOW_DEEP
    tr1[yw >= edge_y + 8.0, :3] = OCHRE_VERGE
    tr1[yw >= edge_y + 24.0, :3] = OCHRE_LIT
    tr1[yw >= edge_y + 34.0, :3] = OCHRE_BASE
    tr1[m_tr, 3] = 255
    trans["trans_ochre_verge_strip"] = Image.fromarray(tr1, "RGBA")

    # T2: Center Boss Causeway cut-stone curb strip (64x256 RGBA, seamless along Y)
    tr2 = np.zeros((256, 64, 4), dtype=np.uint8)
    tr2[:, 12:52, :3] = CAUSEWAY_CURB
    tr2[:, 16:30, :3] = CAUSEWAY_LIT
    tr2[:, 30:48, :3] = CAUSEWAY_SHAD
    tr2[:, 12:52, 3] = 255
    for ky in range(0, 256, 64):
        tr2[ky : ky + 3, 12:52, :3] = (58, 66, 78)
    trans["trans_causeway_stone_curb"] = Image.fromarray(tr2, "RGBA")

    # T3: Road-to-Bastion cobbled apron decal (256x128 RGBA)
    tr3 = Image.new("RGBA", (256, 128), (0, 0, 0, 0))
    d3 = ImageDraw.Draw(tr3)
    d3.pieslice([12, 12, 244, 220], start=180, end=360, fill=(*OCHRE_SHAD, 255))
    d3.pieslice([24, 22, 232, 212], start=180, end=360, fill=(*OCHRE_BASE, 255))
    for cx_s, cy_s in [(76, 68), (128, 56), (178, 70), (104, 92), (154, 92)]:
        d3.rounded_rectangle([cx_s - 18, cy_s - 10, cx_s + 18, cy_s + 10], radius=4, fill=(*PLAZA_STONE, 255), outline=(*CAUSEWAY_SEAM, 255), width=2)
    trans["trans_lane_bastion_apron"] = tr3

    # T4: Organic cel meadow variation patch mask (256x256 RGBA binary alpha)
    tr4 = np.zeros((256, 256, 4), dtype=np.uint8)
    yy, xx = np.mgrid[0:256, 0:256]
    rad = np.sqrt(((xx - 128.0) / 96.0) ** 2 + ((yy - 128.0) / 78.0) ** 2)
    ang = np.arctan2(yy - 128.0, xx - 128.0)
    m_patch = rad <= (0.86 + 0.10 * np.sin(3.0 * ang + 0.7))
    tr4[m_patch, :3] = MEADOW_SHAD
    tr4[rad <= 0.52, :3] = MEADOW_DEEP
    tr4[m_patch, 3] = 255
    trans["trans_meadow_patch_mask"] = Image.fromarray(tr4, "RGBA")

    return tiles, trans
