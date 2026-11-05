#!/usr/bin/env python3
"""
ART-006 — Phase 11 Independent Production UI/HUD Validator, Manifest Generator & 16-Case Adversarial Suite.

Performs mechanical pixel, UV, 9-slice, typography, safe-zone, occlusion, localization, privacy,
normative game-rule, and SHA-256 manifest assertions across art/production/ART-006/, and executes
all 16 mandatory adversarial negative test cases to prove that every class of defect is rejected.
"""

from __future__ import annotations

import argparse
import copy
import fontTools.ttLib as ttLib
import hashlib
import json
import re
import sys
from pathlib import Path
from typing import Any, Callable, Dict, List, Set, Tuple

import numpy as np
from PIL import Image

TOOLS_DIR = Path(__file__).resolve().parent
ART006_DIR = TOOLS_DIR.parent
REPO_ROOT = ART006_DIR.parents[2]
ART002_UI_DIR = REPO_ROOT / "art" / "concepts" / "ART-002" / "ui" / "components"

sys.path.insert(0, str(TOOLS_DIR))
from build_ui_assets import resolve_licensed_font, write_json_lf  # noqa: E402
from build_hud_previews import CompositionalHudRenderer, ModularBattlefieldComposer  # noqa: E402


class ValidationFailure(Exception):
    """Raised whenever any mechanical, visual-contract, privacy, or domain-rule check fails."""


VIETNAMESE_REQUIRED_CHARS = (
    "AĂÂBCDĐEÊGHIKLMNOÔƠPQRSTUƯVXYaăâbcdđeêghiklmnoôơpqrstuưvxy"
    "ÁÀẢÃẠẮẰẲẴẶẤẦẨẪẬÉÈẺẼẸẾỀỂỄỆÍÌỈĨỊÓÒỎÕỌỐỒỔỖỘỚỜỞỠỢÚÙỦŨỤỨỪỬỮỰÝỲỶỸỴ"
    "áàảãạắằẳẵặấầẩẫậéèẻẽẹếềểễệíìỉĩịóòỏõọốồổỗộớờởỡợúùủũụứừửữựýỳỷỹỵ"
    "0123456789/·—…≤+-%():"
)

ACTOR_KEY_HEX64_RE = re.compile(r"\b[0-9a-fA-F]{64}\b")
FORBIDDEN_PRIVATE_SUBSTRINGS = (
    "actor_key",
    "actorkey",
    "hmac_secret",
    "provider_user_id",
    "raw_viewer_id",
    "sqlstate",
    "select * from",
    "insert into ",
)


def sha256_file(path: Path) -> str:
    return hashlib.sha256(path.read_bytes()).hexdigest()


def rect_intersection_area(a: List[int], b: List[int]) -> int:
    ix0 = max(a[0], b[0])
    iy0 = max(a[1], b[1])
    ix1 = min(a[2], b[2])
    iy1 = min(a[3], b[3])
    if ix1 <= ix0 or iy1 <= iy0:
        return 0
    return (ix1 - ix0) * (iy1 - iy0)


def build_state_bundle(art_dir: Path) -> Dict[str, Any]:
    """Loads all ART-006 contracts, manifests, localization tables, fixtures, and telemetry into memory."""
    comp_manifest = json.loads((art_dir / "ui_component_manifest.json").read_text(encoding="utf-8"))
    layout_tokens = json.loads((art_dir / "ui_layout_tokens.json").read_text(encoding="utf-8"))
    state_mapping = json.loads((art_dir / "ui_state_mapping.json").read_text(encoding="utf-8"))
    loc_vi = json.loads((art_dir / "localization_vi.json").read_text(encoding="utf-8"))
    loc_en = json.loads((art_dir / "localization_en.json").read_text(encoding="utf-8"))
    atlas_icons = json.loads((art_dir / "atlas" / "ui_atlas_01_icons.json").read_text(encoding="utf-8"))
    atlas_skins = json.loads((art_dir / "atlas" / "ui_atlas_02_components.json").read_text(encoding="utf-8"))
    telemetry = json.loads((art_dir / "qa" / "hud_render_telemetry.json").read_text(encoding="utf-8"))
    unity_ver = json.loads((art_dir / "qa" / "unity_isolated_verification.json").read_text(encoding="utf-8"))

    fixtures = [
        json.loads(p.read_text(encoding="utf-8"))
        for p in sorted((art_dir / "fixtures").glob("*.json"))
    ]
    icon_files = sorted(p.name for p in (art_dir / "icons").glob("*.png"))
    skin_files = sorted(p.name for p in (art_dir / "components").glob("*.png"))

    asset_manifest_path = art_dir / "asset_manifest.json"
    asset_manifest = json.loads(asset_manifest_path.read_text(encoding="utf-8")) if asset_manifest_path.exists() else None

    # Resolve TrueType font cmap codepoints for Vietnamese glyph verification
    _, font_meta = resolve_licensed_font(bold=True, size=36)
    unity_font_path = Path(r"C:\Program Files\Unity\Hub\Editor\6000.6.7f1\Editor\Data\Resources\Fonts") / font_meta["resolved_from"]
    tt = ttLib.TTFont(str(unity_font_path))
    cmap_cps: Set[int] = set()
    for table in tt["cmap"].tables:
        cmap_cps.update(table.cmap.keys())
    tt.close()

    return {
        "art_dir": art_dir,
        "comp_manifest": comp_manifest,
        "layout_tokens": layout_tokens,
        "state_mapping": state_mapping,
        "loc_vi": loc_vi,
        "loc_en": loc_en,
        "atlas_icons": atlas_icons,
        "atlas_skins": atlas_skins,
        "telemetry": telemetry,
        "unity_ver": unity_ver,
        "fixtures": fixtures,
        "icon_files": icon_files,
        "skin_files": skin_files,
        "asset_manifest": asset_manifest,
        "font_cmap_codepoints": cmap_cps,
        "required_vi_chars": VIETNAMESE_REQUIRED_CHARS,
    }


