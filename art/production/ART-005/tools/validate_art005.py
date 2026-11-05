#!/usr/bin/env python3
"""
ART-005 — Independent Mechanical & 12-Case Adversarial QA Validator
Performs real programmatic assertions across all 26 canonical sprites, 14 companion depth/transition sprites,
both 2048x2048 RGBA32 Unity atlases, 3 deterministic JSON contracts, 8 multi-class 1080x1920 gameplay composites,
mobile/H.264 stream outputs, Unity isolated batch verification report, and SHA-256 asset_manifest.json.
Also runs 12 adversarial negative-injection tests and verifies every single one fails with ValidationFailure.
"""

from __future__ import annotations

import argparse
import copy
import hashlib
import json
import math
import shutil
import tempfile
from pathlib import Path
from typing import Any, Callable, Dict, List, Optional, Tuple

import numpy as np
from PIL import Image

TOOLS_DIR = Path(__file__).resolve().parent
ART005_DIR = TOOLS_DIR.parent
REPO_ROOT = ART005_DIR.parents[2]


class ValidationFailure(AssertionError):
    pass


def require(cond: bool, msg: str) -> None:
    if not cond:
        raise ValidationFailure(msg)


def sha256_file(p: Path) -> str:
    h = hashlib.sha256()
    with p.open("rb") as f:
        for chunk in iter(lambda: f.read(65536), b""):
            h.update(chunk)
    return h.hexdigest()


def max_hsv_saturation(rgb: np.ndarray) -> float:
    f = rgb.astype(np.float32) / 255.0
    cmax = f.max(axis=-1)
    cmin = f.min(axis=-1)
    delta = cmax - cmin
    sat = np.where(cmax > 1e-6, delta / np.maximum(cmax, 1e-6), 0.0)
    return float(sat.max())


def laplacian_sd(rgb: np.ndarray) -> float:
    f = rgb.astype(np.float32)
    luma = 0.2126 * f[..., 0] + 0.7152 * f[..., 1] + 0.0722 * f[..., 2]
    lap = (
        -4.0 * luma
        + np.roll(luma, 1, axis=0)
        + np.roll(luma, -1, axis=0)
        + np.roll(luma, 1, axis=1)
        + np.roll(luma, -1, axis=1)
    )
    return float(np.std(lap))


def select_wall_state(mapping: Dict[str, Any], wall_hp: float, wall_max: float, is_post_loss_recovery: bool) -> str:
    states = sorted(mapping["wall_damage_states"], key=lambda s: int(s["priority_order"]))
    ratio = 0.0 if wall_max <= 0 else (wall_hp / wall_max)
    for st in states:
        key = st["state_key"]
        if key == "post_loss_recovery":
            if is_post_loss_recovery and wall_hp > 0:
                return st["sprite_id"]
            continue
        if key == "breached":
            if wall_hp <= 0:
                return st["sprite_id"]
            continue
        if wall_hp <= 0:
            continue
        lo_inc = st.get("min_ratio_inclusive")
        lo_exc = st.get("min_ratio_exclusive")
        hi_inc = st.get("max_ratio_inclusive")
        hi_exc = st.get("max_ratio_exclusive")
        ok = True
        if lo_inc is not None and ratio < float(lo_inc) - 1e-9:
            ok = False
        if lo_exc is not None and ratio <= float(lo_exc):
            ok = False
        if hi_inc is not None and ratio > float(hi_inc) + 1e-9:
            ok = False
        if hi_exc is not None and ratio >= float(hi_exc) - 1e-9:
            ok = False
        if ok:
            return st["sprite_id"]
    raise ValidationFailure(f"Unmapped wall state: hp={wall_hp}, max={wall_max}, recovery={is_post_loss_recovery}")


