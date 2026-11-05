#!/usr/bin/env python3
"""
ART-004 Complete Character Animation (88 Frames) & Combat VFX (24 Frames) Production Builder.

Produces all 112 real RGBA PNG production frames, 9 deterministic sprite sheets,
5 rig debug inspection boards, and clip_frame_metadata.json under art/production/ART-004/.
"""

from __future__ import annotations

import json
import math
import sys
from pathlib import Path
from typing import Any, Callable, Dict, List, Tuple

import numpy as np
from PIL import Image, ImageDraw, ImageFont

THIS_DIR = Path(__file__).resolve().parent
sys.path.insert(0, str(THIS_DIR))

from art004_rig_lib import (  # noqa: E402
    BOSS_PAL,
    BRUTE_PAL,
    FIGHTER_PAL,
    GUARD_PAL,
    INK_RGB,
    REPO_ROOT,
    RUNNER_PAL,
    build_boss_rig,
    build_brute_rig,
    build_fighter_rig,
    build_guard_rig,
    build_runner_rig,
    draw_tapered_cel_smear,
    finalize_cel_frame,
    hex_to_rgb,
    measure_leg_sole,
    rotate_vec,
    transform_layer,
)

ART004_DIR = REPO_ROOT / "art" / "production" / "ART-004"


def rgb_to_hex(rgb: np.ndarray | Tuple[int, int, int]) -> str:
    return f"#{int(rgb[0]):02X}{int(rgb[1]):02X}{int(rgb[2]):02X}"


# ============================================================================
# POSE RENDERERS FOR ALL 5 CHARACTER CLASSES
# ============================================================================

def render_guard_pose(
    rig: Dict[str, Tuple[Image.Image, Tuple[float, float]]],
    pose: Dict[str, float],
    foot_out: Dict[str, List[float]] | None = None,
) -> Image.Image:
    S = 0.638
    ground_y = 470.0
    center_x = 246.0 + pose.get("root_dx", 0.0) * 4.0

    waist_x = center_x
    waist_y = ground_y - (540.0 - 365.0) * S + pose.get("pelvis_dy", 0.0) * 4.0
    torso_ang = pose.get("torso_ang", 0.0)
    torso_sx = S * pose.get("torso_sx", 1.0)
    torso_sy = S * pose.get("torso_sy", 1.0)

    hx, hy = rotate_vec(166.0 - 162.0, 116.0 - 365.0, torso_ang, torso_sx, torso_sy)
    fax, fay = rotate_vec(88.0 - 162.0, 165.0 - 365.0, torso_ang, torso_sx, torso_sy)
    shx, shy = rotate_vec(228.0 - 162.0, 295.0 - 365.0, torso_ang, torso_sx, torso_sy)

    head_pos = (waist_x + hx + pose.get("head_dx", 0.0) * 4.0, waist_y + hy + pose.get("head_dy", 0.0) * 4.0)
    farm_pos = (waist_x + fax + pose.get("farm_dx", 0.0) * 4.0, waist_y + fay + pose.get("farm_dy", 0.0) * 4.0)
    shield_pos = (waist_x + shx + pose.get("sh_dx", 0.0) * 4.0, waist_y + shy + pose.get("sh_dy", 0.0) * 4.0)

    fleg_pos = (
        center_x + (116.0 - 162.0) * S + pose.get("fleg_dx", 0.0) * 4.0,
        ground_y - (540.0 - 378.0) * S + pose.get("fleg_dy", 0.0) * 4.0,
    )
    bleg_pos = (
        center_x + (212.0 - 162.0) * S + pose.get("bleg_dx", 0.0) * 4.0,
        ground_y - (537.0 - 382.0) * S + pose.get("bleg_dy", 0.0) * 4.0,
    )

    bleg_lyr = transform_layer(rig["back_leg"][0], rig["back_leg"][1], bleg_pos, pose.get("bleg_ang", 0.0), (S, S))
    fleg_lyr = transform_layer(rig["front_leg"][0], rig["front_leg"][1], fleg_pos, pose.get("fleg_ang", 0.0), (S, S))
    if foot_out is not None:
        foot_out["front_foot_xy"] = measure_leg_sole(fleg_lyr, 128)
        foot_out["back_foot_xy"] = measure_leg_sole(bleg_lyr, 128)

    canvas = Image.new("RGBA", (512, 512), (0, 0, 0, 0))
    layers = [
        bleg_lyr,
        fleg_lyr,
        transform_layer(rig["torso"][0], rig["torso"][1], (waist_x, waist_y), torso_ang, (torso_sx, torso_sy)),
        transform_layer(rig["head"][0], rig["head"][1], head_pos, torso_ang * 0.5 + pose.get("head_ang", 0.0), (S, S)),
        transform_layer(rig["front_arm"][0], rig["front_arm"][1], farm_pos, torso_ang + pose.get("farm_ang", 0.0), (S, S)),
        transform_layer(
            rig["shield"][0],
            rig["shield"][1],
            shield_pos,
            torso_ang * 0.35 + pose.get("sh_ang", 0.0),
            (S * pose.get("sh_sx", 1.0), S * pose.get("sh_sy", 1.0)),
        ),
    ]
    for lyr in layers:
        canvas = Image.alpha_composite(canvas, lyr)
    return finalize_cel_frame(canvas, GUARD_PAL, out_dim=128)


def render_fighter_pose(
    rig: Dict[str, Tuple[Image.Image, Tuple[float, float]]],
    pose: Dict[str, float],
    foot_out: Dict[str, List[float]] | None = None,
) -> Image.Image:
    S = 0.675
    ground_y = 470.0
    center_x = 244.0 + pose.get("root_dx", 0.0) * 4.0

    waist_x = center_x
    waist_y = ground_y - (575.0 - 430.0) * S + pose.get("pelvis_dy", 0.0) * 4.0
    torso_ang = pose.get("torso_ang", 0.0)
    torso_sx = S * pose.get("torso_sx", 1.0)
    torso_sy = S * pose.get("torso_sy", 1.0)

    hx, hy = rotate_vec(168.0 - 168.0, 238.0 - 430.0, torso_ang, torso_sx, torso_sy)
    bax, bay = rotate_vec(246.0 - 168.0, 268.0 - 430.0, torso_ang, torso_sx, torso_sy)
    fsx, fsy = rotate_vec(86.0 - 168.0, 266.0 - 430.0, torso_ang, torso_sx, torso_sy)

    head_pos = (waist_x + hx + pose.get("head_dx", 0.0) * 4.0, waist_y + hy + pose.get("head_dy", 0.0) * 4.0)
    barm_pos = (waist_x + bax, waist_y + bay)
    fsh_pos = (waist_x + fsx + pose.get("fsh_dx", 0.0) * 4.0, waist_y + fsy + pose.get("fsh_dy", 0.0) * 4.0)

    fleg_pos = (
        center_x + (118.0 - 168.0) * S + pose.get("fleg_dx", 0.0) * 4.0,
        ground_y - (575.0 - 452.0) * S + pose.get("fleg_dy", 0.0) * 4.0,
    )
    bleg_pos = (
        center_x + (208.0 - 168.0) * S + pose.get("bleg_dx", 0.0) * 4.0,
        ground_y - (566.0 - 452.0) * S + pose.get("bleg_dy", 0.0) * 4.0,
    )

    uarm_ang = torso_ang + pose.get("uarm_ang", 0.0)
    ex, ey = rotate_vec(62.0 - 86.0, 355.0 - 266.0, uarm_ang, S, S)
    elbow_pos = (fsh_pos[0] + ex, fsh_pos[1] + ey)

    farm_ang = uarm_ang + pose.get("farm_ang", 0.0)
    wx, wy = rotate_vec(78.0 - 62.0, 422.0 - 355.0, farm_ang, S, S)
    grip_pos = (elbow_pos[0] + wx, elbow_pos[1] + wy)
    sw_ang = farm_ang + pose.get("wrist_ang", 0.0)

    bleg_lyr = transform_layer(rig["back_leg"][0], rig["back_leg"][1], bleg_pos, pose.get("bleg_ang", 0.0), (S, S))
    fleg_lyr = transform_layer(rig["front_leg"][0], rig["front_leg"][1], fleg_pos, pose.get("fleg_ang", 0.0), (S, S))
    if foot_out is not None:
        foot_out["front_foot_xy"] = measure_leg_sole(fleg_lyr, 128)
        foot_out["back_foot_xy"] = measure_leg_sole(bleg_lyr, 128)

    canvas = Image.new("RGBA", (512, 512), (0, 0, 0, 0))
    for lyr in [
        transform_layer(rig["back_arm"][0], rig["back_arm"][1], barm_pos, torso_ang + pose.get("barm_ang", 0.0), (S, S)),
        bleg_lyr,
        fleg_lyr,
        transform_layer(rig["torso"][0], rig["torso"][1], (waist_x, waist_y), torso_ang, (torso_sx, torso_sy)),
        transform_layer(rig["head"][0], rig["head"][1], head_pos, torso_ang * 0.5 + pose.get("head_ang", 0.0), (S, S)),
    ]:
        canvas = Image.alpha_composite(canvas, lyr)

    if pose.get("swing_arc", 0.0) > 0.5:
        canvas = draw_tapered_cel_smear(
            canvas,
            center_xy=(246.0, 266.0),
            radius_outer=174.0,
            max_thick=32.0,
            deg_start=-96.0,
            deg_end=34.0,
            outer_rgb=hex_to_rgb("#E63946"),
            core_rgb=hex_to_rgb("#F8FAFC"),
            outline_w=4,
        )

    for lyr in [
        transform_layer(rig["sword"][0], rig["sword"][1], grip_pos, sw_ang, (S, S)),
        transform_layer(rig["front_upper_arm"][0], rig["front_upper_arm"][1], fsh_pos, uarm_ang, (S, S)),
        transform_layer(rig["front_forearm"][0], rig["front_forearm"][1], elbow_pos, farm_ang, (S, S)),
    ]:
        canvas = Image.alpha_composite(canvas, lyr)

    return finalize_cel_frame(canvas, FIGHTER_PAL, out_dim=128)