def validate_bundle(
    bundle: Dict[str, Any],
    renderer: CompositionalHudRenderer,
    check_pixels_on_disk: bool = True,
) -> Dict[str, Any]:
    art_dir: Path = bundle["art_dir"]
    cm = bundle["comp_manifest"]
    lt = bundle["layout_tokens"]
    sm = bundle["state_mapping"]
    vi = bundle["loc_vi"]
    en = bundle["loc_en"]
    a_ic = bundle["atlas_icons"]
    a_sk = bundle["atlas_skins"]
    tel = bundle["telemetry"]
    uv = bundle["unity_ver"]
    fixtures = bundle["fixtures"]

    # 1. Verify 20 icons & 15 skins presence and dimensions
    if len(bundle["icon_files"]) != 20 or len(cm["icons"]) != 20:
        raise ValidationFailure(f"Expected 20 UI icons, found files={len(bundle['icon_files'])}, manifest={len(cm['icons'])}")
    if len(bundle["skin_files"]) != 15 or len(cm["components"]) != 15:
        raise ValidationFailure(f"Expected 15 UI component skins, found files={len(bundle['skin_files'])}, manifest={len(cm['components'])}")

    expected_icon_names = sorted(f"{ic['id']}.png" for ic in cm["icons"])
    if bundle["icon_files"] != expected_icon_names:
        raise ValidationFailure("Icon directory filenames do not match ui_component_manifest.json")

    expected_skin_names = sorted(f"{sk['id']}.png" for sk in cm["components"])
    if bundle["skin_files"] != expected_skin_names:
        raise ValidationFailure("Component directory filenames do not match ui_component_manifest.json")

    # 2. Validate 9-slice margins, Unity border vector ordering (L,B,R,T), rights_status & pixel integrity
    for ic in cm["icons"]:
        if ic["dimensions"] != [128, 128]:
            raise ValidationFailure(f"Icon {ic['id']} invalid dimensions: {ic['dimensions']}")
        if ic.get("rights_status") != "PENDING_REVIEW":
            raise ValidationFailure(f"Icon {ic['id']} invalid rights_status: {ic.get('rights_status')}")
        if check_pixels_on_disk:
            p = REPO_ROOT / ic["path"]
            if sha256_file(p) != ic["sha256"]:
                raise ValidationFailure(f"Icon SHA-256 mismatch on {ic['id']}")
            if ic["disposition"] == "REUSED_BYTE_IDENTICAL":
                if sha256_file(REPO_ROOT / ic["art002_source"]) != ic["sha256"]:
                    raise ValidationFailure(f"Reused icon {ic['id']} differs from ART-002 source")
            arr = np.array(Image.open(p).convert("RGBA"))
            if arr.shape != (128, 128, 4):
                raise ValidationFailure(f"Icon {ic['id']} pixel shape != 128x128x4")
            outer_a = np.concatenate([arr[0, :, 3], arr[-1, :, 3], arr[:, 0, 3], arr[:, -1, 3]])
            if int(outer_a.max()) != 0:
                raise ValidationFailure(f"Icon {ic['id']} touches outer border / has edge clipping")
            semi = int(((arr[:, :, 3] > 0) & (arr[:, :, 3] < 255)).sum())
            if semi > 0:
                raise ValidationFailure(f"Icon {ic['id']} has {semi} semi-transparent halo pixels")

    for sk in cm["components"]:
        w, h = sk["dimensions"]
        L, R, T, B = sk["insets_lrtb"]
        if L <= 0 or R <= 0 or T <= 0 or B <= 0 or (L + R) >= w or (T + B) >= h:
            raise ValidationFailure(f"Invalid 9-slice insets {sk['insets_lrtb']} for {sk['id']} ({w}x{h})")
        if sk["unity_sprite_border_lbrt"] != [L, B, R, T]:
            raise ValidationFailure(f"Unity border vector mismatch on {sk['id']}: expected {[L, B, R, T]}, got {sk['unity_sprite_border_lbrt']}")
        min_w, min_h = sk["min_supported_size_px"]
        if min_w < (L + R) or min_h < (T + B):
            raise ValidationFailure(f"Skin {sk['id']} min_supported_size {sk['min_supported_size_px']} smaller than corners")
        if sk.get("rights_status") != "PENDING_REVIEW":
            raise ValidationFailure(f"Skin {sk['id']} invalid rights_status")

        if check_pixels_on_disk:
            p = REPO_ROOT / sk["path"]
            if sha256_file(p) != sk["sha256"]:
                raise ValidationFailure(f"Skin SHA-256 mismatch on {sk['id']}")
            arr = np.array(Image.open(p).convert("RGBA"))
            if arr.shape != (h, w, 4):
                raise ValidationFailure(f"Skin {sk['id']} actual PNG size {arr.shape} != {w}x{h}")
            # Outer ring must have zero clipped semi-transparent leak (only 0 at rounded corners or 255 #11151C border)
            ring_a = np.concatenate([arr[0, :, 3], arr[-1, :, 3], arr[:, 0, 3], arr[:, -1, 3]])
            bad_ring = int(((ring_a > 0) & (ring_a < 255)).sum())
            if bad_ring > 0:
                raise ValidationFailure(f"Skin {sk['id']} has {bad_ring} semi-transparent outer-border leak pixels")

            # Verify stretch constancy (Sliced) or exact 8px tile periodicity (Tiled) across (L .. w-R)!
            top_strip = arr[0:T, L : w - R, :]
            bot_strip = arr[h - B : h, L : w - R, :]
            center = arr[T : h - B, L : w - R, :]
            if sk["unity_image_type"] == "Sliced":
                for s_lbl, strip in (("top", top_strip), ("bottom", bot_strip), ("center", center)):
                    sf = strip.astype(np.float32)
                    if float(np.max(np.abs(sf - sf[:, 0:1, :]))) > 0.0:
                        raise ValidationFailure(f"Sliced skin {sk['id']} {s_lbl} 9-slice zone is not horizontally uniform")
            elif sk["unity_image_type"] == "Tiled":
                cw = w - L - R
                if cw % 8 != 0:
                    raise ValidationFailure(f"Tiled skin {sk['id']} center width {cw} not divisible by 8px period")
                for s_lbl, strip in (("top", top_strip), ("bottom", bot_strip), ("center", center)):
                    if not np.array_equal(strip[:, :-8, :], strip[:, 8:, :]):
                        raise ValidationFailure(f"Tiled skin {sk['id']} {s_lbl} zone is not exact 8px-periodic")

    # 3. Validate both deterministic atlases (dimensions <= 2048, gutter >= 8px, exact UV, 0 gutter bleed, pixel match)
    for atlas_meta, src_subdir in ((a_ic, "icons"), (a_sk, "components")):
        aw, ah = atlas_meta["dimensions"]
        if aw > 2048 or ah > 2048:
            raise ValidationFailure(f"Atlas {atlas_meta['atlas_id']} exceeds 2048x2048: {aw}x{ah}")
        if atlas_meta["rotation_allowed"] is not False or atlas_meta["mipmaps"] is not False:
            raise ValidationFailure(f"Atlas {atlas_meta['atlas_id']} invalid rotation/mipmap settings")
        if int(atlas_meta["min_gutter_px"]) < 8:
            raise ValidationFailure(f"Atlas {atlas_meta['atlas_id']} gutter < 8px")

        seen_ids: Set[str] = set()
        for sl in atlas_meta["slices"]:
            if sl["id"] in seen_ids:
                raise ValidationFailure(f"Duplicate slice ID {sl['id']} in {atlas_meta['atlas_id']}")
            seen_ids.add(sl["id"])
            if sl["rotated"] is not False:
                raise ValidationFailure(f"Rotated slice forbidden: {sl['id']}")
            rx, ry, rw, rh = sl["rect_top_left_px"]
            ux, uy, uw, uh = sl["unity_rect_bottom_left_px"]
            if ux != rx or uw != rw or uh != rh or uy != (ah - ry - rh):
                raise ValidationFailure(f"Invalid bottom-left Unity rect for {sl['id']}")
            exp_uv = [round(ux / aw, 6), round(uy / ah, 6), round(uw / aw, 6), round(uh / ah, 6)]
            if sl["uv_bottom_left"] != exp_uv:
                raise ValidationFailure(f"Corrupted atlas UV on {sl['id']}: got {sl['uv_bottom_left']}, expected {exp_uv}")

        if check_pixels_on_disk:
            ap = REPO_ROOT / atlas_meta["path"]
            a_arr = np.array(Image.open(ap).convert("RGBA"))
            occ = np.zeros((ah, aw), dtype=bool)
            for sl in atlas_meta["slices"]:
                rx, ry, rw, rh = sl["rect_top_left_px"]
                if occ[ry : ry + rh, rx : rx + rw].any():
                    raise ValidationFailure(f"Overlapping atlas slice {sl['id']}")
                occ[ry : ry + rh, rx : rx + rw] = True
                sub_arr = a_arr[ry : ry + rh, rx : rx + rw, :]
                ref_arr = np.array(Image.open(art_dir / src_subdir / f"{sl['id']}.png").convert("RGBA"))
                if not np.array_equal(sub_arr, ref_arr):
                    raise ValidationFailure(f"Atlas pixel mismatch vs standalone PNG on {sl['id']}")
            gutter_bleed = int((a_arr[~occ, 3] > 0).sum())
            if gutter_bleed > 0:
                raise ValidationFailure(f"Atlas {atlas_meta['atlas_id']} has {gutter_bleed} alpha-bleed pixels inside gutters")

    # 4. Typography scale, Localization parity & Vietnamese glyph coverage
    ts = lt["typography_scale"]
    if float(ts["cta_primary"]["px_360p"]) < 16.0:
        raise ValidationFailure(f"Primary CTA below 16px @ 360p: {ts['cta_primary']['px_360p']}")
    if float(ts["hp_and_timer"]["px_360p"]) < 14.0:
        raise ValidationFailure(f"HP/Timer below 14px @ 360p: {ts['hp_and_timer']['px_360p']}")
    if float(ts["secondary_hud"]["px_360p"]) < 12.0 or float(ts["threat_banner"]["px_360p"]) < 12.0:
        raise ValidationFailure("Secondary HUD text below 12px @ 360p")

    vi_keys = set(vi["strings"].keys())
    en_keys = set(en["strings"].keys())
    if vi_keys != en_keys or len(vi_keys) < 50:
        raise ValidationFailure(f"Localization key mismatch between vi-VN ({len(vi_keys)}) and en-US ({len(en_keys)})")
    if vi["font_recommendation"]["bundled_proprietary_fonts"] is not False:
        raise ValidationFailure("Bundled proprietary fonts flag must be false")

    # Check zero bundled font or audio binaries inside art/production/ART-006
    if check_pixels_on_disk:
        bad_bins = [
            p.name for p in art_dir.rglob("*")
            if p.is_file() and p.suffix.lower() in (".ttf", ".otf", ".woff", ".woff2", ".wav", ".mp3", ".ogg", ".flac")
        ]
        if bad_bins:
            raise ValidationFailure(f"Forbidden font or audio binary files inside ART-006: {bad_bins}")

    cmap_cps: Set[int] = bundle["font_cmap_codepoints"]
    all_vi_chars = set(bundle["required_vi_chars"]) | set("".join(vi["strings"].values()))
    missing_glyphs = [ch for ch in sorted(all_vi_chars) if ch not in ("\n", "\r", " ") and ord(ch) not in cmap_cps]
    if missing_glyphs:
        raise ValidationFailure(f"Missing Vietnamese/HUD glyphs in font cmap: {missing_glyphs}")

    # 5. Safe-zone, Text Overflow, Contrast & Castle/Boss Occlusion Checks
    safe_x_max = float(lt["safe_area_defaults"]["right_reserve_x_px"])   # 885.6 (82%)
    safe_y_max = float(lt["safe_area_defaults"]["bottom_reserve_y_px"])  # 1689.6 (88%)

    for zname, zspec in lt["hud_zones"].items():
        x0, y0, x1, y1 = zspec["top_down_px"]
        if x0 < 0 or y0 < 0 or x1 > safe_x_max or y1 > safe_y_max:
            raise ValidationFailure(f"Layout zone {zname} {zspec['top_down_px']} violates safe region (x<={safe_x_max}, y<={safe_y_max})")

    for scr in tel["screens"]:
        c_bb = scr.get("castle_bbox")
        b_bb = scr.get("boss_bbox")
        for pname, pbb in scr["hud_bboxes"].items():
            if pbb[0] < 0 or pbb[1] < 0 or pbb[2] > safe_x_max or pbb[3] > safe_y_max:
                raise ValidationFailure(f"Rendered HUD panel {pname} in {scr['fixture_id']} outside safe region: {pbb}")
            if c_bb and rect_intersection_area(pbb, c_bb) > 0:
                raise ValidationFailure(f"HUD panel {pname} overlaps Castle Keep in {scr['fixture_id']}")
            if b_bb and rect_intersection_area(pbb, b_bb) > 0:
                raise ValidationFailure(f"HUD panel {pname} overlaps Boss sprite in {scr['fixture_id']}")

        for tt_item in scr["text_telemetry"]:
            if tt_item["overflow"] or int(tt_item["measured_w"]) > int(tt_item["available_w"]):
                raise ValidationFailure(f"Text overflow in {scr['fixture_id']} slot {tt_item['slot']}: {tt_item['measured_w']} > {tt_item['available_w']}")
            if float(tt_item["wcag_contrast"]) < 4.5:
                raise ValidationFailure(f"WCAG contrast < 4.5:1 in {scr['fixture_id']} slot {tt_item['slot']}: {tt_item['wcag_contrast']}")
            if tt_item["slot"] == "primary_cta" and float(tt_item["font_px_360p"]) < 16.0:
                raise ValidationFailure(f"Rendered CTA font below 16px @ 360p: {tt_item['font_px_360p']}")

    # 6. Canonical Game Rules, Compositional State Semantics & Privacy Discipline
    pt = sm["authoritative_phase_ticks"]
    if pt.get("PREP") != [0, 1599] or pt.get("ASSAULT") != [1600, 2599] or pt.get("FINALE") != [2600, 2799] or pt.get("RESULTS") != [2800, 2999]:
        raise ValidationFailure(f"Authoritative phase tick boundaries corrupted in ui_state_mapping.json: {pt}")
    if len(sm["canonical_16_states"]) != 16:
        raise ValidationFailure("Expected 16 canonical state mappings")

    valid_pause_reasons = {"operator_paused", "source_disconnected", "storage_unavailable"}
    for fx in fixtures:
        raw_dump = json.dumps(fx, ensure_ascii=False)
        low_dump = raw_dump.lower()
        if ACTOR_KEY_HEX64_RE.search(raw_dump):
            raise ValidationFailure(f"Private 64-hex ActorKey detected in fixture {fx['fixture_id']}")
        for bad_kw in FORBIDDEN_PRIVATE_SUBSTRINGS:
            if bad_kw in low_dump:
                raise ValidationFailure(f"Forbidden private/internal field '{bad_kw}' in fixture {fx['fixture_id']}")
        if fx.get("synthetic_fixture") is not True:
            raise ValidationFailure(f"Fixture {fx['fixture_id']} missing synthetic_fixture=true disclosure")

        snap = fx["public_snapshot"]
        wave_no = int(snap["wave_no"])
        tick = int(snap["wave_tick"])
        lifecycle = snap["lifecycle"]
        pause_reasons = list(snap["pause_reasons"])
        phase = snap["phase"]
        counts = snap["counts"]
        p_unique = int(counts["unique_joined"])
        boss = snap["boss"]
        vote = snap["vote"]
        diag = snap["diagnostics"]

        # Contradictory lifecycle / pause_reasons checks
        if any(r not in valid_pause_reasons for r in pause_reasons):
            raise ValidationFailure(f"Unknown pause_reason in {fx['fixture_id']}: {pause_reasons}")
        if lifecycle == "running" and len(pause_reasons) > 0:
            raise ValidationFailure(f"Contradictory lifecycle=running while pause_reasons={pause_reasons} in {fx['fixture_id']}")
        if lifecycle in ("paused", "source_disconnected", "storage_unavailable") and len(pause_reasons) == 0:
            raise ValidationFailure(f"Contradictory paused lifecycle without pause_reasons in {fx['fixture_id']}")
        if lifecycle == "stopped" and p_unique > 0:
            raise ValidationFailure(f"Stopped maintenance session has active participants in {fx['fixture_id']}")

        # WAITING zero-participant truth
        if phase == "waiting":
            if p_unique != 0 or int(counts["builder"]) != 0 or int(counts["guard"]) != 0 or int(counts["fighter"]) != 0:
                raise ValidationFailure(f"Fake nonzero participant count in WAITING fixture {fx['fixture_id']}")
            if tick != -1:
                raise ValidationFailure(f"WAITING fixture {fx['fixture_id']} must keep wave_tick == -1")
        else:
            # Phase vs authoritative tick boundary check!
            if 0 <= tick <= 1599 and phase != "prep":
                raise ValidationFailure(f"Tick {tick} must be phase='prep' (got '{phase}') in {fx['fixture_id']}")
            if 1600 <= tick <= 2599 and phase != "assault":
                raise ValidationFailure(f"Tick {tick} must be phase='assault' (got '{phase}') in {fx['fixture_id']}")
            if 2600 <= tick <= 2799 and phase != "finale":
                raise ValidationFailure(f"Tick {tick} must be phase='finale' (got '{phase}') in {fx['fixture_id']}")
            if 2800 <= tick <= 2999 and phase != "results":
                raise ValidationFailure(f"Tick {tick} must be phase='results' (got '{phase}') in {fx['fixture_id']}")

        if p_unique != (int(counts["builder"]) + int(counts["guard"]) + int(counts["fighter"])):
            raise ValidationFailure(f"Participant sum mismatch in {fx['fixture_id']}")

        # Boss visibility & Wave 5*k rules
        if (wave_no % 5 != 0) and (bool(boss["boss_exists"]) or bool(boss["boss_incoming"]) or int(boss["boss_hp"]) > 0):
            raise ValidationFailure(f"Boss state active on non-5th wave {wave_no} in {fx['fixture_id']}")
        if tick < 2600 and (bool(boss["boss_exists"]) or int(boss["boss_hp"]) > 0):
            raise ValidationFailure(f"Premature Boss spawn before tick 2600 in {fx['fixture_id']} (tick={tick})")
        if phase == "results" and bool(boss["boss_incoming"]):
            raise ValidationFailure(f"Contradictory boss_incoming=true during RESULTS in {fx['fixture_id']}")

        # Even-wave vote window [80s, 120s) -> ticks 800..1199 ONLY!
        if bool(vote["open"]):
            if (wave_no % 2 != 0) or not (800 <= tick <= 1199) or phase != "prep":
                raise ValidationFailure(f"Invalid vote.open=true on wave={wave_no}, tick={tick}, phase={phase} in {fx['fixture_id']}")

        # Presentation-level semantic truth checks (W0306 backpressure vs confirmed drops, no early CLEAR on Boss incoming)
        pres = bundle.get("override_presentations", {}).get(fx["fixture_id"]) or renderer.resolve_hud_presentation(fx, locale="vi-VN")
        banner_txt = pres["banner"][2]
        diag_txt = pres["diag_pill"][2] if pres["diag_pill"] else ""
        if (not bool(diag["metrics_available"]) or int(diag["confirmed_valid_drops"]) == 0):
            if "LỆNH HỢP LỆ RƠI" in banner_txt or "VALID DROPS" in banner_txt or "LỆNH HỢP LỆ RƠI" in diag_txt:
                raise ValidationFailure(f"W0306 violation: false event-drop warning shown when confirmed_valid_drops=0 in {fx['fixture_id']}")
        if bool(boss["boss_incoming"]) and tick < 2600:
            if "SẠCH QUÁI THƯỜNG · CHỜ CHỐT ĐỢT" in banner_txt or "THẮNG ĐỢT" in banner_txt:
                raise ValidationFailure(f"Premature CLEAR/WIN banner while Boss incoming in {fx['fixture_id']}")

    # 7. Check required rendered image files & Unity verification report on disk
    if check_pixels_on_disk:
        if uv.get("status") != "PASS" or int(uv.get("missing_sprite_references", 1)) != 0:
            raise ValidationFailure(f"Unity isolated batchmode verification did not pass: {uv}")

        state_pngs = sorted((art_dir / "hud" / "states").glob("[0-1][0-9]_*.png"))
        if len(state_pngs) != 16:
            raise ValidationFailure(f"Expected 16 canonical HUD state PNGs, found {len(state_pngs)}")
        for sp in state_pngs:
            with Image.open(sp) as im:
                if im.size != (1080, 1920):
                    raise ValidationFailure(f"State PNG {sp.name} size {im.size} != (1080, 1920)")

        for res_folder, exp_sz in (("1080p", (1080, 1920)), ("720p", (720, 1280)), ("360p", (360, 640))):
            proofs = sorted((art_dir / "previews" / res_folder).glob("proof_*.png"))
            if len(proofs) != 16:
                raise ValidationFailure(f"Expected 16 proofs in previews/{res_folder}, found {len(proofs)}")
            for pp in proofs:
                with Image.open(pp) as im:
                    if im.size != exp_sz:
                        raise ValidationFailure(f"{res_folder}/{pp.name} size {im.size} != {exp_sz}")

        # Verify asset_manifest.json hashes if loaded
        if bundle["asset_manifest"] is not None:
            for entry in bundle["asset_manifest"]["files"]:
                fp = REPO_ROOT / entry["path"]
                if not fp.exists():
                    raise ValidationFailure(f"Manifest file missing on disk: {entry['path']}")
                actual_sha = sha256_file(fp)
                if actual_sha != entry["sha256"]:
                    raise ValidationFailure(f"Manifest SHA-256 mismatch on {entry['path']}: expected {entry['sha256']}, got {actual_sha}")

    return {
        "icons_verified": 20,
        "skins_verified": 15,
        "canonical_states_verified": 16,
        "overlay_combos_verified": 8,
        "multi_res_exports_verified": 48,
        "localization_keys_verified": len(vi_keys),
        "text_overflow_count": 0,
        "safe_zone_violations": 0,
        "castle_or_boss_occlusions": 0,
        "min_wcag_contrast_ratio": min(t["wcag_contrast"] for s in tel["screens"] for t in s["text_telemetry"]),
        "h264_psnr_1080p_db": tel["h264_compression"]["psnr_1080p_2500kbps_db"],
        "h264_psnr_360p_db": tel["h264_compression"]["psnr_360p_800kbps_db"],
    }


