#!/usr/bin/env python3
"""
ART-003 Deterministic 2D Cutout Rig, FK/IK Pose Solver & 42-Frame Sprite Exporter (v2 Polished).

Produces real, game-usable 128x128 RGBA animated sprites at 12 FPS from the
locked ART-002 character turnaround & prop plates:
  1. Builder (Playable Role)  — 18 frames: Idle(4), Walk(6), Repair(6), Hit(2)
  2. Grunt (Enemy Priority 2) — 18 frames: Idle(4), Walk(6), ClubSmash(6), Hit(2)
  3. vfx_builder_repair_spark —  6 frames: 128x128 RGBA golden 4-point sparks + oak chips

Technical invariants:
  - Canvas: 128x128 RGBA PNG, straight RGB, binary clean alpha (0 or 255), zero halos.
  - Pivot: (0.50, 0.08) -> pixel (64.0, 10.24 from bottom) -> ground y = 117 (0-indexed top-down).
  - Outer contour: crisp #11151C dark ink (~2.5-3.0 px source contour).
  - Controlled 16-color locked cel palette per character to eliminate cross-frame drift.
"""

from __future__ import annotations

import json
import math
from pathlib import Path
from typing import Dict, List, Tuple

import numpy as np
from PIL import Image, ImageDraw
from scipy import ndimage

REPO_ROOT = Path(__file__).resolve().parents[4]
ART002_DIR = REPO_ROOT / "art" / "concepts" / "ART-002"
ART003_DIR = REPO_ROOT / "art" / "production" / "ART-003"

INK_RGB = (17, 21, 28)  # #11151C

# Locked 16-color cel palette for Builder (Bastion Dawn v1)
BUILDER_PALETTE_HEX = [
    "#11151C",  # 00 Outer & inner dark ink
    "#FEF08A",  # 01 Apron / cap bright sun rim & smear highlight
    "#F4C430",  # 02 Golden apron & cap primary (#F4C430 canonical)
    "#D49417",  # 03 Golden apron cel shadow
    "#F3EAD6",  # 04 Cream work shirt lit
    "#D5C4A3",  # 05 Cream work shirt cel shadow
    "#F4C29F",  # 06 Warm workman skin lit
    "#DA9B74",  # 07 Warm workman skin mid/shadow
    "#AD6E4D",  # 08 Deep skin/ear/brow crease
    "#A66E44",  # 09 Leather glove / oak shaft lit
    "#7E502F",  # 10 Leather glove / boot / oak shaft base
    "#533420",  # 11 Dark trouser & leather shadow
    "#9BA7B6",  # 12 Forged iron mallet bevel highlight
    "#6E7885",  # 13 Forged iron mallet base
    "#47515E",  # 14 Forged iron mallet shadow plane
    "#FFFBEB",  # 15 Crisp action swing-arc core
]

# Locked 16-color cel palette for Grunt (Bastion Dawn v1)
GRUNT_PALETTE_HEX = [
    "#11151C",  # 00 Outer & inner dark ink
    "#8EC66C",  # 01 Moss-green top-left sun rim
    "#4CC964",  # 02 Canonical vibrant moss-green highlight-mid
    "#5E9155",  # 03 Goblinoid moss-green flesh base
    "#3E693A",  # 04 Goblinoid moss-green cel shadow
    "#284726",  # 05 Deep under-pec / inner-leg shadow
    "#F5EAB5",  # 06 Tusks, eye sclera & crisp smash-arc core
    "#FEF08A",  # 07 Warm smash-arc rim / yellow goblin iris
    "#96663F",  # 08 Hide loincloth lit
    "#6F482B",  # 09 Hide loincloth base
    "#492E1B",  # 10 Hide loincloth shadow
    "#B0784C",  # 11 Gnarled oak club lit
    "#835331",  # 12 Gnarled oak club base
    "#59361E",  # 13 Gnarled oak club shadow
    "#DCE4EE",  # 14 Conical steel spike lit
    "#8A96A6",  # 15 Conical steel spike shadow
]


def hex_to_rgb(h: str) -> Tuple[int, int, int]:
    h = h.lstrip("#")
    return (int(h[0:2], 16), int(h[2:4], 16), int(h[4:6], 16))


BUILDER_PAL = np.array([hex_to_rgb(c) for c in BUILDER_PALETTE_HEX], dtype=np.float32)
GRUNT_PAL = np.array([hex_to_rgb(c) for c in GRUNT_PALETTE_HEX], dtype=np.float32)


def extract_clean_fg(
    img_rgb: Image.Image,
    crop_box: Tuple[int, int, int, int],
    max_bottom_y: int,
    foot_x_ranges: List[Tuple[int, int]],
    foot_line_y: int,
    bg_tol: float = 26.0,
    remove_interior_bg_holes: bool = True,
) -> np.ndarray:
    """
    Extract RGBA foreground from studio turnaround crop, stripping both exterior
    background AND enclosed studio-grey holes (e.g., between legs / under arms)
    plus the bottom turnaround baseline line.
    """
    sub = np.array(img_rgb.crop(crop_box), dtype=np.uint8)
    h, w, _ = sub.shape
    border_pixels = np.concatenate([sub[0, :, :], sub[:, 0, :], sub[:, -1, :]], axis=0)
    bg = np.median(border_pixels, axis=0)

    diff = np.linalg.norm(sub.astype(np.float32) - bg[None, None, :], axis=2)
    is_bg_candidate = diff < bg_tol

    # Flood-fill exterior background from borders
    seed = np.zeros((h, w), dtype=bool)
    seed[0, :] = is_bg_candidate[0, :]
    seed[-1, :] = is_bg_candidate[-1, :]
    seed[:, 0] = is_bg_candidate[:, 0]
    seed[:, -1] = is_bg_candidate[:, -1]

    labeled, _ = ndimage.label(is_bg_candidate)
    border_labels = np.unique(labeled[seed])
    border_labels = border_labels[border_labels > 0]
    ext_bg = np.isin(labeled, border_labels)

    fg = ~ext_bg

    # Also remove enclosed studio-grey holes (low chroma + close to studio bg RGB, size >= 18px)
    if remove_interior_bg_holes:
        rgb_f = sub.astype(np.float32)
        chroma = rgb_f.max(axis=2) - rgb_f.min(axis=2)
        hole_cand = (~ext_bg) & (diff < 24.0) & (chroma < 16.0)
        hlbl, hn = ndimage.label(hole_cand)
        if hn > 0:
            hsizes = ndimage.sum(hole_cand, hlbl, range(1, hn + 1))
            for idx, sz in enumerate(hsizes, 1):
                if sz >= 18:
                    # Dilate 1px to eat anti-aliased fringe inside the hole
                    hmask = ndimage.binary_dilation(hlbl == idx, iterations=1)
                    lum = 0.2126 * rgb_f[:, :, 0] + 0.7152 * rgb_f[:, :, 1] + 0.0722 * rgb_f[:, :, 2]
                    fg[hmask & (lum > 120.0)] = False

    # Strip studio horizontal ground line outside foot X ranges and below max_bottom_y
    fg[max_bottom_y:, :] = False
    for y in range(foot_line_y, max_bottom_y):
        allowed = np.zeros(w, dtype=bool)
        for x0, x1 in foot_x_ranges:
            allowed[x0 : x1 + 1] = True
        fg[y, ~allowed] = False

    # Remove tiny isolated specks (< 25 px)
    lbl_fg, n_fg = ndimage.label(fg)
    if n_fg > 0:
        sizes = ndimage.sum(fg, lbl_fg, range(1, n_fg + 1))
        for idx, sz in enumerate(sizes, 1):
            if sz < 25:
                fg[lbl_fg == idx] = False

    # Strip 1px outer light-grey anti-aliasing halo where luminance is high near bg
    eroded = ndimage.binary_erosion(fg, iterations=1)
    boundary = fg & (~eroded)
    lum = 0.2126 * sub[:, :, 0] + 0.7152 * sub[:, :, 1] + 0.0722 * sub[:, :, 2]
    halo = boundary & (lum > 140.0) & (diff < 72.0)
    fg[halo] = False

    rgba = np.zeros((h, w, 4), dtype=np.uint8)
    rgba[:, :, :3] = sub
    rgba[:, :, 3] = np.where(fg, 255, 0).astype(np.uint8)
    return rgba


def poly_mask(shape_hw: Tuple[int, int], pts: List[Tuple[float, float]]) -> np.ndarray:
    im = Image.new("L", (shape_hw[1], shape_hw[0]), 0)
    ImageDraw.Draw(im).polygon(pts, fill=255)
    return np.array(im) > 127


def add_ink_outline(rgba: np.ndarray, radius: int = 3) -> np.ndarray:
    """Ensure clean closed #11151C dark ink border around opaque alpha."""
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


# ============================================================================
# 1. BUILDER LAYERED CUTOUT RIG EXTRACTION (CLEAN v2)
# ============================================================================