def render_runner_pose(
    rig: Dict[str, Tuple[Image.Image, Tuple[float, float]]],
    pose: Dict[str, float],
    foot_out: Dict[str, List[float]] | None = None,
) -> Image.Image:
    S = 0.5904
    ground_y = 470.0
    center_x = 248.0 + pose.get("root_dx", 0.0) * 4.0

    waist_x = center_x
    waist_y = ground_y - (543.0 - 288.0) * S + pose.get("pelvis_dy", 0.0) * 4.0
    torso_ang = pose.get("torso_ang", 0.0)
    torso_sx = S * pose.get("torso_sx", 1.0)
    torso_sy = S * pose.get("torso_sy", 1.0)

    hx, hy = rotate_vec(182.0 - 158.0, 138.0 - 288.0, torso_ang, torso_sx, torso_sy)
    bax, bay = rotate_vec(214.0 - 158.0, 165.0 - 288.0, torso_ang, torso_sx, torso_sy)
    fsx, fsy = rotate_vec(112.0 - 158.0, 160.0 - 288.0, torso_ang, torso_sx, torso_sy)

    head_pos = (waist_x + hx + pose.get("head_dx", 0.0) * 4.0, waist_y + hy + pose.get("head_dy", 0.0) * 4.0)
    barm_pos = (waist_x + bax + pose.get("barm_dx", 0.0) * 4.0, waist_y + bay + pose.get("barm_dy", 0.0) * 4.0)
    fsh_pos = (waist_x + fsx + pose.get("fsh_dx", 0.0) * 4.0, waist_y + fsy + pose.get("fsh_dy", 0.0) * 4.0)

    fleg_pos = (
        center_x + (126.0 - 158.0) * S + pose.get("fleg_dx", 0.0) * 4.0,
        ground_y - (543.0 - 308.0) * S + pose.get("fleg_dy", 0.0) * 4.0,
    )
    bleg_pos = (
        center_x + (184.0 - 158.0) * S + pose.get("bleg_dx", 0.0) * 4.0,
        ground_y - (524.0 - 308.0) * S + pose.get("bleg_dy", 0.0) * 4.0,
    )

    uarm_ang = torso_ang + pose.get("uarm_ang", 0.0)
    ex, ey = rotate_vec(56.0 - 112.0, 232.0 - 160.0, uarm_ang, S, S)
    elbow_pos = (fsh_pos[0] + ex, fsh_pos[1] + ey)
    farm_ang = uarm_ang + pose.get("farm_ang", 0.0)

    bleg_lyr = transform_layer(rig["back_leg"][0], rig["back_leg"][1], bleg_pos, pose.get("bleg_ang", 0.0), (S, S))
    fleg_lyr = transform_layer(rig["front_leg"][0], rig["front_leg"][1], fleg_pos, pose.get("fleg_ang", 0.0), (S, S))
    if foot_out is not None:
        foot_out["front_foot_xy"] = measure_leg_sole(fleg_lyr, 128)
        foot_out["back_foot_xy"] = measure_leg_sole(bleg_lyr, 128)

    canvas = Image.new("RGBA", (512, 512), (0, 0, 0, 0))
    for lyr in [
        transform_layer(rig["back_arm"][0], rig["back_arm"][1], barm_pos, torso_ang + pose.get("barm_ang", 0.0), (S, S)),
        bleg_lyr,
        fleg_lyr,
        transform_layer(rig["torso"][0], rig["torso"][1], (waist_x, waist_y), torso_ang, (torso_sx, torso_sy)),
        transform_layer(rig["head"][0], rig["head"][1], head_pos, torso_ang * 0.45 + pose.get("head_ang", 0.0), (S, S)),
    ]:
        canvas = Image.alpha_composite(canvas, lyr)

    if pose.get("lunge_smear", 0.0) > 0.5:
        # Crisp forward-pointing twin-blade horizontal thrust streak (never a vertical bow!)
        sm_layer = Image.new("RGBA", (512, 512), (0, 0, 0, 0))
        sdr = ImageDraw.Draw(sm_layer)
        sdr.polygon(
            [(224, 312), (408, 326), (272, 354), (242, 336)],
            fill=(*hex_to_rgb("#C084FC"), 255),
            outline=(*INK_RGB, 255),
            width=4,
        )
        sdr.polygon([(270, 322), (394, 328), (288, 342)], fill=(*hex_to_rgb("#F8FAFC"), 255))
        canvas = Image.alpha_composite(canvas, sm_layer)

    for lyr in [
        transform_layer(rig["front_upper_arm"][0], rig["front_upper_arm"][1], fsh_pos, uarm_ang, (S, S)),
        transform_layer(rig["front_forearm"][0], rig["front_forearm"][1], elbow_pos, farm_ang, (S, S)),
    ]:
        canvas = Image.alpha_composite(canvas, lyr)

    return finalize_cel_frame(canvas, RUNNER_PAL, out_dim=128)


def render_brute_pose(
    rig: Dict[str, Tuple[Image.Image, Tuple[float, float]]],
    pose: Dict[str, float],
    foot_out: Dict[str, List[float]] | None = None,
) -> Image.Image:
    S = 0.857
    ground_y = 470.0
    center_x = 252.0 + pose.get("root_dx", 0.0) * 4.0

    waist_x = center_x
    waist_y = ground_y - (452.0 - 316.0) * S + pose.get("pelvis_dy", 0.0) * 4.0
    torso_ang = pose.get("torso_ang", 0.0)
    torso_sx = S * pose.get("torso_sx", 1.0)
    torso_sy = S * pose.get("torso_sy", 1.0)

    hx, hy = rotate_vec(184.0 - 186.0, 96.0 - 316.0, torso_ang, torso_sx, torso_sy)
    bax, bay = rotate_vec(252.0 - 186.0, 116.0 - 316.0, torso_ang, torso_sx, torso_sy)
    fsx, fsy = rotate_vec(108.0 - 186.0, 116.0 - 316.0, torso_ang, torso_sx, torso_sy)

    head_pos = (waist_x + hx + pose.get("head_dx", 0.0) * 4.0, waist_y + hy + pose.get("head_dy", 0.0) * 4.0)
    barm_pos = (waist_x + bax + pose.get("barm_dx", 0.0) * 4.0, waist_y + bay + pose.get("barm_dy", 0.0) * 4.0)
    fsh_pos = (waist_x + fsx + pose.get("fsh_dx", 0.0) * 4.0, waist_y + fsy + pose.get("fsh_dy", 0.0) * 4.0)

    fleg_pos = (
        center_x + (142.0 - 186.0) * S + pose.get("fleg_dx", 0.0) * 4.0,
        ground_y - (452.0 - 332.0) * S + pose.get("fleg_dy", 0.0) * 4.0,
    )
    bleg_pos = (
        center_x + (224.0 - 186.0) * S + pose.get("bleg_dx", 0.0) * 4.0,
        ground_y - (450.0 - 332.0) * S + pose.get("bleg_dy", 0.0) * 4.0,
    )

    uarm_ang = torso_ang + pose.get("uarm_ang", 0.0)
    ex, ey = rotate_vec(72.0 - 108.0, 214.0 - 116.0, uarm_ang, S, S)
    elbow_pos = (fsh_pos[0] + ex, fsh_pos[1] + ey)
    farm_ang = uarm_ang + pose.get("farm_ang", 0.0)

    bleg_lyr = transform_layer(rig["back_leg"][0], rig["back_leg"][1], bleg_pos, pose.get("bleg_ang", 0.0), (S, S))
    fleg_lyr = transform_layer(rig["front_leg"][0], rig["front_leg"][1], fleg_pos, pose.get("fleg_ang", 0.0), (S, S))
    if foot_out is not None:
        foot_out["front_foot_xy"] = measure_leg_sole(fleg_lyr, 128)
        foot_out["back_foot_xy"] = measure_leg_sole(bleg_lyr, 128)

    canvas = Image.new("RGBA", (512, 512), (0, 0, 0, 0))
    for lyr in [
        transform_layer(rig["back_arm"][0], rig["back_arm"][1], barm_pos, torso_ang + pose.get("barm_ang", 0.0), (S, S)),
        bleg_lyr,
        fleg_lyr,
        transform_layer(rig["torso"][0], rig["torso"][1], (waist_x, waist_y), torso_ang, (torso_sx, torso_sy)),
        transform_layer(rig["head"][0], rig["head"][1], head_pos, torso_ang * 0.4 + pose.get("head_ang", 0.0), (S, S)),
    ]:
        canvas = Image.alpha_composite(canvas, lyr)

    if pose.get("crush_arc", 0.0) > 0.5:
        canvas = draw_tapered_cel_smear(
            canvas,
            center_xy=(212.0, 260.0),
            radius_outer=124.0,
            max_thick=34.0,
            deg_start=-96.0,
            deg_end=48.0,
            outer_rgb=hex_to_rgb("#FB923C"),
            core_rgb=hex_to_rgb("#FDBA74"),
            outline_w=4,
        )

    for lyr in [
        transform_layer(rig["front_upper_arm"][0], rig["front_upper_arm"][1], fsh_pos, uarm_ang, (S, S)),
        transform_layer(rig["front_forearm"][0], rig["front_forearm"][1], elbow_pos, farm_ang, (S, S)),
    ]:
        canvas = Image.alpha_composite(canvas, lyr)

    return finalize_cel_frame(canvas, BRUTE_PAL, out_dim=128)


def render_boss_pose(
    rig: Dict[str, Tuple[Image.Image, Tuple[float, float]]],
    pose: Dict[str, float],
    foot_out: Dict[str, List[float]] | None = None,
) -> Image.Image:
    S = 1.1694
    ground_y = 942.0
    center_x = 468.0 + pose.get("root_dx", 0.0) * 4.0
    csz = (1024, 1024)

    waist_x = center_x
    waist_y = ground_y - (606.0 - 350.0) * S + pose.get("pelvis_dy", 0.0) * 4.0
    torso_ang = pose.get("torso_ang", 0.0)
    torso_sx = S * pose.get("torso_sx", 1.0)
    torso_sy = S * pose.get("torso_sy", 1.0)

    hx, hy = rotate_vec(196.0 - 196.0, 138.0 - 350.0, torso_ang, torso_sx, torso_sy)
    bax, bay = rotate_vec(264.0 - 196.0, 185.0 - 350.0, torso_ang, torso_sx, torso_sy)
    fsx, fsy = rotate_vec(118.0 - 196.0, 182.0 - 350.0, torso_ang, torso_sx, torso_sy)

    head_pos = (waist_x + hx + pose.get("head_dx", 0.0) * 4.0, waist_y + hy + pose.get("head_dy", 0.0) * 4.0)
    barm_pos = (waist_x + bax + pose.get("barm_dx", 0.0) * 4.0, waist_y + bay + pose.get("barm_dy", 0.0) * 4.0)
    fsh_pos = (waist_x + fsx + pose.get("fsh_dx", 0.0) * 4.0, waist_y + fsy + pose.get("fsh_dy", 0.0) * 4.0)

    fleg_pos = (
        center_x + (142.0 - 196.0) * S + pose.get("fleg_dx", 0.0) * 4.0,
        ground_y - (606.0 - 405.0) * S + pose.get("fleg_dy", 0.0) * 4.0,
    )
    bleg_pos = (
        center_x + (234.0 - 196.0) * S + pose.get("bleg_dx", 0.0) * 4.0,
        ground_y - (578.0 - 405.0) * S + pose.get("bleg_dy", 0.0) * 4.0,
    )

    uarm_ang = torso_ang + pose.get("uarm_ang", 0.0)
    ex, ey = rotate_vec(88.0 - 118.0, 240.0 - 182.0, uarm_ang, S, S)
    elbow_pos = (fsh_pos[0] + ex, fsh_pos[1] + ey)

    farm_ang = uarm_ang + pose.get("farm_ang", 0.0)
    wx, wy = rotate_vec(98.0 - 88.0, 354.0 - 240.0, farm_ang, S, S)
    grip_pos = (elbow_pos[0] + wx, elbow_pos[1] + wy)
    cl_ang = farm_ang + pose.get("wrist_ang", 0.0)

    bleg_lyr = transform_layer(rig["back_leg"][0], rig["back_leg"][1], bleg_pos, pose.get("bleg_ang", 0.0), (S, S), csz)
    fleg_lyr = transform_layer(rig["front_leg"][0], rig["front_leg"][1], fleg_pos, pose.get("fleg_ang", 0.0), (S, S), csz)
    if foot_out is not None:
        foot_out["front_foot_xy"] = measure_leg_sole(fleg_lyr, 256)
        foot_out["back_foot_xy"] = measure_leg_sole(bleg_lyr, 256)

    canvas = Image.new("RGBA", csz, (0, 0, 0, 0))
    for lyr in [
        transform_layer(rig["back_arm"][0], rig["back_arm"][1], barm_pos, torso_ang + pose.get("barm_ang", 0.0), (S, S), csz),
        bleg_lyr,
        fleg_lyr,
        transform_layer(rig["torso"][0], rig["torso"][1], (waist_x, waist_y), torso_ang, (torso_sx, torso_sy), csz),
        transform_layer(rig["head"][0], rig["head"][1], head_pos, torso_ang * 0.45 + pose.get("head_ang", 0.0), (S, S), csz),
    ]:
        canvas = Image.alpha_composite(canvas, lyr)

    if pose.get("cleave_arc", 0.0) > 0.5:
        canvas = draw_tapered_cel_smear(
            canvas,
            center_xy=(468.0, 505.0),
            radius_outer=296.0,
            max_thick=68.0,
            deg_start=-98.0,
            deg_end=26.0,
            outer_rgb=hex_to_rgb("#D946EF"),
            core_rgb=hex_to_rgb("#FCE7F3"),
            outline_w=6,
        )

    for lyr in [
        transform_layer(rig["cleaver"][0], rig["cleaver"][1], grip_pos, cl_ang, (S, S), csz),
        transform_layer(rig["front_upper_arm"][0], rig["front_upper_arm"][1], fsh_pos, uarm_ang, (S, S), csz),
        transform_layer(rig["front_forearm"][0], rig["front_forearm"][1], elbow_pos, farm_ang, (S, S), csz),
    ]:
        canvas = Image.alpha_composite(canvas, lyr)

    return finalize_cel_frame(canvas, BOSS_PAL, out_dim=256, min_island_px=14)


