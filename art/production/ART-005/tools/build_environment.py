#!/usr/bin/env python3
"""
ART-005 — Master Environment Asset, Layout, State-Mapping & Palette Builder
Generates all 26 canonical Bastion Dawn v1 environment sprites, 10 split wall depth slices,
4 terrain transition strips/masks, 4 inspection sheets, and 3 deterministic JSON contracts.
"""

from __future__ import annotations

import json
import sys
from pathlib import Path
from typing import Any, Dict, List, Tuple

import numpy as np
from PIL import Image, ImageDraw, ImageFont

TOOLS_DIR = Path(__file__).resolve().parent
ART005_DIR = TOOLS_DIR.parent
REPO_ROOT = ART005_DIR.parents[2]
sys.path.insert(0, str(TOOLS_DIR))

from art005_env_lib import (
    INK_RGB,
    build_terrain_tiles_and_transitions,
    render_cottage_variant,
    render_courtyard_prop,
    render_repair_scaffold,
    render_townhall_band,
    render_wall_state_and_layers,
    render_watchtower,
)

PPU = 192.0
REF_W = 1080.0
REF_H = 1920.0


def screen_to_world(x_px: float, y_px: float) -> List[float]:
    """Converts 1080x1920 reference screen px (top-left origin) to Unity world units (Camera.orthographicSize=5, PPU=192)."""
    return [round((x_px - 540.0) / PPU, 5), round((960.0 - y_px) / PPU, 5)]


def load_font(size: int, bold: bool = False) -> ImageFont.FreeTypeFont | ImageFont.ImageFont:
    candidates = (
        ["C:/Windows/Fonts/segoeuib.ttf", "C:/Windows/Fonts/arialbd.ttf"]
        if bold
        else ["C:/Windows/Fonts/segoeui.ttf", "C:/Windows/Fonts/arial.ttf"]
    )
    for p in candidates:
        if Path(p).exists():
            return ImageFont.truetype(p, size)
    return ImageFont.load_default()


def bbox_opaque(im: Image.Image) -> List[int]:
    arr = np.array(im)
    ys, xs = np.where(arr[:, :, 3] > 0)
    if len(xs) == 0:
        return [0, 0, 0, 0]
    return [int(xs.min()), int(ys.min()), int(xs.max()), int(ys.max())]