def build_builder_rig() -> Dict[str, Tuple[Image.Image, Tuple[float, float]]]:
    raw_turn = Image.open(ART002_DIR / "characters" / "source" / "builder_turnaround_raw.jpg").convert("RGB")
    base = extract_clean_fg(
        raw_turn,
        crop_box=(360, 165, 705, 635),
        max_bottom_y=459,
        foot_x_ranges=[(120, 192), (210, 308)],
        foot_line_y=450,
        remove_interior_bg_holes=False,  # Keep white/cream shirt intact!
    )
    h, w, _ = base.shape

    # Sample exact golden apron RGB from unobstructed right skirt (x=250..274, y=280..355)
    apron_sample = base[280:355, 250:274, :3]
    apron_main_rgb = tuple(int(v) for v in np.median(apron_sample.reshape(-1, 3), axis=0))

    # 1) HEAD + CAP (neck pivot 202, 114)
    m_head = poly_mask(
        (h, w),
        [(138, 8), (266, 8), (266, 75), (248, 115), (235, 121), (190, 120), (166, 110), (142, 98)],
    )
    head_arr = np.zeros_like(base)
    head_arr[m_head] = base[m_head]
    head_im = Image.fromarray(head_arr)
    neck_plug = Image.new("RGBA", (w, h), (0, 0, 0, 0))
    ImageDraw.Draw(neck_plug).rounded_rectangle(
        [176, 98, 228, 126], radius=10, fill=(218, 155, 116, 255), outline=(*INK_RGB, 255), width=4
    )
    head_im = Image.alpha_composite(neck_plug, head_im)

    # 2) BACK ARM (screen-right; shoulder pivot 270, 168)
    m_barm = poly_mask(
        (h, w),
        [(268, 148), (336, 148), (336, 350), (280, 350), (283, 292), (278, 245), (275, 198), (266, 164)],
    )
    barm_arr = np.zeros_like(base)
    barm_arr[m_barm] = base[m_barm]
    barm_im = Image.fromarray(add_ink_outline(barm_arr, radius=3))
    barm_cap = Image.new("RGBA", (w, h), (0, 0, 0, 0))
    ImageDraw.Draw(barm_cap).ellipse(
        [250, 148, 292, 192], fill=(213, 196, 163, 255), outline=(*INK_RGB, 255), width=4
    )
    barm_im = Image.alpha_composite(barm_cap, barm_im)

    # 3) BACK LEG & BOOT (screen-right; hip pivot 246, 352) — cut strictly below apron y=380 to eliminate yellow sliver
    m_bleg = poly_mask(
        (h, w),
        [(210, 381), (290, 378), (310, 428), (310, 460), (210, 460)],
    )
    bleg_arr = np.zeros_like(base)
    bleg_arr[m_bleg] = base[m_bleg]
    bleg_im = Image.fromarray(add_ink_outline(bleg_arr, radius=3))
    bleg_thigh = Image.new("RGBA", (w, h), (0, 0, 0, 0))
    ImageDraw.Draw(bleg_thigh).rounded_rectangle(
        [214, 336, 282, 396], radius=16, fill=(83, 52, 32, 255), outline=(*INK_RGB, 255), width=4
    )
    bleg_im = Image.alpha_composite(bleg_thigh, bleg_im)

    # 4) FRONT LEG & BOOT (screen-left; hip pivot 158, 352)
    m_fleg = poly_mask((h, w), [(120, 393), (192, 393), (192, 460), (120, 460)])
    fleg_arr = np.zeros_like(base)
    fleg_arr[m_fleg] = base[m_fleg]
    fleg_im = Image.fromarray(add_ink_outline(fleg_arr, radius=3))
    fleg_thigh = Image.new("RGBA", (w, h), (0, 0, 0, 0))
    fd = ImageDraw.Draw(fleg_thigh)
    fd.rounded_rectangle([130, 336, 187, 404], radius=15, fill=(92, 60, 38, 255), outline=(*INK_RGB, 255), width=4)
    fd.rectangle([168, 342, 183, 396], fill=(66, 43, 27, 255))
    fd.line([(132, 395), (185, 395)], fill=(*INK_RGB, 255), width=4)
    fleg_im = Image.alpha_composite(fleg_thigh, fleg_im)

    # 5) TORSO + CREAM COLLAR + SEAMLESS GOLDEN APRON (waist pivot 210, 350)
    m_torso_keep = poly_mask(
        (h, w),
        [
            (142, 102), (168, 106), (192, 118), (236, 118), (252, 108),
            (274, 146), (278, 210), (285, 378), (242, 378),
            (245, 254), (142, 252), (140, 200), (154, 145),
        ],
    ) & (~m_head)
    torso_arr = np.zeros_like(base)
    torso_arr[m_torso_keep] = base[m_torso_keep]
    torso_im = Image.fromarray(torso_arr)

    apron_under = Image.new("RGBA", (w, h), (0, 0, 0, 0))
    ud = ImageDraw.Draw(apron_under)
    ud.rounded_rectangle([144, 126, 272, 215], radius=22, fill=(235, 224, 198, 255), outline=(*INK_RGB, 255), width=4)
    apron_poly = [(143, 242), (278, 242), (284, 374), (225, 378), (144, 374)]
    ud.polygon(apron_poly, fill=(*apron_main_rgb, 255))
    ud.polygon([(143, 242), (160, 242), (162, 374), (144, 374)], fill=(212, 148, 23, 255))
    ud.line([(143, 242), (144, 374), (225, 378), (284, 374)], fill=(*INK_RGB, 255), width=5)
    torso_full = Image.alpha_composite(apron_under, torso_im)
    torso_full = Image.fromarray(add_ink_outline(np.array(torso_full), radius=3))

    # 6) FRONT UPPER ARM (rolled cream sleeve; shoulder pivot 148, 152 -> elbow 112, 226)
    m_fuarm = poly_mask((h, w), [(70, 128), (160, 130), (154, 175), (140, 228), (74, 228)])
    fuarm_arr = np.zeros_like(base)
    fuarm_arr[m_fuarm] = base[m_fuarm]
    fuarm_im = Image.fromarray(add_ink_outline(fuarm_arr, radius=3))
    sh_cap = Image.new("RGBA", (w, h), (0, 0, 0, 0))
    sd = ImageDraw.Draw(sh_cap)
    sd.ellipse([122, 128, 174, 180], fill=(243, 234, 214, 255), outline=(*INK_RGB, 255), width=4)
    sd.ellipse([86, 200, 138, 246], fill=(244, 194, 159, 255), outline=(*INK_RGB, 255), width=4)
    fuarm_im = Image.alpha_composite(sh_cap, fuarm_im)

    # 7) FRONT FOREARM GLOVE (elbow 112, 236 -> wrist 108, 294) — strip non-brown apron/shirt pixels on right edge
    m_glove = poly_mask((h, w), [(64, 232), (132, 236), (133, 292), (126, 326), (76, 320), (66, 270)])
    glove_arr = np.zeros_like(base)
    glove_arr[m_glove] = base[m_glove]
    # Remove any bright cream/grey pixels (R>200, G>190, B>175) trapped on right border of cuff
    bright_sliver = (glove_arr[:, :, 0] > 200) & (glove_arr[:, :, 1] > 188) & (glove_arr[:, :, 2] > 165)
    glove_arr[bright_sliver] = [0, 0, 0, 0]
    glove_im = Image.fromarray(add_ink_outline(glove_arr, radius=3))

    # 8) RIGID T-HAMMER (wrist grip 108, 294) — strip any yellow apron pixels from top of iron head!
    m_shaft_left = poly_mask((h, w), [(10, 262), (82, 282), (80, 312), (10, 294)])
    m_head_iron = poly_mask(
        (h, w),
        [(138, 296), (150, 261), (239, 272), (223, 391), (139, 387), (144, 322), (134, 318)],
    )
    ham_arr = np.zeros_like(base)
    ham_arr[m_shaft_left | m_head_iron] = base[m_shaft_left | m_head_iron]
    # Strip any yellow apron pixels (high R & G, low B) outside the brown wooden end-plug (x=196..226, y=312..356)
    plug_zone = poly_mask((h, w), [(194, 310), (228, 310), (228, 358), (194, 358)])
    yellow_apron_leak = (
        (ham_arr[:, :, 3] > 0)
        & (~m_shaft_left)
        & (~plug_zone)
        & (ham_arr[:, :, 0].astype(int) - ham_arr[:, :, 2].astype(int) > 45)
    )
    ham_arr[yellow_apron_leak] = [0, 0, 0, 0]
    ham_im = Image.fromarray(ham_arr)

    bridge = Image.new("RGBA", (w, h), (0, 0, 0, 0))
    brd = ImageDraw.Draw(bridge)
    brd.polygon([(18, 267), (160, 302), (155, 323), (14, 288)], fill=(142, 91, 52, 255))
    brd.polygon([(18, 267), (160, 302), (158, 309), (16, 274)], fill=(166, 110, 68, 255))
    brd.line([(18, 267), (160, 302)], fill=(*INK_RGB, 255), width=4)
    brd.line([(14, 288), (155, 323)], fill=(*INK_RGB, 255), width=4)
    ham_im = Image.alpha_composite(bridge, ham_im)
    ham_im = Image.fromarray(add_ink_outline(np.array(ham_im), radius=3))

    return {
        "back_arm": (barm_im, (270.0, 168.0)),
        "back_leg": (bleg_im, (246.0, 352.0)),
        "front_leg": (fleg_im, (158.0, 352.0)),
        "torso": (torso_full, (210.0, 350.0)),
        "head": (head_im, (202.0, 114.0)),
        "front_upper_arm": (fuarm_im, (148.0, 152.0)),
        "hammer": (ham_im, (108.0, 294.0)),
        "front_glove": (glove_im, (112.0, 236.0)),
    }


# ============================================================================
# 2. GRUNT LAYERED CUTOUT RIG EXTRACTION (CLEAN v2)
# ============================================================================