# ============================================================================
# DETERMINISTIC CLIP KEYFRAME TABLES (88 CHARACTER FRAMES @ 12 FPS)
# ============================================================================

GUARD_CLIPS: Dict[str, List[Dict[str, float]]] = {
    # IDLE (4f): Smooth quadrature breathing + subtle protective tower-shield settle (0.0px foot slip!)
    "idle": [
        {"pelvis_dy":  0.00, "torso_sy": 1.000, "torso_sx": 1.000, "torso_ang":  0.0, "head_dy": -0.5, "farm_ang":  3.2, "sh_dy": -0.7, "sh_dx":  0.0},
        {"pelvis_dy":  0.80, "torso_sy": 0.982, "torso_sx": 1.014, "torso_ang":  1.1, "head_dy":  0.0, "farm_ang":  0.0, "sh_dy":  0.0, "sh_dx":  0.5},
        {"pelvis_dy":  0.00, "torso_sy": 1.000, "torso_sx": 1.000, "torso_ang":  0.0, "head_dy":  0.5, "farm_ang": -3.2, "sh_dy":  0.7, "sh_dx":  0.0},
        {"pelvis_dy": -0.80, "torso_sy": 1.018, "torso_sx": 0.988, "torso_ang": -1.1, "head_dy":  0.0, "farm_ang":  0.0, "sh_dy":  0.0, "sh_dx": -0.5},
    ],
    # WALK (6f): Steady armored phalanx advance keeping tower shield forward
    "walk": [
        {"pelvis_dy":  0.8, "torso_ang": 2.8, "head_ang": -1.2, "fleg_dx":  4.6, "fleg_dy":  0.0, "fleg_ang": -15.0, "bleg_dx": -4.8, "bleg_dy": -0.8, "bleg_ang":  15.0, "farm_ang": -14.0, "sh_dx":  0.8, "sh_dy":  0.5},
        {"pelvis_dy":  2.0, "torso_sy": 0.978, "torso_sx": 1.018, "torso_ang": 3.8, "head_ang": -1.6, "fleg_dx":  1.8, "fleg_dy":  0.0, "fleg_ang":  -6.0, "bleg_dx": -1.5, "bleg_dy": -2.8, "bleg_ang":   8.0, "farm_ang":  -8.0, "sh_dx":  1.2, "sh_dy":  1.2},
        {"pelvis_dy": -1.2, "torso_sy": 1.016, "torso_ang": 1.8, "head_ang": -0.8, "fleg_dx": -1.8, "fleg_dy":  0.0, "fleg_ang":   6.0, "bleg_dx":  2.6, "bleg_dy": -3.2, "bleg_ang": -11.0, "farm_ang":   5.0, "sh_dx":  0.0, "sh_dy": -0.8},
        {"pelvis_dy":  0.8, "torso_ang": 2.8, "head_ang": -1.2, "fleg_dx": -4.8, "fleg_dy": -0.8, "fleg_ang":  15.0, "bleg_dx":  4.6, "bleg_dy":  0.0, "bleg_ang": -15.0, "farm_ang":  15.0, "sh_dx": -0.6, "sh_dy":  0.5},
        {"pelvis_dy":  2.0, "torso_sy": 0.978, "torso_sx": 1.018, "torso_ang": 3.8, "head_ang": -1.6, "fleg_dx": -1.8, "fleg_dy": -2.6, "fleg_ang":   9.0, "bleg_dx":  1.8, "bleg_dy":  0.0, "bleg_ang":  -6.0, "farm_ang":   9.0, "sh_dx": -0.2, "sh_dy":  1.2},
        {"pelvis_dy": -1.2, "torso_sy": 1.016, "torso_ang": 1.8, "head_ang": -0.8, "fleg_dx":  2.6, "fleg_dy": -2.8, "fleg_ang": -11.0, "bleg_dx": -1.8, "bleg_dy":  0.0, "bleg_ang":   6.0, "farm_ang":  -5.0, "sh_dx":  0.4, "sh_dy": -0.8},
    ],
    # SHIELDBRACE (6f): Purely defensive tower-shield raise, knee/shoulder lock, impact absorption & push-back recovery!
    "shieldbrace": [
        # f0: Alert stance — widen boots, raise shield slightly & draw armored right arm back to brace
        {"root_dx": -0.6, "pelvis_dy": 1.2, "torso_sy": 0.985, "torso_sx": 1.015, "torso_ang": -2.0, "fleg_dx": -2.2, "bleg_dx": 2.6, "farm_ang": 16.0, "sh_dx": 1.8, "sh_dy": -2.4, "sh_ang": -3.0},
        # f1: Forward defensive step-in — plant tower shield firmly ahead, tuck chin behind upper steel rim
        {"root_dx":  1.4, "pelvis_dy": 2.6, "torso_sy": 0.960, "torso_sx": 1.035, "torso_ang":  5.5, "head_dy": 1.4, "fleg_dx": -2.6, "bleg_dx": 4.2, "farm_ang": 26.0, "sh_dx": 5.2, "sh_dy": -3.8, "sh_ang": -6.0, "sh_sy": 1.02},
        # f2: Full fortified lock — maximum defensive wall posture just as blow arrives (synced w/ vfx_guard_shield_block f00)
        {"root_dx":  2.0, "pelvis_dy": 3.4, "torso_sy": 0.945, "torso_sx": 1.045, "torso_ang":  7.5, "head_dy": 2.0, "fleg_dx": -2.8, "bleg_dx": 4.8, "farm_ang": 32.0, "sh_dx": 6.6, "sh_dy": -4.2, "sh_ang": -7.5, "sh_sy": 1.03},
        # f3: Heavy impact shock absorption — knees compress deep, body slides back 3px holding shield rock-solid! (vfx f01)
        {"root_dx": -1.8, "pelvis_dy": 4.4, "torso_sy": 0.925, "torso_sx": 1.060, "torso_ang": -4.5, "head_dy": 2.2, "fleg_dx": -3.0, "bleg_dx": 4.2, "farm_ang": 38.0, "sh_dx": 3.6, "sh_dy": -2.0, "sh_ang": -11.0, "sh_sy": 1.03},
        # f4: Resolute shoulder counter-brace — drive shield forward-up to repel attacker (vfx f02)
        {"root_dx":  1.2, "pelvis_dy": 2.8, "torso_sy": 0.960, "torso_sx": 1.030, "torso_ang":  5.0, "head_dy": 1.2, "fleg_dx": -2.2, "bleg_dx": 3.8, "farm_ang": 22.0, "sh_dx": 5.4, "sh_dy": -3.2, "sh_ang": -5.0},
        # f5: Smooth recovery to upright guard stance (vfx f03 dissipates)
        {"root_dx":  0.4, "pelvis_dy": 1.0, "torso_sy": 0.990, "torso_sx": 1.008, "torso_ang":  1.5, "head_dy": 0.4, "fleg_dx": -1.0, "bleg_dx": 1.6, "farm_ang":  8.0, "sh_dx": 2.0, "sh_dy": -1.0, "sh_ang": -2.0},
    ],
    # HIT (2f): Heavy knockback behind raised shield without losing silhouette
    "hit": [
        {"root_dx": -3.4, "pelvis_dy": 2.4, "torso_sy": 0.955, "torso_sx": 1.035, "torso_ang": -10.5, "head_ang": -6.0, "fleg_dx": -2.0, "bleg_dx": 2.2, "bleg_dy": -1.6, "farm_ang": 28.0, "sh_dx": -1.5, "sh_dy": -3.0, "sh_ang": -12.0},
        {"root_dx": -1.4, "pelvis_dy": 2.0, "torso_sy": 0.980, "torso_sx": 1.015, "torso_ang":  -4.5, "head_ang": -2.5, "fleg_dx": -1.0, "bleg_dx": 1.0, "bleg_dy":  0.0, "farm_ang": 12.0, "sh_dx": -0.5, "sh_dy": -1.2, "sh_ang":  -5.0},
    ],
}