def draw_checker_card(canvas: Image.Image, box: Tuple[int, int, int, int], step: int = 16) -> None:
    x0, y0, x1, y1 = box
    w, h = x1 - x0, y1 - y0
    yy, xx = np.mgrid[0:h, 0:w]
    even = ((xx // step) + (yy // step)) % 2 == 0
    chk = np.empty((h, w, 3), dtype=np.uint8)
    chk[even] = (46, 56, 72)
    chk[~even] = (34, 42, 56)
    canvas.paste(Image.fromarray(chk, "RGB"), (x0, y0))


def main() -> None:
    castle_dir = ART005_DIR / "castle" / "frames"
    wall_states_dir = ART005_DIR / "wall" / "states"
    wall_layers_dir = ART005_DIR / "wall" / "layers"
    wall_scaf_dir = ART005_DIR / "wall" / "scaffold"
    bldg_wt_dir = ART005_DIR / "buildings" / "watchtower"
    bldg_hs_dir = ART005_DIR / "buildings" / "houses"
    bldg_pr_dir = ART005_DIR / "buildings" / "props"
    tiles_dir = ART005_DIR / "terrain" / "tiles"
    trans_dir = ART005_DIR / "terrain" / "transitions"

    for d in (
        castle_dir,
        wall_states_dir,
        wall_layers_dir,
        wall_scaf_dir,
        bldg_wt_dir,
        bldg_hs_dir,
        bldg_pr_dir,
        tiles_dir,
        trans_dir,
    ):
        d.mkdir(parents=True, exist_ok=True)

    canonical_sprites: List[Dict[str, Any]] = []
    companion_assets: List[Dict[str, Any]] = []

    # ------------------------------------------------------------------------
    # 1. PHASE 2 — FOUR COSMETIC TOWNHALL SPRITES (512x512 RGBA)
    # ------------------------------------------------------------------------
    castle_specs = [
        (1, "env_castle_band1_lv01_09", "Timber Village Hall", 1, 9),
        (2, "env_castle_band2_lv10_24", "Reinforced Stone Keep", 10, 24),
        (3, "env_castle_band3_lv25_39", "Bastion Citadel", 25, 39),
        (4, "env_castle_band4_lv40_50", "Sovereign Dawn Keep", 40, 50),
    ]
    castle_imgs: List[Tuple[str, str, Image.Image]] = []
    for band, stem, title, lv_min, lv_max in castle_specs:
        im = render_townhall_band(band)
        out_p = castle_dir / f"{stem}.png"
        im.save(out_p)
        castle_imgs.append((stem, f"Band {band} (Lv {lv_min}-{lv_max}): {title}", im))
        canonical_sprites.append(
            {
                "id": stem,
                "category": "castle_townhall",
                "band": band,
                "level_range": [lv_min, lv_max],
                "title": title,
                "path": out_p.relative_to(REPO_ROOT).as_posix(),
                "dimensions": [512, 512],
                "normalized_pivot": [0.50, 0.10],
                "unity_sorting_layer": "BastionBack",
                "unity_visual_scale": 0.50,
                "bbox_px": bbox_opaque(im),
                "authoritative_townhall_hp_invariant": 500,
            }
        )

    # Castle Progression Sheet (2160x660)
    f_h = load_font(22, bold=True)
    f_b = load_font(14, bold=True)
    f_s = load_font(12, bold=False)
    c_sheet = Image.new("RGB", (2160, 680), (16, 23, 38))
    cd = ImageDraw.Draw(c_sheet)
    cd.text((28, 16), "ART-005 CASTLE / TOWNHALL COSMETIC PROGRESSION SHEET (4 CANONICAL 512x512 RGBA SPRITES)", font=f_h, fill=(254, 240, 138))
    cd.text((28, 46), "Identical foundation footprint & pivot (0.50, 0.10) across Bands 1-4 -> 0.0 px swap jump | Authoritative townhall_hp = 500 fixed.", font=f_s, fill=(203, 213, 225))
    for idx, (stem, lbl, im) in enumerate(castle_imgs):
        ox = 28 + idx * 530
        oy = 84
        draw_checker_card(c_sheet, (ox, oy, ox + 512, oy + 512))
        c_sheet.paste(im, (ox, oy), im)
        # Pivot crosshair
        px, py = ox + 256, oy + int(round(512 * 0.90))
        cd.line([(px - 18, py), (px + 18, py)], fill=(74, 222, 128), width=2)
        cd.line([(px, py - 12), (px, py + 12)], fill=(74, 222, 128), width=2)
        cd.rectangle([ox, oy, ox + 512, oy + 512], outline=(100, 116, 139), width=2)
        cd.text((ox + 6, oy + 520), lbl, font=f_b, fill=(248, 250, 252))
        cd.text((ox + 6, oy + 542), f"{stem}.png  |  bbox={bbox_opaque(im)}", font=f_s, fill=(148, 163, 184))
    c_sheet.save(ART005_DIR / "castle" / "castle_progression_sheet.png")

    # ------------------------------------------------------------------------
    # 2. PHASE 3 — CURTAIN WALL & GATE (5 STATES + 10 SPLIT SLICES + 1 SCAFFOLD)
    # ------------------------------------------------------------------------
    wall_specs = [
        (1, "env_wall_state1_healthy", "State 1 — Healthy (60-100% HP)", 0.60, 1.00, False),
        (2, "env_wall_state2_damaged", "State 2 — Damaged (25-59% HP)", 0.25, 0.599999, False),
        (3, "env_wall_state3_critical", "State 3 — Critical (>0% to <25% HP)", 0.000001, 0.249999, False),
        (4, "env_wall_state4_breached", "State 4 — Breached (0 HP)", 0.0, 0.0, False),
        (5, "env_wall_state5_recovery", "State 5 — Post-Loss Recovery (250 HP Floor)", None, None, True),
    ]
    wall_imgs: List[Tuple[str, str, Image.Image, Image.Image, Image.Image]] = []
    for st_idx, stem, label, r_min, r_max, is_rec in wall_specs:
        full_im, back_im, front_im = render_wall_state_and_layers(st_idx)
        p_full = wall_states_dir / f"{stem}.png"
        p_back = wall_layers_dir / f"{stem}_back.png"
        p_front = wall_layers_dir / f"{stem}_front.png"
        full_im.save(p_full)
        back_im.save(p_back)
        front_im.save(p_front)
        wall_imgs.append((stem, label, full_im, back_im, front_im))
        canonical_sprites.append(
            {
                "id": stem,
                "category": "defensive_wall_state",
                "state_index": st_idx,
                "title": label,
                "hp_ratio_min": r_min,
                "hp_ratio_max": r_max,
                "is_post_loss_recovery_state": is_rec,
                "path": p_full.relative_to(REPO_ROOT).as_posix(),
                "back_layer_slice": p_back.relative_to(REPO_ROOT).as_posix(),
                "front_layer_slice": p_front.relative_to(REPO_ROOT).as_posix(),
                "dimensions": [512, 512],
                "normalized_pivot": [0.50, 0.14],
                "unity_sorting_layers": ["BastionBack", "BastionFront"],
                "unity_visual_scale": 1.00,
                "bbox_px": bbox_opaque(full_im),
            }
        )
        companion_assets.append(
            {
                "id": f"{stem}_back",
                "category": "wall_depth_slice_back",
                "path": p_back.relative_to(REPO_ROOT).as_posix(),
                "dimensions": [512, 320],
                "crop_in_512": [0, 0, 512, 320],
                "normalized_pivot": [0.50, -0.376],
                "unity_sorting_layer": "BastionBack",
                "bbox_px": bbox_opaque(back_im),
            }
        )
        companion_assets.append(
            {
                "id": f"{stem}_front",
                "category": "wall_depth_slice_front",
                "path": p_front.relative_to(REPO_ROOT).as_posix(),
                "dimensions": [512, 256],
                "crop_in_512": [0, 208, 512, 464],
                "normalized_pivot": [0.50, 0.0925],
                "unity_sorting_layer": "BastionFront",
                "bbox_px": bbox_opaque(front_im),
            }
        )

    scaf_im = render_repair_scaffold()
    p_scaf = wall_scaf_dir / "env_wall_repair_scaffold.png"
    scaf_im.save(p_scaf)
    canonical_sprites.append(
        {
            "id": "env_wall_repair_scaffold",
            "category": "wall_repair_scaffold",
            "title": "Modular Builder Timber Scaffold",
            "path": p_scaf.relative_to(REPO_ROOT).as_posix(),
            "dimensions": [192, 192],
            "normalized_pivot": [0.50, 0.10],
            "unity_sorting_layer": "BastionFront",
            "unity_visual_scale": 0.45,
            "bbox_px": bbox_opaque(scaf_im),
        }
    )

    # Wall States & Scaffold Inspection Sheet (2280x1010)
    w_sheet = Image.new("RGB", (2280, 1010), (16, 23, 38))
    wd = ImageDraw.Draw(w_sheet)
    wd.text((28, 16), "ART-005 DEFENSIVE WALL, SOUTH GATE (5 STATES + SPLIT DEPTH SLICES) & BUILDER SCAFFOLD", font=f_h, fill=(254, 240, 138))
    wd.text((28, 46), "Exact 477px outer ring width (44.17% of 1080p) | Transparent exterior & inner courtyard | Split BastionBack + BastionFront slices.", font=f_s, fill=(203, 213, 225))
    for idx, (stem, lbl, full_im, back_im, front_im) in enumerate(wall_imgs):
        ox = 28 + idx * 444
        oy = 84
        draw_checker_card(w_sheet, (ox, oy, ox + 424, oy + 424))
        thumb = full_im.resize((424, 424), Image.Resampling.NEAREST)
        w_sheet.paste(thumb, (ox, oy), thumb)
        wd.rectangle([ox, oy, ox + 424, oy + 424], outline=(100, 116, 139), width=2)
        wd.text((ox + 6, oy + 432), lbl, font=f_b, fill=(248, 250, 252))
        wd.text((ox + 6, oy + 454), f"bbox={bbox_opaque(full_im)}", font=f_s, fill=(148, 163, 184))

        # Show split back & front slices cleanly below text!
        by = 562
        draw_checker_card(w_sheet, (ox, by, ox + 204, by + 128), step=10)
        b_th = back_im.resize((204, 128), Image.Resampling.NEAREST)
        w_sheet.paste(b_th, (ox, by), b_th)
        wd.rectangle([ox, by, ox + 204, by + 128], outline=(74, 222, 128), width=1)
        wd.text((ox + 4, by + 132), "BastionBack slice", font=f_s, fill=(134, 239, 172))

        fx = ox + 220
        draw_checker_card(w_sheet, (fx, by, fx + 204, by + 128), step=10)
        f_th = front_im.resize((204, 102), Image.Resampling.NEAREST)
        w_sheet.paste(f_th, (fx, by + 13), f_th)
        wd.rectangle([fx, by, fx + 204, by + 128], outline=(251, 146, 60), width=1)
        wd.text((fx + 4, by + 132), "BastionFront slice", font=f_s, fill=(253, 186, 116))

    # Bottom row: Builder Scaffold + ART-003 Builder Repair alignment preview!
    sy0 = 736
    draw_checker_card(w_sheet, (28, sy0, 28 + 240, sy0 + 240))
    sc_up = scaf_im.resize((240, 240), Image.Resampling.NEAREST)
    w_sheet.paste(sc_up, (28, sy0), sc_up)
    wd.rectangle([28, sy0, 268, sy0 + 240], outline=(250, 204, 21), width=2)
    wd.text((290, sy0 + 18), "6TH WALL ASSET: env_wall_repair_scaffold.png (192x192 RGBA)", font=f_b, fill=(254, 240, 138))
    wd.text((290, sy0 + 48), "- Reusable modular pegged oak scaffold overlay (NW / NE / SW / SE wall flank anchors; mirrorable via flipX).", font=f_s, fill=(226, 232, 240))
    wd.text((290, sy0 + 74), "- Independent of wall state texture; pairs cleanly with Damaged, Critical, and Post-Loss Recovery states.", font=f_s, fill=(226, 232, 240))
    wd.text((290, sy0 + 100), "- Calibrated to ART-003 Builder.Repair mallet strike height & vfx_builder_repair_spark impact anchor.", font=f_s, fill=(134, 239, 172))
    w_sheet.save(ART005_DIR / "wall" / "wall_states_sheet.png")

    # ------------------------------------------------------------------------
    # 3. PHASE 4 — WATCHTOWER (1), COURTYARD HOUSES (6) & PROPS (3) = 10 SPRITES
    # ------------------------------------------------------------------------
    bldg_cards: List[Tuple[str, str, Image.Image]] = []

    wt_im = render_watchtower()
    p_wt = bldg_wt_dir / "env_bldg_watchtower.png"
    wt_im.save(p_wt)
    bldg_cards.append(("env_bldg_watchtower", "Watchtower (256x256)", wt_im))
    canonical_sprites.append(
        {
            "id": "env_bldg_watchtower",
            "category": "village_watchtower",
            "title": "Defensive Stone & Timber Watchtower (Cosmetic Landmark Only)",
            "path": p_wt.relative_to(REPO_ROOT).as_posix(),
            "dimensions": [256, 256],
            "normalized_pivot": [0.50, 0.10],
            "unity_sorting_layer": "BastionBack",
            "unity_visual_scale": 0.625,
            "bbox_px": bbox_opaque(wt_im),
        }
    )

    house_specs = [
        (1, "env_house_01_artisan_gable", "House 1 — Artisan Gable & Awning"),
        (2, "env_house_02_granary_round", "House 2 — Stone Granary Roundhouse"),
        (3, "env_house_03_mason_lodge", "House 3 — Limestone Mason Lodge"),
        (4, "env_house_04_bakery_chimney", "House 4 — Bakery Chimney Cottage"),
        (5, "env_house_05_smithy_shed", "House 5 — Open-Front Smithy Shed"),
        (6, "env_house_06_guard_quarters", "House 6 — Cross-Gable Guard Quarters"),
    ]
    for h_idx, h_stem, h_title in house_specs:
        him = render_cottage_variant(h_idx)
        ph = bldg_hs_dir / f"{h_stem}.png"
        him.save(ph)
        bldg_cards.append((h_stem, h_title, him))
        canonical_sprites.append(
            {
                "id": h_stem,
                "category": "village_cottage",
                "variant_index": h_idx,
                "title": h_title,
                "path": ph.relative_to(REPO_ROOT).as_posix(),
                "dimensions": [256, 256],
                "normalized_pivot": [0.50, 0.12],
                "unity_sorting_layer": "BastionBack",
                "unity_visual_scale": 0.34,
                "bbox_px": bbox_opaque(him),
            }
        )

    prop_specs = [
        ("supply_cart", "env_prop_supply_cart", "Prop 1 — Oak Supply Cart"),
        ("stone_crates", "env_prop_stone_crates", "Prop 2 — Cut-Stone Crate Stack"),
        ("sun_banner", "env_prop_sun_banner", "Prop 3 — Golden Sun Standard"),
    ]
    for p_kind, p_stem, p_title in prop_specs:
        pim = render_courtyard_prop(p_kind)
        pp = bldg_pr_dir / f"{p_stem}.png"
        pim.save(pp)
        bldg_cards.append((p_stem, p_title, pim))
        canonical_sprites.append(
            {
                "id": p_stem,
                "category": "courtyard_prop",
                "title": p_title,
                "path": pp.relative_to(REPO_ROOT).as_posix(),
                "dimensions": [128, 128],
                "normalized_pivot": [0.50, 0.12],
                "unity_sorting_layer": "BastionBack",
                "unity_visual_scale": 0.34,
                "bbox_px": bbox_opaque(pim),
            }
        )

    # Buildings & Props Sheet (1960x760)
    b_sheet = Image.new("RGB", (1960, 760), (16, 23, 38))
    bd = ImageDraw.Draw(b_sheet)
    bd.text((28, 16), "ART-005 VILLAGE BUILDINGS & COURTYARD PROPS (10 SEPARATE TRANSPARENT RGBA SPRITES)", font=f_h, fill=(254, 240, 138))
    bd.text((28, 46), "1 Watchtower + 6 Architecturally Distinct Cottages + 3 Low-Profile Courtyard Props — Binary Alpha (0 Halo px), 45-deg Sunlight.", font=f_s, fill=(203, 213, 225))
    for idx, (stem, title, im) in enumerate(bldg_cards):
        r, c = divmod(idx, 5)
        ox = 28 + c * 384
        oy = 88 + r * 324
        draw_checker_card(b_sheet, (ox, oy, ox + 360, oy + 256))
        disp = im.resize((240, 240), Image.Resampling.NEAREST) if im.width == 256 else im.resize((192, 192), Image.Resampling.NEAREST)
        b_sheet.paste(disp, (ox + (360 - disp.width) // 2, oy + (256 - disp.height) // 2), disp)
        bd.rectangle([ox, oy, ox + 360, oy + 256], outline=(100, 116, 139), width=2)
        bd.text((ox + 6, oy + 264), title, font=f_b, fill=(248, 250, 252))
        bd.text((ox + 6, oy + 286), f"{stem}.png ({im.width}x{im.height})", font=f_s, fill=(148, 163, 184))
    b_sheet.save(ART005_DIR / "buildings" / "buildings_and_props_sheet.png")

    # ------------------------------------------------------------------------
    # 4. PHASE 5 — SEAMLESS TERRAIN TILES (6) & TRANSITION ASSETS (4)
    # ------------------------------------------------------------------------
    tiles_map, trans_map = build_terrain_tiles_and_transitions()
    tile_titles = {
        "env_tile_01_meadow_main": "Tile 1 — Sunlit Meadow Main (Toroidal X/Y)",
        "env_tile_02_meadow_variation": "Tile 2 — Lush Meadow Turf Variation (Toroidal X/Y)",
        "env_tile_03_ochre_road_body": "Tile 3 — Ochre Road Body (Toroidal X/Y)",
        "env_tile_04_ochre_road_border": "Tile 4 — Meadow-to-Ochre Verge Transition (Seamless Y)",
        "env_tile_05_limestone_causeway": "Tile 5 — Pale Limestone Boss Causeway (Toroidal X/Y)",
        "env_tile_06_courtyard_plaza": "Tile 6 — Courtyard Packed-Earth Plaza (Toroidal X/Y)",
    }
    for t_stem, tim in tiles_map.items():
        pt = tiles_dir / f"{t_stem}.png"
        tim.save(pt)
        canonical_sprites.append(
            {
                "id": t_stem,
                "category": "terrain_tile",
                "title": tile_titles[t_stem],
                "path": pt.relative_to(REPO_ROOT).as_posix(),
                "dimensions": [256, 256],
                "normalized_pivot": [0.50, 0.50],
                "seamless_axes": ["Y"] if "border" in t_stem else ["X", "Y"],
                "unity_sorting_layer": "Ground" if "meadow" in t_stem else ("BastionBack" if "courtyard" in t_stem else "LanesAndDecals"),
                "unity_visual_scale": 1.00,
                "bbox_px": [0, 0, 255, 255],
            }
        )

    for tr_stem, trim in trans_map.items():
        ptr = trans_dir / f"{tr_stem}.png"
        trim.save(ptr)
        companion_assets.append(
            {
                "id": tr_stem,
                "category": "terrain_transition",
                "path": ptr.relative_to(REPO_ROOT).as_posix(),
                "dimensions": [trim.width, trim.height],
                "normalized_pivot": [0.50, 0.50],
                "unity_sorting_layer": "LanesAndDecals",
                "bbox_px": bbox_opaque(trim),
            }
        )

    # Terrain Tilesheet (2060x820)
    t_sheet = Image.new("RGB", (2060, 820), (16, 23, 38))
    td = ImageDraw.Draw(t_sheet)
    td.text((28, 16), "ART-005 SEAMLESS H.264-SAFE TERRAIN TILES (6 x 256x256) & MATERIAL TRANSITIONS (4)", font=f_h, fill=(254, 240, 138))
    td.text((28, 46), "Zero #11151C black ink on ground | HSV S <= 48% | Toroidal Fourier continuity + explicit verge, stone curb & bastion apron.", font=f_s, fill=(203, 213, 225))
    tile_short_titles = {
        "env_tile_01_meadow_main": "Tile 1 — Sunlit Meadow Main",
        "env_tile_02_meadow_variation": "Tile 2 — Meadow Turf Variation",
        "env_tile_03_ochre_road_body": "Tile 3 — Ochre Road Body",
        "env_tile_04_ochre_road_border": "Tile 4 — Meadow-Ochre Verge",
        "env_tile_05_limestone_causeway": "Tile 5 — Limestone Boss Causeway",
        "env_tile_06_courtyard_plaza": "Tile 6 — Courtyard Plaza",
    }
    for idx, (t_stem, tim) in enumerate(tiles_map.items()):
        ox = 28 + idx * 334
        oy = 88
        t_sheet.paste(tim.convert("RGB"), (ox, oy))
        td.rectangle([ox, oy, ox + 256, oy + 256], outline=(148, 163, 184), width=2)
        td.text((ox, oy + 264), tile_short_titles[t_stem], font=f_b, fill=(248, 250, 252))
        td.text((ox, oy + 286), f"{t_stem}.png (256x256)", font=f_s, fill=(148, 163, 184))

    for idx, (tr_stem, trim) in enumerate(trans_map.items()):
        ox = 28 + idx * 500
        oy = 420
        draw_checker_card(t_sheet, (ox, oy, ox + 460, oy + 300))
        t_sheet.paste(trim, (ox + (460 - trim.width) // 2, oy + (300 - trim.height) // 2), trim)
        td.rectangle([ox, oy, ox + 460, oy + 300], outline=(100, 116, 139), width=2)
        td.text((ox + 8, oy + 310), f"Transition: {tr_stem}.png ({trim.width}x{trim.height})", font=f_b, fill=(134, 239, 172))
    t_sheet.save(ART005_DIR / "terrain" / "terrain_tilesheet.png")

    assert len(canonical_sprites) == 26, f"Expected 26 canonical sprites, got {len(canonical_sprites)}"

    # ------------------------------------------------------------------------
    # 5. DETERMINISTIC RUNTIME CONTRACTS (JSON)
    # ------------------------------------------------------------------------

    # A) environment_state_mapping.json
    state_mapping = {
        "package_id": "ART-005",
        "rules_version_compatibility": "ttg-rules-v1",
        "gameplay_invariants": {
            "townhall_authoritative_hp": 500,
            "townhall_bands_are_cosmetic_only": True,
            "wall_tier_max_hp_formula": "1000 + 200 * (wall_tier - 1)",
            "post_loss_recovery_floor_hp": 250,
            "boss_wave_modulo": 5,
        },
        "townhall_level_bands": [
            {"band": 1, "min_level": 1, "max_level": 9, "sprite_id": "env_castle_band1_lv01_09", "label": "Timber Village Hall"},
            {"band": 2, "min_level": 10, "max_level": 24, "sprite_id": "env_castle_band2_lv10_24", "label": "Reinforced Stone Keep"},
            {"band": 3, "min_level": 25, "max_level": 39, "sprite_id": "env_castle_band3_lv25_39", "label": "Bastion Citadel"},
            {"band": 4, "min_level": 40, "max_level": 50, "sprite_id": "env_castle_band4_lv40_50", "label": "Sovereign Dawn Keep"},
        ],
        "wall_damage_states": [
            {
                "state_index": 5,
                "state_key": "post_loss_recovery",
                "priority_order": 1,
                "condition": "is_post_loss_recovery == true && wall_hp > 0",
                "sprite_id": "env_wall_state5_recovery",
                "back_slice_id": "env_wall_state5_recovery_back",
                "front_slice_id": "env_wall_state5_recovery_front",
            },
            {
                "state_index": 4,
                "state_key": "breached",
                "priority_order": 2,
                "condition": "wall_hp <= 0",
                "min_ratio_inclusive": 0.0,
                "max_ratio_inclusive": 0.0,
                "sprite_id": "env_wall_state4_breached",
                "back_slice_id": "env_wall_state4_breached_back",
                "front_slice_id": "env_wall_state4_breached_front",
            },
            {
                "state_index": 3,
                "state_key": "critical",
                "priority_order": 3,
                "condition": "wall_hp > 0 && (wall_hp / wall_max) < 0.25",
                "min_ratio_exclusive": 0.0,
                "max_ratio_exclusive": 0.25,
                "sprite_id": "env_wall_state3_critical",
                "back_slice_id": "env_wall_state3_critical_back",
                "front_slice_id": "env_wall_state3_critical_front",
            },
            {
                "state_index": 2,
                "state_key": "damaged",
                "priority_order": 4,
                "condition": "(wall_hp / wall_max) >= 0.25 && (wall_hp / wall_max) < 0.60",
                "min_ratio_inclusive": 0.25,
                "max_ratio_exclusive": 0.60,
                "sprite_id": "env_wall_state2_damaged",
                "back_slice_id": "env_wall_state2_damaged_back",
                "front_slice_id": "env_wall_state2_damaged_front",
            },
            {
                "state_index": 1,
                "state_key": "healthy",
                "priority_order": 5,
                "condition": "(wall_hp / wall_max) >= 0.60",
                "min_ratio_inclusive": 0.60,
                "max_ratio_inclusive": 1.00,
                "sprite_id": "env_wall_state1_healthy",
                "back_slice_id": "env_wall_state1_healthy_back",
                "front_slice_id": "env_wall_state1_healthy_front",
            },
        ],
        "scaffold_visibility_rule": {
            "sprite_id": "env_wall_repair_scaffold",
            "active_when": "builders_count > 0 && (phase == 'PREP' || wall_hp < wall_max || is_post_loss_recovery)",
            "paired_character": "chr_role_builder (Repair clip @ 12 FPS)",
            "paired_vfx": "vfx_builder_repair_spark (ART-003)",
        },
    }
    (ART005_DIR / "environment_state_mapping.json").write_text(json.dumps(state_mapping, indent=2), encoding="utf-8")

    # B) environment_layout.json — Exact world & reference screen coordinates
    layout_doc = {
        "package_id": "ART-005",
        "reference_canvas_px": [1080, 1920],
        "unity_camera": {
            "projection": "orthographic",
            "orthographic_size": 5.0,
            "world_position": [0.0, 0.0, -10.0],
            "pixels_per_unit": 192,
            "screen_px_per_world_unit_at_1080p": 192.0,
            "world_bounds": {"x_min": -2.8125, "x_max": 2.8125, "y_min": -5.0, "y_max": 5.0},
            "conversion_formula": "world_x = (screen_x - 540.0) / 192.0; world_y = (960.0 - screen_y) / 192.0",
        },
        "livestream_safe_zones": {
            "right_rail_x_norm_limit": 0.82,
            "right_rail_x_px_limit": 885.6,
            "right_rail_world_x_limit": round((885.6 - 540.0) / PPU, 5),
            "bottom_overlay_y_norm_limit": 0.88,
            "bottom_overlay_y_px_limit": 1689.6,
            "bottom_overlay_world_y_limit": round((960.0 - 1689.6) / PPU, 5),
        },
        "sorting_layers": [
            {"index": 0, "name": "Ground", "order_rule": "0..10 (meadow base & variation patches)"},
            {"index": 1, "name": "LanesAndDecals", "order_rule": "0..30 (West/Center/East lanes, curbs, aprons, dynamic Boss telegraph)"},
            {"index": 2, "name": "BastionBack", "order_rule": "-round(worldY * 100) (courtyard plaza, North wall slice, Watchtower, Townhall, 6 Cottages, 3 Props)"},
            {"index": 3, "name": "Units", "order_rule": "-round(pivotWorldY * 100) (all 7 character classes <= 150 pooled)"},
            {"index": 4, "name": "BastionFront", "order_rule": "fixed 0..20 (low South wall parapet & Gate arch slice, flank scaffolds)"},
            {"index": 5, "name": "VFX", "order_rule": "0..50 (slash arcs, shield blocks, hit puffs, repair sparks <= 20 bursts)"},
        ],
        "bastion_and_landmarks": {
            "bastion_center_screen_px": [454.0, 860.0],
            "bastion_center_world": screen_to_world(454.0, 860.0),
            "wall_pivot_screen_px": [454.0, 998.0],
            "wall_pivot_world": screen_to_world(454.0, 998.0),
            "bastion_outer_width_px": 477,
            "bastion_width_ratio_of_1080": round(477.0 / 1080.0, 4),
            "bastion_x_span_px": [215.5, 692.5],
            "south_gate_x_span_px": [414.0, 494.0],
            "south_gate_y_span_px": [955.0, 1042.0],
            "townhall_pivot_screen_px": [454.0, 796.0],
            "townhall_pivot_world": screen_to_world(454.0, 796.0),
            "watchtower_pivot_screen_px": [352.0, 756.0],
            "watchtower_pivot_world": screen_to_world(352.0, 756.0),
        },
        "lanes": [
            {
                "lane_index": 0,
                "lane_id": "west",
                "name": "West Lane",
                "material": "ochre_earth",
                "ordinary_hp_share_doc": "floor(base_hp * 34 / 100)",
                "width_px": 74,
                "centerline_screen_px": [[58.0, 0.0], [198.0, 340.0], [335.0, 690.0]],
                "centerline_world": [screen_to_world(58.0, 0.0), screen_to_world(198.0, 340.0), screen_to_world(335.0, 690.0)],
                "apron_screen_px": [335.0, 694.0],
            },
            {
                "lane_index": 1,
                "lane_id": "center",
                "name": "Center Boss Causeway",
                "material": "pale_limestone",
                "ordinary_hp_share_doc": "floor(base_hp * 32 / 100) + exclusive Boss pool ceil(base_hp / 5)",
                "width_px": 86,
                "x_span_px": [411.0, 497.0],
                "centerline_screen_px": [[454.0, 0.0], [454.0, 350.0], [454.0, 672.0]],
                "centerline_world": [screen_to_world(454.0, 0.0), screen_to_world(454.0, 350.0), screen_to_world(454.0, 672.0)],
                "boss_spawn_and_telegraph_screen_px": [454.0, 520.0],
                "boss_spawn_and_telegraph_world": screen_to_world(454.0, 520.0),
                "telegraph_baked_into_road": False,
            },
            {
                "lane_index": 2,
                "lane_id": "east",
                "name": "East Lane",
                "material": "ochre_earth",
                "ordinary_hp_share_doc": "base_hp - west - center (~34%)",
                "width_px": 74,
                "centerline_screen_px": [[836.0, 268.0], [705.0, 478.0], [575.0, 690.0]],
                "centerline_world": [screen_to_world(836.0, 268.0), screen_to_world(705.0, 478.0), screen_to_world(575.0, 690.0)],
                "max_right_edge_px": 873.0,
                "strictly_left_of_82pct_rail": True,
                "apron_screen_px": [575.0, 694.0],
            },
        ],
        "courtyard_placements": [
            {"id": "env_house_01_artisan_gable", "screen_px": [314.0, 804.0], "world": screen_to_world(314.0, 804.0), "scale": 0.34},
            {"id": "env_house_02_granary_round", "screen_px": [594.0, 804.0], "world": screen_to_world(594.0, 804.0), "scale": 0.34},
            {"id": "env_house_03_mason_lodge", "screen_px": [306.0, 860.0], "world": screen_to_world(306.0, 860.0), "scale": 0.34},
            {"id": "env_house_04_bakery_chimney", "screen_px": [602.0, 860.0], "world": screen_to_world(602.0, 860.0), "scale": 0.34},
            {"id": "env_house_05_smithy_shed", "screen_px": [364.0, 898.0], "world": screen_to_world(364.0, 898.0), "scale": 0.33},
            {"id": "env_house_06_guard_quarters", "screen_px": [544.0, 898.0], "world": screen_to_world(544.0, 898.0), "scale": 0.33},
            {"id": "env_prop_supply_cart", "screen_px": [406.0, 852.0], "world": screen_to_world(406.0, 852.0), "scale": 0.36},
            {"id": "env_prop_sun_banner", "screen_px": [454.0, 844.0], "world": screen_to_world(454.0, 844.0), "scale": 0.38},
            {"id": "env_prop_stone_crates", "screen_px": [502.0, 852.0], "world": screen_to_world(502.0, 852.0), "scale": 0.36},
        ],
        "scaffold_anchors": [
            {"slot": "SW", "screen_px": [294.0, 996.0], "world": screen_to_world(294.0, 996.0), "flip_x": False, "builder_offset_px": [-18.0, 8.0]},
            {"slot": "SE", "screen_px": [614.0, 996.0], "world": screen_to_world(614.0, 996.0), "flip_x": True, "builder_offset_px": [18.0, 8.0]},
            {"slot": "NW", "screen_px": [266.0, 782.0], "world": screen_to_world(266.0, 782.0), "flip_x": False, "builder_offset_px": [-16.0, 10.0]},
            {"slot": "NE", "screen_px": [642.0, 782.0], "world": screen_to_world(642.0, 782.0), "flip_x": True, "builder_offset_px": [16.0, 10.0]},
        ],
        "canonical_sprites": canonical_sprites,
        "companion_assets": companion_assets,
    }
    (ART005_DIR / "environment_layout.json").write_text(json.dumps(layout_doc, indent=2), encoding="utf-8")

    # C) lighting_palettes.json — Four deterministic environment-only lighting presentations
    lighting_doc = {
        "package_id": "ART-005",
        "scope": "ENVIRONMENT_LAYERS_ONLY (Ground, LanesAndDecals, BastionBack, BastionFront). Never applied to HUD Canvas; optional mild 25% tie-in on Units.",
        "gameplay_clock_independence": "Visual mood only; never alters simulation ticks, wave timers, or combat HP.",
        "formula": "out_rgb = clamp01(((in_rgb - luma) * saturation_scale + luma) * rgb_multiply + rgb_lift)",
        "palettes": [
            {
                "id": "dawn",
                "name": "Sunlit Dawn",
                "recommended_waves": "Waves 1-3 (Default)",
                "rgb_multiply": [1.02, 1.00, 0.96],
                "rgb_lift": [0.014, 0.008, 0.000],
                "saturation_scale": 1.00,
                "ambient_hex": "#FFF6E0",
            },
            {
                "id": "midday",
                "name": "Golden Midday",
                "recommended_waves": "Waves 4, 6-8",
                "rgb_multiply": [1.00, 1.02, 1.03],
                "rgb_lift": [0.000, 0.006, 0.012],
                "saturation_scale": 0.98,
                "ambient_hex": "#F8FAFC",
            },
            {
                "id": "dusk",
                "name": "Amber Dusk Siege",
                "recommended_waves": "Waves 9, 11-14",
                "rgb_multiply": [1.06, 0.92, 0.82],
                "rgb_lift": [0.022, 0.004, 0.010],
                "saturation_scale": 0.94,
                "ambient_hex": "#FDBA74",
            },
            {
                "id": "boss_eclipse",
                "name": "Violet Boss Eclipse",
                "recommended_waves": "Boss Waves (wave_no % 5 == 0, Finale/Boss)",
                "rgb_multiply": [0.84, 0.80, 0.98],
                "rgb_lift": [0.032, 0.010, 0.058],
                "saturation_scale": 0.88,
                "boss_accent_hex": "#D946EF",
                "ambient_hex": "#C084FC",
            },
        ],
    }
    (ART005_DIR / "lighting_palettes.json").write_text(json.dumps(lighting_doc, indent=2), encoding="utf-8")

    print(
        f"[BUILD ENVIRONMENT PASS] 26 canonical sprites, {len(companion_assets)} companion depth/transition assets, "
        "4 inspection sheets, and 3 deterministic JSON contracts written."
    )


if __name__ == "__main__":
    main()