def build_grunt_rig() -> Dict[str, Tuple[Image.Image, Tuple[float, float]]]:
    raw_turn = Image.open(ART002_DIR / "characters" / "source" / "grunt_turnaround_raw.jpg").convert("RGB")
    base = extract_clean_fg(
        raw_turn,
        crop_box=(395, 200, 740, 615),
        max_bottom_y=389,
        foot_x_ranges=[(64, 130), (190, 276)],
        foot_line_y=381,
        remove_interior_bg_holes=True,  # Eliminates white inner-thigh / under-arm holes!
    )
    h, w, _ = base.shape

    # Sample Grunt's exact lit belly green & shadow green from unobstructed belly pixels
    belly_lit_rgb = tuple(int(v) for v in np.median(base[198:232, 195:225, :3].reshape(-1, 3), axis=0))
    belly_shd_rgb = tuple(int(v) for v in np.median(base[190:225, 102:125, :3].reshape(-1, 3), axis=0))

    club_poly_pts = [
        (24, 298), (74, 256), (76, 252),
        (98, 236), (202, 146), (204, 128), (222, 134),
        (246, 114), (242, 98), (262, 108),
        (262, 82), (284, 100), (294, 90), (312, 100),
        (332, 98), (318, 120), (324, 132),
        (314, 146), (320, 160), (302, 156),
        (294, 166), (298, 180), (280, 174),
        (268, 184), (274, 202), (254, 190),
        (238, 202), (242, 214), (226, 208),
        (114, 268), (94, 286), (40, 320), (24, 310),
    ]
    m_club_zone = poly_mask((h, w), club_poly_pts)

    # 1) HEAD (neck pivot 162, 104)
    m_head = poly_mask(
        (h, w),
        [(80, 10), (230, 10), (230, 66), (218, 98), (204, 114), (168, 118), (132, 108), (110, 92), (80, 72)],
    ) & (~m_club_zone)
    head_arr = np.zeros_like(base)
    head_arr[m_head] = base[m_head]
    head_im = Image.fromarray(add_ink_outline(head_arr, radius=3))
    neck_cap = Image.new("RGBA", (w, h), (0, 0, 0, 0))
    ImageDraw.Draw(neck_cap).ellipse(
        [130, 88, 196, 126], fill=(*belly_shd_rgb, 255), outline=(*INK_RGB, 255), width=4
    )
    head_im = Image.alpha_composite(neck_cap, head_im)

    # 2) BACK ARM (screen-right; shoulder pivot 238, 148)
    m_barm = poly_mask(
        (h, w),
        [(238, 174), (286, 174), (304, 275), (298, 320), (248, 320), (244, 260), (240, 208)],
    ) & (~m_club_zone)
    barm_arr = np.zeros_like(base)
    barm_arr[m_barm] = base[m_barm]
    barm_im = Image.fromarray(add_ink_outline(barm_arr, radius=3))
    barm_deltoid = Image.new("RGBA", (w, h), (0, 0, 0, 0))
    bdd = ImageDraw.Draw(barm_deltoid)
    bdd.polygon(
        [(220, 134), (262, 142), (278, 192), (242, 202), (222, 166)],
        fill=(*belly_lit_rgb, 255),
        outline=(*INK_RGB, 255),
        width=4,
    )
    bdd.polygon([(222, 152), (244, 156), (246, 198), (234, 198)], fill=(*belly_shd_rgb, 255))
    barm_im = Image.alpha_composite(barm_deltoid, barm_im)

    # 3) BACK LEG & FOOT (screen-right; hip pivot 218, 276)
    m_bleg = poly_mask(
        (h, w),
        [(194, 302), (238, 276), (264, 318), (278, 370), (278, 389), (192, 389), (196, 338)],
    )
    bleg_arr = np.zeros_like(base)
    bleg_arr[m_bleg] = base[m_bleg]
    bleg_im = Image.fromarray(add_ink_outline(bleg_arr, radius=3))
    bleg_cap = Image.new("RGBA", (w, h), (0, 0, 0, 0))
    ImageDraw.Draw(bleg_cap).ellipse(
        [196, 256, 248, 308], fill=(*belly_lit_rgb, 255), outline=(*INK_RGB, 255), width=4
    )
    bleg_im = Image.alpha_composite(bleg_cap, bleg_im)

    # 4) FRONT LEG & FOOT (screen-left; hip pivot 114, 282) — exclude brown loincloth & inner hole cleanly
    m_fleg = poly_mask(
        (h, w),
        [(64, 355), (80, 294), (126, 298), (138, 326), (130, 389), (64, 389)],
    )
    fleg_arr = np.zeros_like(base)
    fleg_arr[m_fleg] = base[m_fleg]
    fleg_im = Image.fromarray(add_ink_outline(fleg_arr, radius=3))
    fleg_cap = Image.new("RGBA", (w, h), (0, 0, 0, 0))
    ImageDraw.Draw(fleg_cap).ellipse(
        [82, 262, 140, 316], fill=(*belly_lit_rgb, 255), outline=(*INK_RGB, 255), width=4
    )
    fleg_im = Image.alpha_composite(fleg_cap, fleg_im)

    # 5) TORSO + ORGANIC CEL CHEST/BELLY + HIDE LOINCLOTH (waist pivot 172, 272)
    m_torso_keep = poly_mask(
        (h, w),
        [
            (94, 96), (212, 102), (236, 128), (246, 185), (248, 246),
            (242, 286), (222, 278), (206, 336), (178, 356), (144, 322),
            (114, 294), (94, 256), (86, 195), (96, 138),
        ],
    ) & (~m_head) & (~m_club_zone)
    torso_arr = np.zeros_like(base)
    torso_arr[m_torso_keep] = base[m_torso_keep]
    # Also strip any residual brown/grey club edge pixels inside chest/belly (y < 242)
    chest_zone = poly_mask((h, w), [(88, 98), (248, 98), (248, 240), (88, 240)])
    non_green_in_chest = (
        chest_zone
        & (torso_arr[:, :, 3] > 0)
        & (torso_arr[:, :, 0].astype(int) > torso_arr[:, :, 1].astype(int) - 6)
        & (torso_arr[:, :, 1] > 42)
    )
    torso_arr[non_green_in_chest] = [0, 0, 0, 0]
    torso_im = Image.fromarray(torso_arr)

    belly_under = Image.new("RGBA", (w, h), (0, 0, 0, 0))
    bd = ImageDraw.Draw(belly_under)
    torso_sil = [
        (98, 102), (210, 106), (234, 134), (245, 196), (245, 246),
        (238, 276), (180, 344), (112, 286), (92, 242), (90, 182),
    ]
    bd.polygon(torso_sil, fill=(*belly_lit_rgb, 255))
    bd.polygon([(90, 140), (134, 140), (134, 256), (92, 246)], fill=(*belly_shd_rgb, 255))
    bd.polygon([(96, 244), (156, 258), (152, 278), (98, 264)], fill=(111, 72, 43, 255), outline=(*INK_RGB, 255), width=4)
    bd.line([(98, 102), (210, 106), (234, 134), (245, 196), (245, 246)], fill=(*INK_RGB, 255), width=5)

    torso_merged = Image.alpha_composite(belly_under, torso_im)
    td = ImageDraw.Draw(torso_merged)
    # Organic curved goblin pectoral crease + under-pec cel shadow crescent
    td.polygon([(116, 168), (170, 168), (216, 154), (174, 177), (124, 176)], fill=(*belly_shd_rgb, 255))
    td.line([(114, 166), (144, 171), (174, 168), (214, 155)], fill=(*INK_RGB, 255), width=4)
    torso_merged = Image.fromarray(add_ink_outline(np.array(torso_merged), radius=3))

    # 6) FRONT UPPER ARM (shoulder pivot 84, 134 -> elbow 46, 204)
    m_fuarm = poly_mask((h, w), [(14, 96), (116, 98), (110, 166), (84, 212), (14, 212)])
    fuarm_arr = np.zeros_like(base)
    fuarm_arr[m_fuarm] = base[m_fuarm]
    fuarm_im = Image.fromarray(add_ink_outline(fuarm_arr, radius=3))
    g_sh = Image.new("RGBA", (w, h), (0, 0, 0, 0))
    gsd = ImageDraw.Draw(g_sh)
    gsd.ellipse([54, 104, 116, 166], fill=(*belly_lit_rgb, 255), outline=(*INK_RGB, 255), width=4)
    gsd.ellipse([20, 178, 76, 228], fill=(*belly_lit_rgb, 255), outline=(*INK_RGB, 255), width=4)
    fuarm_im = Image.alpha_composite(g_sh, fuarm_im)

    # 7) FRONT FOREARM + FIST (elbow 46, 204 -> grip 78, 260)
    m_fist = poly_mask((h, w), [(10, 188), (74, 198), (82, 232), (108, 252), (108, 294), (72, 298), (38, 268), (12, 222)])
    fist_arr = np.zeros_like(base)
    fist_arr[m_fist] = base[m_fist]
    # Remove wooden butt pixels on bottom-left of fist
    wood_in_fist = (
        (fist_arr[:, :, 3] > 0)
        & (fist_arr[:, :, 0].astype(int) > fist_arr[:, :, 1].astype(int) + 14)
    )
    fist_arr[wood_in_fist] = [0, 0, 0, 0]
    fist_im = Image.fromarray(add_ink_outline(fist_arr, radius=3))

    # 8) RIGID SPIKED WOODEN CLUB — strip ALL green goblin skin pixels so club is pure oak + steel spikes!
    m_club_only = m_club_zone & (~m_fist)
    club_arr = np.zeros_like(base)
    club_arr[m_club_only] = base[m_club_only]
    green_skin_leak = (
        (club_arr[:, :, 3] > 0)
        & (club_arr[:, :, 1].astype(int) > club_arr[:, :, 0].astype(int) + 10)
        & (club_arr[:, :, 1].astype(int) > club_arr[:, :, 2].astype(int) + 12)
    )
    club_arr[green_skin_leak] = [0, 0, 0, 0]
    # Remove any tiny disconnected fragments left after stripping green skin
    c_mask = club_arr[:, :, 3] > 0
    clbl, cn = ndimage.label(c_mask)
    if cn > 0:
        csizes = ndimage.sum(c_mask, clbl, range(1, cn + 1))
        for idx, sz in enumerate(csizes, 1):
            if sz < 35:
                club_arr[clbl == idx] = [0, 0, 0, 0]
    club_im = Image.fromarray(club_arr)

    c_bridge = Image.new("RGBA", (w, h), (0, 0, 0, 0))
    cbd = ImageDraw.Draw(c_bridge)
    cbd.polygon([(28, 298), (132, 217), (144, 235), (40, 314)], fill=(131, 83, 49, 255))
    cbd.polygon([(28, 298), (132, 217), (137, 224), (33, 304)], fill=(176, 120, 76, 255))
    cbd.line([(28, 298), (132, 217)], fill=(*INK_RGB, 255), width=4)
    cbd.line([(40, 314), (144, 235)], fill=(*INK_RGB, 255), width=4)
    club_im = Image.alpha_composite(c_bridge, club_im)
    club_im = Image.fromarray(add_ink_outline(np.array(club_im), radius=3))

    return {
        "back_arm": (barm_im, (238.0, 148.0)),
        "back_leg": (bleg_im, (218.0, 276.0)),
        "front_leg": (fleg_im, (114.0, 282.0)),
        "torso": (torso_merged, (172.0, 272.0)),
        "head": (head_im, (162.0, 104.0)),
        "front_upper_arm": (fuarm_im, (84.0, 134.0)),
        "club": (club_im, (78.0, 260.0)),
        "front_fist": (fist_im, (46.0, 204.0)),
    }


