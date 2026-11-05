#!/usr/bin/env python3
"""
ART-004 Layered Cutout/FK Rig Library for Bastion Dawn v1.

Builds locked, zero-leak, zero-halo layered character rigs from ART-002 canonical turnarounds:
  - Guard   (chr_role_guard,   128x128 RGBA, H_src=86px -> 48px @ 1080p, scale=0.55814)
  - Fighter (chr_role_fighter, 128x128 RGBA, H_src=86px body -> 48px @ 1080p, scale=0.55814, crimson cloth shoulders / zero steel pauldrons)
  - Runner  (chr_enemy_runner, 128x128 RGBA, H_src=80px -> 44px @ 1080p, scale=0.55000, locked 3/4 +X facing, twin curved daggers)
  - Brute   (chr_enemy_brute,  128x128 RGBA, H_src=98px -> 64px @ 1080p, scale=0.65306, siege-orange armor & crusher gauntlets)
  - Boss    (chr_boss_warlord, 256x256 RGBA, H_src=176px -> 102px @ 1080p, scale=0.57955, zero baked ground telegraph ring)
"""

from __future__ import annotations

import math
from pathlib import Path
from typing import Dict, List, Tuple

import numpy as np
from PIL import Image, ImageDraw
from scipy import ndimage

REPO_ROOT = Path(__file__).resolve().parents[4]
ART002_DIR = REPO_ROOT / "art" / "concepts" / "ART-002"
INK_RGB = (17, 21, 28)  # #11151C


def hex_to_rgb(h: str) -> Tuple[int, int, int]:
    h = h.lstrip("#")
    return (int(h[0:2], 16), int(h[2:4], 16), int(h[4:6], 16))


# ============================================================================
# CONTROLLED <= 16-COLOR BASTION DAWN v1 CHARACTER PALETTES
# ============================================================================

GUARD_PAL = np.array(
    [
        hex_to_rgb("#11151C"),  # 00 Outer dark ink contour
        hex_to_rgb("#60A5FA"),  # 01 Royal cobalt cape lit highlight
        hex_to_rgb("#2E7CF6"),  # 02 Canonical royal cobalt cape/shield face (#2E7CF6)
        hex_to_rgb("#1D4ED8"),  # 03 Cobalt mid-shadow
        hex_to_rgb("#1E3A8A"),  # 04 Deep navy under-tunic / cape shadow
        hex_to_rgb("#F8FAFC"),  # 05 Steel rim specular highlight
        hex_to_rgb("#DCE6F2"),  # 06 Bright polished steel (#DCE6F2)
        hex_to_rgb("#94A3B8"),  # 07 Mid armor steel
        hex_to_rgb("#64748B"),  # 08 Cool steel shadow
        hex_to_rgb("#334155"),  # 09 Dark steel recess / under-suit
        hex_to_rgb("#F4C29F"),  # 10 Warm skin lit
        hex_to_rgb("#DA9B74"),  # 11 Warm skin base
        hex_to_rgb("#AD6E4D"),  # 12 Warm skin shadow
        hex_to_rgb("#8B5E3C"),  # 13 Leather glove lit
        hex_to_rgb("#5C3D28"),  # 14 Leather glove shadow
    ],
    dtype=np.float32,
)

FIGHTER_PAL = np.array(
    [
        hex_to_rgb("#11151C"),  # 00 Outer dark ink contour
        hex_to_rgb("#F87171"),  # 01 Crimson cloth/plume sunlit highlight
        hex_to_rgb("#E63946"),  # 02 Canonical crimson surcoat & sleeves (#E63946)
        hex_to_rgb("#B91C1C"),  # 03 Crimson cloth shadow
        hex_to_rgb("#7F1D1D"),  # 04 Deep crimson fold / plume shadow
        hex_to_rgb("#F8FAFC"),  # 05 Broadsword blade edge highlight
        hex_to_rgb("#E2E8F0"),  # 06 Polished steel broadsword / barbute lit
        hex_to_rgb("#94A3B8"),  # 07 Steel barbute / vambrace / greave mid
        hex_to_rgb("#64748B"),  # 08 Steel shadow
        hex_to_rgb("#334155"),  # 09 Dark visor / steel recess
        hex_to_rgb("#FEF08A"),  # 10 Brass crossguard highlight
        hex_to_rgb("#EAB308"),  # 11 Brass crossguard / pommel base
        hex_to_rgb("#A16207"),  # 12 Brass shadow
        hex_to_rgb("#F4C29F"),  # 13 Chin skin lit
        hex_to_rgb("#8B5E3C"),  # 14 Leather belt / under-sleeve mid
        hex_to_rgb("#4A3222"),  # 15 Dark leather glove / boot strap
    ],
    dtype=np.float32,
)

RUNNER_PAL = np.array(
    [
        hex_to_rgb("#11151C"),  # 00 Outer dark ink contour
        hex_to_rgb("#C084FC"),  # 01 Violet goblin skin sunlit highlight
        hex_to_rgb("#9B5DE5"),  # 02 Canonical violet skin (#9B5DE5)
        hex_to_rgb("#7E22CE"),  # 03 Violet skin shadow
        hex_to_rgb("#581C87"),  # 04 Deep violet crease
        hex_to_rgb("#6B3A8C"),  # 05 Dark-plum vest lit
        hex_to_rgb("#4C286E"),  # 06 Canonical dark-plum vest / loincloth (#4C286E)
        hex_to_rgb("#2E1047"),  # 07 Dark-plum shadow
        hex_to_rgb("#F8FAFC"),  # 08 Glowing eye / dagger edge glint
        hex_to_rgb("#CBD5E1"),  # 09 Curved steel dagger lit
        hex_to_rgb("#94A3B8"),  # 10 Curved steel dagger mid
        hex_to_rgb("#475569"),  # 11 Dagger guard / cross-strap slate
        hex_to_rgb("#334155"),  # 12 Dark leather harness
    ],
    dtype=np.float32,
)

BRUTE_PAL = np.array(
    [
        hex_to_rgb("#11151C"),  # 00 Outer dark ink contour
        hex_to_rgb("#FDBA74"),  # 01 Siege-orange armor specular bevel
        hex_to_rgb("#FB923C"),  # 02 Siege-orange sunlit plate
        hex_to_rgb("#F3722C"),  # 03 Canonical siege-orange armor (#F3722C)
        hex_to_rgb("#C2410C"),  # 04 Siege-orange shaded plate
        hex_to_rgb("#7C2D12"),  # 05 Deep burnt-orange recess
        hex_to_rgb("#94A3B8"),  # 06 Bucket helm / gauntlet top rim highlight
        hex_to_rgb("#6E6A66"),  # 07 Crusher gauntlet warm iron mid (#6E6A66)
        hex_to_rgb("#4A525D"),  # 08 Canonical dark-iron helm / gauntlet (#4A525D)
        hex_to_rgb("#333842"),  # 09 Dark-iron shadow
        hex_to_rgb("#785233"),  # 10 Heavy leather under-arm mid
        hex_to_rgb("#4A321F"),  # 11 Heavy leather shadow
    ],
    dtype=np.float32,
)