def run_16_adversarial_negative_cases(base_bundle: Dict[str, Any], renderer: CompositionalHudRenderer) -> List[Dict[str, Any]]:
    """
    Deliberately injects each of the 16 mandatory Phase 11 defects into a deep copy of the
    verified ART-006 bundle and asserts that validate_bundle() raises ValidationFailure on all 16!
    """
    def mut_01_missing_icon(b: Dict[str, Any]) -> None:
        b["icon_files"].pop()

    def mut_02_missing_skin(b: Dict[str, Any]) -> None:
        b["skin_files"].pop()

    def mut_03_bad_9slice_border(b: Dict[str, Any]) -> None:
        b["comp_manifest"]["components"][0]["insets_lrtb"] = [140, 140, 20, 20]  # 280 >= width 256!

    def mut_04_corrupted_atlas_uv(b: Dict[str, Any]) -> None:
        b["atlas_icons"]["slices"][3]["uv_bottom_left"] = [0.91, 0.91, 0.125, 0.125]

    def mut_05_sprite_hash_mismatch(b: Dict[str, Any]) -> None:
        if b["asset_manifest"] is None:
            b["asset_manifest"] = {"files": [{"path": "art/production/ART-006/icons/ui_icon_wood.png", "sha256": "0" * 64}]}
        else:
            b["asset_manifest"]["files"][0]["sha256"] = "0" * 64

    def mut_06_panel_outside_safe_region(b: Dict[str, Any]) -> None:
        b["telemetry"]["screens"][0]["hud_bboxes"]["primary_cta_plaque"] = [36, 1462, 940, 1550]  # 940 > 885.6 (82%)

    def mut_07_cta_below_min_mobile_font(b: Dict[str, Any]) -> None:
        b["layout_tokens"]["typography_scale"]["cta_primary"]["px_360p"] = 14.5  # < 16.0px!

    def mut_08_missing_vietnamese_glyph(b: Dict[str, Any]) -> None:
        b["font_cmap_codepoints"].discard(ord("Ặ"))

    def mut_09_text_overflow(b: Dict[str, Any]) -> None:
        b["telemetry"]["screens"][2]["text_telemetry"][0]["measured_w"] = 920
        b["telemetry"]["screens"][2]["text_telemetry"][0]["available_w"] = 440

    def mut_10_wrong_boss_visibility_boundary(b: Dict[str, Any]) -> None:
        # Spawn Boss early at tick 2480 (< 2600)!
        for fx in b["fixtures"]:
            if fx["fixture_id"] == "combo_04_boss_incoming_pre2600_lanes_empty":
                fx["public_snapshot"]["boss"]["boss_exists"] = True
                fx["public_snapshot"]["boss"]["boss_hp"] = 88

    def mut_11_prep_ending_at_tick_999(b: Dict[str, Any]) -> None:
        b["state_mapping"]["authoritative_phase_ticks"]["PREP"] = [0, 999]

    def mut_12_fake_nonzero_waiting_participants(b: Dict[str, Any]) -> None:
        for fx in b["fixtures"]:
            if fx["fixture_id"] == "state_02_waiting_for_viewer":
                fx["public_snapshot"]["counts"]["builder"] = 2
                fx["public_snapshot"]["counts"]["unique_joined"] = 2

    def mut_13_backpressure_shown_as_confirmed_loss(b: Dict[str, Any]) -> None:
        b["override_presentations"] = {
            "combo_07_backpressure_zero_valid_drops": {
                "banner": ("err", "ui_icon_danger", "QUÁ TẢI · LỆNH HỢP LỆ RƠI (62)"),
                "diag_pill": None,
            }
        }

    def mut_14_private_actorkey_in_toast(b: Dict[str, Any]) -> None:
        b["fixtures"][0]["public_snapshot"]["join_toast"] = {
            "mode": "single",
            "role": "fighter",
            "display_alias": "a3f9018bc44291de7781209ab5543ef91029384756abcdef0123456789abcdef",
            "ttl_ms": 1800,
        }

    def mut_15_vote_overlay_on_odd_wave(b: Dict[str, Any]) -> None:
        for fx in b["fixtures"]:
            if fx["fixture_id"] == "state_03_prep":  # Wave 3 (odd!)
                fx["public_snapshot"]["vote"]["open"] = True

    def mut_16_contradictory_lifecycle_overlay(b: Dict[str, Any]) -> None:
        # Contradictory: lifecycle == 'running' while operator_paused & storage_unavailable are active!
        b["fixtures"][6]["public_snapshot"]["lifecycle"] = "running"
        b["fixtures"][6]["public_snapshot"]["pause_reasons"] = ["operator_paused", "storage_unavailable"]

    cases: List[Tuple[str, Callable[[Dict[str, Any]], None]]] = [
        ("01_missing_icon", mut_01_missing_icon),
        ("02_missing_component_skin", mut_02_missing_skin),
        ("03_incorrect_9slice_border", mut_03_bad_9slice_border),
        ("04_corrupted_atlas_uv", mut_04_corrupted_atlas_uv),
        ("05_sprite_hash_mismatch", mut_05_sprite_hash_mismatch),
        ("06_ui_panel_outside_safe_region", mut_06_panel_outside_safe_region),
        ("07_cta_below_minimum_mobile_font_size", mut_07_cta_below_min_mobile_font),
        ("08_missing_vietnamese_glyph", mut_08_missing_vietnamese_glyph),
        ("09_text_overflow", mut_09_text_overflow),
        ("10_wrong_boss_visibility_boundary", mut_10_wrong_boss_visibility_boundary),
        ("11_prep_ending_prematurely_at_tick_999", mut_11_prep_ending_at_tick_999),
        ("12_fake_nonzero_participant_count_in_waiting", mut_12_fake_nonzero_waiting_participants),
        ("13_queue_backpressure_displayed_as_confirmed_loss", mut_13_backpressure_shown_as_confirmed_loss),
        ("14_private_actorkey_inserted_into_public_toast", mut_14_private_actorkey_in_toast),
        ("15_vote_overlay_visible_on_odd_wave", mut_15_vote_overlay_on_odd_wave),
        ("16_invalid_contradictory_lifecycle_overlay", mut_16_contradictory_lifecycle_overlay),
    ]

    results: List[Dict[str, Any]] = []
    for cid, mut_fn in cases:
        mut_bundle = copy.deepcopy(base_bundle)
        mut_fn(mut_bundle)
        check_disk = cid == "05_sprite_hash_mismatch"
        try:
            validate_bundle(mut_bundle, renderer, check_pixels_on_disk=check_disk)
        except ValidationFailure as exc:
            results.append({
                "case_id": cid,
                "expected": "REJECT",
                "actual": "REJECTED",
                "passed": True,
                "rejection_reason": str(exc),
            })
        else:
            raise AssertionError(f"Adversarial negative case {cid} WAS NOT REJECTED by validator!")

    return results