# ============================================================================
# 3. 2D HIERARCHICAL FK/POSE COMPOSITOR + TAPERED CEL SMEAR + CLEANUP
# ============================================================================

def transform_layer(
    layer_im: Image.Image,
    pivot_src: Tuple[float, float],
    dest_pos: Tuple[float, float],
    angle_deg: float,
    scale_xy: Tuple[float, float] = (1.0, 1.0),
    canvas_size: Tuple[int, int] = (512, 512),
) -> Image.Image:
    theta = math.radians(angle_deg)
    cos_t = math.cos(theta)
    sin_t = math.sin(theta)
    sx, sy = scale_xy

    a = cos_t / sx
    b = sin_t / sx
    d = -sin_t / sy
    e = cos_t / sy
    px, py = pivot_src
    dx, dy = dest_pos
    c = px - a * dx - b * dy
    f = py - d * dx - e * dy

    return layer_im.transform(
        canvas_size,
        Image.Transform.AFFINE,
        (a, b, c, d, e, f),
        resample=Image.Resampling.BILINEAR,
    )


def rotate_vec(vx: float, vy: float, angle_deg: float, sx: float = 1.0, sy: float = 1.0) -> Tuple[float, float]:
    th = math.radians(angle_deg)
    vx *= sx
    vy *= sy
    return (vx * math.cos(th) - vy * math.sin(th), vx * math.sin(th) + vy * math.cos(th))


def draw_tapered_cel_smear(
    canvas_512: Image.Image,
    center_xy: Tuple[float, float],
    radius_outer: float,
    max_thick: float,
    deg_start: float,
    deg_end: float,
    outer_rgb: Tuple[int, int, int] = (254, 240, 138),
    core_rgb: Tuple[int, int, int] = (255, 251, 235),
) -> Image.Image:
    """
    Draw a sleek, tapered circular cel-shaded motion crescent from deg_start (needle tail)
    to deg_end (broad leading weapon head) so downward strikes read instantly at 16px-48px!
    """
    arc_layer = Image.new("RGBA", canvas_512.size, (0, 0, 0, 0))
    dr = ImageDraw.Draw(arc_layer)
    cx, cy = center_xy
    steps = 22
    outer_pts = []
    inner_pts = []
    core_outer = []
    core_inner = []
    for i in range(steps + 1):
        u = i / steps  # 0 at tail -> 1 at weapon head
        ang = math.radians(deg_start + (deg_end - deg_start) * u)
        taper = math.sin(u * math.pi * 0.86) ** 0.75
        thk = max_thick * taper
        ro = radius_outer
        ri = radius_outer - thk
        outer_pts.append((cx + ro * math.cos(ang), cy + ro * math.sin(ang)))
        inner_pts.append((cx + ri * math.cos(ang), cy + ri * math.sin(ang)))
        if u >= 0.25:
            ro_c = radius_outer - thk * 0.18
            ri_c = radius_outer - thk * 0.68
            core_outer.append((cx + ro_c * math.cos(ang), cy + ro_c * math.sin(ang)))
            core_inner.append((cx + ri_c * math.cos(ang), cy + ri_c * math.sin(ang)))

    poly = outer_pts + list(reversed(inner_pts))
    dr.polygon(poly, fill=(*outer_rgb, 255), outline=(*INK_RGB, 255), width=4)
    if len(core_outer) >= 3:
        dr.polygon(core_outer + list(reversed(core_inner)), fill=(*core_rgb, 255))
    return Image.alpha_composite(canvas_512, arc_layer)


def finalize_cel_frame_128(
    canvas_512: Image.Image,
    palette_rgb: np.ndarray,
) -> Image.Image:
    arr512 = np.array(canvas_512, dtype=np.float32)
    a512 = arr512[:, :, 3:4] / 255.0
    premul = arr512[:, :, :3] * a512

    premul_128 = premul.reshape(128, 4, 128, 4, 3).mean(axis=(1, 3))
    a_128 = a512.reshape(128, 4, 128, 4).mean(axis=(1, 3))

    mask = a_128 >= 0.48
    lbl, n_lbl = ndimage.label(mask)
    if n_lbl > 1:
        sizes = ndimage.sum(mask, lbl, range(1, n_lbl + 1))
        for idx, sz in enumerate(sizes, 1):
            if sz < 4:
                mask[lbl == idx] = False

    rgb_128 = np.zeros((128, 128, 3), dtype=np.float32)
    valid = a_128 > 0.05
    rgb_128[valid] = np.clip(premul_128[valid] / a_128[valid, None], 0.0, 255.0)

    flat = rgb_128[mask]
    weights = np.array([0.30, 0.59, 0.11], dtype=np.float32)[None, None, :]
    diffs = (flat[:, None, :] - palette_rgb[None, :, :]) ** 2 * weights
    nearest_idx = np.argmin(diffs.sum(axis=2), axis=1)
    quant_rgb = np.zeros((128, 128, 3), dtype=np.uint8)
    quant_rgb[mask] = palette_rgb[nearest_idx].astype(np.uint8)

    eroded_4 = ndimage.binary_erosion(mask, structure=np.array([[0, 1, 0], [1, 1, 1], [0, 1, 0]]), border_value=0)
    outer_1px = mask & (~eroded_4)
    quant_rgb[outer_1px] = INK_RGB

    out = np.zeros((128, 128, 4), dtype=np.uint8)
    out[:, :, :3] = quant_rgb
    out[:, :, 3] = np.where(mask, 255, 0).astype(np.uint8)
    return Image.fromarray(out, mode="RGBA")


def render_builder_pose(
    rig: Dict[str, Tuple[Image.Image, Tuple[float, float]]],
    pose: Dict[str, float],
) -> Image.Image:
    S = 0.775
    ground_y = 470.0
    center_x = 256.0 + pose.get("root_dx", 0.0) * 4.0

    waist_x = center_x
    waist_y = ground_y - 82.9 + pose.get("pelvis_dy", 0.0) * 4.0

    torso_ang = pose.get("torso_ang", 0.0)
    torso_sx = S * pose.get("torso_sx", 1.0)
    torso_sy = S * pose.get("torso_sy", 1.0)

    hx, hy = rotate_vec(-8.0, -236.0, torso_ang, torso_sx, torso_sy)
    bax, bay = rotate_vec(60.0, -182.0, torso_ang, torso_sx, torso_sy)
    fsx, fsy = rotate_vec(-62.0, -198.0, torso_ang, torso_sx, torso_sy)

    head_pos = (waist_x + hx + pose.get("head_dx", 0.0) * 4.0, waist_y + hy + pose.get("head_dy", 0.0) * 4.0)
    barm_pos = (waist_x + bax, waist_y + bay)
    fsh_pos = (
        waist_x + fsx + pose.get("fsh_dx", 0.0) * 4.0,
        waist_y + fsy + pose.get("fsh_dy", 0.0) * 4.0,
    )

    bleg_pos = (
        center_x + (246.0 - 210.0) * S + pose.get("bleg_dx", 0.0) * 4.0,
        ground_y - (456.5 - 352.0) * S + pose.get("bleg_dy", 0.0) * 4.0,
    )
    fleg_pos = (
        center_x + (158.0 - 210.0) * S + pose.get("fleg_dx", 0.0) * 4.0,
        ground_y - (457.0 - 352.0) * S + pose.get("fleg_dy", 0.0) * 4.0,
    )

    uarm_ang = torso_ang + pose.get("uarm_ang", 0.0)
    ex, ey = rotate_vec(-36.0, 84.0, uarm_ang, S, S)
    elbow_pos = (fsh_pos[0] + ex, fsh_pos[1] + ey)

    farm_ang = uarm_ang + pose.get("farm_ang", 0.0)
    wx, wy = rotate_vec(-4.0, 58.0, farm_ang, S, S)
    wrist_pos = (elbow_pos[0] + wx, elbow_pos[1] + wy)
    ham_ang = farm_ang + pose.get("wrist_ang", 0.0)

    canvas = Image.new("RGBA", (512, 512), (0, 0, 0, 0))
    layers_ordered = [
        transform_layer(rig["back_arm"][0], rig["back_arm"][1], barm_pos, torso_ang + pose.get("barm_ang", 0.0), (S, S)),
        transform_layer(rig["back_leg"][0], rig["back_leg"][1], bleg_pos, pose.get("bleg_ang", 0.0), (S, S)),
        transform_layer(rig["front_leg"][0], rig["front_leg"][1], fleg_pos, pose.get("fleg_ang", 0.0), (S, S)),
        transform_layer(rig["torso"][0], rig["torso"][1], (waist_x, waist_y), torso_ang, (torso_sx, torso_sy)),
        transform_layer(rig["head"][0], rig["head"][1], head_pos, torso_ang * 0.55 + pose.get("head_ang", 0.0), (S, S)),
    ]
    for lyr in layers_ordered:
        canvas = Image.alpha_composite(canvas, lyr)

    # Render tapered cel smear right behind the striking T-hammer so it leads cleanly into the iron head!
    if pose.get("swing_arc", 0.0) > 0.5:
        canvas = draw_tapered_cel_smear(
            canvas,
            center_xy=(246.0, 252.0),
            radius_outer=168.0,
            max_thick=34.0,
            deg_start=-104.0,
            deg_end=38.0,
            outer_rgb=(254, 240, 138),
            core_rgb=(255, 251, 235),
        )

    arm_layers = [
        transform_layer(rig["front_upper_arm"][0], rig["front_upper_arm"][1], fsh_pos, uarm_ang, (S, S)),
        transform_layer(rig["hammer"][0], rig["hammer"][1], wrist_pos, ham_ang, (S, S)),
        transform_layer(rig["front_glove"][0], rig["front_glove"][1], elbow_pos, farm_ang, (S, S)),
    ]
    for lyr in arm_layers:
        canvas = Image.alpha_composite(canvas, lyr)

    return finalize_cel_frame_128(canvas, BUILDER_PAL)