BOSS_PAL = np.array(
    [
        hex_to_rgb("#11151C"),  # 00 Outer dark ink contour
        hex_to_rgb("#FCE7F3"),  # 01 Hot arcane visor / chest rune core (#FCE7F3)
        hex_to_rgb("#F0ABFC"),  # 02 Bright orchid rune glow
        hex_to_rgb("#D946EF"),  # 03 Canonical magenta core & blade runes (#D946EF)
        hex_to_rgb("#A21CAF"),  # 04 Deep magenta rim glow
        hex_to_rgb("#9F8AB8"),  # 05 Crescent horn / blade bevel highlight
        hex_to_rgb("#6D5687"),  # 06 Crescent horn / runic cleaver steel mid
        hex_to_rgb("#4C3B63"),  # 07 Canonical war-cleaver slate-violet (#4C3B63)
        hex_to_rgb("#38264E"),  # 08 Obsidian-violet armor sunlit bevel
        hex_to_rgb("#28183B"),  # 09 Obsidian-violet armor mid
        hex_to_rgb("#1E102B"),  # 10 Canonical obsidian-violet armor (#1E102B)
        hex_to_rgb("#6B215C"),  # 11 Plum loincloth tabard mid
        hex_to_rgb("#4A154B"),  # 12 Plum loincloth tabard shadow
        hex_to_rgb("#5C4033"),  # 13 Dark leather belt / hilt wrap
    ],
    dtype=np.float32,
)


# ============================================================================
# SHARED EXTRACTION, OUTLINE & SUPERSAMPLED CEL QUANTIZATION HELPERS
# ============================================================================

def poly_mask(shape_hw: Tuple[int, int], pts: List[Tuple[float, float]]) -> np.ndarray:
    im = Image.new("L", (shape_hw[1], shape_hw[0]), 0)
    ImageDraw.Draw(im).polygon(pts, fill=255)
    return np.array(im) > 127


def extract_clean_fg(
    raw_rgb: Image.Image,
    crop_box: Tuple[int, int, int, int],
    max_bottom_y: int,
    foot_x_ranges: List[Tuple[int, int]] | None = None,
    foot_line_y: int | None = None,
    remove_interior_bg_holes: bool = True,
    strip_shadow_oval_below_y: int | None = None,
) -> np.ndarray:
    """
    Extract crisp RGBA foreground from turnaround studio grey backdrop with zero grey
    floor-line or ground-shadow leaks.
    """
    sub = np.array(raw_rgb.crop(crop_box), dtype=np.uint8)
    h, w, _ = sub.shape
    corners = np.vstack([sub[0:10, 0:10].reshape(-1, 3), sub[0:10, w - 10 : w].reshape(-1, 3)])
    bg_med = np.median(corners, axis=0)

    diff = np.linalg.norm(sub.astype(np.float32) - bg_med[None, None, :], axis=2)
    chroma = sub.max(axis=2).astype(np.int16) - sub.min(axis=2).astype(np.int16)
    lum = 0.2126 * sub[:, :, 0] + 0.7152 * sub[:, :, 1] + 0.0722 * sub[:, :, 2]

    is_bg_cand = (diff < 22.0) & (chroma < 18)
    labeled, _ = ndimage.label(is_bg_cand)
    border_labels = set(np.unique(labeled[0, :])) | set(np.unique(labeled[h - 1, :])) | set(
        np.unique(labeled[:, 0])
    ) | set(np.unique(labeled[:, w - 1]))
    border_labels.discard(0)
    bg_mask = np.isin(labeled, list(border_labels))

    if remove_interior_bg_holes:
        interior_bg = (diff < 18.0) & (chroma < 14) & (lum > 185.0)
        bg_mask = bg_mask | interior_bg

    fg = ~bg_mask
    if max_bottom_y < h:
        fg[max_bottom_y:, :] = False

    if foot_x_ranges is not None and foot_line_y is not None:
        y = min(h - 1, foot_line_y)
        allowed = np.zeros((w,), dtype=bool)
        for x0, x1 in foot_x_ranges:
            allowed[max(0, x0) : min(w, x1 + 1)] = True
        fg[y:, ~allowed] = False

    if strip_shadow_oval_below_y is not None:
        # Studio shadow ovals are mid-grey (lum 125..195, chroma < 20) outside dark #11151C boot ink!
        y0 = max(0, strip_shadow_oval_below_y)
        sh_region = np.zeros((h, w), dtype=bool)
        sh_region[y0:, :] = True
        shadow_oval = sh_region & (chroma < 20) & (lum > 122.0) & (diff < 68.0)
        # Only strip shadow pixels connected to the outside of boots so interior steel boot highlights stay intact!
        lbl_s, _ = ndimage.label(shadow_oval | (~fg))
        ext_lbls = set(np.unique(lbl_s[h - 1, :])) | set(np.unique(lbl_s[y0:, 0])) | set(np.unique(lbl_s[y0:, w - 1]))
        ext_lbls.discard(0)
        fg[sh_region & np.isin(lbl_s, list(ext_lbls))] = False

    # Remove tiny isolated specks (< 25 px)
    lbl_fg, n_fg = ndimage.label(fg)
    if n_fg > 0:
        sizes = ndimage.sum(fg, lbl_fg, range(1, n_fg + 1))
        for idx, sz in enumerate(sizes, 1):
            if sz < 25:
                fg[lbl_fg == idx] = False

    eroded = ndimage.binary_erosion(fg, iterations=1)
    boundary = fg & (~eroded)
    halo = boundary & (lum > 142.0) & (diff < 72.0)
    fg[halo] = False

    rgba = np.zeros((h, w, 4), dtype=np.uint8)
    rgba[:, :, :3] = sub
    rgba[:, :, 3] = np.where(fg, 255, 0).astype(np.uint8)
    return rgba


def add_ink_outline(rgba: np.ndarray, radius: int = 3) -> np.ndarray:
    out = rgba.copy()
    alpha = out[:, :, 3] > 127
    if not alpha.any():
        return out
    y_idx, x_idx = np.ogrid[-radius : radius + 1, -radius : radius + 1]
    disk = (x_idx * x_idx + y_idx * y_idx) <= (radius * radius)
    eroded = ndimage.binary_erosion(alpha, structure=disk, border_value=0)
    rim = alpha & (~eroded)
    lum = (
        0.2126 * out[:, :, 0].astype(np.float32)
        + 0.7152 * out[:, :, 1].astype(np.float32)
        + 0.0722 * out[:, :, 2].astype(np.float32)
    )
    need_ink = rim & (lum > 48.0)
    out[need_ink, 0] = INK_RGB[0]
    out[need_ink, 1] = INK_RGB[1]
    out[need_ink, 2] = INK_RGB[2]
    return out


def transform_layer(
    layer_img: Image.Image,
    pivot_src: Tuple[float, float],
    target_pos: Tuple[float, float],
    angle_deg: float = 0.0,
    scale_xy: Tuple[float, float] = (1.0, 1.0),
    canvas_size: Tuple[int, int] = (512, 512),
) -> Image.Image:
    sx, sy = scale_xy
    if abs(sx - 1.0) > 1e-4 or abs(sy - 1.0) > 1e-4:
        nw = max(1, int(round(layer_img.width * sx)))
        nh = max(1, int(round(layer_img.height * sy)))
        scaled = layer_img.resize((nw, nh), Image.Resampling.BILINEAR)
        px, py = pivot_src[0] * sx, pivot_src[1] * sy
    else:
        scaled = layer_img
        px, py = pivot_src

    pad = max(scaled.width, scaled.height) + 64
    big = Image.new("RGBA", (pad * 2, pad * 2), (0, 0, 0, 0))
    big.paste(scaled, (int(round(pad - px)), int(round(pad - py))))
    if abs(angle_deg) > 1e-3:
        big = big.rotate(-angle_deg, resample=Image.Resampling.BILINEAR, center=(pad, pad))

    out = Image.new("RGBA", canvas_size, (0, 0, 0, 0))
    out.paste(big, (int(round(target_pos[0] - pad)), int(round(target_pos[1] - pad))))
    return out