FIGHTER_CLIPS: Dict[str, List[Dict[str, float]]] = {
    # IDLE (4f): Smooth quadrature breathing + alert broadsword poise (0.0px foot slip!)
    "idle": [
        {"pelvis_dy":  0.00, "torso_sy": 1.000, "torso_sx": 1.000, "torso_ang":  0.0, "head_dy": -0.5, "head_ang": -1.2, "uarm_ang":  3.2, "farm_ang":  2.0, "wrist_ang":  3.5, "barm_ang":  3.0},
        {"pelvis_dy":  0.80, "torso_sy": 0.982, "torso_sx": 1.014, "torso_ang":  1.2, "head_dy":  0.0, "head_ang":  0.0, "uarm_ang":  0.0, "farm_ang":  0.0, "wrist_ang":  0.0, "barm_ang":  0.0},
        {"pelvis_dy":  0.00, "torso_sy": 1.000, "torso_sx": 1.000, "torso_ang":  0.0, "head_dy":  0.5, "head_ang":  1.2, "uarm_ang": -3.2, "farm_ang": -2.0, "wrist_ang": -3.5, "barm_ang": -3.0},
        {"pelvis_dy": -0.80, "torso_sy": 1.018, "torso_sx": 0.988, "torso_ang": -1.2, "head_dy":  0.0, "head_ang":  0.0, "uarm_ang":  0.0, "farm_ang":  0.0, "wrist_ang":  0.0, "barm_ang":  0.0},
    ],
    # WALK (6f): Brisk melee vanguard stride with alternating grounded sabaton tread
    "walk": [
        {"pelvis_dy":  0.8, "torso_ang": 3.5, "head_ang": -1.5, "fleg_dx":  4.6, "fleg_dy":  0.0, "fleg_ang": -16.0, "bleg_dx": -4.8, "bleg_dy": -0.8, "bleg_ang":  16.0, "uarm_ang": -12.0, "farm_ang": -6.0, "wrist_ang": -8.0, "barm_ang": -18.0},
        {"pelvis_dy":  2.0, "torso_sy": 0.978, "torso_sx": 1.018, "torso_ang": 4.5, "head_ang": -2.0, "fleg_dx":  1.8, "fleg_dy":  0.0, "fleg_ang":  -6.0, "bleg_dx": -1.5, "bleg_dy": -2.8, "bleg_ang":   8.0, "uarm_ang":  -6.0, "farm_ang": -3.0, "wrist_ang": -4.0, "barm_ang": -10.0},
        {"pelvis_dy": -1.4, "torso_sy": 1.016, "torso_ang": 2.2, "head_ang": -1.0, "fleg_dx": -1.8, "fleg_dy":  0.0, "fleg_ang":   6.0, "bleg_dx":  2.6, "bleg_dy": -3.2, "bleg_ang": -11.0, "uarm_ang":   6.0, "farm_ang":  3.0, "wrist_ang":  4.0, "barm_ang":   9.0},
        {"pelvis_dy":  0.8, "torso_ang": 3.5, "head_ang": -1.5, "fleg_dx": -4.8, "fleg_dy": -0.8, "fleg_ang":  16.0, "bleg_dx":  4.6, "bleg_dy":  0.0, "bleg_ang": -16.0, "uarm_ang":  13.0, "farm_ang":  6.0, "wrist_ang":  8.0, "barm_ang":  19.0},
        {"pelvis_dy":  2.0, "torso_sy": 0.978, "torso_sx": 1.018, "torso_ang": 4.5, "head_ang": -2.0, "fleg_dx": -1.8, "fleg_dy": -2.6, "fleg_ang":   9.0, "bleg_dx":  1.8, "bleg_dy":  0.0, "bleg_ang":  -6.0, "uarm_ang":   7.0, "farm_ang":  3.0, "wrist_ang":  4.0, "barm_ang":  11.0},
        {"pelvis_dy": -1.4, "torso_sy": 1.016, "torso_ang": 2.2, "head_ang": -1.0, "fleg_dx":  2.6, "fleg_dy": -2.8, "fleg_ang": -11.0, "bleg_dx": -1.8, "bleg_dy":  0.0, "bleg_ang":   6.0, "uarm_ang":  -5.0, "farm_ang": -2.0, "wrist_ang": -3.0, "barm_ang":  -8.0},
    ],
    # SWORDSLASH (6f): 1. Ready -> 2. Anticipation -> 3. Windup -> 4. Main Sweep -> 5. Follow-Through -> 6. Recovery
    "swordslash": [
        # f0: 1. Ready stance — step into combat stance, raise broadsword diagonally across chest
        {"root_dx": -0.6, "pelvis_dy":  1.0, "torso_ang":  1.5, "fleg_dx": -1.6, "bleg_dx": 2.2, "fsh_dx": 1.5, "fsh_dy": -1.0, "uarm_ang": -22.0, "farm_ang": -18.0, "wrist_ang": -26.0, "barm_ang": -10.0},
        # f1: 2. Anticipation — coil weight onto rear boot, draw broadsword upward toward shoulder guard
        {"root_dx": -2.0, "pelvis_dy":  2.0, "torso_sy": 0.975, "torso_sx": 1.02, "torso_ang": -7.5, "head_ang": 5.0, "fleg_dx": -2.4, "bleg_dx": 3.0, "fsh_dx": 2.5, "fsh_dy": -2.5, "uarm_ang": -52.0, "farm_ang": -26.0, "wrist_ang": -42.0, "barm_ang":  18.0},
        # f2: 3. Full Windup — high Vom-Tag overhead coil beside crimson plume (y>=6, x>=14, zero clip! Synced w/ slash f00)
        {"root_dx": -1.6, "pelvis_dy": -1.2, "torso_sy": 1.025, "torso_sx": 0.985, "torso_ang": -9.5, "head_ang": 7.0, "fleg_dx": -2.2, "bleg_dx": 3.4, "fsh_dx": 3.5, "fsh_dy": -3.5, "uarm_ang": -68.0, "farm_ang": -28.0, "wrist_ang": -48.0, "barm_ang":  26.0},
        # f3: 4. Main Blade Sweep — explosive forward lunge & wide diagonal broadsword cleave + crisp smear! (slash f01)
        {"root_dx": -1.2, "pelvis_dy":  2.2, "torso_sy": 0.960, "torso_sx": 1.035, "torso_ang": 12.0, "head_ang": -4.5, "fleg_dx": -2.4, "bleg_dx": 4.6, "fsh_dx": 2.5, "fsh_dy": -1.0, "uarm_ang": -36.0, "farm_ang":   6.0, "wrist_ang":   4.0, "barm_ang": -26.0, "swing_arc": 1.0},
        # f4: 5. Follow-through — deep committed low-right blade finish stopping cleanly above ground plane (slash f02)
        {"root_dx": -1.0, "pelvis_dy":  3.2, "torso_sy": 0.940, "torso_sx": 1.050, "torso_ang": 14.0, "head_ang": -5.5, "fleg_dx": -2.4, "bleg_dx": 5.0, "fsh_dx": 3.0, "fsh_dy":  0.0, "uarm_ang": -26.0, "farm_ang":  10.0, "wrist_ang":   6.0, "barm_ang": -30.0},
        # f5: 6. Recovery — crisp return toward ready guard (slash f03)
        {"root_dx":  0.0, "pelvis_dy":  1.0, "torso_sy": 0.990, "torso_sx": 1.008, "torso_ang":  4.5, "head_ang": -1.5, "fleg_dx": -1.0, "bleg_dx": 1.8, "fsh_dx": 1.0, "fsh_dy":  0.0, "uarm_ang": -10.0, "farm_ang":  -4.0, "wrist_ang":  -8.0, "barm_ang": -10.0},
    ],
    # HIT (2f): Defensive melee recoil keeping crimson plume & broadsword crisp
    "hit": [
        {"root_dx": -3.4, "pelvis_dy": 1.8, "torso_sy": 0.960, "torso_sx": 1.035, "torso_ang": -11.5, "head_ang": -6.5, "fleg_dx": -2.0, "bleg_dx": 2.0, "bleg_dy": -1.8, "uarm_ang": -24.0, "farm_ang": -18.0, "wrist_ang": -22.0, "barm_ang": 22.0},
        {"root_dx": -1.4, "pelvis_dy": 2.0, "torso_sy": 0.980, "torso_sx": 1.015, "torso_ang":  -5.0, "head_ang": -2.5, "fleg_dx": -1.0, "bleg_dx": 1.0, "bleg_dy":  0.0, "uarm_ang": -10.0, "farm_ang":  -8.0, "wrist_ang": -10.0, "barm_ang": 10.0},
    ],
}

RUNNER_CLIPS: Dict[str, List[Dict[str, float]]] = {
    # IDLE (4f): Agile crouched goblinoid breathing, twin daggers poised, locked 3/4 +X facing!
    "idle": [
        {"pelvis_dy":  0.00, "torso_sy": 1.000, "torso_sx": 1.000, "torso_ang":  1.0, "head_dy": -0.6, "head_ang": -1.5, "uarm_ang":  4.2, "farm_ang":  2.8, "barm_ang":  4.2},
        {"pelvis_dy":  0.95, "torso_sy": 0.978, "torso_sx": 1.020, "torso_ang":  2.6, "head_dy":  0.0, "head_ang":  0.0, "uarm_ang":  0.0, "farm_ang":  0.0, "barm_ang":  0.0},
        {"pelvis_dy":  0.00, "torso_sy": 1.000, "torso_sx": 1.000, "torso_ang":  1.0, "head_dy":  0.6, "head_ang":  1.5, "uarm_ang": -4.2, "farm_ang": -2.8, "barm_ang": -4.2},
        {"pelvis_dy": -0.95, "torso_sy": 1.022, "torso_sx": 0.985, "torso_ang": -0.6, "head_dy":  0.0, "head_ang":  0.0, "uarm_ang":  0.0, "farm_ang":  0.0, "barm_ang":  0.0},
    ],
    # RUN (6f): Fast forward-leaning sprint with wide alternating bare-foot contacts & high passing knee lift!
    "run": [
        {"pelvis_dy":  1.2, "torso_ang": 10.5, "head_ang": -4.5, "fleg_dx":  4.8, "fleg_dy":  0.0, "fleg_ang": -19.0, "bleg_dx": -4.8, "bleg_dy": -1.4, "bleg_ang":  19.0, "uarm_ang": -20.0, "farm_ang": -10.0, "barm_ang": -24.0},
        {"pelvis_dy":  2.8, "torso_sy": 0.965, "torso_sx": 1.030, "torso_ang": 12.5, "head_ang": -5.5, "fleg_dx":  2.0, "fleg_dy":  0.0, "fleg_ang":  -8.0, "bleg_dx": -1.2, "bleg_dy": -4.0, "bleg_ang":  10.0, "uarm_ang": -10.0, "farm_ang":  -5.0, "barm_ang": -12.0},
        {"pelvis_dy": -1.8, "torso_sy": 1.025, "torso_ang":  8.5, "head_ang": -3.5, "fleg_dx": -1.8, "fleg_dy":  0.0, "fleg_ang":   8.0, "bleg_dx":  2.2, "bleg_dy": -4.2, "bleg_ang": -10.0, "uarm_ang":   8.0, "farm_ang":   5.0, "barm_ang":  11.0},
        {"pelvis_dy":  1.2, "torso_ang": 10.5, "head_ang": -4.5, "fleg_dx": -4.8, "fleg_dy": -1.4, "fleg_ang":  19.0, "bleg_dx":  4.8, "bleg_dy":  1.0, "bleg_ang": -19.0, "uarm_ang":  20.0, "farm_ang":  10.0, "barm_ang":  26.0},
        {"pelvis_dy":  2.8, "torso_sy": 0.965, "torso_sx": 1.030, "torso_ang": 12.5, "head_ang": -5.5, "fleg_dx": -1.4, "fleg_dy": -3.8, "fleg_ang":  10.0, "bleg_dx":  2.0, "bleg_dy":  0.5, "bleg_ang":  -8.0, "uarm_ang":  11.0, "farm_ang":   6.0, "barm_ang":  14.0},
        {"pelvis_dy": -1.8, "torso_sy": 1.025, "torso_ang":  8.5, "head_ang": -3.5, "fleg_dx":  1.8, "fleg_dy": -4.0, "fleg_ang":  -8.0, "bleg_dx": -1.4, "bleg_dy":  0.0, "bleg_ang":   6.0, "uarm_ang":  -6.0, "farm_ang":  -3.0, "barm_ang":  -8.0},
    ],
    # DAGGERLUNGE (6f): Deep coil -> explosive twin-dagger forward thrust -> bite -> agile recoil -> crouch recovery!
    "daggerlunge": [
        # f0: Deep crouch compression — pull both curved daggers back beside ribs
        {"root_dx": -2.4, "pelvis_dy": 3.4, "torso_sy": 0.945, "torso_sx": 1.045, "torso_ang": -6.0, "head_ang":  5.0, "fleg_dx": -2.4, "bleg_dx": 2.8, "uarm_ang": 26.0, "farm_ang":  12.0, "barm_ang":  24.0},
        # f1: Spring launch — body uncoils forward, blades level toward target
        {"root_dx":  0.8, "pelvis_dy": 0.8, "torso_sy": 1.015, "torso_sx": 0.990, "torso_ang": 11.0, "head_ang": -4.0, "fleg_dx": -3.2, "bleg_dx": 4.2, "uarm_ang": -22.0, "farm_ang": -12.0, "barm_ang": -18.0},
        # f2: Full twin-dagger lunge thrust + crisp violet-steel lunge smear!
        {"root_dx":  3.6, "pelvis_dy": 2.2, "torso_sy": 0.960, "torso_sx": 1.035, "torso_ang": 17.5, "head_ang": -7.5, "fleg_dx": -3.6, "bleg_dx": 4.8, "fsh_dx": 2.8, "uarm_ang": -44.0, "farm_ang": -16.0, "barm_dx": 0.6, "barm_ang": -32.0, "lunge_smear": 1.0},
        # f3: Impact bite & follow-through — twin curved blades sunk forward-right
        {"root_dx":  4.0, "pelvis_dy": 3.2, "torso_sy": 0.945, "torso_sx": 1.045, "torso_ang": 19.0, "head_ang": -8.0, "fleg_dx": -3.6, "bleg_dx": 5.0, "fsh_dx": 3.2, "uarm_ang": -38.0, "farm_ang": -14.0, "barm_dx": 0.8, "barm_ang": -26.0},
        # f4: Agile push-off hop back toward guard distance
        {"root_dx":  1.2, "pelvis_dy": 0.4, "torso_sy": 1.010, "torso_ang":  8.0, "head_ang": -3.0, "fleg_dx": -1.8, "bleg_dx": 3.0, "uarm_ang": -14.0, "farm_ang":  -8.0, "barm_ang": -10.0},
        # f5: Settle back into crouched ready stance
        {"root_dx": -0.4, "pelvis_dy": 1.4, "torso_sy": 0.985, "torso_ang":  3.0, "head_ang": -1.0, "fleg_dx": -0.8, "bleg_dx": 1.4, "uarm_ang":   4.0, "farm_ang":   2.0, "barm_ang":   4.0},
    ],
    # HIT (2f): Sharp goblinoid recoil preserving locked +X facing & twin daggers
    "hit": [
        {"root_dx": -3.8, "pelvis_dy": 2.2, "torso_sy": 0.950, "torso_sx": 1.040, "torso_ang": -14.0, "head_ang": -7.5, "fleg_dx": -2.0, "bleg_dx": 2.2, "bleg_dy": -2.2, "uarm_ang": 22.0, "farm_ang": 14.0, "barm_ang": 28.0},
        {"root_dx": -1.6, "pelvis_dy": 2.4, "torso_sy": 0.980, "torso_sx": 1.015, "torso_ang":  -5.5, "head_ang": -3.0, "fleg_dx": -1.0, "bleg_dx": 1.0, "bleg_dy":  0.0, "uarm_ang":  8.0, "farm_ang":  5.0, "barm_ang": 12.0},
    ],
}