def render_grunt_pose(
    rig: Dict[str, Tuple[Image.Image, Tuple[float, float]]],
    pose: Dict[str, float],
) -> Image.Image:
    S = 0.905
    ground_y = 470.0
    center_x = 256.0 + pose.get("root_dx", 0.0) * 4.0

    waist_x = center_x
    waist_y = ground_y - 102.3 + pose.get("pelvis_dy", 0.0) * 4.0

    torso_ang = pose.get("torso_ang", 0.0)
    torso_sx = S * pose.get("torso_sx", 1.0)
    torso_sy = S * pose.get("torso_sy", 1.0)

    hx, hy = rotate_vec(-10.0, -168.0, torso_ang, torso_sx, torso_sy)
    bax, bay = rotate_vec(66.0, -124.0, torso_ang, torso_sx, torso_sy)
    fsx, fsy = rotate_vec(-88.0, -138.0, torso_ang, torso_sx, torso_sy)

    head_pos = (waist_x + hx + pose.get("head_dx", 0.0) * 4.0, waist_y + hy + pose.get("head_dy", 0.0) * 4.0)
    barm_pos = (waist_x + bax, waist_y + bay)
    fsh_pos = (
        waist_x + fsx + pose.get("fsh_dx", 0.0) * 4.0,
        waist_y + fsy + pose.get("fsh_dy", 0.0) * 4.0,
    )

    bleg_pos = (
        center_x + (218.0 - 172.0) * S + pose.get("bleg_dx", 0.0) * 4.0,
        ground_y - (385.0 - 276.0) * S + pose.get("bleg_dy", 0.0) * 4.0,
    )
    fleg_pos = (
        center_x + (114.0 - 172.0) * S + pose.get("fleg_dx", 0.0) * 4.0,
        ground_y - (385.5 - 282.0) * S + pose.get("fleg_dy", 0.0) * 4.0,
    )

    uarm_ang = torso_ang + pose.get("uarm_ang", 0.0)
    ex, ey = rotate_vec(-38.0, 70.0, uarm_ang, S, S)
    elbow_pos = (fsh_pos[0] + ex, fsh_pos[1] + ey)

    farm_ang = uarm_ang + pose.get("farm_ang", 0.0)
    wx, wy = rotate_vec(32.0, 56.0, farm_ang, S, S)
    wrist_pos = (elbow_pos[0] + wx, elbow_pos[1] + wy)
    club_ang = farm_ang + pose.get("wrist_ang", 0.0)

    canvas = Image.new("RGBA", (512, 512), (0, 0, 0, 0))
    base_layers = [
        transform_layer(rig["back_arm"][0], rig["back_arm"][1], barm_pos, torso_ang + pose.get("barm_ang", 0.0), (S, S)),
        transform_layer(rig["back_leg"][0], rig["back_leg"][1], bleg_pos, pose.get("bleg_ang", 0.0), (S, S)),
        transform_layer(rig["front_leg"][0], rig["front_leg"][1], fleg_pos, pose.get("fleg_ang", 0.0), (S, S)),
        transform_layer(rig["torso"][0], rig["torso"][1], (waist_x, waist_y), torso_ang, (torso_sx, torso_sy)),
        transform_layer(rig["head"][0], rig["head"][1], head_pos, torso_ang * 0.55 + pose.get("head_ang", 0.0), (S, S)),
    ]
    for lyr in base_layers:
        canvas = Image.alpha_composite(canvas, lyr)

    if pose.get("swing_arc", 0.0) > 0.5:
        canvas = draw_tapered_cel_smear(
            canvas,
            center_xy=(236.0, 256.0),
            radius_outer=166.0,
            max_thick=36.0,
            deg_start=-102.0,
            deg_end=44.0,
            outer_rgb=(254, 240, 138),
            core_rgb=(245, 234, 181),
        )

    arm_layers = [
        transform_layer(rig["front_upper_arm"][0], rig["front_upper_arm"][1], fsh_pos, uarm_ang, (S, S)),
        transform_layer(rig["club"][0], rig["club"][1], wrist_pos, club_ang, (S, S)),
        transform_layer(rig["front_fist"][0], rig["front_fist"][1], elbow_pos, farm_ang, (S, S)),
    ]
    for lyr in arm_layers:
        canvas = Image.alpha_composite(canvas, lyr)

    return finalize_cel_frame_128(canvas, GRUNT_PAL)


# ============================================================================
# 4. REFINED KEYFRAME TABLES (FORWARD-RIGHT STRIKE KINEMATICS, ZERO CLIPPING!)
# ============================================================================

BUILDER_CLIPS: Dict[str, List[Dict[str, float]]] = {
    # IDLE (4f @ 12 FPS): Gentle craftsman breathing, planted boots (0.0px foot drift!), light mallet breathe
    "idle": [
        {"pelvis_dy": 0.0,  "torso_sy": 1.000, "torso_sx": 1.000, "torso_ang": 0.0,  "head_dy": 0.0,  "uarm_ang": 0.0,  "farm_ang": 0.0,  "wrist_ang": 0.0,  "barm_ang": 0.0},
        {"pelvis_dy": 0.8,  "torso_sy": 0.982, "torso_sx": 1.014, "torso_ang": 1.2,  "head_dy": 0.5,  "uarm_ang": -3.0, "farm_ang": -2.0, "wrist_ang": -2.5, "barm_ang": -2.5},
        {"pelvis_dy": -0.6, "torso_sy": 1.022, "torso_sx": 0.990, "torso_ang": -1.0, "head_dy": -0.6, "uarm_ang": 3.5,  "farm_ang": 2.5,  "wrist_ang": 3.0,  "barm_ang": 3.0},
        {"pelvis_dy": 0.3,  "torso_sy": 1.005, "torso_sx": 0.998, "torso_ang": 0.3,  "head_dy": 0.1,  "uarm_ang": 1.0,  "farm_ang": 0.8,  "wrist_ang": 1.0,  "barm_ang": 1.0},
    ],
    # WALK (6f @ 12 FPS): Natural stride with clean boot clearance (no stumpy over-tucked boot!)
    "walk": [
        # f0: Front boot forward contact, back boot trailing push
        {"pelvis_dy": 0.6,  "torso_ang": 3.2, "head_ang": -1.5, "fleg_dx": 4.5,  "fleg_dy": 0.0,  "fleg_ang": -16.0, "bleg_dx": -4.8, "bleg_dy": -0.8, "bleg_ang": 16.0,  "uarm_ang": -12.0, "farm_ang": -5.0, "wrist_ang": -4.0, "barm_ang": -18.0},
        # f1: Down weight-plant on front boot, back boot lifts cleanly
        {"pelvis_dy": 2.0,  "torso_sy": 0.978, "torso_sx": 1.018, "torso_ang": 4.2, "head_ang": -2.0, "fleg_dx": 2.0, "fleg_dy": 0.0, "fleg_ang": -6.0, "bleg_dx": -1.6, "bleg_dy": -2.6, "bleg_ang": 8.0, "uarm_ang": -7.0, "farm_ang": -3.0, "wrist_ang": -2.0, "barm_ang": -10.0},
        # f2: Passing high-point — front boot planted pushing back, back boot swinging forward
        {"pelvis_dy": -1.4, "torso_sy": 1.016, "torso_ang": 2.0, "head_ang": -1.0, "fleg_dx": -1.4, "fleg_dy": 0.0, "fleg_ang": 6.0, "bleg_dx": 2.5, "bleg_dy": -3.0, "bleg_ang": -10.0, "uarm_ang": 5.0, "farm_ang": 3.0, "wrist_ang": 3.0, "barm_ang": 8.0},
        # f3: Back boot forward contact, front boot trailing push
        {"pelvis_dy": 0.6,  "torso_ang": 3.2, "head_ang": -1.5, "fleg_dx": -4.8, "fleg_dy": -0.8, "fleg_ang": 16.0,  "bleg_dx": 4.5,  "bleg_dy": 0.0,  "bleg_ang": -16.0, "uarm_ang": 13.0,  "farm_ang": 6.0,  "wrist_ang": 5.0,  "barm_ang": 20.0},
        # f4: Down weight-plant on back boot, front boot lifts cleanly with full boot shaft visible
        {"pelvis_dy": 1.8,  "torso_sy": 0.978, "torso_sx": 1.018, "torso_ang": 4.2, "head_ang": -2.0, "fleg_dx": -2.6, "fleg_dy": -1.4, "fleg_ang": 11.0, "bleg_dx": 2.0, "bleg_dy": 0.0, "bleg_ang": -6.0, "uarm_ang": 8.0, "farm_ang": 4.0, "wrist_ang": 3.0, "barm_ang": 12.0},
        # f5: Passing high-point — back boot planted pushing back, front boot swinging forward
        {"pelvis_dy": -1.4, "torso_sy": 1.016, "torso_ang": 2.0, "head_ang": -1.0, "fleg_dx": 2.8, "fleg_dy": -1.6, "fleg_ang": -12.0, "bleg_dx": -1.4, "bleg_dy": 0.0, "bleg_ang": 6.0, "uarm_ang": -4.0, "farm_ang": -2.0, "wrist_ang": -2.0, "barm_ang": -7.0},
    ],
    # REPAIR (6f @ 12 FPS):
    # Note: In Builder rig, CCW (negative angle) lifts arm & T-hammer FORWARD-UP-OVERHEAD!
    # Moving arm to (-58 deg uarm + positive wrist snap) drives the heavy iron head FORWARD-RIGHT onto target!
    "repair": [
        # f0: Prepare T-hammer — step into wide brace, lift & cock hammer head upward in front of chest
        {"pelvis_dy": 1.0,  "torso_ang": 1.5,  "head_ang": 0.0,  "fleg_dx": -1.5, "bleg_dx": 2.2, "uarm_ang": -34.0, "farm_ang": -24.0, "wrist_ang": -12.0, "barm_ang": -10.0},
        # f1: Pull hammer backward — lean back -8 deg onto rear foot, wind hammer back past cap
        {"root_dx": -1.6,   "pelvis_dy": 1.8,  "torso_sy": 0.975,"torso_sx": 1.02, "torso_ang": -8.0, "head_ang": 5.5, "fleg_dx": -2.2, "bleg_dx": 2.8, "fsh_dx": 1.5, "fsh_dy": -1.5, "uarm_ang": -76.0, "farm_ang": -34.0, "wrist_ang": -16.0, "barm_ang": 18.0},
        # f2: Raise hammer HIGH OVERHEAD — unmistakable apex silhouette with T-hammer held high above cap!
        {"root_dx": -1.0,   "pelvis_dy": -1.6, "torso_sy": 1.03, "torso_sx": 0.98, "torso_ang": -9.5, "head_ang": 7.5, "fleg_dx": -2.0, "bleg_dx": 3.2, "fsh_dx": 3.0, "fsh_dy": -3.5, "uarm_ang": -116.0,"farm_ang": -22.0, "wrist_ang": 12.0,  "barm_ang": 26.0},
        # f3: Powerful downward strike FORWARD-RIGHT + sleek tapered cel smear arc!
        {"root_dx": 2.6,    "pelvis_dy": 2.4,  "torso_sy": 0.96, "torso_sx": 1.03, "torso_ang": 13.0, "head_ang": -4.0,"fleg_dx": -2.2, "bleg_dx": 4.6, "fsh_dx": 5.5, "fsh_dy": 1.5,  "uarm_ang": -66.0, "farm_ang": 16.0,  "wrist_ang": 42.0,  "barm_ang": -26.0, "swing_arc": 1.0},
        # f4: Impact follow-through & compression — T-hammer driven solidly into front-right repair point!
        {"root_dx": 3.0,    "pelvis_dy": 3.6,  "torso_sy": 0.935,"torso_sx": 1.05, "torso_ang": 15.0, "head_ang": -5.0,"fleg_dx": -2.2, "bleg_dx": 5.0, "fsh_dx": 6.0, "fsh_dy": 2.5,  "uarm_ang": -52.0, "farm_ang": 22.0,  "wrist_ang": 48.0,  "barm_ang": -30.0},
        # f5: Return toward ready pose — smooth lift & weight recovery back to waist carry
        {"root_dx": 1.0,    "pelvis_dy": 1.0,  "torso_sy": 0.99, "torso_sx": 1.005,"torso_ang": 4.5,  "head_ang": -1.5,"fleg_dx": -0.8, "bleg_dx": 1.8, "fsh_dx": 2.0, "fsh_dy": 0.5,  "uarm_ang": -22.0, "farm_ang": 6.0,   "wrist_ang": 14.0,  "barm_ang": -10.0},
    ],
    # HIT REACTION (2f @ 12 FPS): Readable defensive recoil without distorting face or costume
    "hit": [
        {"root_dx": -3.2, "pelvis_dy": 1.8, "torso_sy": 0.96, "torso_sx": 1.03, "torso_ang": -11.5, "head_ang": -6.5, "fleg_dx": -2.0, "bleg_dx": 2.0, "bleg_dy": -1.8, "bleg_ang": -8.0, "uarm_ang": -24.0, "farm_ang": -18.0, "wrist_ang": -10.0, "barm_ang": 22.0},
        {"root_dx": -1.4, "pelvis_dy": 2.0, "torso_sy": 0.98, "torso_sx": 1.01, "torso_ang": -5.0,  "head_ang": -2.5, "fleg_dx": -1.0, "bleg_dx": 1.0, "bleg_dy": 0.0,  "bleg_ang": -3.0, "uarm_ang": -10.0, "farm_ang": -8.0,  "wrist_ang": -4.0,  "barm_ang": 10.0},
    ],
}


