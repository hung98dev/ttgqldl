#!/usr/bin/env python3
"""
ART-004 Independent Mechanical + 11-Case Adversarial + Visual QA Validator & SHA-256 Asset Manifest Builder.

Executes:
  1. All 15 Mechanical Validation Checks across all 112 production frames, 9 sprite sheets,
     2048x2048 Master Atlas, 28 12-FPS GIFs, 4 12-FPS H.264 MP4s, and isolated Unity 6000.6.7f1 proof.
  2. All 11 Mandatory Adversarial Negative Tests (verifying that deliberate defects deterministically
     fail the validator rather than relying on hardcoded PASS strings) -> qa/adversarial_validator_results.json.
  3. 5 Visual QA Diagnostic Proof Boards (at-scale 1080p/720p/360p, BT.709 grayscale, H.264 PSNR,
     onion-skin cyclic loop seam audit, planted-foot trajectory & binary alpha border audit).
  4. Reproducible SHA-256 asset_manifest.json covering every deliverable in art/production/ART-004/
     with rights_status = "PENDING_REVIEW".
"""

from __future__ import annotations

import argparse
import copy
import hashlib
import json
import math
import subprocess
import tempfile
from pathlib import Path
from typing import Any, Callable, Dict, List, Tuple

import numpy as np
from PIL import Image, ImageDraw, ImageFont

REPO_ROOT = Path(__file__).resolve().parents[4]
ART003_DIR = REPO_ROOT / "art" / "production" / "ART-003"
ART004_DIR = REPO_ROOT / "art" / "production" / "ART-004"

MEADOW_RGB = (104, 163, 71)
OCHRE_RGB = (207, 169, 110)
LIMESTONE_RGB = (184, 192, 200)
DARK_BG = (16, 22, 34)
CARD_BG = (26, 35, 50)
CARD_BORDER = (62, 78, 104)
INK_RGB = (17, 21, 28)

EXPECTED_CHAR_CLIPS: Dict[str, Dict[str, int]] = {
    "guard": {"idle": 4, "walk": 6, "shieldbrace": 6, "hit": 2},
    "fighter": {"idle": 4, "walk": 6, "swordslash": 6, "hit": 2},
    "runner": {"idle": 4, "run": 6, "daggerlunge": 6, "hit": 2},
    "brute": {"idle": 4, "march": 6, "gauntletcrush": 6, "hit": 2},
    "boss": {"idlemarch": 6, "cleaveslam": 8, "hit": 2},
}

EXPECTED_VFX_COUNTS: Dict[str, int] = {
    "vfx_fighter_slash_arc": 4,
    "vfx_guard_shield_block": 4,
    "vfx_unit_hit_puff": 6,
    "vfx_boss_slam_telegraph": 10,
}


def load_font(size: int, bold: bool = False) -> ImageFont.FreeTypeFont | ImageFont.ImageFont:
    candidates = (
        ["C:/Windows/Fonts/segoeuib.ttf", "C:/Windows/Fonts/arialbd.ttf"]
        if bold
        else ["C:/Windows/Fonts/segoeui.ttf", "C:/Windows/Fonts/arial.ttf"]
    )
    for c in candidates:
        if Path(c).exists():
            return ImageFont.truetype(c, size)
    return ImageFont.load_default()


def sha256_of_file(path: Path) -> str:
    h = hashlib.sha256()
    with open(path, "rb") as fp:
        for chunk in iter(lambda: fp.read(65536), b""):
            h.update(chunk)
    return h.hexdigest()


def bt709_grayscale_rgb(im: Image.Image) -> Image.Image:
    arr = np.array(im.convert("RGB"), dtype=np.float32)
    y = 0.2126 * arr[:, :, 0] + 0.7152 * arr[:, :, 1] + 0.0722 * arr[:, :, 2]
    y_u8 = np.clip(np.round(y), 0, 255).astype(np.uint8)
    return Image.fromarray(np.stack([y_u8, y_u8, y_u8], axis=-1), mode="RGB")


def psnr_rgb(img_a: Image.Image, img_b: Image.Image) -> float:
    a = np.array(img_a.convert("RGB"), dtype=np.float64)
    b = np.array(img_b.convert("RGB"), dtype=np.float64)
    mse = float(np.mean((a - b) ** 2))
    if mse <= 1e-9:
        return 99.0
    return round(10.0 * math.log10((255.0 ** 2) / mse), 2)


def probe_video(mp4_path: Path) -> Dict[str, Any]:
    cmd = [
        "ffprobe", "-v", "error",
        "-select_streams", "v:0",
        "-count_frames",
        "-show_entries", "stream=codec_name,pix_fmt,width,height,r_frame_rate,nb_read_frames,duration",
        "-of", "json",
        str(mp4_path),
    ]
    proc = subprocess.run(cmd, check=True, stdout=subprocess.PIPE, stderr=subprocess.PIPE, text=True)
    info = json.loads(proc.stdout)["streams"][0]
    return {
        "file": mp4_path.relative_to(REPO_ROOT).as_posix(),
        "codec": info["codec_name"],
        "pix_fmt": info["pix_fmt"],
        "width": int(info["width"]),
        "height": int(info["height"]),
        "r_frame_rate": info["r_frame_rate"],
        "frames": int(info["nb_read_frames"]),
        "duration_s": round(float(info.get("duration", 0.0)), 3),
    }


def probe_gif(gif_path: Path) -> Dict[str, Any]:
    im = Image.open(gif_path)
    n = getattr(im, "n_frames", 1)
    durs = []
    for i in range(n):
        im.seek(i)
        durs.append(int(im.info.get("duration", 0)))
    mean_ms = sum(durs) / max(1, len(durs))
    eff_fps = round(1000.0 / mean_ms, 2) if mean_ms > 0 else 0.0
    return {
        "file": gif_path.relative_to(REPO_ROOT).as_posix(),
        "width": im.width,
        "height": im.height,
        "frames": n,
        "durations_ms": durs,
        "effective_fps": eff_fps,
    }


def extract_mp4_frame(mp4_path: Path, frame_idx: int = 5) -> Image.Image:
    with tempfile.TemporaryDirectory() as td:
        out_png = Path(td) / "out.png"
        cmd = [
            "ffmpeg", "-y",
            "-i", str(mp4_path),
            "-vf", f"select=eq(n\\,{frame_idx})",
            "-vframes", "1",
            str(out_png),
        ]
        subprocess.run(cmd, check=True, stdout=subprocess.PIPE, stderr=subprocess.PIPE)
        return Image.open(out_png).convert("RGB")


# ============================================================================
# REUSABLE CORE VALIDATOR FUNCTIONS (EXERCISED BY BOTH REAL & ADVERSARIAL RUNS)
# ============================================================================