def rotate_vec(vx: float, vy: float, angle_deg: float, sx: float = 1.0, sy: float = 1.0) -> Tuple[float, float]:
    th = math.radians(angle_deg)
    vx *= sx
    vy *= sy
    return (vx * math.cos(th) - vy * math.sin(th), vx * math.sin(th) + vy * math.cos(th))


def draw_tapered_cel_smear(
    canvas_hi: Image.Image,
    center_xy: Tuple[float, float],
    radius_outer: float,
    max_thick: float,
    deg_start: float,
    deg_end: float,
    outer_rgb: Tuple[int, int, int],
    core_rgb: Tuple[int, int, int],
    outline_w: int = 4,
) -> Image.Image:
    arc_layer = Image.new("RGBA", canvas_hi.size, (0, 0, 0, 0))
    dr = ImageDraw.Draw(arc_layer)
    cx, cy = center_xy
    steps = 24
    outer_pts = []
    inner_pts = []
    core_outer = []
    core_inner = []
    for i in range(steps + 1):
        u = i / steps
        ang = math.radians(deg_start + (deg_end - deg_start) * u)
        taper = math.sin(u * math.pi * 0.86) ** 0.75
        thk = max_thick * taper
        ro = radius_outer
        ri = radius_outer - thk
        outer_pts.append((cx + ro * math.cos(ang), cy + ro * math.sin(ang)))
        inner_pts.append((cx + ri * math.cos(ang), cy + ri * math.sin(ang)))
        if u >= 0.24:
            ro_c = radius_outer - thk * 0.20
            ri_c = radius_outer - thk * 0.68
            core_outer.append((cx + ro_c * math.cos(ang), cy + ro_c * math.sin(ang)))
            core_inner.append((cx + ri_c * math.cos(ang), cy + ri_c * math.sin(ang)))

    dr.polygon(outer_pts + list(reversed(inner_pts)), fill=(*outer_rgb, 255), outline=(*INK_RGB, 255), width=outline_w)
    if len(core_outer) >= 3:
        dr.polygon(core_outer + list(reversed(core_inner)), fill=(*core_rgb, 255))
    return Image.alpha_composite(canvas_hi, arc_layer)


def finalize_cel_frame(
    canvas_hi: Image.Image,
    palette_rgb: np.ndarray,
    out_dim: int = 128,
    min_island_px: int = 4,
) -> Image.Image:
    """
    Downsample 4x supersampled canvas (512->128 or 1024->256) with premultiplied alpha,
    threshold binary alpha (0 or 255, zero semi-transparent halos), quantize to locked
    <=16-color palette, and enforce closed #11151C outer ink contour.
    """
    hi_dim = out_dim * 4
    assert canvas_hi.size == (hi_dim, hi_dim), f"Expected {(hi_dim, hi_dim)}, got {canvas_hi.size}"
    arr_hi = np.array(canvas_hi, dtype=np.float32)
    a_hi = arr_hi[:, :, 3:4] / 255.0
    premul = arr_hi[:, :, :3] * a_hi

    premul_lo = premul.reshape(out_dim, 4, out_dim, 4, 3).mean(axis=(1, 3))
    a_lo = a_hi.reshape(out_dim, 4, out_dim, 4).mean(axis=(1, 3))

    mask = a_lo >= 0.48
    lbl, n_lbl = ndimage.label(mask)
    if n_lbl > 1:
        sizes = ndimage.sum(mask, lbl, range(1, n_lbl + 1))
        for idx, sz in enumerate(sizes, 1):
            if sz < min_island_px:
                mask[lbl == idx] = False

    rgb_lo = np.zeros((out_dim, out_dim, 3), dtype=np.float32)
    valid = a_lo > 0.05
    rgb_lo[valid] = np.clip(premul_lo[valid] / a_lo[valid, None], 0.0, 255.0)

    quant_rgb = np.zeros((out_dim, out_dim, 3), dtype=np.uint8)
    if mask.any():
        flat = rgb_lo[mask]
        weights = np.array([0.30, 0.59, 0.11], dtype=np.float32)[None, None, :]
        diffs = (flat[:, None, :] - palette_rgb[None, :, :]) ** 2 * weights
        nearest_idx = np.argmin(diffs.sum(axis=2), axis=1)
        quant_rgb[mask] = palette_rgb[nearest_idx].astype(np.uint8)

        eroded_4 = ndimage.binary_erosion(mask, structure=np.array([[0, 1, 0], [1, 1, 1], [0, 1, 0]]), border_value=0)
        outer_1px = mask & (~eroded_4)
        quant_rgb[outer_1px] = INK_RGB

    out = np.zeros((out_dim, out_dim, 4), dtype=np.uint8)
    out[:, :, :3] = quant_rgb
    out[:, :, 3] = np.where(mask, 255, 0).astype(np.uint8)
    return Image.fromarray(out, mode="RGBA")


def measure_leg_sole(leg_layer_hi: Image.Image, out_dim: int = 128) -> List[float]:
    a_hi = np.array(leg_layer_hi, dtype=np.float32)[:, :, 3] / 255.0
    a_lo = a_hi.reshape(out_dim, 4, out_dim, 4).mean(axis=(1, 3)) >= 0.48
    ys, xs = np.where(a_lo)
    if len(ys) == 0:
        return [out_dim * 0.5, round(out_dim * 0.92, 1)]
    y_max = int(ys.max())
    sole_xs = xs[ys >= max(0, y_max - 2)]
    return [round(float(np.median(sole_xs)), 2), float(y_max)]


# ============================================================================
# 1. GUARD RIG (chr_role_guard — 350x565 crop from guard_turnaround_raw.jpg)
# ============================================================================