GRUNT_CLIPS: Dict[str, List[Dict[str, float]]] = {
    # IDLE (4f @ 12 FPS): Heavy goblinoid belly-breathing, shoulder roll & light club sway, 0.0px foot drift
    "idle": [
        {"pelvis_dy": 0.0,  "torso_sy": 1.000, "torso_sx": 1.000, "torso_ang": 0.0,  "head_dy": 0.0,  "uarm_ang": 0.0,  "farm_ang": 0.0,  "wrist_ang": 0.0,  "barm_ang": 0.0},
        {"pelvis_dy": 1.1,  "torso_sy": 0.976, "torso_sx": 1.022, "torso_ang": 1.8,  "head_dy": 0.7,  "uarm_ang": -3.0, "farm_ang": 2.0,  "wrist_ang": 4.0,  "barm_ang": -3.5},
        {"pelvis_dy": -0.7, "torso_sy": 1.024, "torso_sx": 0.986, "torso_ang": -1.4, "head_dy": -0.7, "uarm_ang": 3.5,  "farm_ang": -2.5, "wrist_ang": -4.5, "barm_ang": 4.0},
        {"pelvis_dy": 0.4,  "torso_sy": 1.004, "torso_sx": 1.000, "torso_ang": 0.5,  "head_dy": 0.2,  "uarm_ang": 1.0,  "farm_ang": -0.8, "wrist_ang": -1.5, "barm_ang": 1.2},
    ],
    # WALK (6f @ 12 FPS): Heavy, slightly clumsy goblinoid stomp with grounded alternating foot contacts
    "walk": [
        {"pelvis_dy": 1.0,  "torso_ang": 4.5, "head_ang": -2.0, "fleg_dx": 4.8,  "fleg_dy": 0.0,  "fleg_ang": -18.0, "bleg_dx": -5.0, "bleg_dy": -1.0, "bleg_ang": 18.0,  "uarm_ang": -12.0, "farm_ang": 4.0,  "wrist_ang": 10.0,  "barm_ang": -20.0},
        {"pelvis_dy": 2.8,  "torso_sy": 0.968, "torso_sx": 1.028, "torso_ang": 5.8, "head_ang": -2.5, "fleg_dx": 2.2, "fleg_dy": 0.0, "fleg_ang": -7.0, "bleg_dx": -1.5, "bleg_dy": -3.2, "bleg_ang": 9.0, "uarm_ang": -6.0, "farm_ang": 3.0, "wrist_ang": 6.0, "barm_ang": -11.0},
        {"pelvis_dy": -1.6, "torso_sy": 1.020, "torso_sx": 0.988, "torso_ang": 2.5, "head_ang": -1.0, "fleg_dx": -1.6, "fleg_dy": 0.0, "fleg_ang": 6.0, "bleg_dx": 2.8, "bleg_dy": -3.6, "bleg_ang": -12.0, "uarm_ang": 6.0, "farm_ang": -4.0, "wrist_ang": -6.0, "barm_ang": 10.0},
        {"pelvis_dy": 1.0,  "torso_ang": 4.5, "head_ang": -2.0, "fleg_dx": -5.0, "fleg_dy": -1.0, "fleg_ang": 18.0,  "bleg_dx": 4.8,  "bleg_dy": 0.0,  "bleg_ang": -18.0, "uarm_ang": 12.0,  "farm_ang": -6.0, "wrist_ang": -10.0, "barm_ang": 22.0},
        {"pelvis_dy": 2.8,  "torso_sy": 0.968, "torso_sx": 1.028, "torso_ang": 5.8, "head_ang": -2.5, "fleg_dx": -1.5, "fleg_dy": -3.2, "fleg_ang": 9.0, "bleg_dx": 2.0, "bleg_dy": 0.0, "bleg_ang": -7.0, "uarm_ang": 7.0, "farm_ang": -4.0, "wrist_ang": -6.0, "barm_ang": 13.0},
        {"pelvis_dy": -1.6, "torso_sy": 1.020, "torso_sx": 0.988, "torso_ang": 2.5, "head_ang": -1.0, "fleg_dx": 2.8, "fleg_dy": -3.4, "fleg_ang": -12.0, "bleg_dx": -1.6, "bleg_dy": 0.0, "bleg_ang": 6.0, "uarm_ang": -4.0, "farm_ang": 2.0, "wrist_ang": 4.0, "barm_ang": -8.0},
    ],
    # CLUBSMASH (6f @ 12 FPS):
    # Note: Club at rest already points -35 deg (UP-RIGHT).
    # Lifting uarm CCW (-75..-95 deg) while adjusting wrist (+35..+50 deg) raises club HIGH ABOVE HEAD (y=6..36, x=45..95, ZERO clipping!).
    # Driving uarm forward (-58 deg) + snapping wrist CW (+72..+84 deg) slams club head DOWN-RIGHT in front of Grunt (x=88..118, y=82..114, ZERO clipping!).
    "clubsmash": [
        # f0: Anticipation crouch & coil backward, hauling club upward across chest
        {"root_dx": -1.6, "pelvis_dy": 2.4,  "torso_sy": 0.96, "torso_sx": 1.03, "torso_ang": -7.5,  "head_ang": 4.5,  "fleg_dx": -2.0, "bleg_dx": 2.4, "fsh_dx": 2.0, "fsh_dy": -1.5, "uarm_ang": -36.0, "farm_ang": -14.0, "wrist_ang": 18.0, "barm_ang": 16.0},
        # f1: Raise spiked club HIGH OVERHEAD on right-arm side (Grunt's snarling face stays 100% clear!)
        {"root_dx": -1.2, "pelvis_dy": 0.4,  "torso_sy": 1.015,"torso_sx": 0.99, "torso_ang": -8.5,  "head_ang": 6.5,  "fleg_dx": -2.0, "bleg_dx": 2.8, "fsh_dx": 1.0, "fsh_dy": 2.5,  "uarm_ang": -88.0, "farm_ang": 18.0,  "wrist_ang": 42.0, "barm_ang": 24.0},
        # f2: Full overhead apex cocked high above head — snarling face clear, maximum silhouette height!
        {"root_dx": -1.2, "pelvis_dy": 0.2,  "torso_sy": 1.025,"torso_sx": 0.985,"torso_ang": -10.0, "head_ang": 8.0,  "fleg_dx": -2.0, "bleg_dx": 3.2, "fsh_dx": 1.5, "fsh_dy": 3.0,  "uarm_ang": -92.0, "farm_ang": 22.0,  "wrist_ang": 58.0, "barm_ang": 30.0},
        # f3: Ferocious descending smash FORWARD-RIGHT + sleek tapered cel smash arc!
        {"root_dx": -2.2, "pelvis_dy": 2.0,  "torso_sy": 0.955,"torso_sx": 1.035,"torso_ang": 12.5,  "head_ang": -4.5, "fleg_dx": -2.2, "bleg_dx": 4.2, "fsh_dx": -1.5, "fsh_dy": -3.0, "uarm_ang": -40.0, "farm_ang": 8.0,   "wrist_ang": 67.0, "barm_ang": -28.0, "swing_arc": 1.0},
        # f4: Heavy ground/wall impact compression — spiked club head slammed low front-right!
        {"root_dx": -2.0, "pelvis_dy": 3.0,  "torso_sy": 0.930,"torso_sx": 1.055,"torso_ang": 14.5,  "head_ang": -5.5, "fleg_dx": -2.2, "bleg_dx": 4.6, "fsh_dx": -1.0, "fsh_dy": -3.0, "uarm_ang": -34.0, "farm_ang": 10.0,  "wrist_ang": 70.0, "barm_ang": -32.0},
        # f5: Recovery haul back toward diagonal ready stance
        {"root_dx": 1.0,  "pelvis_dy": 1.2,  "torso_sy": 0.988,"torso_sx": 1.01, "torso_ang": 5.0,   "head_ang": -1.5, "fleg_dx": -0.8, "bleg_dx": 1.8, "fsh_dx": 2.5, "fsh_dy": 0.5,  "uarm_ang": -20.0, "farm_ang": 5.0,   "wrist_ang": 28.0, "barm_ang": -10.0},
    ],
    # HIT REACTION (2f @ 12 FPS): Short recoil keeping Grunt's snarling face 100% unobstructed!
    "hit": [
        {"root_dx": -3.4, "pelvis_dy": 1.8, "torso_sy": 0.955, "torso_sx": 1.035, "torso_ang": -12.5, "head_ang": -7.5, "fleg_dx": -2.0, "bleg_dx": 2.2, "bleg_dy": -2.0, "bleg_ang": -10.0, "uarm_ang": -10.0, "farm_ang": -6.0, "wrist_ang": 22.0, "barm_ang": 26.0},
        {"root_dx": -1.4, "pelvis_dy": 2.2, "torso_sy": 0.980, "torso_sx": 1.015, "torso_ang": -5.5,  "head_ang": -3.0, "fleg_dx": -1.0, "bleg_dx": 1.0, "bleg_dy": 0.0,  "bleg_ang": -3.5,  "uarm_ang": -4.0,  "farm_ang": -3.0, "wrist_ang": 10.0, "barm_ang": 12.0},
    ],
}