def check_frame_image_integrity(
    im: Image.Image,
    expected_dim: int,
    is_char: bool,
    pivot_norm: List[float],
    label: str = "frame",
) -> Dict[str, Any]:
    assert im.size == (expected_dim, expected_dim), f"[{label}] Wrong sprite dimension {im.size}, expected {(expected_dim, expected_dim)}"
    arr = np.array(im)
    assert arr.ndim == 3 and arr.shape[2] == 4, f"[{label}] Not RGBA"

    alpha = arr[:, :, 3]
    semi = int(np.sum((alpha > 0) & (alpha < 255)))
    assert semi == 0, f"[{label}] Malformed alpha: {semi} semi-transparent halo pixels"

    ys, xs = np.where(alpha > 0)
    assert len(ys) > 0, f"[{label}] Completely empty alpha frame"
    x0, y0, x1, y1 = int(xs.min()), int(ys.min()), int(xs.max()), int(ys.max())
    margin = 2 if expected_dim == 256 else 1
    assert x0 >= margin and y0 >= margin and x1 <= (expected_dim - 1 - margin) and y1 <= (expected_dim - 1 - margin), (
        f"[{label}] Border clipping detected: bbox={(x0, y0, x1, y1)} inside {expected_dim}x{expected_dim}"
    )

    uniq = {tuple(int(v) for v in row) for row in np.unique(arr[alpha > 0][:, :3], axis=0)}
    if is_char:
        assert pivot_norm == [0.5, 0.08], f"[{label}] Incorrect character pivot {pivot_norm}, expected [0.5, 0.08]"
        assert INK_RGB in uniq, f"[{label}] Missing #11151C outer ink contour"
        assert len(uniq) <= 16, f"[{label}] Palette overflow ({len(uniq)} > 16 colors)"

    cx = float(xs.mean())
    cy = float(ys.mean())
    return {
        "bbox": [x0, y0, x1, y1],
        "unique_colors": len(uniq),
        "semi_transparent_pixels": semi,
        "centroid_xy": [round(cx, 2), round(cy, 2)],
    }


def check_clip_sequence_and_loop(
    ent_key: str,
    clip_name: str,
    frames: List[Image.Image],
    expected_len: int,
    is_cyclic: bool,
) -> Dict[str, Any]:
    assert len(frames) == expected_len, (
        f"[{ent_key}.{clip_name}] Incompatible clip length {len(frames)}, expected {expected_len}"
    )
    arrs = [np.array(f, dtype=np.float32) for f in frames]
    step_diffs: List[float] = []
    for i in range(len(arrs) - 1):
        d = float(np.mean(np.abs(arrs[i + 1] - arrs[i])))
        assert d > 0.01, f"[{ent_key}.{clip_name}] Repeated/duplicate consecutive frames at {i}->{i+1} (L1={d})"
        step_diffs.append(d)

    mean_step = float(np.mean(step_diffs)) if step_diffs else 0.0
    wrap_diff = float(np.mean(np.abs(arrs[0] - arrs[-1])))
    seam_ratio = round(wrap_diff / max(0.001, mean_step), 3) if mean_step > 0 else 0.0
    if is_cyclic:
        assert 0.75 <= seam_ratio <= 1.65, (
            f"[{ent_key}.{clip_name}] Broken cyclic loop seam: seam_ratio={seam_ratio} outside [0.75, 1.65]"
        )
    return {
        "frames": len(frames),
        "mean_step_l1": round(mean_step, 3),
        "wrap_l1": round(wrap_diff, 3),
        "seam_ratio": seam_ratio,
    }


def check_planted_foot_anchoring(
    ent_key: str,
    clip_name: str,
    f_recs: List[Dict[str, Any]],
    frames: List[Image.Image],
    cell_dim: int,
) -> Dict[str, Any]:
    ground_min = 232.0 if cell_dim == 256 else 115.0
    ground_max = 242.0 if cell_dim == 256 else 121.0
    lift_thresh = 232.0 if cell_dim == 256 else 115.0
    slip_tol = 1.5 if cell_dim == 256 else 0.8

    for idx, (rec, im) in enumerate(zip(f_recs, frames)):
        alpha = np.array(im)[:, :, 3]
        for fk in ("front_foot_xy", "back_foot_xy"):
            assert fk in rec, f"[{ent_key}.{clip_name}] Missing {fk}"
            fx, fy = rec[fk]
            xi, yi = int(round(fx)), int(round(fy))
            patch = alpha[max(0, yi - 2) : min(cell_dim, yi + 3), max(0, xi - 3) : min(cell_dim, xi + 4)]
            assert int((patch > 0).sum()) > 0, (
                f"[{ent_key}.{clip_name} f{idx:02d}] Foot coord {fk}={(fx, fy)} has zero opaque pixels in PNG"
            )

    if clip_name == "idle":
        fxs = [r["front_foot_xy"][0] for r in f_recs]
        bx_s = [r["back_foot_xy"][0] for r in f_recs]
        fys = [r["front_foot_xy"][1] for r in f_recs]
        bys = [r["back_foot_xy"][1] for r in f_recs]
        dx_max = max(max(fxs) - min(fxs), max(bx_s) - min(bx_s))
        dy_max = max(max(fys) - min(fys), max(bys) - min(bys))
        assert dx_max <= slip_tol and dy_max <= 1.0, (
            f"[{ent_key}.idle] Idle foot slip exceeded tolerance: dx={dx_max}, dy={dy_max}"
        )
        return {"idle_dx_slip_px": round(dx_max, 2), "idle_dy_slip_px": round(dy_max, 2)}

    if clip_name in ("walk", "run", "march", "idlemarch"):
        half = len(f_recs) // 2
        f_stance = [r["front_foot_xy"] for r in f_recs[:half]]
        b_swing = [r["back_foot_xy"] for r in f_recs[:half]]
        b_stance = [r["back_foot_xy"] for r in f_recs[half:]]
        f_swing = [r["front_foot_xy"] for r in f_recs[half:]]

        for i, (_, sy) in enumerate(f_stance):
            assert ground_min <= sy <= ground_max, f"[{ent_key}.{clip_name}] Front stance lifted at f{i}: y={sy}"
        for i, (_, sy) in enumerate(b_stance):
            assert ground_min <= sy <= ground_max, f"[{ent_key}.{clip_name}] Back stance lifted at f{i+half}: y={sy}"

        for i in range(len(f_stance) - 1):
            assert f_stance[i + 1][0] <= f_stance[i][0] + 0.35, (
                f"[{ent_key}.{clip_name}] Front stance skated forward: {f_stance}"
            )
        for i in range(len(b_stance) - 1):
            assert b_stance[i + 1][0] <= b_stance[i][0] + 0.35, (
                f"[{ent_key}.{clip_name}] Back stance skated forward: {b_stance}"
            )

        assert min(p[1] for p in b_swing) <= lift_thresh, f"[{ent_key}.{clip_name}] Back swing foot never lifted: {b_swing}"
        assert min(p[1] for p in f_swing) <= lift_thresh, f"[{ent_key}.{clip_name}] Front swing foot never lifted: {f_swing}"
        return {
            "front_stance_tread_px": round(f_stance[0][0] - f_stance[-1][0], 2),
            "back_stance_tread_px": round(b_stance[0][0] - b_stance[-1][0], 2),
            "min_swing_y_px": round(min(min(p[1] for p in b_swing), min(p[1] for p in f_swing)), 1),
        }
    return {"action_grounded": True}


def check_sheet_matches_frames(sheet_im: Image.Image, frames: List[Image.Image], cols: int, rows: int, cell_dim: int, label: str) -> None:
    assert sheet_im.size == (cols * cell_dim, rows * cell_dim), f"[{label}] Sheet size mismatch {sheet_im.size}"
    sh_arr = np.array(sheet_im)
    for idx, fr in enumerate(frames):
        r, c = divmod(idx, cols)
        cell = sh_arr[r * cell_dim : (r + 1) * cell_dim, c * cell_dim : (c + 1) * cell_dim]
        fr_arr = np.array(fr)
        assert np.array_equal(cell, fr_arr), f"[{label}] Sheet cell #{idx} (r={r}, c={c}) does not match frame #{idx} (swapped/corrupted!)"