BRUTE_CLIPS: Dict[str, List[Dict[str, float]]] = {
    # IDLE (4f): Heavy armored juggernaut breathing & crusher-gauntlet heft (0.0px foot slip!)
    "idle": [
        {"pelvis_dy":  0.00, "torso_sy": 1.000, "torso_sx": 1.000, "torso_ang":  0.0, "head_dy": -0.45, "uarm_ang":  3.0, "farm_ang":  2.4, "barm_ang":  3.2},
        {"pelvis_dy":  0.85, "torso_sy": 0.980, "torso_sx": 1.018, "torso_ang":  1.2, "head_dy":  0.00, "uarm_ang":  0.0, "farm_ang":  0.0, "barm_ang":  0.0},
        {"pelvis_dy":  0.00, "torso_sy": 1.000, "torso_sx": 1.000, "torso_ang":  0.0, "head_dy":  0.45, "uarm_ang": -3.0, "farm_ang": -2.4, "barm_ang": -3.2},
        {"pelvis_dy": -0.85, "torso_sy": 1.020, "torso_sx": 0.986, "torso_ang": -1.2, "head_dy":  0.00, "uarm_ang":  0.0, "farm_ang":  0.0, "barm_ang":  0.0},
    ],
    # MARCH (6f): Slow, earth-shaking heavy siege march with deliberate weight transfer
    "march": [
        {"pelvis_dy":  1.2, "torso_ang": 3.5, "head_ang": -1.2, "fleg_dx":  4.2, "fleg_dy":  0.0, "fleg_ang": -14.0, "bleg_dx": -4.4, "bleg_dy": -0.8, "bleg_ang":  14.0, "uarm_ang": -12.0, "farm_ang": -6.0, "barm_ang": -16.0},
        {"pelvis_dy":  3.0, "torso_sy": 0.966, "torso_sx": 1.028, "torso_ang": 5.0, "head_ang": -2.0, "fleg_dx":  1.6, "fleg_dy":  0.0, "fleg_ang":  -6.0, "bleg_dx": -1.4, "bleg_dy": -2.8, "bleg_ang":   7.0, "uarm_ang":  -6.0, "farm_ang": -3.0, "barm_ang":  -9.0},
        {"pelvis_dy": -1.4, "torso_sy": 1.018, "torso_ang": 2.2, "head_ang": -0.8, "fleg_dx": -1.8, "fleg_dy":  0.0, "fleg_ang":   6.0, "bleg_dx":  2.4, "bleg_dy": -3.2, "bleg_ang": -10.0, "uarm_ang":   6.0, "farm_ang":  3.0, "barm_ang":   8.0},
        {"pelvis_dy":  1.2, "torso_ang": 3.5, "head_ang": -1.2, "fleg_dx": -4.4, "fleg_dy": -0.8, "fleg_ang":  14.0, "bleg_dx":  4.2, "bleg_dy":  0.0, "bleg_ang": -14.0, "uarm_ang":  12.0, "farm_ang":  6.0, "barm_ang":  16.0},
        {"pelvis_dy":  3.0, "torso_sy": 0.966, "torso_sx": 1.028, "torso_ang": 5.0, "head_ang": -2.0, "fleg_dx": -1.4, "fleg_dy": -2.6, "fleg_ang":   7.0, "bleg_dx":  1.6, "bleg_dy":  0.0, "bleg_ang":  -6.0, "uarm_ang":   6.0, "farm_ang":  3.0, "barm_ang":   9.0},
        {"pelvis_dy": -1.4, "torso_sy": 1.018, "torso_ang": 2.2, "head_ang": -0.8, "fleg_dx":  2.4, "fleg_dy": -2.8, "fleg_ang": -10.0, "bleg_dx": -1.8, "bleg_dy":  0.0, "bleg_ang":   6.0, "uarm_ang":  -5.0, "farm_ang": -2.0, "barm_ang":  -7.0},
    ],
    # GAUNTLETCRUSH (6f): Heavy crouch coil -> massive two-fisted overhead raise -> devastating downward crush -> follow-through -> recovery
    "gauntletcrush": [
        # f0: Heavy crouch anticipation — brace wide, cock iron crusher gauntlets upward
        {"root_dx": -1.6, "pelvis_dy":  2.4, "torso_sy": 0.960, "torso_sx": 1.035, "torso_ang": -6.0, "head_ang":  4.0, "fleg_dx": -2.0, "bleg_dx": 2.4, "uarm_ang": -32.0, "farm_ang": -24.0, "barm_ang": -28.0},
        # f1: Heavy arm raise — haul massive gauntlets up beside bucket helm
        {"root_dx": -2.4, "pelvis_dy":  0.8, "torso_sy": 1.018, "torso_sx": 0.990, "torso_ang": -8.5, "head_ang":  6.0, "fleg_dx": -2.0, "bleg_dx": 2.8, "fsh_dy": -1.0, "uarm_ang": -88.0, "farm_ang": -32.0, "barm_dy": -1.5, "barm_ang": -76.0},
        # f2: Full overhead apex — twin iron crusher fists poised high above bucket helm (strictly inside 128x128!)
        {"root_dx": -2.6, "pelvis_dy":  1.2, "torso_sy": 1.028, "torso_sx": 0.985, "torso_ang": -9.5, "head_ang":  7.0, "fleg_dx": -2.0, "bleg_dx": 3.2, "fsh_dy": -1.5, "uarm_ang": -102.0, "farm_ang": -34.0, "barm_dy": -2.0, "barm_ang": -90.0},
        # f3: Devastating downward crusher slam + crisp siege-orange impact arc!
        {"root_dx":  0.4, "pelvis_dy":  3.2, "torso_sy": 0.940, "torso_sx": 1.050, "torso_ang": 13.5, "head_ang": -5.0, "fleg_dx": -2.4, "bleg_dx": 4.2, "fsh_dx": 2.0, "fsh_dy":  1.5, "uarm_ang": -48.0, "farm_ang": -12.0, "barm_dx":  1.5, "barm_ang": -32.0, "crush_arc": 1.0},
        # f4: Maximum seismic follow-through & mass compression — gauntlets driven low front-right
        {"root_dx":  0.8, "pelvis_dy":  4.4, "torso_sy": 0.918, "torso_sx": 1.065, "torso_ang": 15.5, "head_ang": -6.0, "fleg_dx": -2.4, "bleg_dx": 4.6, "fsh_dx": 2.0, "fsh_dy":  2.5, "uarm_ang": -32.0, "farm_ang":   4.0, "barm_dx":  2.0, "barm_ang": -28.0},
        # f5: Deliberate heavy recovery haul back to siege stance
        {"root_dx":  0.8, "pelvis_dy":  1.4, "torso_sy": 0.985, "torso_sx": 1.012, "torso_ang":  5.0, "head_ang": -2.0, "fleg_dx": -1.0, "bleg_dx": 1.8, "fsh_dx": 1.2, "uarm_ang":  -8.0, "farm_ang":   6.0, "barm_ang":  -6.0},
    ],
    # HIT (2f): Stubborn heavy-armor flinch
    "hit": [
        {"root_dx": -2.8, "pelvis_dy": 2.0, "torso_sy": 0.960, "torso_sx": 1.035, "torso_ang": -9.5, "head_ang": -5.0, "fleg_dx": -1.8, "bleg_dx": 2.0, "bleg_dy": -1.4, "uarm_ang": -18.0, "farm_ang": -14.0, "barm_ang": 18.0},
        {"root_dx": -1.2, "pelvis_dy": 2.2, "torso_sy": 0.982, "torso_sx": 1.015, "torso_ang": -4.0, "head_ang": -2.0, "fleg_dx": -0.8, "bleg_dx": 1.0, "bleg_dy":  0.0, "uarm_ang":  -8.0, "farm_ang":  -6.0, "barm_ang":  8.0},
    ],
}

BOSS_CLIPS: Dict[str, List[Dict[str, float]]] = {
    # IDLEMARCH (6f @ 12 FPS, cyclic): Commanding dread warlord march with grounded alternating sabaton tread
    "idlemarch": [
        {"pelvis_dy":  1.6, "torso_ang": 2.8, "head_ang": -1.2, "fleg_dx":  6.8, "fleg_dy":  0.0, "fleg_ang": -13.0, "bleg_dx": -7.0, "bleg_dy": -1.6, "bleg_ang":  13.0, "uarm_ang": -10.0, "farm_ang": -5.0, "wrist_ang": -6.0, "barm_ang": -14.0},
        {"pelvis_dy":  4.2, "torso_sy": 0.972, "torso_sx": 1.022, "torso_ang": 4.0, "head_ang": -1.8, "fleg_dx":  2.6, "fleg_dy":  0.0, "fleg_ang":  -5.0, "bleg_dx": -2.2, "bleg_dy": -4.8, "bleg_ang":   7.0, "uarm_ang":  -5.0, "farm_ang": -2.5, "wrist_ang": -3.0, "barm_ang":  -8.0},
        {"pelvis_dy": -2.2, "torso_sy": 1.018, "torso_ang": 1.8, "head_ang": -0.8, "fleg_dx": -2.8, "fleg_dy":  0.0, "fleg_ang":   6.0, "bleg_dx":  4.0, "bleg_dy": -5.4, "bleg_ang": -10.0, "uarm_ang":   5.0, "farm_ang":  2.5, "wrist_ang":  3.0, "barm_ang":   8.0},
        {"pelvis_dy":  1.6, "torso_ang": 2.8, "head_ang": -1.2, "fleg_dx": -7.0, "fleg_dy": -1.6, "fleg_ang":  13.0, "bleg_dx":  6.8, "bleg_dy":  0.0, "bleg_ang": -13.0, "uarm_ang":  10.0, "farm_ang":  5.0, "wrist_ang":  6.0, "barm_ang":  15.0},
        {"pelvis_dy":  4.2, "torso_sy": 0.972, "torso_sx": 1.022, "torso_ang": 4.0, "head_ang": -1.8, "fleg_dx": -2.4, "fleg_dy": -4.6, "fleg_ang":   7.0, "bleg_dx":  2.6, "bleg_dy":  0.0, "bleg_ang":  -5.0, "uarm_ang":   6.0, "farm_ang":  3.0, "wrist_ang":  3.5, "barm_ang":   9.0},
        {"pelvis_dy": -2.2, "torso_sy": 1.018, "torso_ang": 1.8, "head_ang": -0.8, "fleg_dx":  4.0, "fleg_dy": -5.0, "fleg_ang": -10.0, "bleg_dx": -2.8, "bleg_dy":  0.0, "bleg_ang":   6.0, "uarm_ang":  -4.0, "farm_ang": -2.0, "wrist_ang": -2.5, "barm_ang":  -7.0},
    ],
    # CLEAVESLAM (8f @ 12 FPS): Anticipation -> Windup -> High Rise -> Overhead Commitment -> Cleave -> Slam Impact -> Settle -> Recovery
    "cleaveslam": [
        # f0: Menacing anticipation crouch & war-cleaver lift across magenta core
        {"root_dx": -2.0, "pelvis_dy":  3.2, "torso_sy": 0.968, "torso_sx": 1.025, "torso_ang": -4.5, "head_ang":  3.0, "fleg_dx": -3.0, "bleg_dx": 3.5, "uarm_ang": -20.0, "farm_ang": -14.0, "wrist_ang": -22.0, "barm_ang": 12.0},
        # f1: Heavy windup haul — coil torso back, swing colossal cleaver past spiked shoulder
        {"root_dx": -3.6, "pelvis_dy":  1.8, "torso_sy": 0.985, "torso_sx": 1.012, "torso_ang": -7.5, "head_ang":  5.0, "fleg_dx": -3.8, "bleg_dx": 4.6, "fsh_dx": 2.5, "fsh_dy": -2.5, "uarm_ang": -44.0, "farm_ang": -22.0, "wrist_ang": -36.0, "barm_ang": 22.0},
        # f2: High overhead rise — war-cleaver climbs above crescent horns (strictly inside 256x256!)
        {"root_dx": -3.0, "pelvis_dy": -1.6, "torso_sy": 1.020, "torso_sx": 0.988, "torso_ang": -9.5, "head_ang":  6.5, "fleg_dx": -3.6, "bleg_dx": 5.2, "fsh_dx": 4.0, "fsh_dy": -4.5, "uarm_ang": -60.0, "farm_ang": -26.0, "wrist_ang": -44.0, "barm_ang": 30.0},
        # f3: Full overhead commitment (apex hold) — towering dread silhouette before strike!
        {"root_dx": -2.4, "pelvis_dy": -2.8, "torso_sy": 1.032, "torso_sx": 0.980, "torso_ang": -10.5,"head_ang":  7.5, "fleg_dx": -3.4, "bleg_dx": 5.6, "fsh_dx": 5.0, "fsh_dy": -5.5, "uarm_ang": -68.0, "farm_ang": -28.0, "wrist_ang": -48.0, "barm_ang": 34.0},
        # f4: Ferocious descending runic cleave + colossal magenta/orchid arc smear!
        {"root_dx":  3.2, "pelvis_dy":  3.6, "torso_sy": 0.955, "torso_sx": 1.038, "torso_ang": 12.5, "head_ang": -4.5, "fleg_dx": -3.6, "bleg_dx": 7.0, "fsh_dx": 5.0, "fsh_dy":  1.5, "uarm_ang": -38.0, "farm_ang":  -8.0, "wrist_ang": -12.0, "barm_ang": -26.0, "cleave_arc": 1.0},
        # f5: Cataclysmic ground slam impact (synced w/ vfx_boss_slam_telegraph f05 eruption!)
        {"root_dx":  4.6, "pelvis_dy":  6.2, "torso_sy": 0.928, "torso_sx": 1.058, "torso_ang": 16.0, "head_ang": -6.5, "fleg_dx": -3.8, "bleg_dx": 7.8, "fsh_dx": 6.0, "fsh_dy":  3.0, "uarm_ang": -24.0, "farm_ang":   2.0, "wrist_ang":  -2.0, "barm_ang": -34.0},
        # f6: Heavy impact follow-through & tremor settle
        {"root_dx":  3.8, "pelvis_dy":  4.8, "torso_sy": 0.945, "torso_sx": 1.042, "torso_ang": 13.0, "head_ang": -5.0, "fleg_dx": -3.2, "bleg_dx": 6.4, "fsh_dx": 4.5, "fsh_dy":  2.0, "uarm_ang": -20.0, "farm_ang":   4.0, "wrist_ang":   2.0, "barm_ang": -24.0},
        # f7: Deliberate warlord recovery haul back toward march carry
        {"root_dx":  1.4, "pelvis_dy":  2.0, "torso_sy": 0.985, "torso_sx": 1.012, "torso_ang":  5.0, "head_ang": -2.0, "fleg_dx": -1.4, "bleg_dx": 2.8, "fsh_dx": 1.8, "fsh_dy":  0.5, "uarm_ang":  -8.0, "farm_ang":   2.0, "wrist_ang":   2.0, "barm_ang": -10.0},
    ],
    # HIT (2f @ 12 FPS): Heavy warlord recoil keeping horns & runic war-cleaver commanding
    "hit": [
        {"root_dx": -4.6, "pelvis_dy": 3.0, "torso_sy": 0.955, "torso_sx": 1.038, "torso_ang": -10.0, "head_ang": -5.5, "fleg_dx": -2.8, "bleg_dx": 3.0, "bleg_dy": -2.4, "uarm_ang": -18.0, "farm_ang": -12.0, "wrist_ang": -14.0, "barm_ang": 24.0},
        {"root_dx": -2.0, "pelvis_dy": 2.8, "torso_sy": 0.982, "torso_sx": 1.015, "torso_ang":  -4.2, "head_ang": -2.2, "fleg_dx": -1.2, "bleg_dx": 1.4, "bleg_dy":  0.0, "uarm_ang":  -8.0, "farm_ang":  -5.0, "wrist_ang":  -6.0, "barm_ang": 10.0},
    ],
}