def build_asset_manifest(pkg_dir: Path, repo_root: Path) -> Dict[str, Any]:
    files_out: List[Dict[str, Any]] = []
    for p in sorted(pkg_dir.rglob("*")):
        if not p.is_file():
            continue
        if "__pycache__" in p.parts or p.suffix == ".pyc":
            continue
        if p.name == "asset_manifest.json":
            continue
        rel = p.relative_to(repo_root).as_posix()
        entry: Dict[str, Any] = {
            "path": rel,
            "bytes": p.stat().st_size,
            "sha256": sha256_file(p),
        }
        if p.suffix.lower() == ".png":
            with Image.open(p) as im:
                entry["dimensions"] = [im.width, im.height]
                entry["mode"] = im.mode
        files_out.append(entry)

    return {
        "package_id": "ART-005",
        "title": "Bastion Dawn v1 — Production Environment, Castle, Wall & Three-Lane Battlefield",
        "style_lock": "Bastion Dawn v1",
        "engine_target": "Unity 6000.6.7f1 (Built-in 2D, Orthographic Size 5.0, PPU 192)",
        "normative_disambiguation": "Work package ART-005 (environment production); distinct from requirement ID ART005 (combat VFX) in docs/plan/07_UNITY_ART_UX.md.",
        "m06_integration_status": "NOT_STARTED",
        "rights_status": "PENDING_REVIEW",
        "canonical_sprite_count": 26,
        "companion_sprite_count": 14,
        "atlas_count": 2,
        "gameplay_composite_count": 8,
        "total_tracked_files": len(files_out),
        "files": files_out,
    }