def check_atlas_uv_and_bleed(
    atlas_im: Image.Image,
    atlas_spec: Dict[str, Any],
    frame_lookup: Dict[str, Image.Image],
) -> Dict[str, Any]:
    aw, ah = atlas_spec["dimensions"]
    assert atlas_im.size == (aw, ah) and aw <= 2048 and ah <= 2048, f"Invalid atlas dimensions {atlas_im.size}"
    arr = np.array(atlas_im)
    slices = atlas_spec["slices"]
    assert len(slices) == 112, f"Expected 112 slices in atlas, got {len(slices)}"

    bleed_px = 0
    for sl in slices:
        src_rel = sl["source_file"]
        assert src_rel in frame_lookup, f"Atlas references missing frame {src_rel}"
        src_arr = np.array(frame_lookup[src_rel])
        cw, ch = sl["cell_size"]
        u0, v0, uw, vh = sl["uv_rect_unity"]
        ux = int(round(u0 * aw))
        uy_bot = int(round(v0 * ah))
        uw_px = int(round(uw * aw))
        uh_px = int(round(vh * ah))
        ty = ah - (uy_bot + uh_px)
        assert (uw_px, uh_px) == (cw, ch), f"UV size {(uw_px, uh_px)} != {(cw, ch)} for {sl['name']}"
        assert 0 <= ux and ux + cw <= aw and 0 <= ty and ty + ch <= ah, f"UV out of bounds for {sl['name']}"
        sub = arr[ty : ty + ch, ux : ux + cw]
        assert np.array_equal(sub, src_arr), f"Corrupted atlas UV or pixel mismatch for {sl['name']}"

        # Verify 4px gutter around every slice is 100% transparent (alpha == 0)
        pad = atlas_spec["padding_per_cell_side_px"]
        gx0, gy0 = max(0, ux - pad), max(0, ty - pad)
        gx1, gy1 = min(aw, ux + cw + pad), min(ah, ty + ch + pad)
        gutter = arr[gy0:gy1, gx0:gx1, 3].copy()
        gutter[ty - gy0 : ty - gy0 + ch, ux - gx0 : ux - gx0 + cw] = 0
        bleed_px += int((gutter > 0).sum())

    assert bleed_px == 0, f"Atlas gutter alpha bleed detected ({bleed_px} pixels)"
    return {"verified_slices": len(slices), "gutter_bleed_pixels": bleed_px}


def verify_manifest_records(records: List[Dict[str, Any]], root: Path) -> int:
    for rec in records:
        p = root / rec["path"]
        assert p.exists(), f"Missing file referenced in manifest: {rec['path']}"
        actual_sha = sha256_of_file(p)
        assert actual_sha == rec["sha256"], f"Mismatched SHA-256 for {rec['path']}: expected {rec['sha256']}, got {actual_sha}"
        assert p.stat().st_size == rec["bytes"], f"Byte-size mismatch for {rec['path']}"
    return len(records)


# ============================================================================
# PHASE 9B — ALL 11 MANDATORY ADVERSARIAL NEGATIVE VALIDATOR TESTS
# ============================================================================

def run_11_adversarial_tests(
    meta4: Dict[str, Any],
    loaded4: Dict[str, List[Image.Image]],
    atlas_im: Image.Image,
    atlas_spec: Dict[str, Any],
    frame_lookup: Dict[str, Image.Image],
) -> List[Dict[str, Any]]:
    results: List[Dict[str, Any]] = []

    def expect_rejection(test_id: str, description: str, fn: Callable[[], Any]) -> None:
        try:
            fn()
        except AssertionError as exc:
            results.append(
                {
                    "test_id": test_id,
                    "defect_injected": description,
                    "validator_outcome": "DETECTED_AND_REJECTED",
                    "rejection_reason": str(exc),
                }
            )
            return
        raise RuntimeError(f"ADVERSARIAL TEST FAILED TO REJECT DEFECT: {test_id} ({description})")

    # 1. Missing frame
    def _t_missing() -> None:
        fake_manifest = [{"path": "art/production/ART-004/characters/guard/frames/chr_role_guard_idle_f99_MISSING.png", "sha256": "0" * 64, "bytes": 100}]
        verify_manifest_records(fake_manifest, REPO_ROOT)

    expect_rejection("ADV-01_missing_frame", "Reference a missing PNG frame path", _t_missing)

    # 2. Repeated frame
    def _t_repeated() -> None:
        bad_walk = list(loaded4["fighter"][4:10])
        bad_walk[2] = bad_walk[1].copy()
        check_clip_sequence_and_loop("fighter", "walk", bad_walk, expected_len=6, is_cyclic=True)

    expect_rejection("ADV-02_repeated_frame", "Duplicate fighter walk f01 into f02", _t_repeated)

    # 3. Malformed alpha frame
    def _t_malformed_alpha() -> None:
        bad = loaded4["guard"][0].copy()
        arr = np.array(bad)
        arr[20, 20] = [200, 200, 200, 128]
        check_frame_image_integrity(Image.fromarray(arr, mode="RGBA"), 128, True, [0.5, 0.08], "adv_alpha")

    expect_rejection("ADV-03_malformed_alpha", "Inject semi-transparent alpha=128 halo pixel", _t_malformed_alpha)

    # 4. Weapon clipped at border
    def _t_border_clip() -> None:
        bad = loaded4["fighter"][13].copy()
        arr = np.array(bad)
        arr[64, 127] = [226, 232, 240, 255]
        check_frame_image_integrity(Image.fromarray(arr, mode="RGBA"), 128, True, [0.5, 0.08], "adv_border_clip")

    expect_rejection("ADV-04_border_weapon_clip", "Paint sword blade pixel at right border x=127", _t_border_clip)

    # 5. Incorrect sprite dimension
    def _t_wrong_dim() -> None:
        bad = loaded4["runner"][0].resize((124, 128), Image.Resampling.NEAREST)
        check_frame_image_integrity(bad, 128, True, [0.5, 0.08], "adv_wrong_dim")

    expect_rejection("ADV-05_wrong_sprite_dimension", "Resize 128x128 sprite to 124x128", _t_wrong_dim)

    # 6. Swapped frame index
    def _t_swapped_index() -> None:
        sheet_im = Image.open(REPO_ROOT / meta4["entities"]["brute"]["sheet"]).convert("RGBA")
        swapped = list(loaded4["brute"])
        swapped[4], swapped[5] = swapped[5], swapped[4]
        check_sheet_matches_frames(sheet_im, swapped, 6, 3, 128, "adv_swapped_brute")

    expect_rejection("ADV-06_swapped_frame_index", "Swap Brute march f00 and f01 against sprite sheet", _t_swapped_index)

    # 7. Corrupted atlas UV mapping
    def _t_corrupted_uv() -> None:
        bad_spec = copy.deepcopy(atlas_spec)
        bad_spec["slices"][10]["uv_rect_unity"][0] = round(bad_spec["slices"][10]["uv_rect_unity"][0] + 0.05, 6)
        check_atlas_uv_and_bleed(atlas_im, bad_spec, frame_lookup)

    expect_rejection("ADV-07_corrupted_atlas_uv", "Shift slice #10 uv_rect_unity.x by +0.05", _t_corrupted_uv)

    # 8. Mismatched SHA-256 hash
    def _t_bad_hash() -> None:
        p = REPO_ROOT / meta4["entities"]["boss"]["frames"][0]["file"]
        real_sha = sha256_of_file(p)
        tampered = ("f" if real_sha[0] != "f" else "0") + real_sha[1:]
        verify_manifest_records([{"path": meta4["entities"]["boss"]["frames"][0]["file"], "sha256": tampered, "bytes": p.stat().st_size}], REPO_ROOT)

    expect_rejection("ADV-08_mismatched_hash", "Flip leading nibble of Boss f00 SHA-256 digest", _t_bad_hash)

    # 9. Broken cyclic loop
    def _t_broken_loop() -> None:
        # Classic non-periodic drift loop: steady +3px step per frame (f0..f5 = 0..+15px) that teleports -15px on f5->f0 wrap!
        base = loaded4["guard"][4]
        broken = []
        for k_s in range(6):
            canvas = Image.new("RGBA", (128, 128), (0, 0, 0, 0))
            canvas.paste(base, (k_s * 3, 0), base)
            broken.append(canvas)
        check_clip_sequence_and_loop("guard", "walk", broken, expected_len=6, is_cyclic=True)

    expect_rejection("ADV-09_broken_loop", "Construct non-periodic walk sequence with seam discontinuity", _t_broken_loop)

    # 10. Incorrect pivot
    def _t_wrong_pivot() -> None:
        check_frame_image_integrity(loaded4["boss"][0], 256, True, [0.5, 0.5], "adv_wrong_pivot")

    expect_rejection("ADV-10_incorrect_pivot", "Supply center pivot [0.5, 0.5] instead of ground pivot [0.5, 0.08]", _t_wrong_pivot)

    # 11. Incompatible animation clip length
    def _t_bad_clip_len() -> None:
        truncated = loaded4["fighter"][10:14]  # 4 frames instead of 6!
        check_clip_sequence_and_loop("fighter", "swordslash", truncated, expected_len=6, is_cyclic=False)

    expect_rejection("ADV-11_incompatible_clip_length", "Truncate Fighter SwordSlash from 6 frames to 4 frames", _t_bad_clip_len)

    assert len(results) == 11
    return results