# ============================================================================
# PHASE 7 — 4 COMBAT VFX ARCHETYPES (24 REAL RGBA FRAMES @ 12 FPS)
# ============================================================================

SLASH_VFX_PAL = np.array(
    [
        hex_to_rgb("#11151C"),
        hex_to_rgb("#FFFFFF"),
        hex_to_rgb("#FFE4E6"),
        hex_to_rgb("#FB7185"),
        hex_to_rgb("#E63946"),
        hex_to_rgb("#B91C1C"),
        hex_to_rgb("#7F1D1D"),
    ],
    dtype=np.float32,
)

SHIELD_VFX_PAL = np.array(
    [
        hex_to_rgb("#11151C"),
        hex_to_rgb("#FFFFFF"),
        hex_to_rgb("#DBEAFE"),
        hex_to_rgb("#60A5FA"),
        hex_to_rgb("#2E7CF6"),
        hex_to_rgb("#1D4ED8"),
        hex_to_rgb("#1E3A8A"),
    ],
    dtype=np.float32,
)

PUFF_VFX_PAL = np.array(
    [
        hex_to_rgb("#11151C"),
        hex_to_rgb("#FFFBEB"),
        hex_to_rgb("#E2E8F0"),
        hex_to_rgb("#CBD5E1"),
        hex_to_rgb("#94A3B8"),
        hex_to_rgb("#64748B"),
        hex_to_rgb("#475569"),
    ],
    dtype=np.float32,
)