def build_guard_rig() -> Dict[str, Tuple[Image.Image, Tuple[float, float]]]:
    raw_turn = Image.open(ART002_DIR / "characters" / "source" / "guard_turnaround_raw.jpg").convert("RGB")
    base = extract_clean_fg(
        raw_turn,
        crop_box=(375, 130, 725, 695),
        max_bottom_y=553,
        foot_x_ranges=[(80, 146), (178, 276)],
        foot_line_y=520,
        strip_shadow_oval_below_y=515,
    )
    h, w, _ = base.shape

    # 1) HEAD + KETTLE HELM (neck pivot 166, 116)
    m_head = poly_mask((h, w), [(76, 0), (256, 0), (256, 84), (218, 118), (150, 122), (120, 102), (76, 86)])
    head_arr = np.zeros_like(base)
    head_arr[m_head] = base[m_head]
    head_im = Image.fromarray(head_arr)
    neck_plug = Image.new("RGBA", (w, h), (0, 0, 0, 0))
    ImageDraw.Draw(neck_plug).rounded_rectangle(
        [144, 98, 200, 128], radius=10, fill=(173, 110, 77, 255), outline=(*INK_RGB, 255), width=4
    )
    head_im = Image.alpha_composite(neck_plug, head_im)

    # 2) TOWER SHIELD (rectangular steel-rimmed cobalt shield; grip pivot 228, 295)
    m_shield = poly_mask((h, w), [(151, 131), (338, 131), (338, 473), (151, 473)])
    sh_arr = np.zeros_like(base)
    sh_arr[m_shield] = base[m_shield]
    shield_im = Image.fromarray(add_ink_outline(sh_arr, radius=3))

    # 3) FRONT ARM (screen-left armored right arm, clenched gauntlet, ZERO weapon; shoulder pivot 88, 165)
    m_farm = poly_mask(
        (h, w),
        [(8, 126), (112, 126), (132, 165), (126, 212), (106, 246), (110, 315), (114, 372), (48, 378), (8, 298)],
    )
    farm_arr = np.zeros_like(base)
    farm_arr[m_farm & (~m_head)] = base[m_farm & (~m_head)]
    farm_im = Image.fromarray(add_ink_outline(farm_arr, radius=3))
    sh_plug = Image.new("RGBA", (w, h), (0, 0, 0, 0))
    ImageDraw.Draw(sh_plug).ellipse(
        [56, 134, 118, 198], fill=(148, 163, 184, 255), outline=(*INK_RGB, 255), width=4
    )
    farm_im = Image.alpha_composite(sh_plug, farm_im)

    # 4) FRONT LEG (screen-left steel greave + sabaton; hip pivot 116, 378)
    m_fleg = poly_mask((h, w), [(80, 376), (151, 376), (151, 540), (80, 540)])
    fleg_arr = np.zeros_like(base)
    fleg_arr[m_fleg & (~m_shield)] = base[m_fleg & (~m_shield)]
    fleg_im = Image.fromarray(add_ink_outline(fleg_arr, radius=3))
    fleg_thigh = Image.new("RGBA", (w, h), (0, 0, 0, 0))
    ImageDraw.Draw(fleg_thigh).rounded_rectangle(
        [88, 352, 146, 412], radius=14, fill=(51, 65, 85, 255), outline=(*INK_RGB, 255), width=4
    )
    fleg_im = Image.alpha_composite(fleg_thigh, fleg_im)

    # 5) BACK LEG (screen-right steel sabaton + upper greave extension; hip pivot 212, 382)
    m_bleg = poly_mask((h, w), [(176, 464), (276, 464), (276, 537), (176, 537)])
    bleg_arr = np.zeros_like(base)
    bleg_arr[m_bleg & (~m_shield)] = base[m_bleg & (~m_shield)]
    bleg_im = Image.fromarray(add_ink_outline(bleg_arr, radius=3))
    bleg_upper = Image.new("RGBA", (w, h), (0, 0, 0, 0))
    bdr = ImageDraw.Draw(bleg_upper)
    bdr.rounded_rectangle([184, 358, 240, 480], radius=14, fill=(100, 116, 139, 255), outline=(*INK_RGB, 255), width=4)
    bdr.rounded_rectangle([190, 398, 236, 442], radius=10, fill=(148, 163, 184, 255), outline=(*INK_RGB, 255), width=3)
    bleg_im = Image.alpha_composite(bleg_upper, bleg_im)

    # 6) TORSO + COBALT CAPE (waist pivot 162, 365) — extend breastplate/cape behind shield edge so zero gap appears
    m_torso_box = poly_mask((h, w), [(72, 96), (242, 96), (242, 462), (72, 462)])
    m_torso = m_torso_box & (~m_head) & (~m_shield) & (~m_farm) & (~m_fleg) & (~m_bleg)
    torso_arr = np.zeros_like(base)
    torso_arr[m_torso] = base[m_torso]
    torso_fill = Image.new("RGBA", (w, h), (0, 0, 0, 0))
    tdr = ImageDraw.Draw(torso_fill)
    # Royal cobalt cape extension across shoulders
    tdr.rounded_rectangle([84, 102, 232, 176], radius=20, fill=(46, 124, 246, 255), outline=(*INK_RGB, 255), width=4)
    # Steel breastplate & tassets behind left edge of shield
    tdr.rounded_rectangle([104, 152, 220, 382], radius=18, fill=(100, 116, 139, 255), outline=(*INK_RGB, 255), width=4)
    tdr.rounded_rectangle([112, 162, 196, 290], radius=14, fill=(148, 163, 184, 255))
    tdr.line([(106, 305), (218, 305)], fill=(*INK_RGB, 255), width=4)
    tdr.line([(106, 342), (218, 342)], fill=(*INK_RGB, 255), width=4)
    torso_comp = Image.alpha_composite(torso_fill, Image.fromarray(torso_arr))
    torso_im = Image.fromarray(add_ink_outline(np.array(torso_comp), radius=3))

    return {
        "head": (head_im, (166.0, 116.0)),
        "front_arm": (farm_im, (88.0, 165.0)),
        "front_leg": (fleg_im, (116.0, 378.0)),
        "back_leg": (bleg_im, (212.0, 382.0)),
        "torso": (torso_im, (162.0, 365.0)),
        "shield": (shield_im, (228.0, 295.0)),
    }


# ============================================================================
# 2. FIGHTER RIG (chr_role_fighter — 340x605 crop, NO STEEL PAULDRONS!)
# ============================================================================

def convert_shoulder_steel_to_crimson_cloth(rgba: np.ndarray, cap_mask: np.ndarray) -> np.ndarray:
    """
    Phase 3 invariant + Owner Decision A2: 'No steel shoulder pauldrons.'
    Converts Fighter's cool-grey steel shoulder caps into crimson #E63946 cloth surcoat
    sleeves while preserving 45-deg top-left cel highlight/shadow bands and #11151C ink contours.
    """
    out = rgba.copy()
    sub = out[:, :, :3].astype(np.float32)
    lum = 0.2126 * sub[:, :, 0] + 0.7152 * sub[:, :, 1] + 0.0722 * sub[:, :, 2]
    # Cool steel has R - B < 35 and lum > 50 (leaving #11151C dark ink & warm brown leather untouched)
    steel = cap_mask & (out[:, :, 3] > 0) & (lum > 50.0) & ((sub[:, :, 0] - sub[:, :, 2]) < 35.0)
    lit = steel & (lum >= 172.0)
    mid = steel & (lum >= 122.0) & (lum < 172.0)
    shd = steel & (lum < 122.0)
    out[lit, :3] = hex_to_rgb("#F87171")
    out[mid, :3] = hex_to_rgb("#E63946")
    out[shd, :3] = hex_to_rgb("#B91C1C")
    return out