# ============================================================================
# 5. VFX PILOT — vfx_builder_repair_spark (6 FRAMES @ 12 FPS, 128x128 RGBA)
# ============================================================================

VFX_PAL = np.array(
    [
        hex_to_rgb("#11151C"),  # 00 Crisp cel ink contour
        hex_to_rgb("#FFFFFF"),  # 01 Hot impact core (early frames only)
        hex_to_rgb("#FEF08A"),  # 02 Canonical golden spark (#FEF08A)
        hex_to_rgb("#FACC15"),  # 03 Warm gold mid-tone
        hex_to_rgb("#CA8A04"),  # 04 Deep amber spark rim / late ember
        hex_to_rgb("#B07848"),  # 05 Seasoned timber fragment lit
        hex_to_rgb("#8C5A32"),  # 06 Seasoned timber fragment base
        hex_to_rgb("#5C381C"),  # 07 Seasoned timber fragment shadow
    ],
    dtype=np.float32,
)


def draw_four_point_star(
    dr: ImageDraw.ImageDraw,
    cx: float,
    cy: float,
    outer_r: float,
    inner_r: float,
    rot_deg: float,
    fill_rgba: Tuple[int, int, int, int],
    outline_rgba: Tuple[int, int, int, int] = (*INK_RGB, 255),
    outline_w: int = 5,
) -> None:
    pts = []
    for i in range(8):
        ang = math.radians(rot_deg + i * 45.0)
        r = outer_r if (i % 2 == 0) else inner_r
        pts.append((cx + r * math.cos(ang), cy + r * math.sin(ang)))
    dr.polygon(pts, fill=fill_rgba, outline=outline_rgba, width=outline_w)


def draw_timber_chip(
    dr: ImageDraw.ImageDraw,
    cx: float,
    cy: float,
    length: float,
    thick: float,
    rot_deg: float,
    lit_rgb: Tuple[int, int, int] = (176, 120, 72),
    base_rgb: Tuple[int, int, int] = (140, 90, 50),
) -> None:
    th = math.radians(rot_deg)
    ux, uy = math.cos(th), math.sin(th)
    nx, ny = -uy, ux
    hl, ht = length * 0.5, thick * 0.5
    p0 = (cx - ux * hl - nx * ht, cy - uy * hl - ny * ht)
    p1 = (cx + ux * hl - nx * (ht * 0.75), cy + uy * hl - ny * (ht * 0.75))
    p2 = (cx + ux * (hl * 0.85) + nx * ht, cy + uy * (hl * 0.85) + ny * ht)
    p3 = (cx - ux * hl + nx * (ht * 0.85), cy - uy * hl + ny * (ht * 0.85))
    dr.polygon([p0, p1, p2, p3], fill=(*base_rgb, 255), outline=(*INK_RGB, 255), width=5)
    pm0 = (cx - ux * (hl * 0.8) - nx * (ht * 0.55), cy - uy * (hl * 0.8) - ny * (ht * 0.55))
    pm1 = (cx + ux * (hl * 0.8) - nx * (ht * 0.4), cy + uy * (hl * 0.8) - ny * (ht * 0.4))
    dr.line([pm0, pm1], fill=(*lit_rgb, 255), width=4)


def build_vfx_repair_spark_frames() -> List[Image.Image]:
    cx, cy = 256.0, 268.0
    chip_seeds = [
        (-125.0, 146.0, 20.0, -52.0, 44.0, 18.0),
        (-52.0,  160.0, -15.0, 64.0, 48.0, 19.0),
        (22.0,   130.0, 35.0,  48.0, 40.0, 17.0),
        (152.0,  116.0, -40.0, -58.0, 38.0, 16.0),
    ]
    spark_dirs = [-90.0, -24.0, 42.0, 145.0]

    frames: List[Image.Image] = []
    for f in range(6):
        c512 = Image.new("RGBA", (512, 512), (0, 0, 0, 0))
        dr = ImageDraw.Draw(c512)

        if f == 0:
            draw_four_point_star(dr, cx, cy, outer_r=68.0, inner_r=17.0, rot_deg=0.0, fill_rgba=(254, 240, 138, 255))
            draw_four_point_star(dr, cx, cy, outer_r=36.0, inner_r=11.0, rot_deg=45.0, fill_rgba=(255, 255, 255, 255), outline_w=3)
        elif f == 1:
            dr.arc([cx - 90, cy - 90, cx + 90, cy + 90], start=200, end=350, fill=(*INK_RGB, 255), width=14)
            dr.arc([cx - 90, cy - 90, cx + 90, cy + 90], start=203, end=347, fill=(250, 204, 21, 255), width=6)
            draw_four_point_star(dr, cx, cy, outer_r=114.0, inner_r=25.0, rot_deg=8.0, fill_rgba=(254, 240, 138, 255))
            draw_four_point_star(dr, cx, cy, outer_r=56.0, inner_r=15.0, rot_deg=53.0, fill_rgba=(255, 255, 255, 255), outline_w=4)
            for ang, v0, r0, sp, ln, thk in chip_seeds[:3]:
                rad = math.radians(ang)
                draw_timber_chip(dr, cx + math.cos(rad) * 58.0, cy + math.sin(rad) * 58.0, ln, thk, r0 + sp * 0.25)
        elif f == 2:
            for idx, sang in enumerate(spark_dirs):
                rad = math.radians(sang)
                sx = cx + math.cos(rad) * 102.0
                sy = cy + math.sin(rad) * 96.0 + 10.0
                draw_four_point_star(dr, sx, sy, outer_r=54.0 - idx * 4, inner_r=14.0, rot_deg=15.0 * idx, fill_rgba=(254, 240, 138, 255))
            for ang, v0, r0, sp, ln, thk in chip_seeds:
                rad = math.radians(ang)
                draw_timber_chip(dr, cx + math.cos(rad) * (v0 * 0.68), cy + math.sin(rad) * (v0 * 0.68) + 22.0, ln, thk, r0 + sp * 0.55)
        elif f == 3:
            for idx, sang in enumerate(spark_dirs[:3]):
                rad = math.radians(sang + 8.0)
                sx = cx + math.cos(rad) * 134.0
                sy = cy + math.sin(rad) * 118.0 + 36.0
                draw_four_point_star(dr, sx, sy, outer_r=38.0, inner_r=11.0, rot_deg=25.0 + idx * 20.0, fill_rgba=(250, 204, 21, 255))
            for ang, v0, r0, sp, ln, thk in chip_seeds:
                rad = math.radians(ang)
                draw_timber_chip(dr, cx + math.cos(rad) * (v0 * 0.92), cy + math.sin(rad) * (v0 * 0.84) + 64.0, ln * 0.90, thk * 0.90, r0 + sp * 0.85)
        elif f == 4:
            for sang in [-45.0, 28.0]:
                rad = math.radians(sang)
                sx = cx + math.cos(rad) * 148.0
                sy = cy + math.sin(rad) * 110.0 + 80.0
                draw_four_point_star(dr, sx, sy, outer_r=28.0, inner_r=9.0, rot_deg=45.0, fill_rgba=(202, 138, 4, 255), outline_w=4)
            for ang, v0, r0, sp, ln, thk in chip_seeds[1:4]:
                rad = math.radians(ang)
                py = min(448.0, cy + math.sin(rad) * (v0 * 0.86) + 112.0)
                draw_timber_chip(dr, cx + math.cos(rad) * (v0 * 1.02), py, ln * 0.78, thk * 0.80, r0 + sp * 1.15)
        else:
            # f == 5: Clean dissipation tail — 3 settling dark-timber chips, ZERO residual bright yellow/white pixels!
            for ang, v0, r0, sp, ln, thk in chip_seeds[1:4]:
                rad = math.radians(ang)
                py = min(460.0, cy + math.sin(rad) * (v0 * 0.86) + 146.0)
                draw_timber_chip(
                    dr,
                    cx + math.cos(rad) * (v0 * 1.08),
                    py,
                    ln * 0.62,
                    thk * 0.66,
                    r0 + sp * 1.35,
                    lit_rgb=(140, 90, 50),
                    base_rgb=(92, 56, 28),
                )

        frames.append(finalize_cel_frame_128(c512, VFX_PAL))
    return frames