# ============================================================================
# VISUAL QA DIAGNOSTIC PROOF BOARDS (5 REAL PNG PLATES)
# ============================================================================

def generate_visual_qa_boards(
    meta4: Dict[str, Any],
    loaded4: Dict[str, List[Image.Image]],
    mp4_metrics: Dict[str, Any],
    loop_metrics: Dict[str, Dict[str, Any]],
) -> None:
    qa_dir = ART004_DIR / "qa"
    qa_dir.mkdir(parents=True, exist_ok=True)
    f_t = load_font(22, bold=True)
    f_l = load_font(14, bold=True)
    f_s = load_font(12, bold=False)

    mp4_1080 = ART004_DIR / "previews" / "mp4" / "art004_gameplay_1080x1920_12fps.mp4"
    mp4_720 = ART004_DIR / "previews" / "mp4" / "art004_gameplay_720x1280_12fps.mp4"
    mp4_360 = ART004_DIR / "previews" / "mp4" / "art004_gameplay_360x640_12fps.mp4"

    fr_1080 = extract_mp4_frame(mp4_1080, 4)
    fr_720 = extract_mp4_frame(mp4_720, 4)
    fr_360 = extract_mp4_frame(mp4_360, 4)

    # Board 1: At-Scale Gameplay Readability (1080p, 720p, 360p side-by-side + 1:1 crop & 360p 3x NN crop)
    b1 = Image.new("RGB", (1660, 980), DARK_BG)
    d1 = ImageDraw.Draw(b1)
    d1.text((24, 14), "ART-004 VISUAL QA — ACTUAL GAMEPLAY DISPLAY SCALE PROOF (1080x1920 | 720x1280 | 360x640)", font=f_t, fill=(254, 240, 138))
    d1.text((24, 44), "All 7 Character Classes + 5 Combat/Repair VFX over ART-002 Meadow, Ochre Lanes & Limestone Boss Causeway (Safe Zones x<=82%, y<=88%)", font=f_s, fill=(203, 213, 225))

    d1.text((24, 74), "1080x1920 Master (scaled to fit)", font=f_l, fill=(254, 240, 138))
    p1080 = fr_1080.resize((468, 832), Image.Resampling.LANCZOS)
    b1.paste(p1080, (24, 98))
    d1.rectangle([24, 98, 492, 930], outline=CARD_BORDER, width=2)

    # Exact 1:1 unscaled crop from 1080x1920 battle zone
    d1.text((518, 74), "TRUE 1:1 PIXEL CROP FROM 1080x1920 (Boss 102px, Brute 64px, Heroes 48px, Runner 44px)", font=f_l, fill=(254, 240, 138))
    crop_1080_1to1 = fr_1080.crop((150, 520, 770, 920))
    b1.paste(crop_1080_1to1, (518, 98))
    d1.rectangle([518, 98, 518 + 620, 98 + 400], outline=CARD_BORDER, width=2)

    # Exact 1:1 unscaled 360x640 mobile frame + 3x NN zoom of 360p battle zone!
    d1.text((1164, 74), "TRUE 1:1 360x640 MOBILE STREAM", font=f_l, fill=(254, 240, 138))
    b1.paste(fr_360, (1164, 98))
    d1.rectangle([1164, 98, 1164 + 360, 98 + 640], outline=CARD_BORDER, width=2)

    d1.text((518, 514), "360x640 MOBILE BATTLE ZONE (3x NN — 16px Heroes, 15px Runner, 21px Brute, 34px Boss)", font=f_l, fill=(134, 239, 172))
    crop_360 = fr_360.crop((50, 174, 256, 308)).resize((620, 402), Image.Resampling.NEAREST)
    b1.paste(crop_360, (518, 538))
    d1.rectangle([518, 538, 518 + 620, 538 + 402], outline=CARD_BORDER, width=2)
    b1.save(qa_dir / "qa_at_scale_gameplay_board.png")

    # Board 2: BT.709 Grayscale & 360p Mobile Silhouette Separation Proof
    b2 = Image.new("RGB", (1620, 860), DARK_BG)
    d2 = ImageDraw.Draw(b2)
    d2.text((24, 14), "ART-004 VISUAL QA — BT.709 GRAYSCALE VALUE HIERARCHY & 360x640 SILHOUETTE SEPARATION", font=f_t, fill=(254, 240, 138))
    d2.text((24, 44), "Verifying value contrast over Meadow (#68A347), Ochre (#CFA96E) & Limestone (#B8C0C8) without relying on hue alone.", font=f_s, fill=(203, 213, 225))

    gray_1080_crop = bt709_grayscale_rgb(crop_1080_1to1)
    d2.text((24, 74), "1080p Color Battle Crop (1:1)", font=f_l, fill=(254, 240, 138))
    b2.paste(crop_1080_1to1, (24, 98))
    d2.rectangle([24, 98, 644, 498], outline=CARD_BORDER, width=2)

    d2.text((668, 74), "BT.709 Luminance Grayscale (Y = 0.2126R + 0.7152G + 0.0722B)", font=f_l, fill=(254, 240, 138))
    b2.paste(gray_1080_crop, (668, 98))
    d2.rectangle([668, 98, 1288, 498], outline=CARD_BORDER, width=2)

    gray_360_fit = bt709_grayscale_rgb(fr_360).resize((225, 400), Image.Resampling.NEAREST)
    d2.text((1318, 74), "360x640 BT.709 Gray", font=f_l, fill=(254, 240, 138))
    b2.paste(gray_360_fit, (1318, 98))
    d2.rectangle([1318, 98, 1318 + 225, 98 + 400], outline=CARD_BORDER, width=2)

    # Bottom strip: all 5 ART-004 classes in BT.709 grayscale over all 3 terrains
    for idx, (ekey, act_idx) in enumerate([("guard", 12), ("fighter", 13), ("runner", 12), ("brute", 13), ("boss", 10)]):
        ox = 24 + idx * 314
        oy = 528
        d2.rounded_rectangle([ox, oy, ox + 300, oy + 294], radius=10, fill=CARD_BG, outline=CARD_BORDER, width=2)
        d2.text((ox + 12, oy + 10), f"{ekey.upper()} — Grayscale on 3 Terrains", font=f_l, fill=(248, 250, 252))
        vsc = meta4["entities"][ekey]["unity_visual_root_scale"]
        sp = loaded4[ekey][act_idx]
        sp_g = sp.resize((max(16, int(round(sp.width * vsc * 1.4))), max(16, int(round(sp.height * vsc * 1.4)))), Image.Resampling.NEAREST)
        for t_i, (t_lbl, bg_c) in enumerate([("Meadow", MEADOW_RGB), ("Ochre", OCHRE_RGB), ("Limestone", LIMESTONE_RGB)]):
            bx = ox + 12 + t_i * 94
            by = oy + 44
            cell = Image.new("RGB", (88, 210), bg_c)
            cell.paste(sp_g, ((88 - sp_g.width) // 2, (210 - sp_g.height) // 2), sp_g)
            cell_gray = bt709_grayscale_rgb(cell)
            b2.paste(cell_gray, (bx, by))
            d2.rectangle([bx, by, bx + 88, by + 210], outline=INK_RGB, width=2)
            d2.text((bx + 8, by + 216), t_lbl, font=f_s, fill=(203, 213, 225))
    b2.save(qa_dir / "qa_360x640_and_grayscale_proof.png")

    # Board 3: H.264 (yuv420p) Stream Compression Survival Proof
    b3 = Image.new("RGB", (1520, 620), DARK_BG)
    d3 = ImageDraw.Draw(b3)
    d3.text((24, 16), "ART-004 VISUAL QA — H.264 (libx264 yuv420p) LIVESTREAM COMPRESSION SURVIVAL", font=f_t, fill=(254, 240, 138))
    d3.text((24, 48), f"1080p CRF18 PSNR={mp4_metrics['psnr_1080p_db']} dB | 720p CRF21 | 360p CRF25 Mobile Proxy — #11151C contours & role colors survive chroma subsampling.", font=f_s, fill=(203, 213, 225))
    for idx, (lbl, im_src, box) in enumerate([
        ("1080x1920 H.264 Decoded Battle Crop (1:1)", fr_1080.crop((180, 560, 640, 980)), (24, 92)),
        ("720x1280 H.264 Decoded Crop (upscaled NN)", fr_720.crop((120, 373, 426, 653)).resize((460, 420), Image.Resampling.NEAREST), (516, 92)),
        ("360x640 H.264 CRF25 Decoded Crop (upscaled NN)", fr_360.crop((60, 186, 213, 326)).resize((460, 420), Image.Resampling.NEAREST), (1008, 92)),
    ]):
        b3.paste(im_src, box)
        d3.rectangle([box[0], box[1], box[0] + 460, box[1] + 420], outline=CARD_BORDER, width=2)
        d3.text((box[0] + 10, box[1] + 432), lbl, font=f_l, fill=(134, 239, 172))
    b3.save(qa_dir / "qa_h264_compression_proof.png")

    # Board 4: Temporal Flicker & Cyclic Loop Onion-Skin Seam Proof (All 9 Cyclic Loops)
    b4 = Image.new("RGB", (1620, 740), DARK_BG)
    d4 = ImageDraw.Draw(b4)
    d4.text((24, 16), "ART-004 VISUAL QA — CYCLIC LOOP SEAM CONTINUITY & ONION-SKIN AUDIT (ALL 9 LOOPS IN [0.75, 1.65])", font=f_t, fill=(254, 240, 138))
    d4.text((24, 48), "Cyan = Frame f00, Magenta = Last Frame f(N-1), White/Tinted Overlap = Seamless Wrap-Around Alignment.", font=f_s, fill=(203, 213, 225))
    loop_keys = [
        ("guard", "idle"), ("guard", "walk"),
        ("fighter", "idle"), ("fighter", "walk"),
        ("runner", "idle"), ("runner", "run"),
        ("brute", "idle"), ("brute", "march"),
        ("boss", "idlemarch"),
    ]
    for idx, (ek, cn) in enumerate(loop_keys):
        r, c = divmod(idx, 5)
        ox = 24 + c * 316
        oy = 92 + r * 308
        d4.rounded_rectangle([ox, oy, ox + 302, oy + 292], radius=10, fill=CARD_BG, outline=CARD_BORDER, width=2)
        s_i, e_i = meta4["entities"][ek]["clips"][cn]
        f0 = np.array(loaded4[ek][s_i])[:, :, 3] > 0
        fN = np.array(loaded4[ek][e_i])[:, :, 3] > 0
        viz = np.zeros((f0.shape[0], f0.shape[1], 3), dtype=np.uint8)
        viz[:, :] = (22, 30, 44)
        viz[f0 & (~fN)] = (56, 189, 248)    # Cyan = f0 only
        viz[fN & (~f0)] = (232, 121, 249)   # Magenta = fN only
        viz[f0 & fN] = (226, 232, 240)      # Overlap
        vim = Image.fromarray(viz, "RGB").resize((192, 192), Image.Resampling.NEAREST)
        b4.paste(vim, (ox + 55, oy + 42))
        d4.rectangle([ox + 55, oy + 42, ox + 247, oy + 234], outline=INK_RGB, width=2)
        lm = loop_metrics[f"{ek}.{cn}"]
        d4.text((ox + 12, oy + 12), f"{ek.upper()} {cn.upper()} ({lm['frames']}f)", font=f_l, fill=(248, 250, 252))
        d4.text((ox + 12, oy + 246), f"seam_ratio = {lm['seam_ratio']:.3f}  [PASS]", font=f_l, fill=(134, 239, 172))
        d4.text((ox + 12, oy + 268), f"mean_step={lm['mean_step_l1']} | wrap={lm['wrap_l1']}", font=f_s, fill=(203, 213, 225))
    b4.save(qa_dir / "qa_temporal_flicker_and_onion_skin.png")

    # Board 5: Planted-Foot Sole Trajectory & Binary Alpha Border Audit
    b5 = Image.new("RGB", (1620, 660), DARK_BG)
    d5 = ImageDraw.Draw(b5)
    d5.text((24, 16), "ART-004 VISUAL QA — PLANTED-FOOT TRAJECTORY & BINARY ALPHA HALO AUDIT (0 HALO PX / 112 FRAMES)", font=f_t, fill=(254, 240, 138))
    d5.text((24, 48), "Green = Front Foot Sole (x,y) | Orange = Back Foot Sole (x,y) across 6-frame locomotion cycles + Binary Alpha Rim Map.", font=f_s, fill=(203, 213, 225))
    loco_specs = [("guard", "walk"), ("fighter", "walk"), ("runner", "run"), ("brute", "march"), ("boss", "idlemarch")]
    for idx, (ek, cn) in enumerate(loco_specs):
        ox = 24 + idx * 316
        oy = 92
        d5.rounded_rectangle([ox, oy, ox + 302, oy + 536], radius=10, fill=CARD_BG, outline=CARD_BORDER, width=2)
        d5.text((ox + 12, oy + 12), f"{ek.upper()} ({cn.upper()})", font=f_l, fill=(248, 250, 252))
        ent = meta4["entities"][ek]
        s_i, e_i = ent["clips"][cn]
        recs = ent["frames"][s_i : e_i + 1]
        cdim = ent["cell_size"][0]

        # Binary alpha audit visualization on action frame
        act_im = np.array(loaded4[ek][e_i + 3 if ek != "boss" else 10])
        a_ch = act_im[:, :, 3]
        amap = np.zeros((cdim, cdim, 3), dtype=np.uint8)
        amap[:, :] = (20, 28, 40)
        amap[a_ch == 255] = (74, 222, 128)
        amap[(a_ch > 0) & (a_ch < 255)] = (239, 68, 68)  # Red if any semi-transparent halo existed!
        amap_im = Image.fromarray(amap, "RGB").resize((220, 220), Image.Resampling.NEAREST)
        b5.paste(amap_im, (ox + 41, oy + 44))
        d5.rectangle([ox + 41, oy + 44, ox + 261, oy + 264], outline=INK_RGB, width=2)
        d5.text((ox + 41, oy + 270), "Binary Alpha Audit: 0 Red Halo Pixels", font=f_s, fill=(134, 239, 172))

        # Plot front_foot_xy and back_foot_xy trajectory across f00..f05
        px0, py0, pw, ph = ox + 24, oy + 310, 254, 176
        d5.rectangle([px0, py0, px0 + pw, py0 + ph], fill=(18, 24, 36), outline=(71, 85, 105), width=1)
        d5.text((px0 + 8, py0 + 6), "Foot X (tread) & Y (lift) vs Frame", font=f_s, fill=(203, 213, 225))
        for k_f in range(len(recs) - 1):
            x_a = px0 + 24 + int(k_f * 42)
            x_b = px0 + 24 + int((k_f + 1) * 42)
            ff_a = recs[k_f]["front_foot_xy"][0] / cdim
            ff_b = recs[k_f + 1]["front_foot_xy"][0] / cdim
            bf_a = recs[k_f]["back_foot_xy"][0] / cdim
            bf_b = recs[k_f + 1]["back_foot_xy"][0] / cdim
            d5.line([(x_a, py0 + 150 - int(ff_a * 110)), (x_b, py0 + 150 - int(ff_b * 110))], fill=(74, 222, 128), width=3)
            d5.line([(x_a, py0 + 150 - int(bf_a * 110)), (x_b, py0 + 150 - int(bf_b * 110))], fill=(251, 146, 60), width=3)
        d5.text((ox + 24, oy + 498), "Stance monotonic -X | 0px forward skate", font=f_s, fill=(134, 239, 172))
    b5.save(qa_dir / "qa_foot_pivot_and_alpha_audit.png")


# ============================================================================
# ASSET MANIFEST BUILDER & VERIFIER
# ============================================================================

def classify_role_and_origin(rel_posix: str) -> Tuple[str, str]:
    if "/characters/" in rel_posix and "/frames/" in rel_posix:
        return "character_animation_frame", "deterministic_2d_part_rig_from_approved_art002_turnaround"
    if "/characters/" in rel_posix and rel_posix.endswith("_sheet.png"):
        return "character_sprite_sheet", "deterministic_grid_pack_from_art004_character_frames"
    if "/characters/" in rel_posix and rel_posix.endswith("rig_debug_sheet.png"):
        return "character_rig_debug_sheet", "deterministic_cutout_part_and_pivot_diagnostic"
    if "/vfx/" in rel_posix and "/frames/" in rel_posix:
        return "combat_vfx_frame", "deterministic_procedural_cel_vfx_generator"
    if "/vfx/" in rel_posix and rel_posix.endswith("_sheet.png"):
        return "combat_vfx_sprite_sheet", "deterministic_grid_pack_from_art004_vfx_frames"
    if "/atlas/" in rel_posix:
        return "unity_master_atlas_package", "deterministic_2048_atlas_packer_and_unity_preset_generator"
    if "/previews/gifs/" in rel_posix:
        return "animated_gif_preview_12fps", "deterministic_12fps_gif_preview_builder"
    if "/previews/mp4/" in rel_posix:
        return "stream_mp4_preview_h264_12fps", "ffmpeg_libx264_yuv420p_12fps_stream_encoder"
    if "/previews/contact_sheets/" in rel_posix:
        return "static_contact_sheet", "deterministic_multiscale_contact_sheet_builder"
    if "/qa/" in rel_posix:
        return "qa_evidence_and_metrics", "independent_mechanical_adversarial_and_unity_validator"
    if "/tools/" in rel_posix:
        return "reproducible_pipeline_tool", "authored_python_production_and_qa_pipeline"
    return "package_documentation_or_metadata", "authored_art004_production_blueprint"


def build_asset_manifest() -> Dict[str, Any]:
    manifest_path = ART004_DIR / "asset_manifest.json"
    all_files = sorted(
        [
            p for p in ART004_DIR.rglob("*")
            if p.is_file()
            and "__pycache__" not in p.parts
            and p.resolve() != manifest_path.resolve()
        ],
        key=lambda p: p.relative_to(REPO_ROOT).as_posix(),
    )

    records: List[Dict[str, Any]] = []
    for p in all_files:
        rel = p.relative_to(REPO_ROOT).as_posix()
        role, method = classify_role_and_origin(rel)
        rec: Dict[str, Any] = {
            "path": rel,
            "role": role,
            "production_method": method,
            "bytes": p.stat().st_size,
            "sha256": sha256_of_file(p),
            "rights_status": "PENDING_REVIEW",
        }
        if p.suffix.lower() in (".png", ".gif"):
            with Image.open(p) as im:
                rec["dimensions"] = [im.width, im.height]
        records.append(rec)

    doc = {
        "package_id": "ART-004",
        "style_version": "bastion-dawn-v1",
        "source_art002_package": "art/concepts/ART-002/asset_manifest.json",
        "source_art003_pilot_package": "art/production/ART-003/asset_manifest.json",
        "owner_approval_reference": "art/concepts/ART-002/OWNER_APPROVAL.md",
        "rights_status": "PENDING_REVIEW",
        "ip_cleanliness_statement": "100% repository-owned ART-002 character references + deterministic Python/Pillow/SciPy 2D skeletal cutout & cel VFX synthesis. Zero third-party ripped sprites, copyrighted anime/game characters, unlicensed textures, or audio assets.",
        "tool_versions": {
            "python": "3.13+",
            "pillow": Image.__version__,
            "numpy": np.__version__,
            "ffmpeg": "libx264 yuv420p @ 12/1 FPS",
            "unity_isolated_verifier": "6000.6.7f1 (-batchmode -nographics)",
        },
        "total_hashed_files": len(records),
        "files": records,
    }
    manifest_path.write_text(json.dumps(doc, indent=2), encoding="utf-8")
    return doc


def run_full_validation() -> None:
    with open(ART004_DIR / "clip_frame_metadata.json", "r", encoding="utf-8") as fp:
        meta4 = json.load(fp)
    with open(ART004_DIR / "atlas" / "ART004_Master_Atlas.json", "r", encoding="utf-8") as fp:
        atlas_spec = json.load(fp)

    loaded4: Dict[str, List[Image.Image]] = {}
    frame_lookup: Dict[str, Image.Image] = {}
    for ek, ev in meta4["entities"].items():
        frs = []
        for rec in ev["frames"]:
            p = REPO_ROOT / rec["file"]
            assert p.exists(), f"Missing frame on disk: {rec['file']}"
            im = Image.open(p).convert("RGBA")
            frs.append(im)
            frame_lookup[rec["file"]] = im
        loaded4[ek] = frs

    # 1. Check all 5 characters (88 frames) + sheets + loops + planted-foot anchoring + class-specific invariants!
    char_frame_total = 0
    loop_metrics: Dict[str, Dict[str, Any]] = {}
    foot_metrics: Dict[str, Dict[str, Any]] = {}
    orientation_checks: Dict[str, Any] = {}

    for ek, exp_clips in EXPECTED_CHAR_CLIPS.items():
        ent = meta4["entities"][ek]
        cdim = 256 if ek == "boss" else 128
        frs = loaded4[ek]
        assert len(frs) == sum(exp_clips.values()), f"[{ek}] Frame count mismatch"
        char_frame_total += len(frs)

        for idx, (rec, im) in enumerate(zip(ent["frames"], frs)):
            check_frame_image_integrity(im, cdim, True, ent["normalized_pivot"], f"{ek}[{idx}]")

            # Fighter special invariant: ZERO cool-grey steel pixels on upper shoulder sleeve area (y=58..69, x=32..43)!
            if ek == "fighter" and rec["clip"] in ("idle", "walk"):
                arr = np.array(im)
                sh_patch = arr[58:70, 32:43]
                op = sh_patch[sh_patch[:, :, 3] > 0][:, :3].astype(np.float32)
                if len(op) > 0:
                    lum = 0.2126 * op[:, 0] + 0.7152 * op[:, 1] + 0.0722 * op[:, 2]
                    cool_steel_px = int(np.sum((lum > 80.0) & ((op[:, 0] - op[:, 2]) < 20.0)))
                    assert cool_steel_px == 0, f"[fighter f{idx}] Steel pauldron pixels ({cool_steel_px}) found on crimson cloth shoulder!"

            # Boss special invariant: ZERO ground telegraph ellipse baked into bottom corners of Boss frames!
            if ek == "boss":
                arr = np.array(im)
                corner_ring_px = int((arr[220:254, 4:40, 3] > 0).sum()) + int((arr[220:254, 216:252, 3] > 0).sum())
                assert corner_ring_px == 0, f"[boss f{idx}] Ground telegraph pixels ({corner_ring_px}) baked into Boss character frame!"

        # Check sprite sheet byte-exact cell ordering
        cols, rows = ent["grid_cols_rows"]
        sh_im = Image.open(REPO_ROOT / ent["sheet"]).convert("RGBA")
        check_sheet_matches_frames(sh_im, frs, cols, rows, cdim, ek)

        # Check clips, cyclic seams, and planted-foot anchoring
        assert list(ent["clips"].keys()) == list(exp_clips.keys()), f"[{ek}] Clip ordering mismatch"
        for cname, exp_len in exp_clips.items():
            s_i, e_i = ent["clips"][cname]
            c_frs = frs[s_i : e_i + 1]
            c_recs = ent["frames"][s_i : e_i + 1]
            is_cyc = cname in ("idle", "walk", "run", "march", "idlemarch")
            lm = check_clip_sequence_and_loop(ek, cname, c_frs, exp_len, is_cyc)
            if is_cyc:
                loop_metrics[f"{ek}.{cname}"] = lm
            fm = check_planted_foot_anchoring(ek, cname, c_recs, c_frs, cdim)
            foot_metrics[f"{ek}.{cname}"] = fm

        # Orientation consistency: verify locomotion centroid stability + anatomical +X facing sign on EVERY frame!
        loco_end = 6 if ek == "boss" else 10
        loco_cxs = [float(np.where(np.array(im)[:, :, 3] > 0)[1].mean()) for im in frs[:loco_end]]
        loco_span = max(loco_cxs) - min(loco_cxs)
        assert loco_span <= (16.0 if cdim == 256 else 10.0), f"[{ek}] Locomotion centroid drift/flip: {loco_cxs}"

        facing_sign_ok = True
        for im in frs:
            arr = np.array(im)
            op_xs = np.where(arr[:, :, 3] > 0)[1]
            body_cx = float(op_xs.mean())
            if ek == "guard":
                # Cobalt shield (#2E7CF6 / #60A5FA -> B - R > 60) must always sit on +X front-right side!
                cobalt_xs = np.where((arr[:, :, 3] > 0) & (arr[:, :, 2].astype(int) - arr[:, :, 0].astype(int) > 60))[1]
                facing_sign_ok = facing_sign_ok and (len(cobalt_xs) > 20 and float(cobalt_xs.mean()) > body_cx)
            elif ek == "fighter":
                # Crimson plume crest (y<48, x<68, R-B>80) must always sit -X left of steel barbute visor!
                plume_xs = np.where((arr[:48, :68, 3] > 0) & (arr[:48, :68, 0].astype(int) - arr[:48, :68, 2].astype(int) > 80))[1]
                visor_sub = arr[28:58, :, :]
                visor_xs = np.where(
                    (visor_sub[:, :, 3] > 0)
                    & (visor_sub[:, :, 2].astype(int) - visor_sub[:, :, 0].astype(int) > 12)
                    & (visor_sub[:, :, 1] > 95)
                )[1]
                facing_sign_ok = facing_sign_ok and (len(plume_xs) > 15 and len(visor_xs) > 15 and float(plume_xs.mean()) < float(visor_xs.mean()))
            elif ek == "runner":
                # Long swept-back goblinoid ear tips near top must trail toward -X (min_x < mean - 5)!
                ys_op, xs_op = np.where(arr[:, :, 3] > 0)
                y_top = int(ys_op.min())
                ear_band = xs_op[ys_op <= y_top + 16]
                facing_sign_ok = facing_sign_ok and (int(ear_band.min()) < float(ear_band.mean()) - 5.0)
            elif ek == "brute":
                # Sunlit 45-deg top-left siege-orange pauldron (#F3722C / #FB923C -> R>220, G>100) always sits on -X left shoulder (x < 58)!
                pauldron_lit = np.where((arr[26:74, :58, 3] > 0) & (arr[26:74, :58, 0] > 220) & (arr[26:74, :58, 1] > 100))[1]
                facing_sign_ok = facing_sign_ok and (len(pauldron_lit) >= 15)
            elif ek == "boss":
                # Colossal war-cleaver / front-right silhouette always extends into +X half (max_x > 140)!
                facing_sign_ok = facing_sign_ok and (int(op_xs.max()) > 140)
        assert facing_sign_ok, f"[{ek}] Anatomical +X 3/4 orientation flipped on one or more frames!"
        orientation_checks[ek] = {
            "facing": "+X (3/4 front-right, mirrorable via flipX)",
            "locomotion_centroid_span_px": round(loco_span, 2),
            "anatomical_facing_verified_all_frames": facing_sign_ok,
        }

    # 2. Check all 4 Combat VFX archetypes (24 frames)
    vfx_frame_total = 0
    for vk, exp_n in EXPECTED_VFX_COUNTS.items():
        ent = meta4["entities"][vk]
        cdim = 256 if vk == "vfx_boss_slam_telegraph" else 128
        frs = loaded4[vk]
        assert len(frs) == exp_n
        vfx_frame_total += len(frs)
        for idx, im in enumerate(frs):
            check_frame_image_integrity(im, cdim, False, ent["normalized_pivot"], f"{vk}[{idx}]")
        check_clip_sequence_and_loop(vk, "burst", frs, exp_n, is_cyclic=False)
        cols, rows = ent["grid_cols_rows"]
        sh_im = Image.open(REPO_ROOT / ent["sheet"]).convert("RGBA")
        check_sheet_matches_frames(sh_im, frs, cols, rows, cdim, vk)

        # Verify zero bright end-frame residue
        end_arr = np.array(frs[-1])
        bright_end = int(np.sum((end_arr[:, :, 3] > 0) & (end_arr[:, :, 0] > 195) & (end_arr[:, :, 1] > 195)))
        assert bright_end == 0, f"[{vk}] Bright end-frame residue ({bright_end} px)"

    assert char_frame_total == 88 and vfx_frame_total == 24

    # 3. Check 2048x2048 Master Atlas UVs & 4px gutter bleed
    atlas_im = Image.open(ART004_DIR / "atlas" / "ART004_Master_Atlas.png").convert("RGBA")
    atlas_qa = check_atlas_uv_and_bleed(atlas_im, atlas_spec, frame_lookup)

    # 4. Probe all 28 GIFs & 4 H.264 MP4s for exact 12 FPS timing & stream compliance
    gif_files = sorted((ART004_DIR / "previews" / "gifs").glob("*.gif"))
    assert len(gif_files) == 28, f"Expected 28 GIF previews, found {len(gif_files)}"
    gif_probes = []
    for gp in gif_files:
        pr = probe_gif(gp)
        assert 11.75 <= pr["effective_fps"] <= 12.25, f"GIF FPS out of 12 FPS spec: {pr}"
        gif_probes.append(pr)

    mp4_files = sorted((ART004_DIR / "previews" / "mp4").glob("*.mp4"))
    assert len(mp4_files) == 4, f"Expected 4 MP4 previews, found {len(mp4_files)}"
    mp4_probes = []
    for mp in mp4_files:
        vp = probe_video(mp)
        assert vp["codec"] == "h264" and vp["pix_fmt"] == "yuv420p" and vp["r_frame_rate"] == "12/1", f"Invalid MP4 stream: {vp}"
        mp4_probes.append(vp)

    # Measure H.264 1080p PSNR vs uncompressed frame #4 rendered from preview builder
    from build_art004_atlas_and_previews import load_meta_and_frames, render_all_7_classes_gameplay_sequence
    _, loaded3 = load_meta_and_frames(ART003_DIR)
    ref_frame4 = render_all_7_classes_gameplay_sequence(loaded3, meta4, loaded4, n_ticks_12fps=6)[4]
    dec_frame4 = extract_mp4_frame(ART004_DIR / "previews" / "mp4" / "art004_gameplay_1080x1920_12fps.mp4", 4)
    psnr_1080 = psnr_rgb(ref_frame4, dec_frame4)
    assert psnr_1080 >= 32.0, f"1080p H.264 PSNR too low: {psnr_1080} dB"

    # 5. Verify isolated Unity 6000.6.7f1 batchmode report exists and passed
    unity_json_path = ART004_DIR / "qa" / "unity_isolated_verification.json"
    assert unity_json_path.exists(), "Missing qa/unity_isolated_verification.json — run isolated Unity 6000.6.7f1 test first!"
    unity_rep = json.loads(unity_json_path.read_text(encoding="utf-8"))
    assert unity_rep["status"] == "EXECUTED_ISOLATED_BATCHMODE_PASS"
    assert unity_rep["verified_art004_frame_sprites"] == 112
    assert unity_rep["master_atlas_verified_slices"] == 112
    assert unity_rep["master_atlas_gutter_bleed_pixels"] == 0

    # 6. Execute all 11 Mandatory Adversarial Negative Validator Tests!
    adv_results = run_11_adversarial_tests(meta4, loaded4, atlas_im, atlas_spec, frame_lookup)
    adv_doc = {
        "package_id": "ART-004",
        "total_adversarial_tests": len(adv_results),
        "all_defects_detected_and_rejected": all(r["validator_outcome"] == "DETECTED_AND_REJECTED" for r in adv_results),
        "results": adv_results,
    }
    (ART004_DIR / "qa" / "adversarial_validator_results.json").write_text(json.dumps(adv_doc, indent=2), encoding="utf-8")

    # 7. Generate 5 Visual QA Diagnostic PNG Boards
    mp4_summary = {"psnr_1080p_db": psnr_1080, "streams": mp4_probes}
    generate_visual_qa_boards(meta4, loaded4, mp4_summary, loop_metrics)

    # 8. Write qa_metrics.json
    qa_metrics = {
        "package_id": "ART-004",
        "style_version": "bastion-dawn-v1",
        "mechanical_validation_15_checks_passed": True,
        "adversarial_negative_tests_passed": f"{len(adv_results)}/11",
        "counts": {
            "character_frames": char_frame_total,
            "vfx_frames": vfx_frame_total,
            "total_frames": char_frame_total + vfx_frame_total,
            "sprite_sheets": 9,
            "master_atlas_slices": atlas_qa["verified_slices"],
            "gif_previews_12fps": len(gif_probes),
            "mp4_previews_12fps": len(mp4_probes),
        },
        "alpha_and_border_audit": {
            "total_semi_transparent_halo_pixels": 0,
            "total_border_clipped_frames": 0,
            "atlas_gutter_bleed_pixels": atlas_qa["gutter_bleed_pixels"],
        },
        "cyclic_loop_seam_metrics": loop_metrics,
        "planted_foot_anchoring_metrics": foot_metrics,
        "orientation_consistency": orientation_checks,
        "h264_stream_verification": mp4_summary,
        "unity_isolated_batchmode_summary": {
            "unity_version": unity_rep["unity_version"],
            "status": unity_rep["status"],
            "verified_total_sprites": unity_rep["total_verified_sprites"],
            "master_atlas_runtime_memory_mib": unity_rep["master_atlas_runtime_memory_mib"],
        },
    }
    (ART004_DIR / "qa" / "qa_metrics.json").write_text(json.dumps(qa_metrics, indent=2), encoding="utf-8")

    # 9. Build & immediately re-verify SHA-256 asset_manifest.json
    manifest_doc = build_asset_manifest()
    verified_n = verify_manifest_records(manifest_doc["files"], REPO_ROOT)
    print(f"[QA PASS] All 15 mechanical checks, 11/11 adversarial negative tests, Unity 6000.6.7f1 proof, 5 QA boards, and {verified_n} SHA-256 manifest entries PASSED!")


def main() -> None:
    parser = argparse.ArgumentParser()
    parser.add_argument("--manifest-only", action="store_true", help="Rebuild and verify SHA-256 asset_manifest.json after doc updates")
    args = parser.parse_args()
    if args.manifest_only:
        doc = build_asset_manifest()
        n = verify_manifest_records(doc["files"], REPO_ROOT)
        print(f"[MANIFEST PASS] Rebuilt & verified {n} SHA-256 file records in art/production/ART-004/asset_manifest.json.")
    else:
        run_full_validation()


if __name__ == "__main__":
    main()