def build_fighter_rig() -> Dict[str, Tuple[Image.Image, Tuple[float, float]]]:
    raw_turn = Image.open(ART002_DIR / "characters" / "source" / "fighter_turnaround_raw.jpg").convert("RGB")
    base = extract_clean_fg(
        raw_turn,
        crop_box=(375, 80, 715, 685),
        max_bottom_y=577,
        foot_x_ranges=[(82, 150), (174, 260)],
        foot_line_y=540,
        strip_shadow_oval_below_y=535,
    )
    h, w, _ = base.shape

    # Convert both shoulder steel caps to crimson #E63946 cloth sleeves!
    m_fcap = poly_mask((h, w), [(28, 214), (138, 214), (138, 326), (98, 338), (28, 304)])
    m_bcap = poly_mask((h, w), [(224, 216), (284, 216), (284, 336), (236, 336)])
    base = convert_shoulder_steel_to_crimson_cloth(base, m_fcap | m_bcap)

    # 1) HEAD + BARBUTE HELM + CRIMSON PLUME (neck pivot 168, 238)
    m_head = poly_mask(
        (h, w),
        [(44, 0), (242, 0), (242, 235), (210, 256), (156, 256), (110, 224), (44, 196)],
    )
    head_arr = np.zeros_like(base)
    head_arr[m_head] = base[m_head]
    head_im = Image.fromarray(add_ink_outline(head_arr, radius=3))

    # 2) BROADSWORD (unobstructed pommel + brass crossguard + polished steel blade; grip pivot 78, 422)
    m_sword_pommel = poly_mask((h, w), [(8, 390), (50, 404), (44, 434), (8, 420)])
    m_sword_blade = poly_mask(
        (h, w),
        [(74, 398), (124, 398), (118, 418), (328, 502), (328, 518), (278, 528), (94, 456), (88, 480), (72, 474)],
    )
    m_sword = m_sword_pommel | m_sword_blade
    sw_arr = np.zeros_like(base)
    sw_arr[m_sword] = base[m_sword]
    sw_im = Image.fromarray(sw_arr)
    # Connect hilt handle smoothly inside the grip (42..86) so sword is one solid rigid steel/brass piece
    sw_hilt = Image.new("RGBA", (w, h), (0, 0, 0, 0))
    ImageDraw.Draw(sw_hilt).polygon(
        [(38, 405), (92, 424), (86, 444), (32, 424)],
        fill=(92, 61, 40, 255),
        outline=(*INK_RGB, 255),
        width=4,
    )
    sword_im = Image.fromarray(add_ink_outline(np.array(Image.alpha_composite(sw_hilt, sw_im)), radius=3))

    # 3) FRONT UPPER ARM (crimson cloth shoulder sleeve + brown upper arm; shoulder pivot 86, 266 -> elbow 62, 355)
    m_fuarm = poly_mask((h, w), [(28, 214), (136, 214), (136, 326), (102, 362), (34, 356)])
    fuarm_arr = np.zeros_like(base)
    fuarm_arr[m_fuarm & (~m_head)] = base[m_fuarm & (~m_head)]
    fuarm_im = Image.fromarray(add_ink_outline(fuarm_arr, radius=3))

    # 4) FRONT FOREARM + GAUNTLET (steel vambrace & gauntlet; elbow pivot 62, 355 -> grip 78, 422)
    m_ffarm = poly_mask((h, w), [(20, 342), (106, 354), (98, 402), (88, 450), (42, 448), (20, 386)])
    ffarm_arr = np.zeros_like(base)
    ffarm_arr[m_ffarm & (~m_sword)] = base[m_ffarm & (~m_sword)]
    ffarm_im = Image.fromarray(add_ink_outline(ffarm_arr, radius=3))
    el_plug = Image.new("RGBA", (w, h), (0, 0, 0, 0))
    ImageDraw.Draw(el_plug).ellipse([38, 336, 90, 380], fill=(148, 163, 184, 255), outline=(*INK_RGB, 255), width=4)
    ffarm_im = Image.alpha_composite(el_plug, ffarm_im)

    # 5) BACK ARM (screen-right crimson sleeve + steel vambrace + leather fist; shoulder pivot 246, 268)
    m_barm = poly_mask((h, w), [(228, 218), (296, 218), (296, 446), (242, 446), (232, 348)])
    barm_arr = np.zeros_like(base)
    barm_arr[m_barm & (~m_head) & (~m_sword)] = base[m_barm & (~m_head) & (~m_sword)]
    barm_im = Image.fromarray(add_ink_outline(barm_arr, radius=3))

    # 6) FRONT LEG (screen-left steel knee cop + greave + sabaton; hip pivot 118, 452)
    m_sword_pad = ndimage.binary_dilation(m_sword, iterations=5)
    m_fleg = poly_mask((h, w), [(82, 464), (154, 464), (154, 577), (82, 577)])
    fleg_arr = np.zeros_like(base)
    fleg_arr[m_fleg & (~m_sword_pad)] = base[m_fleg & (~m_sword_pad)]
    fleg_base = Image.new("RGBA", (w, h), (0, 0, 0, 0))
    fdr = ImageDraw.Draw(fleg_base)
    fdr.rounded_rectangle([90, 434, 146, 496], radius=14, fill=(74, 50, 34, 255), outline=(*INK_RGB, 255), width=4)
    fleg_comp = Image.alpha_composite(fleg_base, Image.fromarray(fleg_arr))
    fleg_im = Image.fromarray(add_ink_outline(np.array(fleg_comp), radius=3))

    # 7) BACK LEG (screen-right steel greave + sabaton, healed behind sword blade; hip pivot 208, 452)
    m_bleg = poly_mask((h, w), [(174, 504), (236, 504), (258, 545), (258, 566), (174, 566)])
    bleg_arr = np.zeros_like(base)
    bleg_arr[m_bleg & (~m_sword_pad)] = base[m_bleg & (~m_sword_pad)]
    bleg_base = Image.new("RGBA", (w, h), (0, 0, 0, 0))
    bdr = ImageDraw.Draw(bleg_base)
    bdr.rounded_rectangle([178, 434, 236, 532], radius=14, fill=(100, 116, 139, 255), outline=(*INK_RGB, 255), width=4)
    bdr.rounded_rectangle([184, 456, 230, 502], radius=10, fill=(148, 163, 184, 255), outline=(*INK_RGB, 255), width=3)
    bleg_comp = Image.alpha_composite(bleg_base, Image.fromarray(bleg_arr))
    bleg_im = Image.fromarray(add_ink_outline(np.array(bleg_comp), radius=3))

    # 8) TORSO (crimson #E63946 cloth surcoat + belt + pleated skirt healed behind sword; waist pivot 168, 430)
    m_torso_keep = poly_mask((h, w), [(96, 224), (242, 224), (244, 472), (96, 472)]) & (~m_head) & (~m_fuarm) & (~m_ffarm) & (~m_barm) & (~m_sword)
    torso_arr = np.zeros_like(base)
    torso_arr[m_torso_keep] = base[m_torso_keep]
    skirt_patch = Image.new("RGBA", (w, h), (0, 0, 0, 0))
    sdr = ImageDraw.Draw(skirt_patch)
    # Pleated crimson surcoat skirt behind lifted broadsword
    sdr.polygon([(98, 392), (236, 392), (244, 468), (102, 468)], fill=(230, 57, 70, 255), outline=(*INK_RGB, 255), width=4)
    sdr.polygon([(192, 394), (236, 394), (244, 468), (196, 468)], fill=(185, 28, 28, 255), outline=(*INK_RGB, 255), width=3)
    sdr.line([(150, 396), (148, 466)], fill=(*INK_RGB, 255), width=4)
    sdr.rounded_rectangle([102, 372, 232, 398], radius=6, fill=(139, 94, 60, 255), outline=(*INK_RGB, 255), width=4)
    torso_comp = Image.alpha_composite(skirt_patch, Image.fromarray(torso_arr))
    torso_im = Image.fromarray(add_ink_outline(np.array(torso_comp), radius=3))

    return {
        "head": (head_im, (168.0, 238.0)),
        "back_arm": (barm_im, (246.0, 268.0)),
        "front_leg": (fleg_im, (118.0, 452.0)),
        "back_leg": (bleg_im, (208.0, 452.0)),
        "torso": (torso_im, (168.0, 430.0)),
        "front_upper_arm": (fuarm_im, (86.0, 266.0)),
        "front_forearm": (ffarm_im, (62.0, 355.0)),
        "sword": (sword_im, (78.0, 422.0)),
    }