def validate_bundle(
    pkg_dir: Path,
    repo_root: Path,
    check_manifest: bool = True,
    overrides: Optional[Dict[str, Any]] = None,
) -> Dict[str, Any]:
    ov = overrides or {}

    layout = copy.deepcopy(ov.get("layout") or json.loads((pkg_dir / "environment_layout.json").read_text(encoding="utf-8")))
    mapping = copy.deepcopy(ov.get("mapping") or json.loads((pkg_dir / "environment_state_mapping.json").read_text(encoding="utf-8")))
    lighting = copy.deepcopy(ov.get("lighting") or json.loads((pkg_dir / "lighting_palettes.json").read_text(encoding="utf-8")))
    a1_meta = copy.deepcopy(ov.get("atlas1_meta") or json.loads((pkg_dir / "atlas/environment_atlas_01_fortress.json").read_text(encoding="utf-8")))
    a2_meta = copy.deepcopy(ov.get("atlas2_meta") or json.loads((pkg_dir / "atlas/environment_atlas_02_layers_and_terrain.json").read_text(encoding="utf-8")))
    telemetry = copy.deepcopy(ov.get("telemetry") or json.loads((pkg_dir / "previews/gameplay_composites/composite_scene_telemetry.json").read_text(encoding="utf-8")))

    def resolve_path(rel_repo_path: str) -> Path:
        if "file_map" in ov and rel_repo_path in ov["file_map"]:
            return Path(ov["file_map"][rel_repo_path])
        return repo_root / rel_repo_path

    # ------------------------------------------------------------------------
    # 1. CANONICAL 26 + COMPANION 14 SPRITE CHECKS
    # ------------------------------------------------------------------------
    canon = layout["canonical_sprites"]
    comp = layout["companion_assets"]
    require(len(canon) == 26, f"Expected exactly 26 canonical sprites, got {len(canon)}")
    require(len(comp) == 14, f"Expected exactly 14 companion sprites, got {len(comp)}")

    expected_pivots = {
        "castle_townhall": [0.50, 0.10],
        "defensive_wall_state": [0.50, 0.14],
        "wall_repair_scaffold": [0.50, 0.10],
        "village_watchtower": [0.50, 0.10],
        "village_cottage": [0.50, 0.12],
        "courtyard_prop": [0.50, 0.12],
        "terrain_tile": [0.50, 0.50],
    }

    castle_bottom_ys: List[int] = []
    wall_spans: List[Tuple[int, int]] = []
    tile_arrays: Dict[str, np.ndarray] = {}
    cutout_halo_total = 0

    for sp in canon + comp:
        sid = sp["id"]
        cat = sp["category"]
        p = resolve_path(sp["path"])
        require(p.exists(), f"Missing sprite file: {sp['path']}")
        with Image.open(p) as im_raw:
            require(im_raw.mode == "RGBA", f"Sprite {sid} must be RGBA, got {im_raw.mode}")
            require([im_raw.width, im_raw.height] == sp["dimensions"], f"Dimension mismatch on {sid}")
            arr = np.array(im_raw)

        if cat in expected_pivots:
            require(sp["normalized_pivot"] == expected_pivots[cat], f"Invalid pivot on {sid}: {sp['normalized_pivot']} != {expected_pivots[cat]}")

        alpha = arr[:, :, 3]
        if cat == "terrain_tile":
            require(int((alpha == 255).sum()) == arr.shape[0] * arr.shape[1], f"Terrain tile {sid} must be 100% opaque")
            # Zero #11151C black ink on ground/road tiles!
            ink_px = int(((arr[:, :, 0] == 17) & (arr[:, :, 1] == 21) & (arr[:, :, 2] == 28)).sum())
            require(ink_px == 0, f"Terrain tile {sid} contains {ink_px} forbidden #11151C black ink pixels")
            sat = max_hsv_saturation(arr[:, :, :3])
            require(sat <= 0.485, f"Terrain tile {sid} exceeded calm saturation ceiling: {sat:.3f}")
            lsd = laplacian_sd(arr[:, :, :3])
            require(lsd < 16.0, f"Terrain tile {sid} high-frequency Laplacian SD too noisy for H.264: {lsd:.2f}")
            if sid == "env_tile_05_limestone_causeway":
                # Verify Boss purple telegraph is NOT baked into causeway tile
                purple_px = int(((arr[:, :, 0] > 160) & (arr[:, :, 2] > 190) & (arr[:, :, 1] < 120)).sum())
                require(purple_px == 0, f"Boss telegraph magenta/purple baked into causeway tile: {purple_px} px")
            tile_arrays[sid] = arr
        else:
            # Cutout / transition sprite: strict binary alpha (0 halo pixels!) & genuine transparent background
            semi = int(((alpha > 0) & (alpha < 255)).sum())
            cutout_halo_total += semi
            require(semi == 0, f"Sprite {sid} has {semi} semi-transparent halo pixels")
            fill_ratio = float((alpha == 255).sum()) / float(alpha.size)
            require(0.02 <= fill_ratio <= 0.85, f"Cutout sprite {sid} invalid fill ratio {fill_ratio:.3f} (non-transparent background or empty)")

            ys, xs = np.where(alpha > 0)
            x0, y0, x1, y1 = int(xs.min()), int(ys.min()), int(xs.max()), int(ys.max())
            w, h = arr.shape[1], arr.shape[0]

            # Border clipping checks (allow intentional tiling axes on verge/curb strips)
            if sid == "trans_ochre_verge_strip":
                require(y0 >= 2, f"{sid} clipped at top border")
            elif sid == "trans_causeway_stone_curb":
                require(x0 >= 2 and x1 <= w - 3, f"{sid} clipped at X border")
            else:
                require(x0 >= 2 and y0 >= 2 and x1 <= w - 3 and y1 <= h - 3, f"Sprite {sid} clipped at canvas border: bbox=({x0},{y0},{x1},{y1}) in {w}x{h}")

            if cat == "castle_townhall":
                castle_bottom_ys.append(y1)
            elif cat == "defensive_wall_state":
                wall_spans.append((x0, x1))
                # Verify courtyard center (256, 256) and exterior corner (4, 4) are genuinely transparent!
                require(int(alpha[256, 256]) == 0 and int(alpha[4, 4]) == 0, f"Wall sprite {sid} interior courtyard or exterior is not transparent")

    require(len(set(castle_bottom_ys)) == 1, f"Castle progression foundation drift detected across bands: {castle_bottom_ys}")
    for wx0, wx1 in wall_spans:
        require((wx1 - wx0 + 1) == 477, f"Wall outer width must be 477 px (44.17% of 1080p), got {wx1 - wx0 + 1}")

    # ------------------------------------------------------------------------
    # 2. TOWNHALL LEVEL-BAND & WALL HP STATE MAPPING CONTRACTS
    # ------------------------------------------------------------------------
    require(mapping["gameplay_invariants"]["townhall_authoritative_hp"] == 500, "Authoritative townhall_hp must remain 500")
    require(mapping["gameplay_invariants"]["townhall_bands_are_cosmetic_only"] is True, "Townhall bands must be cosmetic only")

    bands = sorted(mapping["townhall_level_bands"], key=lambda b: int(b["band"]))
    require(len(bands) == 4, "Must define exactly 4 Townhall level bands")
    covered_levels: List[int] = []
    canon_ids = {s["id"] for s in canon}
    for b in bands:
        require(b["sprite_id"] in canon_ids, f"Unknown castle sprite_id in mapping: {b['sprite_id']}")
        lo, hi = int(b["min_level"]), int(b["max_level"])
        require(lo <= hi, f"Invalid band range {lo}..{hi}")
        covered_levels.extend(range(lo, hi + 1))
    require(covered_levels == list(range(1, 51)), f"Townhall bands must partition levels 1..50 without gap/overlap, got count={len(covered_levels)}")

    # Exhaustive Wall HP mapping test across all 20 tiers (1000..4800 HP)
    for tier in range(1, 21):
        wmax = 1000.0 + 200.0 * (tier - 1)
        require(select_wall_state(mapping, wmax, wmax, False) == "env_wall_state1_healthy", "100% HP must map to healthy")
        require(select_wall_state(mapping, 0.60 * wmax, wmax, False) == "env_wall_state1_healthy", "60% HP must map to healthy")
        require(select_wall_state(mapping, 0.59 * wmax, wmax, False) == "env_wall_state2_damaged", "59% HP must map to damaged")
        require(select_wall_state(mapping, 0.25 * wmax, wmax, False) == "env_wall_state2_damaged", "25% HP must map to damaged")
        require(select_wall_state(mapping, 0.24 * wmax, wmax, False) == "env_wall_state3_critical", "24% HP must map to critical")
        require(select_wall_state(mapping, 1.0, wmax, False) == "env_wall_state3_critical", "1 HP must map to critical")
        require(select_wall_state(mapping, 0.0, wmax, False) == "env_wall_state4_breached", "0 HP must map to breached")
        require(select_wall_state(mapping, 250.0, wmax, True) == "env_wall_state5_recovery", "post_loss_recovery floor must map to state 5")

    # ------------------------------------------------------------------------
    # 3. SEAMLESS TERRAIN TILE WRAP & ROAD-TO-MEADOW TRANSITION CONTINUITY
    # ------------------------------------------------------------------------
    seam_metrics: Dict[str, Any] = {}
    for tid, tarr in tile_arrays.items():
        rgb = tarr[:, :, :3].astype(np.float32)
        int_dx = float(np.mean(np.abs(rgb[:, 1:, :] - rgb[:, :-1, :])))
        int_dy = float(np.mean(np.abs(rgb[1:, :, :] - rgb[:-1, :, :])))
        wrap_y = float(np.mean(np.abs(rgb[0, :, :] - rgb[-1, :, :])))
        max_wy = float(np.max(np.abs(rgb[0, :, :] - rgb[-1, :, :])))
        require(max_wy <= 16.0, f"Vertical tile seam jump in {tid}: max_wrap_y={max_wy:.1f}")
        if tid != "env_tile_04_ochre_road_border":
            wrap_x = float(np.mean(np.abs(rgb[:, 0, :] - rgb[:, -1, :])))
            max_wx = float(np.max(np.abs(rgb[:, 0, :] - rgb[:, -1, :])))
            require(max_wx <= 16.0, f"Horizontal tile seam jump in {tid}: max_wrap_x={max_wx:.1f}")
            seam_metrics[tid] = {
                "wrap_x_mae": round(wrap_x, 3),
                "wrap_y_mae": round(wrap_y, 3),
                "max_wrap_x_diff": round(max_wx, 1),
                "max_wrap_y_diff": round(max_wy, 1),
                "interior_dx_mae": round(int_dx, 3),
                "interior_dy_mae": round(int_dy, 3),
            }
        else:
            seam_metrics[tid] = {"wrap_y_mae": round(wrap_y, 3), "max_wrap_y_diff": round(max_wy, 1)}

    # Explicit cross-tile transition verification:
    # (a) env_tile_01_meadow_main right edge (x=255) -> env_tile_04_ochre_road_border left edge (x=0)
    m1_rgb = tile_arrays["env_tile_01_meadow_main"][:, :, :3].astype(np.float32)
    m2_rgb = tile_arrays["env_tile_02_meadow_variation"][:, :, :3].astype(np.float32)
    b4_rgb = tile_arrays["env_tile_04_ochre_road_border"][:, :, :3].astype(np.float32)
    r3_rgb = tile_arrays["env_tile_03_ochre_road_body"][:, :, :3].astype(np.float32)

    meadow_to_border_diff = float(np.max(np.abs(m1_rgb[:, -1, :] - b4_rgb[:, 0, :])))
    border_to_road_diff = float(np.max(np.abs(b4_rgb[:, -1, :] - r3_rgb[:, 0, :])))
    meadow_cross_diff = float(np.max(np.abs(m1_rgb[:, -1, :] - m2_rgb[:, 0, :])))
    require(meadow_to_border_diff <= 14.0, f"Broken road-to-meadow transition (meadow->border max diff={meadow_to_border_diff:.1f})")
    require(border_to_road_diff <= 14.0, f"Broken road-to-meadow transition (border->ochre max diff={border_to_road_diff:.1f})")
    require(meadow_cross_diff <= 14.0, f"Broken cross-meadow transition (tile_01->tile_02 max diff={meadow_cross_diff:.1f})")

    # ------------------------------------------------------------------------
    # 4. ATLAS UV, BYTE-EXACT SLICE MATCH & ZERO GUTTER BLEED
    # ------------------------------------------------------------------------
    for ameta in (a1_meta, a2_meta):
        require(ameta["dimensions"] == [2048, 2048], f"Atlas {ameta['atlas_id']} exceeds/mismatches 2048x2048")
        ap = resolve_path(ameta["atlas_path"])
        require(ap.exists(), f"Missing atlas PNG: {ameta['atlas_path']}")
        at_arr = np.array(Image.open(ap).convert("RGBA"))
        occ = np.zeros((2048, 2048), dtype=bool)
        for sl in ameta["slices"]:
            rx, ry, rw, rh = sl["rect_top_left_px"]
            require(0 <= rx and 0 <= ry and rx + rw <= 2048 and ry + rh <= 2048, f"UV rect out of bounds for {sl['id']}")
            require(not occ[ry : ry + rh, rx : rx + rw].any(), f"Overlapping slice rect in {ameta['atlas_id']} at {sl['id']}")
            occ[ry : ry + rh, rx : rx + rw] = True

            # Verify Unity bottom-left UV math matches top-left pixel rect
            uy = 2048 - (ry + rh)
            require(sl["rect_unity_bottom_left_px"] == [rx, uy, rw, rh], f"Unity bottom-left rect mismatch on {sl['id']}")

            # Byte-exact texel match against standalone source PNG!
            src_arr = np.array(Image.open(resolve_path(sl["source_path"])).convert("RGBA"))
            sub_arr = at_arr[ry : ry + rh, rx : rx + rw]
            require(src_arr.shape == sub_arr.shape and np.array_equal(src_arr, sub_arr), f"Atlas UV/texel mismatch on {sl['id']}")

        bleed = int((at_arr[:, :, 3][~occ] > 0).sum())
        require(bleed == 0, f"Atlas {ameta['atlas_id']} has {bleed} gutter bleed pixels")

    # ------------------------------------------------------------------------
    # 5. SORTING LAYERS, BOSS VISIBILITY & LIVESTREAM SAFE-ZONE ENVELOPE
    # ------------------------------------------------------------------------
    sl_names = [s["name"] for s in sorted(layout["sorting_layers"], key=lambda x: int(x["index"]))]
    require(
        sl_names == ["Ground", "LanesAndDecals", "BastionBack", "Units", "BastionFront", "VFX"],
        f"Invalid sorting layer stack: {sl_names}",
    )
    sl_rank = {name: i for i, name in enumerate(sl_names)}

    safe_x_px = float(layout["livestream_safe_zones"]["right_rail_x_px_limit"])
    safe_y_px = float(layout["livestream_safe_zones"]["bottom_overlay_y_px_limit"])
    require(abs(safe_x_px - 885.6) < 0.2 and abs(safe_y_px - 1689.6) < 0.2, "Safe zone limits must match 82% (885.6px) and 88% (1689.6px)")

    for ln in layout["lanes"]:
        if "max_right_edge_px" in ln:
            require(float(ln["max_right_edge_px"]) <= safe_x_px, f"Lane {ln['lane_id']} violates right 82% safe zone: {ln['max_right_edge_px']}")
        for pt in ln["centerline_screen_px"]:
            require(float(pt[0]) <= safe_x_px and float(pt[1]) <= safe_y_px, f"Lane centerline point {pt} violates safe zone")

    for pl in layout["courtyard_placements"]:
        sx, sy = float(pl["screen_px"][0]), float(pl["screen_px"][1])
        require(sx <= safe_x_px and sy <= safe_y_px, f"Courtyard placement {pl['id']} violates safe zone at ({sx}, {sy})")

    # Verify all 8 mandatory gameplay composites + Boss sorting + Safe-Zone bboxes
    require(len(telemetry) == 8, f"Expected 8 gameplay composites in telemetry, got {len(telemetry)}")
    boss_seen = False
    for sc_key, sc_data in telemetry.items():
        cp = resolve_path(sc_data["path"])
        require(cp.exists(), f"Missing gameplay composite {sc_data['path']}")
        with Image.open(cp) as cim:
            require(cim.size == (1080, 1920), f"Composite {sc_key} must be 1080x1920")

        for b in sc_data["buildings"]:
            bx0, by0, bx1, by1 = b["bbox"]
            require(bx1 <= safe_x_px and by1 <= safe_y_px, f"Building {b['id']} in {sc_key} enters restricted safe zone: {b['bbox']}")

        for u in sc_data["units"]:
            ux0, uy0, ux1, uy1 = u["bbox"]
            require(ux1 <= safe_x_px and uy1 <= safe_y_px, f"Unit {u['class']} in {sc_key} enters restricted safe zone: {u['bbox']}")
            u_layer = u["sorting_layer"]
            require(u_layer in sl_rank, f"Unknown unit sorting layer {u_layer}")
            if u["class"] == "boss":
                boss_seen = True
                require(
                    sl_rank[u_layer] > sl_rank["BastionBack"] and sl_rank[u_layer] > sl_rank["LanesAndDecals"],
                    f"Boss sorting layer {u_layer} hides Boss behind environment/lanes!",
                )
                # Also verify Boss is north of BastionFront parapet (never occluded by front wall!)
                require(uy1 < 760, f"Boss bbox {u['bbox']} overlaps BastionFront southern wall slice!")

    require(boss_seen, "Boss must be verified in at least one gameplay composite")

    # Verify 4 lighting palettes & Unity isolated verification report
    require(len(lighting["palettes"]) == 4, "Must provide 4 lighting palettes")
    unity_rep_path = pkg_dir / "qa/unity_isolated_verification.json"
    require(unity_rep_path.exists(), "Missing qa/unity_isolated_verification.json")
    unity_rep = json.loads(unity_rep_path.read_text(encoding="utf-8"))
    require(unity_rep.get("status") == "PASS" and unity_rep.get("verified_canonical_sprites") == 26, "Unity isolated verification did not pass 26 canonical sprites")

    # Verify mobile & H.264 stream deliverables exist
    for req_rel in (
        "previews/mobile/mobile_720p_assault_three_lanes.png",
        "previews/mobile/mobile_360p_assault_three_lanes.png",
        "previews/mobile/mobile_360p_grayscale_readability.png",
        "previews/mobile/reduced_motion_boss_eclipse_proof.png",
        "previews/mobile/stream_turntable_h264.mp4",
        "previews/mobile/h264_mobile_stream_proof.png",
        "previews/comparison/art002_vs_art005_blueprint_comparison.png",
    ):
        require((pkg_dir / req_rel).exists(), f"Missing required preview file: {req_rel}")

    # ------------------------------------------------------------------------
    # 6. SHA-256 ASSET MANIFEST INTEGRITY
    # ------------------------------------------------------------------------
    if check_manifest:
        man_path = pkg_dir / "asset_manifest.json"
        require(man_path.exists(), "Missing asset_manifest.json")
        man = ov.get("manifest") or json.loads(man_path.read_text(encoding="utf-8"))
        require(man.get("rights_status") == "PENDING_REVIEW", "rights_status must remain PENDING_REVIEW")
        require(man.get("m06_integration_status") == "NOT_STARTED", "m06_integration_status must remain NOT_STARTED")
        for f_rec in man["files"]:
            fp = resolve_path(f_rec["path"])
            require(fp.exists(), f"Manifest file missing on disk: {f_rec['path']}")
            actual_sha = sha256_file(fp)
            require(actual_sha == f_rec["sha256"], f"SHA-256 mismatch for {f_rec['path']}: {actual_sha} != {f_rec['sha256']}")
            require(fp.stat().st_size == int(f_rec["bytes"]), f"Byte size mismatch for {f_rec['path']}")

    return {
        "status": "PASS",
        "canonical_sprites_verified": 26,
        "companion_sprites_verified": 14,
        "cutout_halo_pixels_total": cutout_halo_total,
        "castle_foundation_bottom_y_px": castle_bottom_ys[0],
        "castle_swap_drift_px": 0,
        "wall_outer_width_px": 477,
        "wall_width_pct_of_1080p": 44.17,
        "meadow_to_border_max_diff": round(meadow_to_border_diff, 2),
        "border_to_road_max_diff": round(border_to_road_diff, 2),
        "meadow_cross_tile_max_diff": round(meadow_cross_diff, 2),
        "tile_seam_metrics": seam_metrics,
        "atlas_01_slices": len(a1_meta["slices"]),
        "atlas_02_slices": len(a2_meta["slices"]),
        "atlas_gutter_bleed_pixels": 0,
        "gameplay_composites_verified": 8,
        "unity_version_verified": unity_rep["unity_version"],
    }