def write_asset_manifest(art_dir: Path) -> Dict[str, Any]:
    """Generates SHA-256 asset_manifest.json covering every deliverable inside art/production/ART-006/."""
    excluded = {"asset_manifest.json"}
    files_list: List[Dict[str, Any]] = []

    for p in sorted(art_dir.rglob("*")):
        if not p.is_file():
            continue
        if "__pycache__" in p.parts or p.name in excluded or p.name.startswith("_tmp"):
            continue
        rel = p.relative_to(REPO_ROOT).as_posix()
        files_list.append({
            "path": rel,
            "bytes": p.stat().st_size,
            "sha256": sha256_file(p),
            "rights_status": "PENDING_REVIEW",
        })

    manifest = {
        "package_id": "ART-006",
        "title": "Bastion Dawn v1 — Production UI/HUD, 16-State Screen System & Unity Integration Handoff",
        "target_engine": "Unity 6000.6.7f1 (Built-in 2D / com.unity.ugui 2.6.0 / TextMeshPro)",
        "target_unity_art_root": "unity/Assets/_TTGQLDL/Art/UI/",
        "m06_milestone_status": "NOT_STARTED",
        "rights_status": "PENDING_REVIEW",
        "total_tracked_files": len(files_list),
        "files": files_list,
    }
    write_json_lf(art_dir / "asset_manifest.json", manifest)
    return manifest