# ============================================================================
# 3. RUNNER RIG (chr_enemy_runner — 300x565 crop, locked 3/4 +X facing, twin daggers)
# ============================================================================

def build_runner_rig() -> Dict[str, Tuple[Image.Image, Tuple[float, float]]]:
    raw_turn = Image.open(ART002_DIR / "characters" / "source" / "runner_turnaround_raw.jpg").convert("RGB")
    base = extract_clean_fg(
        raw_turn,
        crop_box=(390, 125, 690, 690),
        max_bottom_y=546,
        foot_x_ranges=[(64, 130), (176, 276)],
        foot_line_y=492,
        strip_shadow_oval_below_y=488,
    )
    h, w, _ = base.shape

    # 1) HEAD + SWEPT-BACK EARS (neck pivot 182, 138)
    m_head = poly_mask((h, w), [(74, 0), (256, 0), (256, 152), (222, 155), (175, 146), (144, 108), (74, 48)])
    head_arr = np.zeros_like(base)
    head_arr[m_head] = base[m_head]
    head_im = Image.fromarray(add_ink_outline(head_arr, radius=3))

    # 2) BACK ARM + CURVED DAGGER #2 (shoulder pivot 214, 165)
    m_barm_limb = poly_mask((h, w), [(208, 152), (288, 152), (288, 336), (235, 336), (212, 222)])
    m_bdagger = poly_mask((h, w), [(192, 262), (248, 294), (288, 310), (288, 334), (232, 326), (192, 286)])
    # Keep steel/hilt pixels on dagger #2 (chroma < 48 or dark ink) so violet thigh doesn't leak onto blade
    chroma = base[:, :, :3].max(axis=2).astype(np.int16) - base[:, :, :3].min(axis=2).astype(np.int16)
    lum = 0.2126 * base[:, :, 0] + 0.7152 * base[:, :, 1] + 0.0722 * base[:, :, 2]
    m_barm = m_barm_limb | (m_bdagger & ((chroma < 46) | (lum < 48.0)))
    barm_arr = np.zeros_like(base)
    barm_arr[m_barm & (~m_head)] = base[m_barm & (~m_head)]
    barm_im = Image.fromarray(add_ink_outline(barm_arr, radius=3))
    bsh_plug = Image.new("RGBA", (w, h), (0, 0, 0, 0))
    ImageDraw.Draw(bsh_plug).ellipse([196, 142, 246, 194], fill=(126, 34, 206, 255), outline=(*INK_RGB, 255), width=4)
    barm_im = Image.alpha_composite(bsh_plug, barm_im)

    # 3) FRONT UPPER ARM (shoulder pivot 112, 160 -> elbow 56, 232)
    m_fuarm = poly_mask((h, w), [(36, 138), (126, 138), (114, 212), (82, 248), (36, 238)])
    fuarm_arr = np.zeros_like(base)
    fuarm_arr[m_fuarm & (~m_head)] = base[m_fuarm & (~m_head)]
    fuarm_im = Image.fromarray(add_ink_outline(fuarm_arr, radius=3))

    # 4) FRONT FOREARM + CURVED DAGGER #1 (elbow pivot 56, 232)
    m_ffarm_limb = poly_mask((h, w), [(10, 224), (86, 224), (88, 344), (10, 344)])
    m_fdagger = poly_mask((h, w), [(58, 286), (96, 286), (198, 336), (198, 356), (156, 366), (58, 348)])
    m_ffarm = m_ffarm_limb | (m_fdagger & ((chroma < 46) | (lum < 48.0)))
    ffarm_arr = np.zeros_like(base)
    ffarm_arr[m_ffarm] = base[m_ffarm]
    ffarm_im = Image.fromarray(add_ink_outline(ffarm_arr, radius=3))
    el_plug = Image.new("RGBA", (w, h), (0, 0, 0, 0))
    ImageDraw.Draw(el_plug).ellipse([28, 204, 86, 260], fill=(155, 93, 229, 255), outline=(*INK_RGB, 255), width=4)
    ffarm_im = Image.alpha_composite(el_plug, ffarm_im)

    # Dilate dagger masks by 6px when cutting legs so zero dark dagger-rim barbs remain on thighs!
    m_ffarm_pad = ndimage.binary_dilation(m_ffarm | m_fdagger, iterations=5)
    m_barm_pad = ndimage.binary_dilation(m_barm | m_bdagger, iterations=5)

    # 5) FRONT LEG (screen-left crouched violet leg, healed behind dagger #1; hip pivot 126, 308)
    m_fleg = poly_mask((h, w), [(64, 290), (154, 290), (154, 546), (64, 546)]) & (~m_ffarm_pad)
    fleg_arr = np.zeros_like(base)
    fleg_arr[m_fleg] = base[m_fleg]
    fleg_patch = Image.new("RGBA", (w, h), (0, 0, 0, 0))
    ImageDraw.Draw(fleg_patch).polygon(
        [(100, 292), (150, 296), (144, 386), (98, 382)],
        fill=(155, 93, 229, 255),
        outline=(*INK_RGB, 255),
        width=4,
    )
    fleg_comp = Image.alpha_composite(fleg_patch, Image.fromarray(fleg_arr))
    fleg_im = Image.fromarray(add_ink_outline(np.array(fleg_comp), radius=3))

    # 6) BACK LEG (screen-right crouched violet leg, healed behind dagger #2; hip pivot 184, 308)
    m_bleg = poly_mask((h, w), [(155, 290), (276, 290), (276, 526), (155, 526)]) & (~m_barm_pad) & (~m_ffarm_pad)
    bleg_arr = np.zeros_like(base)
    bleg_arr[m_bleg] = base[m_bleg]
    bleg_patch = Image.new("RGBA", (w, h), (0, 0, 0, 0))
    ImageDraw.Draw(bleg_patch).polygon(
        [(154, 282), (222, 284), (252, 384), (188, 388)],
        fill=(155, 93, 229, 255),
        outline=(*INK_RGB, 255),
        width=4,
    )
    bleg_comp = Image.alpha_composite(bleg_patch, Image.fromarray(bleg_arr))
    bleg_im = Image.fromarray(add_ink_outline(np.array(bleg_comp), radius=3))

    # 7) TORSO (dark-plum #4C286E vest, cross-strap, belt & pelvis; waist pivot 158, 288)
    m_torso = poly_mask((h, w), [(100, 108), (226, 108), (226, 336), (100, 336)]) & (~m_head) & (~m_fuarm) & (~m_barm) & (~m_ffarm)
    torso_arr = np.zeros_like(base)
    torso_arr[m_torso] = base[m_torso]
    pelvis_plug = Image.new("RGBA", (w, h), (0, 0, 0, 0))
    ImageDraw.Draw(pelvis_plug).rounded_rectangle(
        [102, 266, 220, 340], radius=14, fill=(126, 34, 206, 255), outline=(*INK_RGB, 255), width=4
    )
    torso_comp = Image.alpha_composite(pelvis_plug, Image.fromarray(torso_arr))
    torso_im = Image.fromarray(add_ink_outline(np.array(torso_comp), radius=3))

    return {
        "head": (head_im, (182.0, 138.0)),
        "back_arm": (barm_im, (214.0, 165.0)),
        "front_leg": (fleg_im, (126.0, 308.0)),
        "back_leg": (bleg_im, (184.0, 308.0)),
        "torso": (torso_im, (158.0, 288.0)),
        "front_upper_arm": (fuarm_im, (112.0, 160.0)),
        "front_forearm": (ffarm_im, (56.0, 232.0)),
    }