BOSS_VFX_PAL = np.array(
    [
        hex_to_rgb("#11151C"),
        hex_to_rgb("#FCE7F3"),
        hex_to_rgb("#F5D0FE"),
        hex_to_rgb("#E879F9"),
        hex_to_rgb("#D946EF"),
        hex_to_rgb("#A21CAF"),
        hex_to_rgb("#7E22CE"),
        hex_to_rgb("#581C87"),
        hex_to_rgb("#3B0764"),
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
    outline_w: int = 5,
) -> None:
    pts = []
    for i in range(8):
        ang = math.radians(rot_deg + i * 45.0)
        r = outer_r if (i % 2 == 0) else inner_r
        pts.append((cx + r * math.cos(ang), cy + r * math.sin(ang)))
    dr.polygon(pts, fill=fill_rgba, outline=(*INK_RGB, 255), width=outline_w)


def build_vfx_fighter_slash_arc() -> List[Image.Image]:
    """4 frames @ 12 FPS, 128x128 RGBA — white/crimson crescent sweep synced to Fighter SwordSlash f02..f05."""
    frames: List[Image.Image] = []
    cx, cy = 204.0, 268.0
    for f in range(4):
        c512 = Image.new("RGBA", (512, 512), (0, 0, 0, 0))
        dr = ImageDraw.Draw(c512)
        if f == 0:
            # Ignition crescent at top of slash
            c512 = draw_tapered_cel_smear(
                c512, (cx, cy), radius_outer=168.0, max_thick=36.0, deg_start=-98.0, deg_end=-18.0,
                outer_rgb=hex_to_rgb("#E63946"), core_rgb=hex_to_rgb("#FFFFFF"), outline_w=5
            )
            dr = ImageDraw.Draw(c512)
            draw_four_point_star(dr, cx + 152.0, cy - 52.0, 42.0, 12.0, 18.0, (*hex_to_rgb("#FFE4E6"), 255), outline_w=4)
        elif f == 1:
            # Peak full crescent blade sweep (aligned with Fighter SwordSlash f03)
            c512 = draw_tapered_cel_smear(
                c512, (cx, cy), radius_outer=206.0, max_thick=64.0, deg_start=-84.0, deg_end=48.0,
                outer_rgb=hex_to_rgb("#E63946"), core_rgb=hex_to_rgb("#FFFFFF"), outline_w=6
            )
            dr = ImageDraw.Draw(c512)
            draw_four_point_star(dr, cx + 196.0, cy + 42.0, 56.0, 15.0, 28.0, (*hex_to_rgb("#FFFFFF"), 255), outline_w=4)
        elif f == 2:
            # Splitting crimson follow-through ribbons (aligned with Fighter SwordSlash f04)
            c512 = draw_tapered_cel_smear(
                c512, (cx, cy), radius_outer=224.0, max_thick=34.0, deg_start=-32.0, deg_end=56.0,
                outer_rgb=hex_to_rgb("#B91C1C"), core_rgb=hex_to_rgb("#FB7185"), outline_w=5
            )
            c512 = draw_tapered_cel_smear(
                c512, (cx, cy), radius_outer=172.0, max_thick=22.0, deg_start=-12.0, deg_end=46.0,
                outer_rgb=hex_to_rgb("#E63946"), core_rgb=hex_to_rgb("#FB7185"), outline_w=4
            )
        else:
            # Clean dark-crimson dissipating tail wisps (zero bright pixels!)
            for a0, a1, rad, thk in [(-6.0, 24.0, 232.0, 18.0), (30.0, 58.0, 226.0, 16.0), (14.0, 40.0, 184.0, 14.0)]:
                c512 = draw_tapered_cel_smear(
                    c512, (cx, cy), radius_outer=rad, max_thick=thk, deg_start=a0, deg_end=a1,
                    outer_rgb=hex_to_rgb("#7F1D1D"), core_rgb=hex_to_rgb("#B91C1C"), outline_w=4
                )
        frames.append(finalize_cel_frame(c512, SLASH_VFX_PAL, out_dim=128))
    return frames


def build_vfx_guard_shield_block() -> List[Image.Image]:
    """4 frames @ 12 FPS, 128x128 RGBA — compact steel-blue impact ring synced to Guard ShieldBrace f02..f05."""
    frames: List[Image.Image] = []
    cx, cy = 264.0, 256.0
    for f in range(4):
        c512 = Image.new("RGBA", (512, 512), (0, 0, 0, 0))
        dr = ImageDraw.Draw(c512)
        if f == 0:
            # Crisp defensive contact burst on shield face
            dr.ellipse([cx - 64, cy - 92, cx + 64, cy + 92], fill=(*hex_to_rgb("#2E7CF6"), 255), outline=(*INK_RGB, 255), width=6)
            dr.ellipse([cx - 38, cy - 58, cx + 38, cy + 58], fill=(*hex_to_rgb("#DBEAFE"), 255))
            draw_four_point_star(dr, cx + 10, cy, 94.0, 22.0, 0.0, (*hex_to_rgb("#FFFFFF"), 255), outline_w=5)
        elif f == 1:
            # Expanding 3/4 tilted steel-cobalt shock ring + deflection ticks
            dr.ellipse([cx - 108, cy - 152, cx + 108, cy + 152], outline=(*INK_RGB, 255), width=26)
            dr.ellipse([cx - 102, cy - 146, cx + 102, cy + 146], outline=(*hex_to_rgb("#2E7CF6"), 255), width=16)
            dr.ellipse([cx - 96, cy - 140, cx + 96, cy + 140], outline=(*hex_to_rgb("#DBEAFE"), 255), width=6)
            draw_four_point_star(dr, cx + 16, cy, 62.0, 15.0, 45.0, (*hex_to_rgb("#60A5FA"), 255), outline_w=4)
        elif f == 2:
            # Segmented steel-blue shockwave arcs opening outward
            for s_deg, e_deg in [(-75, -15), (15, 75), (115, 165), (195, 245)]:
                dr.arc([cx - 142, cy - 188, cx + 142, cy + 188], start=s_deg, end=e_deg, fill=(*INK_RGB, 255), width=22)
                dr.arc([cx - 142, cy - 188, cx + 142, cy + 188], start=s_deg + 3, end=e_deg - 3, fill=(*hex_to_rgb("#2E7CF6"), 255), width=11)
                dr.arc([cx - 142, cy - 188, cx + 142, cy + 188], start=s_deg + 8, end=e_deg - 8, fill=(*hex_to_rgb("#60A5FA"), 255), width=5)
        else:
            # Muted deep-cobalt dissipating arcs (zero bright pixels!)
            for s_deg, e_deg in [(-62, -28), (28, 62), (130, 156), (204, 230)]:
                dr.arc([cx - 166, cy - 208, cx + 166, cy + 208], start=s_deg, end=e_deg, fill=(*INK_RGB, 255), width=16)
                dr.arc([cx - 166, cy - 208, cx + 166, cy + 208], start=s_deg + 3, end=e_deg - 3, fill=(*hex_to_rgb("#1E3A8A"), 255), width=7)
        frames.append(finalize_cel_frame(c512, SHIELD_VFX_PAL, out_dim=128))
    return frames


def build_vfx_unit_hit_puff() -> List[Image.Image]:
    """6 frames @ 12 FPS, 128x128 RGBA — low-noise cel dust/impact puff, ring-opening center, zero gore."""
    frames: List[Image.Image] = []
    cx, cy = 256.0, 256.0
    puff_dirs = [-145.0, -75.0, -15.0, 45.0, 115.0, 175.0]
    for f in range(6):
        c512 = Image.new("RGBA", (512, 512), (0, 0, 0, 0))
        dr = ImageDraw.Draw(c512)
        if f == 0:
            # Compact crisp impact pop
            draw_four_point_star(dr, cx, cy, 78.0, 22.0, 22.0, (*hex_to_rgb("#E2E8F0"), 255), outline_w=5)
            draw_four_point_star(dr, cx, cy, 42.0, 13.0, 67.0, (*hex_to_rgb("#FFFBEB"), 255), outline_w=4)
        elif f in (1, 2, 3, 4):
            # Expanding ring of 6 cel dust puffs so center opens cleanly (actor silhouette stays visible!)
            dist = 54.0 + f * 28.0
            rad = [38.0, 44.0, 36.0, 24.0][f - 1]
            fill_main = [hex_to_rgb("#E2E8F0"), hex_to_rgb("#CBD5E1"), hex_to_rgb("#94A3B8"), hex_to_rgb("#64748B")][f - 1]
            fill_hi = [hex_to_rgb("#FFFBEB"), hex_to_rgb("#E2E8F0"), hex_to_rgb("#CBD5E1"), hex_to_rgb("#94A3B8")][f - 1]
            for idx, ang_deg in enumerate(puff_dirs):
                a = math.radians(ang_deg + f * 4.0)
                px = cx + math.cos(a) * dist
                py = cy + math.sin(a) * dist * 0.85
                r_i = rad * (1.0 if idx % 2 == 0 else 0.82)
                dr.ellipse([px - r_i, py - r_i, px + r_i, py + r_i], fill=(*fill_main, 255), outline=(*INK_RGB, 255), width=5)
                if f <= 2:
                    dr.ellipse([px - r_i * 0.55, py - r_i * 0.65, px + r_i * 0.35, py + r_i * 0.25], fill=(*fill_hi, 255))
        else:
            # f5: Tiny dark-slate dissipating dust motes (zero bright residue!)
            dist = 178.0
            for ang_deg in puff_dirs[::2]:
                a = math.radians(ang_deg + 20.0)
                px = cx + math.cos(a) * dist
                py = cy + math.sin(a) * dist * 0.85
                dr.ellipse([px - 14, py - 14, px + 14, py + 14], fill=(*hex_to_rgb("#475569"), 255), outline=(*INK_RGB, 255), width=4)
        frames.append(finalize_cel_frame(c512, PUFF_VFX_PAL, out_dim=128))
    return frames


def draw_iso_ellipse_ring(
    dr: ImageDraw.ImageDraw,
    cx: float,
    cy: float,
    rx: float,
    ry: float,
    ring_w: int,
    fill_rgb: Tuple[int, int, int],
    outline_w: int = 5,
) -> None:
    dr.ellipse([cx - rx, cy - ry, cx + rx, cy + ry], outline=(*INK_RGB, 255), width=ring_w + outline_w * 2)
    dr.ellipse([cx - rx, cy - ry, cx + rx, cy + ry], outline=(*fill_rgb, 255), width=ring_w)


def build_vfx_boss_slam_telegraph() -> List[Image.Image]:
    """
    10 frames @ 12 FPS, 256x256 RGBA — independent ground-plane violet-magenta concentric rune telegraph & shockwave.
      f00..f04 (5f): Restrained concentric rune warning phase (never implies premature damage, monotonic lock-in)
      f05..f07 (3f): Active ground slam shockwave eruption (synced with Boss CleaveSlam f05)
      f08..f09 (2f): Clean dark-violet dissipation (0 persistent bright residue)
    """
    frames: List[Image.Image] = []
    cx, cy = 512.0, 580.0
    for f in range(10):
        c1024 = Image.new("RGBA", (1024, 1024), (0, 0, 0, 0))
        dr = ImageDraw.Draw(c1024)

        if f <= 4:
            # Phase A (f00..f04): Restrained 2:1 isometric rune warning ring locking inward monotonically
            u = f / 4.0
            rx_outer = 340.0
            ry_outer = 166.0
            rx_inner = 300.0 - u * 165.0  # Closing timing indicator ring!
            ry_inner = rx_inner * 0.49
            col_outer = hex_to_rgb("#7E22CE") if f < 3 else hex_to_rgb("#A21CAF")
            col_inner = hex_to_rgb("#A21CAF") if f < 4 else hex_to_rgb("#D946EF")

            draw_iso_ellipse_ring(dr, cx, cy, rx_outer, ry_outer, ring_w=14, fill_rgb=col_outer, outline_w=5)
            draw_iso_ellipse_ring(dr, cx, cy, rx_inner, ry_inner, ring_w=10, fill_rgb=col_inner, outline_w=4)

            # Subtle flat inscribed isometric 6-point Warlord rune motif inside the warning ring (matching ART-002!)
            tri_col = hex_to_rgb("#581C87") if f < 3 else hex_to_rgb("#7E22CE")
            for base_rot in (0.0, 60.0):
                pts_tri = []
                for k_v in range(3):
                    a_v = math.radians(base_rot + k_v * 120.0 + f * 6.0)
                    pts_tri.append((cx + math.cos(a_v) * (rx_outer * 0.76), cy + math.sin(a_v) * (ry_outer * 0.76)))
                pts_tri.append(pts_tri[0])
                dr.line(pts_tri, fill=(*INK_RGB, 255), width=11)
                dr.line(pts_tri, fill=(*tri_col, 255), width=5)

            # 6 subtle rotating isometric rune ticks between rings (flat on ground, zero vertical spikes!)
            for k in range(6):
                ang = math.radians(k * 60.0 + f * 9.0)
                p0 = (cx + math.cos(ang) * (rx_outer - 28.0), cy + math.sin(ang) * (ry_outer - 14.0))
                p1 = (cx + math.cos(ang) * (rx_outer + 18.0), cy + math.sin(ang) * (ry_outer + 9.0))
                dr.line([p0, p1], fill=(*INK_RGB, 255), width=14)
                dr.line([p0, p1], fill=(*col_inner, 255), width=6)

        elif f <= 7:
            # Phase B (f05..f07): Active shockwave eruption!
            k_imp = f - 5  # 0, 1, 2
            rx = [356.0, 418.0, 456.0][k_imp]
            ry = rx * 0.49
            ring_col = [hex_to_rgb("#F5D0FE"), hex_to_rgb("#D946EF"), hex_to_rgb("#A21CAF")][k_imp]
            core_col = [hex_to_rgb("#FCE7F3"), hex_to_rgb("#F5D0FE"), hex_to_rgb("#D946EF")][k_imp]

            draw_iso_ellipse_ring(dr, cx, cy, rx, ry, ring_w=28 - k_imp * 6, fill_rgb=ring_col, outline_w=6)
            if k_imp <= 1:
                draw_iso_ellipse_ring(dr, cx, cy, rx * 0.62, ry * 0.62, ring_w=18 - k_imp * 4, fill_rgb=core_col, outline_w=5)

            # Crisp cel-shaded upward shockwave crown teeth around the ground impact rim
            n_spikes = 8
            spike_h = [118.0, 84.0, 44.0][k_imp]
            for s_i in range(n_spikes):
                ang = math.radians(s_i * (360.0 / n_spikes) + 22.5)
                bx = cx + math.cos(ang) * (rx * 0.88)
                by = cy + math.sin(ang) * (ry * 0.88)
                pts = [(bx - 22.0, by), (bx, by - spike_h), (bx + 22.0, by)]
                dr.polygon(pts, fill=(*core_col, 255), outline=(*INK_RGB, 255), width=5)

        else:
            # Phase C (f08..f09): Clean dark-violet dissipation (zero bright residue on f09!)
            k_dis = f - 8  # 0, 1
            rx = 468.0 + k_dis * 10.0
            ry = rx * 0.49
            col_d = hex_to_rgb("#581C87") if k_dis == 0 else hex_to_rgb("#3B0764")
            w_d = 12 if k_dis == 0 else 8
            for s_i in range(6):
                s_deg = s_i * 60 + 12 + k_dis * 8
                e_deg = s_deg + (28 if k_dis == 0 else 16)
                dr.arc([cx - rx, cy - ry, cx + rx, cy + ry], start=s_deg, end=e_deg, fill=(*INK_RGB, 255), width=w_d + 8)
                dr.arc([cx - rx, cy - ry, cx + rx, cy + ry], start=s_deg + 2, end=e_deg - 2, fill=(*col_d, 255), width=w_d)

        frames.append(finalize_cel_frame(c1024, BOSS_VFX_PAL, out_dim=256, min_island_px=8))
    return frames


# ============================================================================
# SHEET PACKING, RIG DEBUG PLATES & STRICT FRAME ASSERTIONS
# ============================================================================

def pack_fixed_grid(frames: List[Image.Image], cols: int, rows: int, cell_w: int, cell_h: int) -> Image.Image:
    assert len(frames) <= cols * rows
    sheet = Image.new("RGBA", (cols * cell_w, rows * cell_h), (0, 0, 0, 0))
    for idx, im in enumerate(frames):
        assert im.size == (cell_w, cell_h)
        r, c = divmod(idx, cols)
        sheet.paste(im, (c * cell_w, r * cell_h))
    return sheet


def save_rig_debug_sheet(
    title: str,
    rig: Dict[str, Tuple[Image.Image, Tuple[float, float]]],
    sample_frames: List[Image.Image],
    out_path: Path,
) -> None:
    board = Image.new("RGB", (1480, 560), (20, 26, 38))
    dr = ImageDraw.Draw(board)
    try:
        f_b = ImageFont.truetype("C:/Windows/Fonts/segoeuib.ttf", 20)
        f_s = ImageFont.truetype("C:/Windows/Fonts/segoeui.ttf", 13)
    except Exception:
        f_b = ImageFont.load_default()
        f_s = ImageFont.load_default()

    dr.text((20, 14), f"ART-004 RIG & KEY-POSE DEBUG SHEET — {title}", font=f_b, fill=(254, 240, 138))
    dr.text((20, 42), "Top row: Extracted & healed rigid cutout parts + FK pivots (red crosshairs). Bottom row: 4x/2x nearest-neighbor key poses.", font=f_s, fill=(203, 213, 225))

    for idx, (part_name, (pim, piv)) in enumerate(rig.items()):
        x0 = 20 + idx * 178
        y0 = 74
        dr.rounded_rectangle([x0, y0, x0 + 168, y0 + 210], radius=8, fill=(32, 42, 58), outline=(71, 85, 105), width=2)
        scale = min(150.0 / pim.width, 168.0 / pim.height)
        tw, th = max(1, int(round(pim.width * scale))), max(1, int(round(pim.height * scale)))
        thumb = pim.resize((tw, th), Image.Resampling.BILINEAR)
        px = x0 + 84 - tw // 2
        py = y0 + 18
        board.paste(thumb, (px, py), thumb)
        jx = int(round(px + piv[0] * scale))
        jy = int(round(py + piv[1] * scale))
        dr.line([(jx - 6, jy), (jx + 6, jy)], fill=(248, 113, 113), width=2)
        dr.line([(jx, jy - 6), (jx, jy + 6)], fill=(248, 113, 113), width=2)
        dr.text((x0 + 8, y0 + 188), part_name, font=f_s, fill=(248, 250, 252))

    for idx, fim in enumerate(sample_frames[:6]):
        x0 = 20 + idx * 240
        y0 = 304
        dr.rounded_rectangle([x0, y0, x0 + 226, y0 + 236], radius=8, fill=(32, 42, 58), outline=(71, 85, 105), width=2)
        z = fim.resize((192, 192), Image.Resampling.NEAREST)
        board.paste(z, (x0 + 17, y0 + 14), z)
        # Pivot crosshair at (0.50, 0.08) -> (96, 177) inside 192x192
        dr.line([(x0 + 17, y0 + 14 + 177), (x0 + 209, y0 + 14 + 177)], fill=(56, 189, 248), width=1)
        dr.line([(x0 + 17 + 96, y0 + 14), (x0 + 17 + 96, y0 + 206)], fill=(56, 189, 248), width=1)
        dr.text((x0 + 10, y0 + 212), f"Key Pose #{idx}", font=f_s, fill=(203, 213, 225))

    out_path.parent.mkdir(parents=True, exist_ok=True)
    board.save(out_path)


def verify_and_record_frame(
    im: Image.Image,
    fpath: Path,
    clip_name: str,
    local_idx: int,
    global_idx: int,
    cell_dim: int,
    is_char: bool,
    foot_info: Dict[str, Any] | None = None,
) -> Dict[str, Any]:
    fpath.parent.mkdir(parents=True, exist_ok=True)
    im.save(fpath)

    arr = np.array(im)
    assert im.size == (cell_dim, cell_dim), f"Wrong size {im.size} for {fpath}"
    alpha = arr[:, :, 3]
    semi = int(np.sum((alpha > 0) & (alpha < 255)))
    assert semi == 0, f"Semi-transparent halo pixels ({semi}) in {fpath}"

    ys, xs = np.where(alpha > 0)
    assert len(ys) > 0, f"Empty frame {fpath}"
    x0, y0, x1, y1 = int(xs.min()), int(ys.min()), int(xs.max()), int(ys.max())
    margin = 2 if cell_dim == 256 else 1
    assert x0 >= margin and y0 >= margin and x1 <= (cell_dim - 1 - margin) and y1 <= (cell_dim - 1 - margin), (
        f"Border clipping in {fpath}: bbox={(x0, y0, x1, y1)} in {cell_dim}x{cell_dim}"
    )

    uniq = {tuple(int(v) for v in row) for row in np.unique(arr[alpha > 0][:, :3], axis=0)}
    if is_char:
        assert INK_RGB in uniq, f"Missing #11151C ink contour in {fpath}"
        assert len(uniq) <= 16, f"Palette overflow ({len(uniq)} > 16) in {fpath}"

    rec: Dict[str, Any] = {
        "frame_index": global_idx,
        "clip": clip_name,
        "clip_frame_index": local_idx,
        "file": fpath.relative_to(REPO_ROOT).as_posix(),
        "cell_size": [cell_dim, cell_dim],
        "bbox": [x0, y0, x1, y1],
        "pixel_height": int(y1 - y0 + 1),
        "pixel_width": int(x1 - x0 + 1),
        "opaque_pixels": int((alpha > 0).sum()),
        "semi_transparent_pixels": semi,
        "unique_colors": len(uniq),
    }
    if foot_info is not None:
        rec.update(foot_info)
    return rec


def stance_for_clip(clip_name: str, local_idx: int) -> str:
    if clip_name == "idle":
        return "both"
    if clip_name in ("walk", "run", "march", "idlemarch"):
        return "front" if local_idx < 3 else "back"
    return "action"


def main() -> None:
    char_specs: List[Tuple[str, str, int, float, Callable[[], Any], Callable[..., Image.Image], Dict[str, List[Dict[str, float]]], np.ndarray, Tuple[int, int]]] = [
        ("guard", "chr_role_guard", 128, 48.0, build_guard_rig, render_guard_pose, GUARD_CLIPS, GUARD_PAL, (6, 3)),
        ("fighter", "chr_role_fighter", 128, 48.0, build_fighter_rig, render_fighter_pose, FIGHTER_CLIPS, FIGHTER_PAL, (6, 3)),
        ("runner", "chr_enemy_runner", 128, 44.0, build_runner_rig, render_runner_pose, RUNNER_CLIPS, RUNNER_PAL, (6, 3)),
        ("brute", "chr_enemy_brute", 128, 64.0, build_brute_rig, render_brute_pose, BRUTE_CLIPS, BRUTE_PAL, (6, 3)),
        ("boss", "chr_boss_warlord", 256, 102.0, build_boss_rig, render_boss_pose, BOSS_CLIPS, BOSS_PAL, (4, 4)),
    ]

    entities_meta: Dict[str, Any] = {}
    total_char_frames = 0

    for folder, prefix, cdim, target_1080p, rig_fn, render_fn, clip_dict, pal, (gcols, grows) in char_specs:
        rig = rig_fn()
        cdir = ART004_DIR / "characters" / folder
        fdir = cdir / "frames"

        all_frames: List[Image.Image] = []
        frame_records: List[Dict[str, Any]] = []
        clip_ranges: Dict[str, List[int]] = {}
        g_idx = 0

        for cname, poses in clip_dict.items():
            start_idx = g_idx
            for l_idx, pose in enumerate(poses):
                foot_out: Dict[str, Any] = {"stance_leg": stance_for_clip(cname, l_idx)}
                fim = render_fn(rig, pose, foot_out=foot_out)
                fname = f"{prefix}_{cname}_f{l_idx:02d}.png"
                rec = verify_and_record_frame(
                    fim, fdir / fname, cname, l_idx, g_idx, cdim, is_char=True, foot_info=foot_out
                )
                all_frames.append(fim)
                frame_records.append(rec)
                g_idx += 1
            clip_ranges[cname] = [start_idx, g_idx - 1]

        sheet_path = cdir / f"{prefix}_sheet.png"
        sheet_im = pack_fixed_grid(all_frames, gcols, grows, cdim, cdim)
        sheet_im.save(sheet_path)

        # Pick 6 representative frames across clips for rig_debug_sheet.png
        sample_indices = [0, 4, 6, 8, 10, 14] if len(all_frames) == 18 else [0, 3, 6, 9, 11, 14]
        save_rig_debug_sheet(
            f"{folder.upper()} ({prefix})",
            rig,
            [all_frames[i] for i in sample_indices],
            cdir / "rig_debug_sheet.png",
        )

        # Compute exact Unity VisualRoot scale from actual source standing height!
        # Note: For Fighter, body helm-crown height is 86px while tall crimson plume adds ~14px above barbute crown!
        standing_ref_h = 86 if folder == "fighter" else frame_records[0]["pixel_height"]
        vis_scale = round(target_1080p / float(standing_ref_h), 5)
        disp_1080p = round(frame_records[0]["pixel_height"] * vis_scale, 1)
        disp_720p = round(disp_1080p * (720.0 / 1080.0), 1)
        disp_360p = round(disp_1080p * (360.0 / 1080.0), 1)

        entities_meta[folder] = {
            "entity_id": prefix,
            "kind": "boss_character" if folder == "boss" else "character",
            "cell_size": [cdim, cdim],
            "normalized_pivot": [0.5, 0.08],
            "pivot_pixel_top_down": [round(cdim * 0.5, 2), round(cdim * 0.92, 2)],
            "sheet": sheet_path.relative_to(REPO_ROOT).as_posix(),
            "sheet_dimensions": [sheet_im.width, sheet_im.height],
            "grid_cols_rows": [gcols, grows],
            "palette_hex": [rgb_to_hex(row) for row in pal],
            "standing_source_body_px": standing_ref_h,
            "standing_source_total_bbox_px": frame_records[0]["pixel_height"],
            "target_body_1080p_px": target_1080p,
            "unity_ppu": 192,
            "unity_visual_root_scale": vis_scale,
            "displayed_total_height_px": {
                "1080x1920": disp_1080p,
                "720x1280": disp_720p,
                "360x640": disp_360p,
            },
            "clips": clip_ranges,
            "frames": frame_records,
        }
        total_char_frames += len(all_frames)
        print(
            f"[CHAR PASS] {prefix}: {len(all_frames)} frames ({cdim}x{cdim}), sheet={sheet_im.size}, "
            f"H_src={frame_records[0]['pixel_height']}px, VisualRoot={vis_scale} -> 1080p={disp_1080p}px / 360p={disp_360p}px"
        )

    # Build 4 Combat VFX Archetypes (24 frames)
    vfx_specs = [
        ("fighter_slash_arc", "vfx_fighter_slash_arc", 128, (4, 1), SLASH_VFX_PAL, build_vfx_fighter_slash_arc, "fighter.swordslash[f02..f05]"),
        ("guard_shield_block", "vfx_guard_shield_block", 128, (4, 1), SHIELD_VFX_PAL, build_vfx_guard_shield_block, "guard.shieldbrace[f02..f05]"),
        ("unit_hit_puff", "vfx_unit_hit_puff", 128, (6, 1), PUFF_VFX_PAL, build_vfx_unit_hit_puff, "any_unit.hit[f00..f01]+dissipation"),
        ("boss_slam_telegraph", "vfx_boss_slam_telegraph", 256, (5, 2), BOSS_VFX_PAL, build_vfx_boss_slam_telegraph, "boss.cleaveslam[f00..f04 warn, f05..f07 impact, f08..f09 dissipate]"),
    ]
    total_vfx_frames = 0
    for folder, prefix, cdim, (gcols, grows), pal, vfx_fn, sync_note in vfx_specs:
        vdir = ART004_DIR / "vfx" / folder
        fdir = vdir / "frames"
        vframes = vfx_fn()
        vrecords: List[Dict[str, Any]] = []
        for idx, vim in enumerate(vframes):
            fname = f"{prefix}_f{idx:02d}.png"
            rec = verify_and_record_frame(vim, fdir / fname, "burst", idx, idx, cdim, is_char=False)
            vrecords.append(rec)

        # Verify zero residual bright pixels on final VFX frame
        last_arr = np.array(vframes[-1])
        bright_end = int(np.sum((last_arr[:, :, 3] > 0) & (last_arr[:, :, 0] > 195) & (last_arr[:, :, 1] > 195)))
        assert bright_end == 0, f"Persistent bright pixels ({bright_end}) on final frame of {prefix}"

        sheet_path = vdir / f"{prefix}_sheet.png"
        sheet_im = pack_fixed_grid(vframes, gcols, grows, cdim, cdim)
        sheet_im.save(sheet_path)

        entities_meta[prefix] = {
            "entity_id": prefix,
            "kind": "combat_vfx",
            "cell_size": [cdim, cdim],
            "normalized_pivot": [0.5, 0.5] if prefix != "vfx_boss_slam_telegraph" else [0.5, 0.43],
            "synchronized_with": sync_note,
            "sheet": sheet_path.relative_to(REPO_ROOT).as_posix(),
            "sheet_dimensions": [sheet_im.width, sheet_im.height],
            "grid_cols_rows": [gcols, grows],
            "palette_hex": [rgb_to_hex(row) for row in pal],
            "clips": {"burst": [0, len(vframes) - 1]},
            "frames": vrecords,
        }
        total_vfx_frames += len(vframes)
        print(f"[VFX PASS]  {prefix}: {len(vframes)} frames ({cdim}x{cdim}), sheet={sheet_im.size}, end_bright_px=0")

    assert total_char_frames == 88, f"Expected 88 character frames, got {total_char_frames}"
    assert total_vfx_frames == 24, f"Expected 24 VFX frames, got {total_vfx_frames}"

    meta_doc = {
        "package_id": "ART-004",
        "style_version": "bastion-dawn-v1",
        "fps": 12,
        "frame_duration_ms": 83.333,
        "gif_centisecond_cadence_ms": [80, 80, 90],
        "target_ppu": 192,
        "camera_orthographic_size": 5.0,
        "reference_1080p_pixels_per_world_unit": 192.0,
        "counts": {
            "character_classes": 5,
            "character_frames": total_char_frames,
            "vfx_archetypes": 4,
            "vfx_frames": total_vfx_frames,
            "total_frames": total_char_frames + total_vfx_frames,
            "sprite_sheets": 9,
        },
        "entities": entities_meta,
    }
    meta_path = ART004_DIR / "clip_frame_metadata.json"
    with open(meta_path, "w", encoding="utf-8") as fp:
        json.dump(meta_doc, fp, indent=2)
    print(f"\nALL 112 PRODUCTION FRAMES ({total_char_frames} CHAR + {total_vfx_frames} VFX) & 9 SHEETS GENERATED SUCCESSFULLY!")


if __name__ == "__main__":
    main()