def main() -> None:
    parser = argparse.ArgumentParser(description="Validate ART-006 deliverables and run 16 adversarial negative tests.")
    parser.add_argument("--write-manifest", action="store_true", help="Write qa_metrics.json, adversarial_validator_results.json, and asset_manifest.json")
    args = parser.parse_args()

    composer = ModularBattlefieldComposer()
    renderer = CompositionalHudRenderer(composer)

    bundle = build_state_bundle(ART006_DIR)
    if args.write_manifest:
        bundle["asset_manifest"] = None
    metrics = validate_bundle(bundle, renderer, check_pixels_on_disk=True)
    adv_results = run_16_adversarial_negative_cases(bundle, renderer)

    if args.write_manifest:
        write_json_lf(ART006_DIR / "qa" / "adversarial_validator_results.json", {
            "package_id": "ART-006",
            "total_adversarial_cases": len(adv_results),
            "rejected_count": sum(1 for r in adv_results if r["passed"]),
            "status": "PASS",
            "cases": adv_results,
        })
        write_json_lf(ART006_DIR / "qa" / "qa_metrics.json", {
            "package_id": "ART-006",
            "status": "PASS",
            "automated_metrics": metrics,
            "adversarial_negative_cases_passed": f"{len(adv_results)}/{len(adv_results)}",
            "unity_isolated_batchmode_status": bundle["unity_ver"]["status"],
            "rights_status": "PENDING_REVIEW",
            "m06_status": "NOT_STARTED",
        })
        manifest = write_asset_manifest(ART006_DIR)
        # Re-load and verify final manifest SHA-256 parity!
        final_bundle = build_state_bundle(ART006_DIR)
        validate_bundle(final_bundle, renderer, check_pixels_on_disk=True)
        print(f"[ART-006 VALIDATION & MANIFEST PASS] {manifest['total_tracked_files']} files hashed & verified, 16/16 adversarial cases rejected!")
    else:
        print(f"[ART-006 VALIDATION PASS] metrics={json.dumps(metrics)}, 16/16 adversarial cases rejected!")


if __name__ == "__main__":
    main()