# ============================================================================
# 4. BRUTE RIG (chr_enemy_brute — 365x480 crop, siege-orange armor & crusher gauntlets)
# ============================================================================

def build_brute_rig() -> Dict[str, Tuple[Image.Image, Tuple[float, float]]]:
    raw_turn = Image.open(ART002_DIR / "characters" / "source" / "brute_turnaround_raw.jpg").convert("RGB")
    base = extract_clean_fg(
        raw_turn,
        crop_box=(405, 145, 770, 625),
        max_bottom_y=453,
        foot_x_ranges=[(100, 178), (186, 284)],
        foot_line_y=432,
    )
    h, w, _ = base.shape

    # 1) HEAD — FLAT DARK-IRON BUCKET HELM (neck pivot 184, 96)
    m_head = poly_mask((h, w), [(138, 0), (230, 0), (230, 108), (186, 114), (142, 98)])
    head_arr = np.zeros_like(base)
    head_arr[m_head] = base[m_head]
    head_im = Image.fromarray(add_ink_outline(head_arr, radius=3))

    # 2) BACK ARM + CRUSHER GAUNTLET (shoulder pivot 252, 116)
    m_barm = poly_mask(
        (h, w),
        [(226, 54), (322, 84), (358, 304), (332, 364), (260, 364), (248, 245), (255, 165), (226, 105)],
    )
    barm_arr = np.zeros_like(base)
    barm_arr[m_barm & (~m_head)] = base[m_barm & (~m_head)]
    barm_im = Image.fromarray(add_ink_outline(barm_arr, radius=3))

    # 3) FRONT UPPER ARM / SIEGE-ORANGE PAULDRON (shoulder pivot 108, 116 -> elbow 72, 214)
    m_fuarm = poly_mask((h, w), [(26, 50), (168, 66), (168, 152), (124, 224), (36, 224)])
    fuarm_arr = np.zeros_like(base)
    fuarm_arr[m_fuarm & (~m_head)] = base[m_fuarm & (~m_head)]
    fuarm_im = Image.fromarray(add_ink_outline(fuarm_arr, radius=3))

    # 4) FRONT MASSIVE IRON CRUSHER GAUNTLET (elbow pivot 72, 214)
    m_fgaunt = poly_mask((h, w), [(8, 200), (122, 206), (138, 298), (134, 368), (36, 372), (8, 266)])
    fgaunt_arr = np.zeros_like(base)
    fgaunt_arr[m_fgaunt] = base[m_fgaunt]
    fgaunt_im = Image.fromarray(add_ink_outline(fgaunt_arr, radius=3))
    el_plug = Image.new("RGBA", (w, h), (0, 0, 0, 0))
    ImageDraw.Draw(el_plug).ellipse([38, 188, 106, 242], fill=(120, 82, 51, 255), outline=(*INK_RGB, 255), width=4)
    fgaunt_im = Image.alpha_composite(el_plug, fgaunt_im)

    # 5) FRONT LEG (hip pivot 142, 332)
    m_fleg = poly_mask((h, w), [(100, 326), (182, 326), (182, 454), (100, 454)]) & (~m_fgaunt)
    fleg_arr = np.zeros_like(base)
    fleg_arr[m_fleg] = base[m_fleg]
    fleg_plug = Image.new("RGBA", (w, h), (0, 0, 0, 0))
    ImageDraw.Draw(fleg_plug).rounded_rectangle(
        [110, 308, 174, 362], radius=12, fill=(120, 82, 51, 255), outline=(*INK_RGB, 255), width=4
    )
    fleg_im = Image.fromarray(add_ink_outline(np.array(Image.alpha_composite(fleg_plug, Image.fromarray(fleg_arr))), radius=3))

    # 6) BACK LEG (hip pivot 224, 332)
    m_bleg = poly_mask((h, w), [(184, 326), (286, 326), (286, 452), (184, 452)]) & (~m_barm)
    bleg_arr = np.zeros_like(base)
    bleg_arr[m_bleg] = base[m_bleg]
    bleg_plug = Image.new("RGBA", (w, h), (0, 0, 0, 0))
    ImageDraw.Draw(bleg_plug).rounded_rectangle(
        [192, 308, 252, 362], radius=12, fill=(120, 82, 51, 255), outline=(*INK_RGB, 255), width=4
    )
    bleg_im = Image.fromarray(add_ink_outline(np.array(Image.alpha_composite(bleg_plug, Image.fromarray(bleg_arr))), radius=3))

    # 7) TORSO (wide square siege-orange breastplate + tassets; waist pivot 186, 316)
    m_torso = poly_mask((h, w), [(104, 72), (274, 72), (274, 344), (104, 344)]) & (~m_head) & (~m_fuarm) & (~m_fgaunt) & (~m_barm)
    torso_arr = np.zeros_like(base)
    torso_arr[m_torso] = base[m_torso]
    chest_fill = Image.new("RGBA", (w, h), (0, 0, 0, 0))
    ImageDraw.Draw(chest_fill).rounded_rectangle(
        [116, 98, 258, 324], radius=18, fill=(243, 114, 44, 255), outline=(*INK_RGB, 255), width=4
    )
    torso_comp = Image.alpha_composite(chest_fill, Image.fromarray(torso_arr))
    torso_im = Image.fromarray(add_ink_outline(np.array(torso_comp), radius=3))

    return {
        "head": (head_im, (184.0, 96.0)),
        "back_arm": (barm_im, (252.0, 116.0)),
        "front_leg": (fleg_im, (142.0, 332.0)),
        "back_leg": (bleg_im, (224.0, 332.0)),
        "torso": (torso_im, (186.0, 316.0)),
        "front_upper_arm": (fuarm_im, (108.0, 116.0)),
        "front_forearm": (fgaunt_im, (72.0, 214.0)),
    }


# ============================================================================
# 5. BOSS RIG (chr_boss_warlord — 395x640 crop -> 256x256 RGBA, ZERO GROUND RING!)
# ============================================================================