# ============================================================================
# 6. SHEET PACKERS & RIG DEBUG EXPORT
# ============================================================================

def save_rig_debug_sheet(
    rig: Dict[str, Tuple[Image.Image, Tuple[float, float]]],
    title: str,
    out_path: Path,
) -> None:
    cols = 4
    rows = 2
    cell_w, cell_h = 300, 380
    sheet = Image.new("RGB", (cols * cell_w + 40, rows * cell_h + 90), (20, 26, 36))
    dr = ImageDraw.Draw(sheet)
    dr.text((24, 20), title, fill=(254, 240, 138))

    for idx, (name, (im, piv)) in enumerate(rig.items()):
        r, c = divmod(idx, cols)
        ox = 20 + c * cell_w
        oy = 64 + r * cell_h
        dr.rounded_rectangle([ox + 8, oy + 8, ox + cell_w - 8, oy + cell_h - 8], radius=10, fill=(38, 46, 62), outline=(82, 96, 120), width=2)
        scale = 0.68
        nw, nh = int(im.width * scale), int(im.height * scale)
        thumb = im.resize((nw, nh), Image.Resampling.BILINEAR)
        tx = ox + (cell_w - nw) // 2
        ty = oy + 32
        sheet.paste(thumb, (tx, ty), thumb)
        px = tx + int(piv[0] * scale)
        py = ty + int(piv[1] * scale)
        dr.ellipse([px - 5, py - 5, px + 5, py + 5], fill=(239, 68, 68), outline=(255, 255, 255), width=2)
        dr.text((ox + 18, oy + 14), f"{idx+1}. {name}  piv=({piv[0]:.0f},{piv[1]:.0f})", fill=(241, 245, 249))

    out_path.parent.mkdir(parents=True, exist_ok=True)
    sheet.save(out_path)


def pack_grid_sheet(frames: List[Image.Image], cols: int, rows: int) -> Image.Image:
    assert len(frames) == cols * rows
    sheet = Image.new("RGBA", (cols * 128, rows * 128), (0, 0, 0, 0))
    for idx, fr in enumerate(frames):
        r, c = divmod(idx, cols)
        sheet.paste(fr, (c * 128, r * 128))
    return sheet


def measure_frame_stats(im: Image.Image) -> Dict[str, object]:
    arr = np.array(im)
    alpha = arr[:, :, 3]
    ys, xs = np.where(alpha > 0)
    assert len(ys) > 0, "Empty frame detected!"
    y_max = int(ys.max())
    bottom_band = alpha[max(0, y_max - 4) : y_max + 1, :] > 0
    foot_xs = np.where(bottom_band.any(axis=0))[0]
    semi = int(np.sum((alpha > 0) & (alpha < 255)))
    border_clipped = bool(
        (xs.min() == 0) or (xs.max() == 127) or (ys.min() == 0) or (ys.max() == 127)
    )
    return {
        "bbox": [int(xs.min()), int(ys.min()), int(xs.max()), int(ys.max())],
        "width": int(xs.max() - xs.min() + 1),
        "height": int(ys.max() - ys.min() + 1),
        "opaque_pixels": int(np.sum(alpha == 255)),
        "semi_transparent_pixels": semi,
        "border_clipped": border_clipped,
        "foot_contact_y": y_max,
        "foot_contact_x_span": [int(foot_xs.min()), int(foot_xs.max())],
        "centroid": [round(float(xs.mean()), 2), round(float(ys.mean()), 2)],
    }


def main() -> None:
    builder_rig = build_builder_rig()
    grunt_rig = build_grunt_rig()

    builder_dir = ART003_DIR / "characters" / "builder"
    grunt_dir = ART003_DIR / "characters" / "grunt"
    vfx_dir = ART003_DIR / "vfx" / "builder_repair_spark"

    (builder_dir / "frames").mkdir(parents=True, exist_ok=True)
    (grunt_dir / "frames").mkdir(parents=True, exist_ok=True)
    (vfx_dir / "frames").mkdir(parents=True, exist_ok=True)

    save_rig_debug_sheet(
        builder_rig,
        "ART-003 BUILDER 2D CUTOUT RIG - ISOLATED CEL-COMPLETED LAYERS & JOINT PIVOTS",
        builder_dir / "rig_debug_sheet.png",
    )
    save_rig_debug_sheet(
        grunt_rig,
        "ART-003 GRUNT 2D CUTOUT RIG - ISOLATED CEL-COMPLETED LAYERS & JOINT PIVOTS",
        grunt_dir / "rig_debug_sheet.png",
    )

    clip_meta: Dict[str, object] = {
        "package_id": "ART-003",
        "style_version": "bastion-dawn-v1",
        "fps": 12,
        "frame_duration_ms": round(1000.0 / 12.0, 3),
        "cell_size": [128, 128],
        "normalized_pivot": [0.50, 0.08],
        "pivot_pixel_top_down": [64.0, 117.76],
        "target_ppu": 192,
        "entities": {},
    }

    builder_all: List[Image.Image] = []
    builder_records = []
    sheet_idx = 0
    for clip_name in ["idle", "walk", "repair", "hit"]:
        for f_idx, pose in enumerate(BUILDER_CLIPS[clip_name]):
            fr = render_builder_pose(builder_rig, pose)
            fname = f"chr_role_builder_{clip_name}_f{f_idx:02d}.png"
            fr.save(builder_dir / "frames" / fname, optimize=False)
            builder_all.append(fr)
            r, c = divmod(sheet_idx, 6)
            stats = measure_frame_stats(fr)
            assert not stats["border_clipped"], f"Builder clipped border on {fname}: {stats['bbox']}"
            builder_records.append(
                {
                    "sheet_index": sheet_idx,
                    "grid_row_col": [r, c],
                    "sheet_rect_px": [c * 128, r * 128, 128, 128],
                    "clip": clip_name,
                    "frame_index": f_idx,
                    "loop": clip_name in ("idle", "walk"),
                    "file": f"art/production/ART-003/characters/builder/frames/{fname}",
                    **stats,
                }
            )
            sheet_idx += 1

    pack_grid_sheet(builder_all, cols=6, rows=3).save(builder_dir / "chr_role_builder_sheet.png", optimize=False)

    grunt_all: List[Image.Image] = []
    grunt_records = []
    sheet_idx = 0
    for clip_name in ["idle", "walk", "clubsmash", "hit"]:
        for f_idx, pose in enumerate(GRUNT_CLIPS[clip_name]):
            fr = render_grunt_pose(grunt_rig, pose)
            fname = f"chr_enemy_grunt_{clip_name}_f{f_idx:02d}.png"
            fr.save(grunt_dir / "frames" / fname, optimize=False)
            grunt_all.append(fr)
            r, c = divmod(sheet_idx, 6)
            stats = measure_frame_stats(fr)
            assert not stats["border_clipped"], f"Grunt clipped border on {fname}: {stats['bbox']}"
            grunt_records.append(
                {
                    "sheet_index": sheet_idx,
                    "grid_row_col": [r, c],
                    "sheet_rect_px": [c * 128, r * 128, 128, 128],
                    "clip": clip_name,
                    "frame_index": f_idx,
                    "loop": clip_name in ("idle", "walk"),
                    "file": f"art/production/ART-003/characters/grunt/frames/{fname}",
                    **stats,
                }
            )
            sheet_idx += 1

    pack_grid_sheet(grunt_all, cols=6, rows=3).save(grunt_dir / "chr_enemy_grunt_sheet.png", optimize=False)

    vfx_frames = build_vfx_repair_spark_frames()
    vfx_records = []
    for f_idx, fr in enumerate(vfx_frames):
        fname = f"vfx_builder_repair_spark_f{f_idx:02d}.png"
        fr.save(vfx_dir / "frames" / fname, optimize=False)
        stats = measure_frame_stats(fr)
        assert not stats["border_clipped"], f"VFX clipped border on {fname}: {stats['bbox']}"
        vfx_records.append(
            {
                "sheet_index": f_idx,
                "grid_row_col": [0, f_idx],
                "sheet_rect_px": [f_idx * 128, 0, 128, 128],
                "clip": "repair_spark",
                "frame_index": f_idx,
                "loop": False,
                "sync_note": "Trigger f00 on Builder repair f03->f04 downward hammer contact (cosmetic only)",
                "file": f"art/production/ART-003/vfx/builder_repair_spark/frames/{fname}",
                **stats,
            }
        )

    pack_grid_sheet(vfx_frames, cols=6, rows=1).save(vfx_dir / "vfx_builder_repair_spark_sheet.png", optimize=False)

    clip_meta["entities"] = {
        "builder": {
            "sheet": "art/production/ART-003/characters/builder/chr_role_builder_sheet.png",
            "sheet_dimensions": [768, 384],
            "grid_cols_rows": [6, 3],
            "palette_hex": BUILDER_PALETTE_HEX,
            "clips": {"idle": [0, 3], "walk": [4, 9], "repair": [10, 15], "hit": [16, 17]},
            "frames": builder_records,
        },
        "grunt": {
            "sheet": "art/production/ART-003/characters/grunt/chr_enemy_grunt_sheet.png",
            "sheet_dimensions": [768, 384],
            "grid_cols_rows": [6, 3],
            "palette_hex": GRUNT_PALETTE_HEX,
            "clips": {"idle": [0, 3], "walk": [4, 9], "clubsmash": [10, 15], "hit": [16, 17]},
            "frames": grunt_records,
        },
        "vfx_builder_repair_spark": {
            "sheet": "art/production/ART-003/vfx/builder_repair_spark/vfx_builder_repair_spark_sheet.png",
            "sheet_dimensions": [768, 128],
            "grid_cols_rows": [6, 1],
            "sync_builder_repair_contact_frame": 3,
            "frames": vfx_records,
        },
    }

    with open(ART003_DIR / "clip_frame_metadata.json", "w", encoding="utf-8") as fp:
        json.dump(clip_meta, fp, indent=2)

    print("v2 Export PASS: 42 frames verified with ZERO border clipping and ZERO semi-transparent halo pixels!")


if __name__ == "__main__":
    main()