def run_12_adversarial_tests(pkg_dir: Path, repo_root: Path) -> List[Dict[str, Any]]:
    results: List[Dict[str, Any]] = []
    tmp_dir = Path(tempfile.mkdtemp(prefix="art005_adv_"))
    try:
        # 1. Missing castle sprite
        def case_01() -> Dict[str, Any]:
            return {"file_map": {"art/production/ART-005/castle/frames/env_castle_band3_lv25_39.png": str(tmp_dir / "nonexistent.png")}}

        # 2. Incorrect wall damage mapping
        def case_02() -> Dict[str, Any]:
            m = json.loads((pkg_dir / "environment_state_mapping.json").read_text(encoding="utf-8"))
            for st in m["wall_damage_states"]:
                if st["state_key"] == "damaged":
                    st["min_ratio_inclusive"] = 0.35  # leaves 0.25..0.349 unmapped!
            return {"mapping": m}

        # 3. Townhall level-band mismatch
        def case_03() -> Dict[str, Any]:
            m = json.loads((pkg_dir / "environment_state_mapping.json").read_text(encoding="utf-8"))
            m["townhall_level_bands"][1]["max_level"] = 22  # leaves Lv 23-24 unmapped!
            return {"mapping": m}

        # 4. Corrupted terrain tile seam
        def case_04() -> Dict[str, Any]:
            p = tmp_dir / "bad_seam_tile.png"
            arr = np.array(Image.open(pkg_dir / "terrain/tiles/env_tile_01_meadow_main.png").convert("RGBA"))
            arr[:, -1, :3] = (235, 245, 190)  # corrupt right border seam
            Image.fromarray(arr, "RGBA").save(p)
            return {"file_map": {"art/production/ART-005/terrain/tiles/env_tile_01_meadow_main.png": str(p)}}

        # 5. Non-transparent building background
        def case_05() -> Dict[str, Any]:
            p = tmp_dir / "opaque_house.png"
            arr = np.array(Image.open(pkg_dir / "buildings/houses/env_house_01_artisan_gable.png").convert("RGBA"))
            arr[:, :, 3] = 255  # opaque background!
            Image.fromarray(arr, "RGBA").save(p)
            return {"file_map": {"art/production/ART-005/buildings/houses/env_house_01_artisan_gable.png": str(p)}}

        # 6. Wrong pivot
        def case_06() -> Dict[str, Any]:
            lay = json.loads((pkg_dir / "environment_layout.json").read_text(encoding="utf-8"))
            lay["canonical_sprites"][3]["normalized_pivot"] = [0.50, 0.50]
            return {"layout": lay}

        # 7. Atlas UV corruption
        def case_07() -> Dict[str, Any]:
            a1 = json.loads((pkg_dir / "atlas/environment_atlas_01_fortress.json").read_text(encoding="utf-8"))
            rx, ry, rw, rh = a1["slices"][0]["rect_top_left_px"]
            a1["slices"][0]["rect_top_left_px"] = [rx + 12, ry, rw, rh]
            a1["slices"][0]["rect_unity_bottom_left_px"] = [rx + 12, 2048 - (ry + rh), rw, rh]
            return {"atlas1_meta": a1}

        # 8. Sprite clipped at border
        def case_08() -> Dict[str, Any]:
            p = tmp_dir / "clipped_watchtower.png"
            arr = np.array(Image.open(pkg_dir / "buildings/watchtower/env_bldg_watchtower.png").convert("RGBA"))
            arr[0, 120:136] = (40, 60, 90, 255)  # clipped at y=0 top edge!
            Image.fromarray(arr, "RGBA").save(p)
            return {"file_map": {"art/production/ART-005/buildings/watchtower/env_bldg_watchtower.png": str(p)}}

        # 9. Incorrect SHA-256 hash
        def case_09() -> Dict[str, Any]:
            man = json.loads((pkg_dir / "asset_manifest.json").read_text(encoding="utf-8"))
            orig = man["files"][0]["sha256"]
            man["files"][0]["sha256"] = ("0" if orig[0] != "0" else "f") + orig[1:]
            return {"manifest": man}

        # 10. Sorting layer that hides the Boss
        def case_10() -> Dict[str, Any]:
            tel = json.loads((pkg_dir / "previews/gameplay_composites/composite_scene_telemetry.json").read_text(encoding="utf-8"))
            for u in tel["07_boss_approach_center_lane"]["units"]:
                if u["class"] == "boss":
                    u["sorting_layer"] = "Ground"  # hides Boss under LanesAndDecals & BastionBack!
            return {"telemetry": tel}

        # 11. Environment element entering restricted gameplay safe zone (x > 82%)
        def case_11() -> Dict[str, Any]:
            lay = json.loads((pkg_dir / "environment_layout.json").read_text(encoding="utf-8"))
            lay["courtyard_placements"][1]["screen_px"] = [925.0, 804.0]  # x > 885.6px (82% limit)!
            return {"layout": lay}

        # 12. Broken transition between road and meadow
        def case_12() -> Dict[str, Any]:
            p = tmp_dir / "broken_road_border.png"
            arr = np.array(Image.open(pkg_dir / "terrain/tiles/env_tile_04_ochre_road_border.png").convert("RGBA"))
            arr[:, 0, :3] = (215, 165, 95)  # left edge no longer matches meadow_main right edge!
            Image.fromarray(arr, "RGBA").save(p)
            return {"file_map": {"art/production/ART-005/terrain/tiles/env_tile_04_ochre_road_border.png": str(p)}}

        cases: List[Tuple[str, Callable[[], Dict[str, Any]]]] = [
            ("01_missing_castle_sprite", case_01),
            ("02_incorrect_wall_damage_mapping", case_02),
            ("03_townhall_level_band_mismatch", case_03),
            ("04_corrupted_terrain_tile_seam", case_04),
            ("05_non_transparent_building_background", case_05),
            ("06_wrong_pivot", case_06),
            ("07_atlas_uv_corruption", case_07),
            ("08_sprite_clipped_at_border", case_08),
            ("09_incorrect_sha256_hash", case_09),
            ("10_sorting_layer_hides_boss", case_10),
            ("11_safe_zone_violation_x_gt_82pct", case_11),
            ("12_broken_road_to_meadow_transition", case_12),
        ]

        for name, fn in cases:
            ov = fn()
            chk_man = (name == "09_incorrect_sha256_hash")
            caught = False
            reason = ""
            try:
                validate_bundle(pkg_dir, repo_root, check_manifest=chk_man, overrides=ov)
            except ValidationFailure as exc:
                caught = True
                reason = str(exc)
            require(caught, f"Adversarial test {name} FAILED TO REJECT corrupted input!")
            results.append({"test_id": name, "rejected_as_expected": True, "rejection_reason": reason})
    finally:
        shutil.rmtree(tmp_dir, ignore_errors=True)

    return results