def build_boss_rig() -> Dict[str, Tuple[Image.Image, Tuple[float, float]]]:
    raw_turn = Image.open(ART002_DIR / "characters" / "source" / "boss_turnaround_raw.jpg").convert("RGB")
    base = extract_clean_fg(
        raw_turn,
        crop_box=(380, 65, 775, 705),
        max_bottom_y=612,
        foot_x_ranges=[(94, 154), (204, 306)],
        foot_line_y=542,
    )
    h, w, _ = base.shape

    # Strip 100% of the pink/magenta baked ground telegraph ring below y=536 outside the armored sabaton polygons!
    m_front_boot_keep = poly_mask((h, w), [(96, 530), (153, 530), (152, 606), (124, 610), (96, 602)])
    m_back_boot_keep = poly_mask((h, w), [(206, 530), (274, 530), (305, 568), (304, 579), (208, 577)])
    below_536 = np.zeros((h, w), dtype=bool)
    below_536[536:, :] = True
    ring_strip = below_536 & (~m_front_boot_keep) & (~m_back_boot_keep)
    base[ring_strip, 3] = 0
    # Also strip any bright pink ring glow pixels (R>170, B>180, lum>120) clinging to boot sole rims below y=536
    lum = 0.2126 * base[:, :, 0] + 0.7152 * base[:, :, 1] + 0.0722 * base[:, :, 2]
    pink_rim = below_536 & (base[:, :, 0] > 165) & (base[:, :, 2] > 175) & (lum > 110.0)
    base[pink_rim, 3] = 0

    # 1) HEAD + CRESCENT HORNS + MAGENTA VISOR (neck pivot 196, 138)
    m_head = poly_mask((h, w), [(102, 0), (276, 0), (276, 112), (235, 152), (196, 158), (156, 146), (102, 102)])
    head_arr = np.zeros_like(base)
    head_arr[m_head] = base[m_head]
    head_im = Image.fromarray(add_ink_outline(head_arr, radius=3))

    # 2) COLOSSAL NOTCHED RUNIC WAR-CLEAVER (grip pivot 98, 354)
    m_cleaver_pommel = poly_mask((h, w), [(8, 294), (78, 324), (72, 356), (8, 316)])
    m_cleaver_blade = poly_mask(
        (h, w),
        [(98, 316), (154, 326), (158, 355), (326, 438), (390, 410), (390, 446), (364, 516), (316, 568), (306, 530), (278, 502), (126, 424), (96, 428)],
    )
    m_cleaver = m_cleaver_pommel | m_cleaver_blade
    cl_arr = np.zeros_like(base)
    cl_arr[m_cleaver] = base[m_cleaver]
    cl_hilt = Image.new("RGBA", (w, h), (0, 0, 0, 0))
    ImageDraw.Draw(cl_hilt).polygon(
        [(60, 324), (118, 352), (110, 374), (52, 346)],
        fill=(92, 64, 51, 255),
        outline=(*INK_RGB, 255),
        width=5,
    )
    cleaver_im = Image.fromarray(add_ink_outline(np.array(Image.alpha_composite(cl_hilt, Image.fromarray(cl_arr))), radius=3))

    # 3) FRONT SPIKED SHOULDER PAULDRON / UPPER ARM (shoulder pivot 118, 182 -> elbow 88, 240)
    m_fuarm = poly_mask((h, w), [(58, 106), (156, 116), (154, 228), (124, 256), (58, 232)])
    fuarm_arr = np.zeros_like(base)
    fuarm_arr[m_fuarm & (~m_head)] = base[m_fuarm & (~m_head)]
    fuarm_im = Image.fromarray(add_ink_outline(fuarm_arr, radius=3))

    # 4) FRONT FOREARM + GAUNTLET (elbow pivot 88, 240 -> grip 98, 354)
    m_ffarm = poly_mask((h, w), [(48, 222), (124, 240), (126, 378), (72, 386), (48, 300)])
    ffarm_arr = np.zeros_like(base)
    ffarm_arr[m_ffarm & (~m_cleaver)] = base[m_ffarm & (~m_cleaver)]
    ffarm_im = Image.fromarray(add_ink_outline(ffarm_arr, radius=3))

    # 5) BACK ARM (screen-right spiked shoulder + gauntlet; shoulder pivot 264, 185)
    m_barm = poly_mask((h, w), [(248, 122), (296, 122), (324, 198), (324, 384), (272, 384), (254, 285), (252, 185)])
    barm_arr = np.zeros_like(base)
    barm_arr[m_barm & (~m_head) & (~m_cleaver)] = base[m_barm & (~m_head) & (~m_cleaver)]
    barm_im = Image.fromarray(add_ink_outline(barm_arr, radius=3))

    m_cleaver_pad = ndimage.binary_dilation(m_cleaver, iterations=5)

    # 6) FRONT LEG (screen-left obsidian greave + sabaton, healed behind cleaver; hip pivot 142, 405)
    m_fleg = poly_mask((h, w), [(96, 390), (178, 390), (178, 610), (96, 610)]) & (~m_cleaver_pad)
    fleg_arr = np.zeros_like(base)
    fleg_arr[m_fleg] = base[m_fleg]
    fleg_heal = Image.new("RGBA", (w, h), (0, 0, 0, 0))
    ImageDraw.Draw(fleg_heal).rounded_rectangle(
        [108, 376, 170, 456], radius=16, fill=(40, 24, 59, 255), outline=(*INK_RGB, 255), width=5
    )
    fleg_im = Image.fromarray(add_ink_outline(np.array(Image.alpha_composite(fleg_heal, Image.fromarray(fleg_arr))), radius=3))

    # 7) BACK LEG (screen-right obsidian greave + sabaton, healed behind cleaver; hip pivot 234, 405)
    m_bleg = poly_mask((h, w), [(194, 395), (308, 395), (308, 582), (194, 582)]) & (~m_cleaver_pad)
    bleg_arr = np.zeros_like(base)
    bleg_arr[m_bleg] = base[m_bleg]
    bleg_heal = Image.new("RGBA", (w, h), (0, 0, 0, 0))
    bdr = ImageDraw.Draw(bleg_heal)
    bdr.rounded_rectangle([198, 382, 266, 510], radius=16, fill=(40, 24, 59, 255), outline=(*INK_RGB, 255), width=5)
    bdr.rounded_rectangle([206, 422, 258, 474], radius=10, fill=(56, 38, 78, 255), outline=(*INK_RGB, 255), width=3)
    bleg_im = Image.fromarray(add_ink_outline(np.array(Image.alpha_composite(bleg_heal, Image.fromarray(bleg_arr))), radius=3))

    # 8) TORSO (obsidian-violet chestplate + glowing magenta core + healed waist tassets/loincloth; waist pivot 196, 350)
    m_torso = poly_mask((h, w), [(116, 128), (280, 128), (280, 428), (116, 428)]) & (~m_head) & (~m_fuarm) & (~m_ffarm) & (~m_barm) & (~m_cleaver)
    torso_arr = np.zeros_like(base)
    torso_arr[m_torso] = base[m_torso]
    waist_heal = Image.new("RGBA", (w, h), (0, 0, 0, 0))
    wdr = ImageDraw.Draw(waist_heal)
    wdr.polygon([(122, 312), (272, 312), (278, 412), (120, 412)], fill=(40, 24, 59, 255), outline=(*INK_RGB, 255), width=5)
    wdr.polygon([(176, 326), (228, 326), (234, 424), (172, 424)], fill=(107, 33, 92, 255), outline=(*INK_RGB, 255), width=4)
    torso_comp = Image.alpha_composite(waist_heal, Image.fromarray(torso_arr))
    torso_im = Image.fromarray(add_ink_outline(np.array(torso_comp), radius=3))

    return {
        "head": (head_im, (196.0, 138.0)),
        "back_arm": (barm_im, (264.0, 185.0)),
        "front_leg": (fleg_im, (142.0, 405.0)),
        "back_leg": (bleg_im, (234.0, 405.0)),
        "torso": (torso_im, (196.0, 350.0)),
        "front_upper_arm": (fuarm_im, (118.0, 182.0)),
        "front_forearm": (ffarm_im, (88.0, 240.0)),
        "cleaver": (cleaver_im, (98.0, 354.0)),
    }