def main() -> None:
    ap = argparse.ArgumentParser()
    ap.add_argument("--write-manifest", action="store_true", help="Write/refresh asset_manifest.json and QA JSON metrics")
    args = ap.parse_args()

    qa_dir = ART005_DIR / "qa"
    qa_dir.mkdir(parents=True, exist_ok=True)

    if args.write_manifest:
        # First pass without manifest check, write qa_metrics + adversarial results, then freeze manifest!
        metrics = validate_bundle(ART005_DIR, REPO_ROOT, check_manifest=False)
        (qa_dir / "qa_metrics.json").write_text(json.dumps(metrics, indent=2), encoding="utf-8", newline="\n")

        # Write preliminary manifest so adversarial test #9 has real SHA-256 manifest to mutate
        man_pre = build_asset_manifest(ART005_DIR, REPO_ROOT)
        (ART005_DIR / "asset_manifest.json").write_text(json.dumps(man_pre, indent=2), encoding="utf-8", newline="\n")

        adv = run_12_adversarial_tests(ART005_DIR, REPO_ROOT)
        (qa_dir / "adversarial_validator_results.json").write_text(
            json.dumps({"package_id": "ART-005", "total_negative_tests": len(adv), "all_rejected": True, "cases": adv}, indent=2),
            encoding="utf-8",
            newline="\n",
        )

        # Refresh final manifest including qa_metrics.json & adversarial_validator_results.json
        man_final = build_asset_manifest(ART005_DIR, REPO_ROOT)
        (ART005_DIR / "asset_manifest.json").write_text(json.dumps(man_final, indent=2), encoding="utf-8", newline="\n")

    # Full end-to-end strict check including SHA-256 manifest + all 12 negative tests!
    final_metrics = validate_bundle(ART005_DIR, REPO_ROOT, check_manifest=True)
    adv_check = run_12_adversarial_tests(ART005_DIR, REPO_ROOT)
    print(
        f"[ART-005 VALIDATOR PASS] {final_metrics['canonical_sprites_verified']} canonical + "
        f"{final_metrics['companion_sprites_verified']} companion sprites, "
        f"{final_metrics['atlas_01_slices']}+{final_metrics['atlas_02_slices']} atlas slices (0 bleed), "
        f"8 gameplay composites, SHA-256 manifest, and {len(adv_check)}/12 adversarial negative tests verified!"
    )


if __name__ == "__main__":
    main()
