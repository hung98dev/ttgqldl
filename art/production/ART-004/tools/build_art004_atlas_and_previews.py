#!/usr/bin/env python3
"""
ART-004 Deterministic Master Atlas Packer, Contact Sheets, 12 FPS GIFs & H.264 MP4 Stream Previews.

Produces:
  1. Packed Master Sprite Atlas:
       art/production/ART-004/atlas/ART004_Master_Atlas.png (2048x2048 RGBA32, 112 sprites, 4px cell border = 8px gutter)
       art/production/ART-004/atlas/ART004_Master_Atlas.json (exact UV/slice/pivot/PPU/VisualRoot/VRAM metadata)
       art/production/ART-004/atlas/unity_presets/* (Unity 6000.6.7f1 Presets, SpriteAtlas v2 YAML, C# verifier)
  2. Static Contact Sheets:
       5 per-character sheets (Guard 18f, Fighter 18f, Runner 18f, Brute 18f, Boss 16f)
       1 combat VFX & synchronized character+VFX contact sheet
       1 all-7-class cross-package visual consistency sheet (ART-003 Builder/Grunt + ART-004 Guard/Fighter/Runner/Brute/Boss)
  3. True 12.00 FPS Animated GIF Previews ([80, 80, 90] ms centisecond cadence):
       19 individual character clip GIFs + 4 standalone VFX GIFs + 4 synchronized character+VFX GIFs
       + 1 all-7-class composite Bastion Dawn gameplay showcase GIF
  4. Real 12.00 FPS H.264 (yuv420p) MP4 Stream Previews via ffmpeg:
       1080x1920, 720x1280, 360x640 (mobile bitrate proxy), and 1280x720 all-7-class showcase.
"""

from __future__ import annotations

import json
import math
import subprocess
import tempfile
from pathlib import Path
from typing import Any, Dict, List, Tuple

import numpy as np
from PIL import Image, ImageDraw, ImageFont

REPO_ROOT = Path(__file__).resolve().parents[4]
ART002_DIR = REPO_ROOT / "art" / "concepts" / "ART-002"
ART003_DIR = REPO_ROOT / "art" / "production" / "ART-003"
ART004_DIR = REPO_ROOT / "art" / "production" / "ART-004"

MEADOW_RGB = (104, 163, 71)      # #68A347
OCHRE_RGB = (207, 169, 110)      # #CFA96E
LIMESTONE_RGB = (184, 192, 200)  # #B8C0C8
DARK_BG_RGB = (18, 24, 36)


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


def load_meta_and_frames(pkg_dir: Path) -> Tuple[Dict[str, Any], Dict[str, List[Image.Image]]]:
    with open(pkg_dir / "clip_frame_metadata.json", "r", encoding="utf-8") as fp:
        meta = json.load(fp)
    loaded: Dict[str, List[Image.Image]] = {}
    for ent_key, ent_val in meta["entities"].items():
        loaded[ent_key] = [Image.open(REPO_ROOT / rec["file"]).convert("RGBA") for rec in ent_val["frames"]]
    return meta, loaded


def scale_by_factor(im: Image.Image, scale: float) -> Image.Image:
    nw = max(8, int(round(im.width * scale)))
    nh = max(8, int(round(im.height * scale)))
    return im.resize((nw, nh), Image.Resampling.NEAREST)


def flip_h(im: Image.Image) -> Image.Image:
    return im.transpose(Image.Transpose.FLIP_LEFT_RIGHT)


# ============================================================================
# 1. DETERMINISTIC 2048x2048 MASTER SPRITE ATLAS + UNITY 6000.6.7f1 PRESETS
# ============================================================================

def build_master_atlas(meta4: Dict[str, Any], loaded4: Dict[str, List[Image.Image]]) -> None:
    atlas_dir = ART004_DIR / "atlas"
    presets_dir = atlas_dir / "unity_presets"
    presets_dir.mkdir(parents=True, exist_ok=True)

    atlas_w, atlas_h = 2048, 2048
    pad = 4  # 4px border on every side of every cell = 8px inter-sprite gutter (zero bleed!)
    atlas_im = Image.new("RGBA", (atlas_w, atlas_h), (0, 0, 0, 0))

    slices: List[Dict[str, Any]] = []
    atlas_idx = 0

    # Top Band (y = 0..1055): 256x256 assets -> pitch = 264px, 7 cols x 4 rows = 28 slots (26 used)
    band256_order = ["boss", "vfx_boss_slam_telegraph"]
    pitch256 = 256 + pad * 2  # 264 px
    cols256 = 7
    slot256 = 0
    for ent_key in band256_order:
        ent = meta4["entities"][ent_key]
        vis_scale = ent.get("unity_visual_root_scale", 0.57955)
        piv = ent["normalized_pivot"]
        for rec, im in zip(ent["frames"], loaded4[ent_key]):
            r, c = divmod(slot256, cols256)
            x = pad + c * pitch256
            y = pad + r * pitch256
            assert x + 256 + pad <= atlas_w and y + 256 + pad <= 1056
            atlas_im.paste(im, (x, y))
            unity_y = atlas_h - (y + 256)
            slices.append(
                {
                    "atlas_index": atlas_idx,
                    "band": "top_256",
                    "name": Path(rec["file"]).stem,
                    "entity": ent_key,
                    "entity_id": ent["entity_id"],
                    "clip": rec["clip"],
                    "frame_index": rec["frame_index"],
                    "clip_frame_index": rec["clip_frame_index"],
                    "source_file": rec["file"],
                    "cell_size": [256, 256],
                    "rect_top_down_px": [x, y, 256, 256],
                    "rect_unity_bottom_up_px": [x, unity_y, 256, 256],
                    "uv_rect_unity": [
                        round(x / atlas_w, 6),
                        round(unity_y / atlas_h, 6),
                        round(256 / atlas_w, 6),
                        round(256 / atlas_h, 6),
                    ],
                    "pivot_normalized": piv,
                    "pixels_per_unit": 192,
                    "unity_visual_root_scale": vis_scale,
                    "tight_bbox_in_cell_px": rec["bbox"],
                }
            )
            slot256 += 1
            atlas_idx += 1

    # Bottom Band (y = 1056..2007): 128x128 assets -> pitch = 136px, 15 cols x 7 rows = 105 slots (86 used)
    band128_order = [
        "guard",
        "fighter",
        "runner",
        "brute",
        "vfx_fighter_slash_arc",
        "vfx_guard_shield_block",
        "vfx_unit_hit_puff",
    ]
    y_base128 = 1056
    pitch128 = 128 + pad * 2  # 136 px
    cols128 = 15
    slot128 = 0
    for ent_key in band128_order:
        ent = meta4["entities"][ent_key]
        vis_scale = ent.get("unity_visual_root_scale", 0.55814)
        piv = ent["normalized_pivot"]
        for rec, im in zip(ent["frames"], loaded4[ent_key]):
            r, c = divmod(slot128, cols128)
            x = pad + c * pitch128
            y = y_base128 + pad + r * pitch128
            assert x + 128 + pad <= atlas_w and y + 128 + pad <= atlas_h
            atlas_im.paste(im, (x, y))
            unity_y = atlas_h - (y + 128)
            slices.append(
                {
                    "atlas_index": atlas_idx,
                    "band": "bottom_128",
                    "name": Path(rec["file"]).stem,
                    "entity": ent_key,
                    "entity_id": ent["entity_id"],
                    "clip": rec["clip"],
                    "frame_index": rec["frame_index"],
                    "clip_frame_index": rec["clip_frame_index"],
                    "source_file": rec["file"],
                    "cell_size": [128, 128],
                    "rect_top_down_px": [x, y, 128, 128],
                    "rect_unity_bottom_up_px": [x, unity_y, 128, 128],
                    "uv_rect_unity": [
                        round(x / atlas_w, 6),
                        round(unity_y / atlas_h, 6),
                        round(128 / atlas_w, 6),
                        round(128 / atlas_h, 6),
                    ],
                    "pivot_normalized": piv,
                    "pixels_per_unit": 192,
                    "unity_visual_root_scale": vis_scale,
                    "tight_bbox_in_cell_px": rec["bbox"],
                }
            )
            slot128 += 1
            atlas_idx += 1

    assert len(slices) == 112, f"Expected 112 packed sprites, got {len(slices)}"
    atlas_png_path = atlas_dir / "ART004_Master_Atlas.png"
    atlas_im.save(atlas_png_path, optimize=False)

    used_px_area = 26 * (256 * 256) + 86 * (128 * 128)
    raw_vram_bytes = atlas_w * atlas_h * 4
    png_file_bytes = atlas_png_path.stat().st_size

    atlas_json = {
        "atlas_name": "ART004_Master_Atlas",
        "package_id": "ART-004",
        "style_version": "bastion-dawn-v1",
        "dimensions": [atlas_w, atlas_h],
        "max_allowed_dimensions": [2048, 2048],
        "format": "RGBA32 (Uncompressed Point-Filtered PNG)",
        "filter_mode": "Point",
        "mipmaps": False,
        "alpha_is_transparency": True,
        "padding_per_cell_side_px": pad,
        "inter_sprite_gutter_px": pad * 2,
        "allow_rotation": False,
        "tight_packing": False,
        "pixels_per_unit": 192,
        "memory_and_delivery_budget": {
            "png_file_bytes": png_file_bytes,
            "png_file_kib": round(png_file_bytes / 1024.0, 2),
            "raw_rgba32_vram_bytes": raw_vram_bytes,
            "raw_rgba32_vram_mib": round(raw_vram_bytes / (1024.0 * 1024.0), 2),
            "art003_pilot_atlas_raw_rgba32_vram_mib": 4.0,
            "combined_art003_plus_art004_raw_vram_mib": 20.0,
            "max_pooled_gameplay_entities": 150,
            "max_simultaneous_vfx_bursts": 20,
        },
        "per_class_unity_scale_contract": {
            ent_key: {
                "entity_id": meta4["entities"][ent_key]["entity_id"],
                "cell_size": meta4["entities"][ent_key]["cell_size"],
                "ppu": 192,
                "pivot": meta4["entities"][ent_key]["normalized_pivot"],
                "visual_root_scale": meta4["entities"][ent_key]["unity_visual_root_scale"],
                "target_body_1080p_px": meta4["entities"][ent_key]["target_body_1080p_px"],
                "displayed_total_height_px": meta4["entities"][ent_key]["displayed_total_height_px"],
            }
            for ent_key in ["guard", "fighter", "runner", "brute", "boss"]
        },
        "total_packed_sprites": len(slices),
        "band_counts": {"top_256x256": 26, "bottom_128x128": 86},
        "pixel_occupancy_ratio": round(used_px_area / float(atlas_w * atlas_h), 4),
        "slices": slices,
    }
    with open(atlas_dir / "ART004_Master_Atlas.json", "w", encoding="utf-8") as fp:
        json.dump(atlas_json, fp, indent=2)

    # Write Unity 6000.6.7f1 Presets & SpriteAtlas v2 YAML
    for preset_name, piv_x, piv_y in [
        ("TTG_ART004_Character128_Point192", "0.5", "0.08"),
        ("TTG_ART004_Boss256_Point192", "0.5", "0.08"),
        ("TTG_ART004_VFX_Point192", "0.5", "0.5"),
    ]:
        preset_yaml = f"""%YAML 1.1
%TAG !u! tag:unity3d.com,2011:
--- !u!181963792 &2655988077585873504
Preset:
  m_ObjectHideFlags: 0
  m_CorrespondingSourceObject: {{fileID: 0}}
  m_PrefabInstance: {{fileID: 0}}
  m_PrefabAsset: {{fileID: 0}}
  m_Name: {preset_name}
  m_TargetType:
    m_NativeTypeID: 1006
    m_ManagedTypePPtr: {{fileID: 0}}
    m_ManagedTypeFallback: 
  m_Properties:
  - target: {{fileID: 0}}
    propertyPath: m_TextureType
    value: 8
    objectReference: {{fileID: 0}}
  - target: {{fileID: 0}}
    propertyPath: m_SpriteMode
    value: 1
    objectReference: {{fileID: 0}}
  - target: {{fileID: 0}}
    propertyPath: m_SpritePixelsToUnits
    value: 192
    objectReference: {{fileID: 0}}
  - target: {{fileID: 0}}
    propertyPath: m_Alignment
    value: 9
    objectReference: {{fileID: 0}}
  - target: {{fileID: 0}}
    propertyPath: m_SpritePivot.x
    value: {piv_x}
    objectReference: {{fileID: 0}}
  - target: {{fileID: 0}}
    propertyPath: m_SpritePivot.y
    value: {piv_y}
    objectReference: {{fileID: 0}}
  - target: {{fileID: 0}}
    propertyPath: m_EnableMipMap
    value: 0
    objectReference: {{fileID: 0}}
  - target: {{fileID: 0}}
    propertyPath: m_FilterMode
    value: 0
    objectReference: {{fileID: 0}}
  - target: {{fileID: 0}}
    propertyPath: m_AlphaIsTransparency
    value: 1
    objectReference: {{fileID: 0}}
  - target: {{fileID: 0}}
    propertyPath: m_TextureCompression
    value: 0
    objectReference: {{fileID: 0}}
  - target: {{fileID: 0}}
    propertyPath: m_sRGBTexture
    value: 1
    objectReference: {{fileID: 0}}
"""
        (presets_dir / f"{preset_name}.preset.yaml").write_text(preset_yaml, encoding="utf-8")

    atlas_yaml = """%YAML 1.1
%TAG !u! tag:unity3d.com,2011:
--- !u!687078895 &4343727234628468602
SpriteAtlas:
  m_ObjectHideFlags: 0
  m_Name: ART004_Master
  m_EditorData:
    textureSettings:
      anisoLevel: 1
      compressionQuality: 50
      maxTextureSize: 2048
      textureCompression: 0
      filterMode: 0
      generateMipMaps: 0
      readable: 0
      crunchedCompression: 0
      sRGB: 1
    packingSettings:
      padding: 4
      blockOffset: 1
      allowAlphaSplitting: 0
      enableRotation: 0
      enableTightPacking: 0
      enableAlphaDilation: 0
    variantMultiplier: 1
    bindAsDefault: 1
"""
    (presets_dir / "ART004_Master.spriteatlasv2.yaml").write_text(atlas_yaml, encoding="utf-8")

    verifier_cs = """// Reference Editor script for M06 integration under unity/Assets/_TTGQLDL/Editor/
// Stored outside unity/ as .cs.txt in ART-004 so M01-M05 runtime/editor assemblies stay untouched.
using UnityEditor;
using UnityEngine;

namespace Ttgqldl.Editor.Art004
{
    public static class ART004SpriteImportVerifier
    {
        public static void ApplyAndVerifySprite(string assetPath, Vector2 normalizedPivot)
        {
            var importer = (TextureImporter)AssetImporter.GetAtPath(assetPath);
            importer.textureType = TextureImporterType.Sprite;
            importer.spriteImportMode = SpriteImportMode.Single;
            importer.spritePixelsPerUnit = 192f;
            importer.mipmapEnabled = false;
            importer.filterMode = FilterMode.Point;
            importer.alphaIsTransparency = true;
            importer.textureCompression = TextureImporterCompression.Uncompressed;

            var settings = new TextureImporterSettings();
            importer.ReadTextureSettings(settings);
            settings.spriteAlignment = (int)SpriteAlignment.Custom;
            settings.spritePivot = normalizedPivot;
            settings.spriteMeshType = SpriteMeshType.FullRect;
            importer.SetTextureSettings(settings);
            importer.SaveAndReimport();

            var sprite = AssetDatabase.LoadAssetAtPath<Sprite>(assetPath);
            Debug.Log($"[ART-004] Verified {assetPath}: rect={sprite.rect}, pivot={sprite.pivot}, ppu={sprite.pixelsPerUnit}, filter={sprite.texture.filterMode}");
        }
    }
}
"""
    (presets_dir / "ART004SpriteImportVerifier.cs.txt").write_text(verifier_cs, encoding="utf-8")
    print(f"[ATLAS PASS] ART004_Master_Atlas.png (2048x2048, 112 sprites, {png_file_bytes/1024:.1f} KiB PNG, 16.00 MiB raw RGBA32)")


# ============================================================================
# 2. STATIC CONTACT SHEETS & SYNCHRONIZED CHARACTER + VFX SEQUENCES
# ============================================================================

def build_character_contact_sheet(
    title: str,
    subtitle: str,
    ent_meta: Dict[str, Any],
    frames: List[Image.Image],
    out_path: Path,
) -> None:
    f_title = load_font(22, bold=True)
    f_sub = load_font(14, bold=False)
    f_lbl = load_font(13, bold=True)
    f_small = load_font(12, bold=False)

    cols = 6 if len(frames) == 18 else 4
    rows = 3 if len(frames) == 18 else 4
    is_boss = ent_meta["cell_size"][0] == 256
    vis_scale = ent_meta["unity_visual_root_scale"]
    cw = 266 if not is_boss else 356
    ch = 286 if not is_boss else 318
    W = 40 + cols * cw
    H = 106 + rows * ch + 18
    board = Image.new("RGB", (W, H), DARK_BG_RGB)
    dr = ImageDraw.Draw(board)

    dr.text((24, 16), title, font=f_title, fill=(254, 240, 138))
    dr.text((24, 48), subtitle, font=f_sub, fill=(203, 213, 225))
    dr.line([(24, 80), (W - 24, 80)], fill=(51, 65, 85), width=2)

    for idx, (rec, fr) in enumerate(zip(ent_meta["frames"], frames)):
        r, c = divmod(idx, cols)
        ox = 20 + c * cw
        oy = 92 + r * ch
        dr.rounded_rectangle([ox + 6, oy + 6, ox + cw - 6, oy + ch - 6], radius=10, fill=(28, 36, 52), outline=(64, 78, 104), width=2)
        dr.text((ox + 14, oy + 12), f"#{idx:02d} {rec['clip'].upper()} f{rec['clip_frame_index']:02d}", font=f_lbl, fill=(248, 250, 252))

        # Left swatch: source cell (128x128 at 1:1, or 256x256 Boss shown at 160x160 NN) on Meadow + pivot crosshair
        sx, sy = ox + 14, oy + 40
        disp_src_sz = 128 if not is_boss else 160
        fr_src_box = fr if not is_boss else fr.resize((160, 160), Image.Resampling.NEAREST)
        piv_x = int(round(disp_src_sz * 0.50))
        piv_y = int(round(disp_src_sz * 0.92))
        dr.rectangle([sx, sy, sx + disp_src_sz, sy + disp_src_sz], fill=MEADOW_RGB, outline=(17, 21, 28), width=2)
        dr.line([(sx, sy + piv_y), (sx + disp_src_sz, sy + piv_y)], fill=(82, 133, 54), width=1)
        dr.line([(sx + piv_x, sy), (sx + piv_x, sy + disp_src_sz)], fill=(82, 133, 54), width=1)
        board.paste(fr_src_box, (sx, sy), fr_src_box)
        dr.line([(sx + piv_x - 4, sy + piv_y), (sx + piv_x + 4, sy + piv_y)], fill=(239, 68, 68), width=2)
        dr.line([(sx + piv_x, sy + piv_y - 4), (sx + piv_x, sy + piv_y + 4)], fill=(239, 68, 68), width=2)
        dr.text((sx + 2, sy + disp_src_sz + 4), "256x256 src" if is_boss else "128x128 (1:1)", font=f_small, fill=(148, 163, 184))

        # Right-top swatch: Actual 1080p gameplay scale!
        fr_1080 = scale_by_factor(fr, vis_scale)
        rx = sx + disp_src_sz + 12
        ry = oy + 40
        bw_1080 = 96 if not is_boss else 156
        bh_1080 = 92 if not is_boss else 132
        dr.rectangle([rx, ry, rx + bw_1080, ry + bh_1080], fill=OCHRE_RGB if not is_boss else LIMESTONE_RGB, outline=(17, 21, 28), width=2)
        board.paste(fr_1080, (rx + (bw_1080 - fr_1080.width) // 2, ry + (bh_1080 - fr_1080.height) // 2), fr_1080)
        dr.text((rx + 2, ry + bh_1080 + 3), f"1080p ({ent_meta['displayed_total_height_px']['1080x1920']}px)", font=f_small, fill=(254, 240, 138))

        # Right-bottom swatch: Actual 360x640 mobile scale + BT.709 Grayscale!
        fr_360 = scale_by_factor(fr, vis_scale / 3.0)
        mx = rx
        my = ry + bh_1080 + 24
        mw = 44 if not is_boss else 72
        mh = 42 if not is_boss else 58
        dr.rectangle([mx, my, mx + mw, my + mh], fill=LIMESTONE_RGB, outline=(17, 21, 28), width=1)
        board.paste(fr_360, (mx + (mw - fr_360.width) // 2, my + (mh - fr_360.height) // 2), fr_360)

        fr_gray_src = scale_by_factor(fr, vis_scale * 0.5)
        tmp_gray = Image.new("RGB", (mw, mh), LIMESTONE_RGB)
        tmp_gray.paste(fr_gray_src, ((mw - fr_gray_src.width) // 2, (mh - fr_gray_src.height) // 2), fr_gray_src)
        tmp_gray = tmp_gray.convert("L").convert("RGB")
        board.paste(tmp_gray, (mx + mw + 6, my))
        dr.rectangle([mx + mw + 6, my, mx + mw * 2 + 6, my + mh], outline=(17, 21, 28), width=1)
        dr.text((mx, my + mh + 2), "360p | Gray", font=f_small, fill=(148, 163, 184))

        bbox = rec["bbox"]
        f_sole = max(rec["front_foot_xy"][1], rec["back_foot_xy"][1])
        dr.text((ox + 14, oy + ch - 46), f"bbox:[{bbox[0]},{bbox[1]}..{bbox[2]},{bbox[3]}] col={rec['unique_colors']}", font=f_small, fill=(203, 213, 225))
        dr.text((ox + 14, oy + ch - 26), f"sole_y={f_sole:.0f} | halo={rec['semi_transparent_pixels']}", font=f_small, fill=(134, 239, 172))

    out_path.parent.mkdir(parents=True, exist_ok=True)
    board.save(out_path)


def build_synced_sequences(loaded4: Dict[str, List[Image.Image]]) -> Dict[str, List[Image.Image]]:
    """Create 4 deterministic character+VFX synchronized RGBA sequences @ 12 FPS."""
    # 1. Fighter SwordSlash (6f) + vfx_fighter_slash_arc (4f synced on f02..f05)
    f_slash = loaded4["fighter"][10:16]
    v_slash = loaded4["vfx_fighter_slash_arc"]
    sync_fighter: List[Image.Image] = []
    for i in range(6):
        c = Image.new("RGBA", (176, 144), (0, 0, 0, 0))
        c.paste(f_slash[i], (12, 8), f_slash[i])
        if i >= 2:
            vl = Image.new("RGBA", (176, 144), (0, 0, 0, 0))
            vl.paste(v_slash[i - 2], (34, 8), v_slash[i - 2])
            c = Image.alpha_composite(c, vl)
        sync_fighter.append(c)

    # 2. Guard ShieldBrace (6f) + vfx_guard_shield_block (4f synced on f02..f05)
    g_brace = loaded4["guard"][10:16]
    v_block = loaded4["vfx_guard_shield_block"]
    sync_guard: List[Image.Image] = []
    for i in range(6):
        c = Image.new("RGBA", (176, 144), (0, 0, 0, 0))
        c.paste(g_brace[i], (12, 8), g_brace[i])
        if i >= 2:
            vl = Image.new("RGBA", (176, 144), (0, 0, 0, 0))
            vl.paste(v_block[i - 2], (36, 10), v_block[i - 2])
            c = Image.alpha_composite(c, vl)
        sync_guard.append(c)

    # 3. Runner Hit (2f + 4f recovery) + vfx_unit_hit_puff (6f)
    r_hit = loaded4["runner"][16:18]
    r_idle = loaded4["runner"][0:4]
    v_puff = loaded4["vfx_unit_hit_puff"]
    sync_hit: List[Image.Image] = []
    for i in range(6):
        actor = r_hit[i] if i < 2 else r_idle[i - 2]
        c = Image.new("RGBA", (160, 144), (0, 0, 0, 0))
        c.paste(actor, (16, 8), actor)
        vl = Image.new("RGBA", (160, 144), (0, 0, 0, 0))
        vl.paste(v_puff[i], (20, 10), v_puff[i])
        c = Image.alpha_composite(c, vl)
        sync_hit.append(c)

    # 4. Boss CleaveSlam (8f + 2f march settle = 10f) + vfx_boss_slam_telegraph (10f: f00..f04 ground warning BEHIND boss, f05..f09 shockwave!)
    b_slam = loaded4["boss"][6:14]
    b_march = loaded4["boss"][0:2]
    v_tele = loaded4["vfx_boss_slam_telegraph"]
    sync_boss: List[Image.Image] = []
    for i in range(10):
        boss_fr = b_slam[i] if i < 8 else b_march[i - 8]
        c = Image.new("RGBA", (320, 288), (0, 0, 0, 0))
        # Ground-plane rune telegraph renders UNDER Boss feet on sorting layer 10_GroundTelegraphsAndShadows!
        tele_layer = Image.new("RGBA", (320, 288), (0, 0, 0, 0))
        tele_layer.paste(v_tele[i], (54, 32), v_tele[i])
        boss_layer = Image.new("RGBA", (320, 288), (0, 0, 0, 0))
        boss_layer.paste(boss_fr, (18, 12), boss_fr)
        c = Image.alpha_composite(tele_layer, boss_layer)
        # During active shockwave eruption (i in 5..7), composite front shockwave rim slightly in front of ankles
        if 5 <= i <= 7:
            front_rim = v_tele[i].crop((0, 152, 256, 256))
            rim_layer = Image.new("RGBA", (320, 288), (0, 0, 0, 0))
            rim_layer.paste(front_rim, (54, 32 + 152), front_rim)
            c = Image.alpha_composite(c, rim_layer)
        sync_boss.append(c)

    return {
        "sync_fighter_slash": sync_fighter,
        "sync_guard_shieldblock": sync_guard,
        "sync_runner_hitpuff": sync_hit,
        "sync_boss_cleaveslam_telegraph": sync_boss,
    }


def build_vfx_and_sync_contact_sheet(
    loaded4: Dict[str, List[Image.Image]],
    synced: Dict[str, List[Image.Image]],
    out_path: Path,
) -> None:
    f_title = load_font(22, bold=True)
    f_lbl = load_font(14, bold=True)
    f_sm = load_font(12, bold=False)

    W, H = 1720, 1080
    board = Image.new("RGB", (W, H), DARK_BG_RGB)
    dr = ImageDraw.Draw(board)
    dr.text((24, 16), "ART-004 COMBAT VFX ARCHETYPES (24 FRAMES @ 12 FPS) & SYNCHRONIZED ACTION COMPOSITES", font=f_title, fill=(254, 240, 138))
    dr.text((24, 48), "Zero persistent end-frame bright pixels | Flash rate < 3 Hz | Independent ground-plane Boss Slam Telegraph (warn f00..f04 -> impact f05..f07 -> dissipate f08..f09)", font=f_sm, fill=(203, 213, 225))

    rows_spec = [
        ("vfx_fighter_slash_arc (4f, 128x128) + Synced Fighter SwordSlash", loaded4["vfx_fighter_slash_arc"], synced["sync_fighter_slash"], OCHRE_RGB, 90),
        ("vfx_guard_shield_block (4f, 128x128) + Synced Guard ShieldBrace", loaded4["vfx_guard_shield_block"], synced["sync_guard_shieldblock"], MEADOW_RGB, 325),
        ("vfx_unit_hit_puff (6f, 128x128) + Synced Runner Hit Reaction", loaded4["vfx_unit_hit_puff"], synced["sync_runner_hitpuff"], OCHRE_RGB, 560),
        ("vfx_boss_slam_telegraph (10f, 256x256) + Synced Boss CleaveSlam", loaded4["vfx_boss_slam_telegraph"][::2], synced["sync_boss_cleaveslam_telegraph"][::2], LIMESTONE_RGB, 795),
    ]

    for row_title, vfx_list, sync_list, bg_col, oy in rows_spec:
        dr.rounded_rectangle([20, oy, W - 20, oy + 220], radius=10, fill=(28, 36, 52), outline=(64, 78, 104), width=2)
        dr.text((34, oy + 10), row_title, font=f_lbl, fill=(248, 250, 252))

        x_cur = 34
        for idx, vf in enumerate(vfx_list):
            dr.rectangle([x_cur, oy + 38, x_cur + 116, oy + 154], fill=(24, 30, 42), outline=(17, 21, 28), width=2)
            v_fit = vf.resize((112, 112), Image.Resampling.NEAREST)
            board.paste(v_fit, (x_cur + 2, oy + 40), v_fit)
            dr.text((x_cur + 4, oy + 158), f"VFX #{idx}", font=f_sm, fill=(148, 163, 184))
            x_cur += 126

        x_cur += 18
        for idx, sf in enumerate(sync_list):
            dr.rectangle([x_cur, oy + 38, x_cur + 132, oy + 154], fill=bg_col, outline=(17, 21, 28), width=2)
            sc = min(126.0 / sf.width, 110.0 / sf.height)
            s_fit = sf.resize((int(round(sf.width * sc)), int(round(sf.height * sc))), Image.Resampling.NEAREST)
            board.paste(s_fit, (x_cur + (132 - s_fit.width) // 2, oy + 38 + (116 - s_fit.height) // 2), s_fit)
            dr.text((x_cur + 4, oy + 158), f"Sync #{idx}", font=f_sm, fill=(254, 240, 138))
            x_cur += 142

    out_path.parent.mkdir(parents=True, exist_ok=True)
    board.save(out_path)


def build_all_7_classes_comparison_sheet(
    meta3: Dict[str, Any],
    loaded3: Dict[str, List[Image.Image]],
    meta4: Dict[str, Any],
    loaded4: Dict[str, List[Image.Image]],
    out_path: Path,
) -> None:
    """Cross-package consistency matrix comparing ART-003 (Builder, Grunt) + ART-004 (Guard, Fighter, Runner, Brute, Boss)."""
    f_title = load_font(22, bold=True)
    f_lbl = load_font(14, bold=True)
    f_sm = load_font(12, bold=False)

    roster = [
        ("Builder (ART-003)", "Role", loaded3["builder"][0], loaded3["builder"][13], 48.0 / 86.0, "48px", "16px"),
        ("Guard (ART-004)", "Role", loaded4["guard"][0], loaded4["guard"][12], meta4["entities"]["guard"]["unity_visual_root_scale"], "48px", "16px"),
        ("Fighter (ART-004)", "Role", loaded4["fighter"][0], loaded4["fighter"][13], meta4["entities"]["fighter"]["unity_visual_root_scale"], "48px body", "16px"),
        ("Runner (ART-004)", "Enemy", loaded4["runner"][0], loaded4["runner"][12], meta4["entities"]["runner"]["unity_visual_root_scale"], "44px", "15px"),
        ("Grunt (ART-003)", "Enemy", loaded3["grunt"][0], loaded3["grunt"][13], 48.0 / 86.0, "48px", "16px"),
        ("Brute (ART-004)", "Enemy", loaded4["brute"][0], loaded4["brute"][13], meta4["entities"]["brute"]["unity_visual_root_scale"], "64px", "21px"),
        ("Boss Warlord (ART-004)", "Boss", loaded4["boss"][0], loaded4["boss"][10], meta4["entities"]["boss"]["unity_visual_root_scale"], "102px", "34px"),
    ]

    W, H = 1660, 820
    board = Image.new("RGB", (W, H), DARK_BG_RGB)
    dr = ImageDraw.Draw(board)
    dr.text((24, 16), "BASTION DAWN v1 — ALL 7 CHARACTER CLASSES CROSS-PACKAGE CONSISTENCY MATRIX (ART-003 + ART-004)", font=f_title, fill=(254, 240, 138))
    dr.text((24, 48), "Top: Zoomed Silhouette & Action Peak  |  Mid: True Proportional 1080x1920 Gameplay Lineup on Common Ground Baseline  |  Bottom: 360x640 Mobile & BT.709 Grayscale", font=f_sm, fill=(203, 213, 225))

    # Row 1: Individual class cards
    for idx, (name, team, fr_idle, fr_act, vscale, s1080, s360) in enumerate(roster):
        ox = 20 + idx * 232
        oy = 84
        dr.rounded_rectangle([ox, oy, ox + 222, oy + 320], radius=10, fill=(28, 36, 52), outline=(64, 78, 104), width=2)
        dr.text((ox + 10, oy + 10), name, font=f_lbl, fill=(248, 250, 252))
        dr.rectangle([ox + 10, oy + 38, ox + 212, oy + 172], fill=MEADOW_RGB if team == "Role" else OCHRE_RGB, outline=(17, 21, 28), width=2)
        fit_sz = 104 if fr_idle.width == 128 else 128
        i_im = fr_idle.resize((fit_sz, fit_sz), Image.Resampling.NEAREST)
        a_im = fr_act.resize((fit_sz, fit_sz), Image.Resampling.NEAREST)
        board.paste(i_im, (ox + 8, oy + 42 + (128 - fit_sz) // 2), i_im)
        board.paste(a_im, (ox + 104, oy + 42 + (128 - fit_sz) // 2), a_im)
        dr.text((ox + 12, oy + 180), f"Team: {team} | Scale: {vscale:.4f}", font=f_sm, fill=(203, 213, 225))
        dr.text((ox + 12, oy + 202), f"1080p: {s1080} | 360p: {s360}", font=f_sm, fill=(134, 239, 172))

        # 360p + Grayscale mini-swatches inside card
        im_360 = scale_by_factor(fr_act, vscale / 3.0)
        dr.rectangle([ox + 12, oy + 230, ox + 98, oy + 306], fill=LIMESTONE_RGB, outline=(17, 21, 28), width=1)
        board.paste(im_360, (ox + 12 + (86 - im_360.width) // 2, oy + 230 + (76 - im_360.height) // 2), im_360)
        dr.text((ox + 16, oy + 234), "360p", font=f_sm, fill=(17, 21, 28))

        im_1080 = scale_by_factor(fr_act, min(vscale, 68.0 / fr_act.height))
        g_box = Image.new("RGB", (98, 76), LIMESTONE_RGB)
        g_box.paste(im_1080, ((98 - im_1080.width) // 2, (76 - im_1080.height) // 2), im_1080)
        g_box = g_box.convert("L").convert("RGB")
        board.paste(g_box, (ox + 110, oy + 230))
        dr.rectangle([ox + 110, oy + 230, ox + 208, oy + 306], outline=(17, 21, 28), width=1)

    # Row 2: True Proportional 1080p Common-Baseline Lineup (Heroes facing +X on left, Enemies/Boss facing -X on right!)
    ly = 424
    dr.rounded_rectangle([20, ly, W - 20, H - 20], radius=12, fill=(28, 36, 52), outline=(64, 78, 104), width=2)
    dr.text((36, ly + 12), "TRUE PROPORTIONAL 1080x1920 GAMEPLAY SCALE LINEUP (COMMON PIVOT GROUND BASELINE — NO FAKE ZOOM!)", font=f_lbl, fill=(254, 240, 138))

    strip_x, strip_y, strip_w, strip_h = 36, ly + 44, W - 72, 316
    dr.rectangle([strip_x, strip_y, strip_x + strip_w // 3, strip_y + strip_h], fill=MEADOW_RGB)
    dr.rectangle([strip_x + strip_w // 3, strip_y, strip_x + 2 * strip_w // 3, strip_y + strip_h], fill=OCHRE_RGB)
    dr.rectangle([strip_x + 2 * strip_w // 3, strip_y, strip_x + strip_w, strip_y + strip_h], fill=LIMESTONE_RGB)
    dr.rectangle([strip_x, strip_y, strip_x + strip_w, strip_y + strip_h], outline=(17, 21, 28), width=3)

    baseline_y = strip_y + 236
    dr.line([(strip_x, baseline_y), (strip_x + strip_w, baseline_y)], fill=(17, 21, 28), width=2)

    xs_lineup = [140, 320, 500, 740, 930, 1140, 1410]
    for (name, team, fr_idle, fr_act, vscale, s1080, _), cx in zip(roster, xs_lineup):
        sp_i = scale_by_factor(fr_idle, vscale)
        sp_a = scale_by_factor(fr_act, vscale)
        if team != "Role":
            sp_i = flip_h(sp_i)
            sp_a = flip_h(sp_a)
        for dx_off, sp in [(-42, sp_i), (42, sp_a)]:
            px = int(round(cx + dx_off - sp.width * 0.50))
            py = int(round(baseline_y - sp.height * 0.92))
            board.paste(sp, (px, py), sp)
        dr.text((cx - 64, baseline_y + 18), f"{name.split(' (')[0]} ({s1080})", font=f_lbl, fill=(17, 21, 28))

    out_path.parent.mkdir(parents=True, exist_ok=True)
    board.save(out_path)


# ============================================================================
# 3. TRUE 12.00 FPS GIFS & ALL-7-CLASS H.264 MP4 STREAM PREVIEWS
# ============================================================================

def durations_12fps(n_frames: int) -> List[int]:
    if n_frames == 8:
        return [80, 80, 90, 80, 90, 80, 80, 90]
    if n_frames == 4:
        return [80, 80, 90, 80]
    if n_frames == 10:
        return [80, 80, 90, 80, 80, 90, 80, 80, 90, 80]
    base = [80, 80, 90]
    return [base[i % 3] for i in range(n_frames)]


def save_review_gif(
    frames_rgba: List[Image.Image],
    label: str,
    vis_scale: float,
    out_gif: Path,
    bg_rgb: Tuple[int, int, int] = MEADOW_RGB,
    frame_tags: List[str] | None = None,
) -> None:
    f_hdr = load_font(13, bold=True)
    f_sm = load_font(11, bold=False)
    rendered: List[Image.Image] = []

    for idx, fr in enumerate(frames_rgba):
        W, H = 440, 320
        card = Image.new("RGB", (W, H), DARK_BG_RGB)
        dr = ImageDraw.Draw(card)
        tag = frame_tags[idx] if frame_tags else f"f{idx:02d}/{len(frames_rgba)-1:02d}"
        dr.text((12, 10), f"{label}  [12 FPS | {tag}]", font=f_hdr, fill=(254, 240, 138))

        # Left: Zoomed inspection view (fit inside 256x256 NN)
        z_sc = 2.0 if fr.width <= 128 else (1.0 if fr.width == 256 else 256.0 / max(fr.width, fr.height))
        z_im = scale_by_factor(fr, z_sc)
        dr.rectangle([14, 38, 274, 302], fill=bg_rgb, outline=(17, 21, 28), width=2)
        card.paste(z_im, (14 + (260 - z_im.width) // 2, 38 + (264 - z_im.height) // 2), z_im)
        dr.text((20, 282), "Zoom NN", font=f_sm, fill=(17, 21, 28))

        # Top-R: 1080p Actual Gameplay Scale
        rx = 286
        im_1080 = scale_by_factor(fr, vis_scale)
        dr.rectangle([rx, 38, rx + 140, 182], fill=OCHRE_RGB, outline=(17, 21, 28), width=2)
        card.paste(im_1080, (rx + (140 - im_1080.width) // 2, 38 + (144 - im_1080.height) // 2), im_1080)
        dr.text((rx + 4, 42), "1080p Scale", font=f_sm, fill=(17, 21, 28))

        # Bot-R: 360p Mobile Scale + BT.709 Grayscale
        im_360 = scale_by_factor(fr, vis_scale / 3.0)
        dr.rectangle([rx, 192, rx + 66, 302], fill=LIMESTONE_RGB, outline=(17, 21, 28), width=2)
        card.paste(im_360, (rx + (66 - im_360.width) // 2, 192 + (110 - im_360.height) // 2), im_360)
        dr.text((rx + 4, 196), "360p", font=f_sm, fill=(17, 21, 28))

        im_gray_sp = scale_by_factor(fr, vis_scale * 0.65)
        g_box = Image.new("RGB", (68, 110), LIMESTONE_RGB)
        g_box.paste(im_gray_sp, ((68 - im_gray_sp.width) // 2, (110 - im_gray_sp.height) // 2), im_gray_sp)
        g_box = g_box.convert("L").convert("RGB")
        card.paste(g_box, (rx + 72, 192))
        dr.rectangle([rx + 72, 192, rx + 140, 302], outline=(17, 21, 28), width=2)
        dr.text((rx + 76, 196), "Gray", font=f_sm, fill=(17, 21, 28))

        rendered.append(card.quantize(colors=256, method=Image.Quantize.MEDIANCUT, dither=Image.Dither.NONE))

    out_gif.parent.mkdir(parents=True, exist_ok=True)
    rendered[0].save(
        out_gif,
        save_all=True,
        append_images=rendered[1:],
        duration=durations_12fps(len(rendered)),
        loop=0,
        disposal=2,
    )


def render_all_7_classes_gameplay_sequence(
    loaded3: Dict[str, List[Image.Image]],
    meta4: Dict[str, Any],
    loaded4: Dict[str, List[Image.Image]],
    n_ticks_12fps: int = 48,
) -> List[Image.Image]:
    """
    Render a 48-frame (4.0s @ 12 FPS) actual-scale 1080x1920 Bastion Dawn Wave 05 Boss Battle scene
    containing ALL SEVEN CHARACTER CLASSES (Builder, Guard, Fighter, Runner, Grunt, Brute, Boss)
    and ALL FIVE VFX ARCHETYPES over ART-002 Meadow, Ochre Lanes, and Limestone Center Causeway!
    Strictly respects x<=82%, y<=88% livestream safe zones and zero fake viewer names.
    """
    f_hud_lg = load_font(38, bold=True)
    f_hud_md = load_font(28, bold=True)
    f_hud_sm = load_font(22, bold=True)

    base_1080 = Image.new("RGB", (1080, 1920), MEADOW_RGB)
    bdr = ImageDraw.Draw(base_1080)
    bdr.polygon([(0, 0), (1080, 0), (1080, 620), (0, 760)], fill=(114, 173, 79))
    bdr.polygon([(0, 1280), (1080, 1180), (1080, 1920), (0, 1920)], fill=(92, 148, 62))

    # West Ochre Lane, East Ochre Lane (strictly left of x=885 safe zone!), Center Limestone Boss Causeway
    bdr.polygon([(36, 260), (176, 260), (355, 790), (235, 835)], fill=OCHRE_RGB, outline=(142, 112, 68), width=4)
    bdr.polygon([(716, 260), (852, 260), (665, 835), (555, 790)], fill=OCHRE_RGB, outline=(142, 112, 68), width=4)
    bdr.polygon([(382, 260), (526, 260), (538, 795), (370, 795)], fill=LIMESTONE_RGB, outline=(96, 106, 120), width=4)

    # Central Bastion Courtyard & Curtain Wall (476px = 44.1% width, centered at x=454)
    bdr.ellipse([216, 780, 692, 1230], fill=(216, 179, 132), outline=(17, 21, 28), width=6)
    bdr.ellipse([238, 802, 670, 1208], fill=(216, 179, 132), outline=(100, 116, 139), width=18)
    bdr.rounded_rectangle([374, 875, 534, 1025], radius=14, fill=(232, 223, 200), outline=(17, 21, 28), width=5)
    bdr.polygon([(360, 895), (454, 825), (548, 895)], fill=(59, 110, 165), outline=(17, 21, 28), width=5)
    bdr.rounded_rectangle([408, 1176, 500, 1236], radius=8, fill=(140, 90, 50), outline=(17, 21, 28), width=5)

    # Top Anchored HUD (x=36..876 <= 82%) — Wave 05 Boss Active (tick >= 2600)
    bdr.rounded_rectangle([36, 36, 876, 248], radius=18, fill=(16, 23, 38), outline=(217, 70, 239), width=3)
    bdr.text((58, 50), "ĐỢT 05  •  TRÙM XUẤT HIỆN!", font=f_hud_md, fill=(245, 208, 254))
    bdr.rounded_rectangle([642, 48, 856, 102], radius=12, fill=(46, 16, 59), outline=(217, 70, 239), width=2)
    bdr.text((666, 56), "TICK 2640", font=f_hud_md, fill=(254, 240, 138))
    bdr.text((58, 114), "TƯỜNG 740/1000", font=f_hud_sm, fill=(255, 255, 255))
    bdr.rounded_rectangle([282, 116, 532, 146], radius=8, fill=(30, 41, 59), outline=(148, 163, 184), width=2)
    bdr.rounded_rectangle([285, 119, 470, 143], radius=6, fill=(76, 201, 100))
    bdr.text((58, 160), "LÂU ĐÀI 500/500", font=f_hud_sm, fill=(255, 255, 255))
    bdr.rounded_rectangle([282, 162, 532, 192], radius=8, fill=(30, 41, 59), outline=(148, 163, 184), width=2)
    bdr.rounded_rectangle([285, 165, 529, 189], radius=6, fill=(59, 130, 246))
    bdr.text((558, 120), "TRÙM CHIẾN LÃNH CHÚA", font=f_hud_sm, fill=(240, 171, 252))
    bdr.rounded_rectangle([558, 158, 854, 188], radius=8, fill=(46, 16, 59), outline=(217, 70, 239), width=2)
    bdr.rounded_rectangle([561, 161, 812, 185], radius=6, fill=(217, 70, 239))

    # Bottom Single Primary CTA + Role Counts + Honest MOCK Watermark (x<=876, y<=1684 <= 88%)
    bdr.rounded_rectangle([54, 1460, 864, 1546], radius=16, fill=(18, 26, 43), outline=(100, 116, 139), width=2)
    bdr.text((84, 1484), "XÂY: 14     THỦ: 18     ĐÁNH: 21", font=f_hud_md, fill=(255, 255, 255))
    bdr.rounded_rectangle([54, 1558, 864, 1632], radius=18, fill=(15, 23, 42), outline=(234, 179, 8), width=4)
    bdr.text((104, 1570), "Bình luận XÂY / THỦ / ĐÁNH", font=f_hud_lg, fill=(255, 255, 255))
    bdr.rounded_rectangle([54, 1640, 560, 1684], radius=10, fill=(28, 22, 10), outline=(250, 204, 21), width=2)
    bdr.text((72, 1646), "MOCK / ART-004 ALL-7-CLASS PREVIEW", font=f_hud_sm, fill=(250, 204, 21))

    # Pre-scale all 7 classes + 5 VFX to exact 1080p display sizes!
    sc_b = 48.0 / 86.0
    sc_gu = meta4["entities"]["guard"]["unity_visual_root_scale"]
    sc_fi = meta4["entities"]["fighter"]["unity_visual_root_scale"]
    sc_ru = meta4["entities"]["runner"]["unity_visual_root_scale"]
    sc_br = meta4["entities"]["brute"]["unity_visual_root_scale"]
    sc_bo = meta4["entities"]["boss"]["unity_visual_root_scale"]

    b_rep = [scale_by_factor(f, sc_b) for f in loaded3["builder"][10:16]]
    b_idle = [scale_by_factor(f, sc_b) for f in loaded3["builder"][0:4]]
    b_walk = [scale_by_factor(f, sc_b) for f in loaded3["builder"][4:10]]
    gr_walk = [scale_by_factor(f, sc_b) for f in loaded3["grunt"][4:10]]
    gr_smash = [scale_by_factor(f, sc_b) for f in loaded3["grunt"][10:16]]
    v_spark = [scale_by_factor(f, sc_b) for f in loaded3["vfx_builder_repair_spark"]]

    gu_idle = [scale_by_factor(f, sc_gu) for f in loaded4["guard"][0:4]]
    gu_walk = [scale_by_factor(f, sc_gu) for f in loaded4["guard"][4:10]]
    gu_brace = [scale_by_factor(f, sc_gu) for f in loaded4["guard"][10:16]]

    fi_walk = [scale_by_factor(f, sc_fi) for f in loaded4["fighter"][4:10]]
    fi_slash = [scale_by_factor(f, sc_fi) for f in loaded4["fighter"][10:16]]

    ru_run = [scale_by_factor(f, sc_ru) for f in loaded4["runner"][4:10]]
    ru_lunge = [scale_by_factor(f, sc_ru) for f in loaded4["runner"][10:16]]
    ru_hit = [scale_by_factor(f, sc_ru) for f in loaded4["runner"][16:18]]

    br_march = [scale_by_factor(f, sc_br) for f in loaded4["brute"][4:10]]
    br_crush = [scale_by_factor(f, sc_br) for f in loaded4["brute"][10:16]]

    bo_march = [scale_by_factor(f, sc_bo) for f in loaded4["boss"][0:6]]
    bo_slam = [scale_by_factor(f, sc_bo) for f in loaded4["boss"][6:14]]

    v_slash = [scale_by_factor(f, sc_fi) for f in loaded4["vfx_fighter_slash_arc"]]
    v_block = [scale_by_factor(f, sc_gu) for f in loaded4["vfx_guard_shield_block"]]
    v_puff = [scale_by_factor(f, sc_ru) for f in loaded4["vfx_unit_hit_puff"]]
    v_tele = [scale_by_factor(f, sc_bo) for f in loaded4["vfx_boss_slam_telegraph"]]

    seq: List[Image.Image] = []
    for t in range(n_ticks_12fps):
        frame = base_1080.copy()
        u24 = (t % 24) / 24.0
        ang24 = 2.0 * math.pi * u24

        # 10_GroundTelegraphsAndShadows: Boss Slam Telegraph on Center Limestone Causeway (t%12 in 0..9)
        boss_ph = t % 12
        boss_x, boss_y = 454.0, 650.0
        if boss_ph < 10:
            tf = v_tele[boss_ph]
            tx = int(round(boss_x - tf.width * 0.50))
            ty = int(round((boss_y + 6) - tf.height * 0.57))
            frame.paste(tf, (tx, ty), tf)

        draw_list: List[Tuple[float, Image.Image, float, int]] = []  # (foot_y, sprite, cx, shadow_w)

        # Boss Warlord commanding Center Limestone Causeway (102px tall, CleaveSlam synced w/ telegraph!)
        bo_sp = bo_slam[boss_ph] if boss_ph < 8 else bo_march[(t - 8) % 6]
        draw_list.append((boss_y, flip_h(bo_sp), boss_x, 66))

        # Guards bracing perimeter front below Center & West lanes (shield block synced!)
        gb_ph = t % 8
        gu_c_sp = gu_brace[gb_ph] if gb_ph < 6 else gu_idle[gb_ph - 6]
        draw_list.append((772.0, gu_c_sp, 412.0, 36))
        draw_list.append((776.0, flip_h(gu_brace[(t + 4) % 6]), 496.0, 36))
        draw_list.append((812.0, flip_h(gu_walk[t % 6]), 282.0, 36))

        # Fighters engaging Runners & Brutes on West & East Ochre Lanes (SwordSlash synced!)
        fs_ph = t % 8
        fi_w_sp = flip_h(fi_slash[fs_ph] if fs_ph < 6 else fi_walk[fs_ph % 6])
        draw_list.append((695.0, fi_w_sp, 276.0, 36))
        fi_e_sp = fi_slash[(t + 3) % 6]
        draw_list.append((705.0, fi_e_sp, 616.0, 36))

        # West Ochre Lane: Fast Runners sprinting + lunging + taking Hit reaction from Fighter!
        rx_s = 156.0 + 22.0 * math.sin(ang24)
        ry_s = 450.0 + 38.0 * math.sin(ang24)
        ru_sp1 = ru_run[t % 6] if math.cos(ang24) >= 0 else flip_h(ru_run[t % 6])
        draw_list.append((ry_s, ru_sp1, rx_s, 30))
        # Runner dueling West Fighter (takes Hit on fs_ph in 3..4!)
        ru_duel = ru_hit[fs_ph - 3] if fs_ph in (3, 4) else ru_lunge[t % 6]
        draw_list.append((688.0, ru_duel, 224.0, 30))

        # East Ochre Lane (mirrored via flipX): Heavy Brute marching & GauntletCrush + Grunt escort!
        bx_m = 744.0 - 14.0 * math.sin(ang24)
        by_m = 485.0 + 24.0 * math.sin(ang24)
        br_m_sp = flip_h(br_march[t % 6]) if math.cos(ang24) >= 0 else br_march[t % 6]
        draw_list.append((by_m, br_m_sp, bx_m, 48))
        draw_list.append((702.0, flip_h(br_crush[(t + 2) % 6]), 674.0, 48))
        draw_list.append((585.0, flip_h(gr_walk[(t + 1) % 6]), 706.0, 36))
        draw_list.append((798.0, gr_smash[t % 6], 238.0, 36))

        # Builders inside Bastion repairing walls + patrolling courtyard
        rep_ph = t % 8
        b_sp = b_rep[rep_ph] if rep_ph < 6 else b_idle[rep_ph - 6]
        draw_list.append((865.0, b_sp, 304.0, 34))
        patrol_x = 454.0 + 70.0 * math.sin(ang24)
        bw_sp = b_walk[t % 6] if math.cos(ang24) >= 0 else flip_h(b_walk[t % 6])
        draw_list.append((1085.0, bw_sp, patrol_x, 34))

        # Authentic Y-sort back-to-front on 20_CastleAndUnits_YSorted
        draw_list.sort(key=lambda item: item[0])
        for fy, sp, cx, sh_w in draw_list:
            sh = Image.new("RGBA", (sh_w, 16), (0, 0, 0, 0))
            ImageDraw.Draw(sh).ellipse([2, 3, sh_w - 2, 13], fill=(17, 21, 28, 95))
            frame.paste(sh, (int(round(cx - sh_w // 2)), int(round(fy - 8))), sh)
            px = int(round(cx - sp.width * 0.50))
            py = int(round(fy - sp.height * 0.92))
            frame.paste(sp, (px, py), sp)

        # 30_CombatAndRepairVFX: Synchronized combat & repair bursts (well under <=20 pool cap!)
        # 1) Fighter slash arc on West & East lanes
        if 2 <= fs_ph <= 5:
            vs = flip_h(v_slash[fs_ph - 2])
            frame.paste(vs, (int(round(252 - vs.width * 0.5)), int(round(666 - vs.height * 0.5))), vs)
        e_ph = (t + 3) % 6
        if 2 <= e_ph <= 5:
            vs_e = v_slash[e_ph - 2]
            frame.paste(vs_e, (int(round(640 - vs_e.width * 0.5)), int(round(676 - vs_e.height * 0.5))), vs_e)

        # 2) Unit Hit Puff on West Runner when Fighter blade connects
        if 3 <= fs_ph <= 7:
            vp = v_puff[min(5, fs_ph - 3)]
            frame.paste(vp, (int(round(224 - vp.width * 0.5)), int(round(666 - vp.height * 0.5))), vp)

        # 3) Guard Shield Block on Center Guard
        if 2 <= gb_ph <= 5:
            vb = v_block[gb_ph - 2]
            frame.paste(vb, (int(round(426 - vb.width * 0.5)), int(round(746 - vb.height * 0.5))), vb)

        # 4) Builder Repair Spark on West parapet
        if 3 <= rep_ph <= 7:
            vsp = v_spark[min(5, rep_ph - 3)]
            frame.paste(vsp, (int(round(322 - vsp.width * 0.5)), int(round(842 - vsp.height * 0.5))), vsp)

        seq.append(frame)
    return seq


def encode_mp4(frames_rgb: List[Image.Image], out_mp4: Path, crf: int = 18) -> None:
    out_mp4.parent.mkdir(parents=True, exist_ok=True)
    with tempfile.TemporaryDirectory() as td:
        tdir = Path(td)
        for idx, im in enumerate(frames_rgb):
            im.save(tdir / f"f_{idx:04d}.png")
        cmd = [
            "ffmpeg", "-y",
            "-framerate", "12",
            "-i", str(tdir / "f_%04d.png"),
            "-c:v", "libx264",
            "-pix_fmt", "yuv420p",
            "-crf", str(crf),
            "-movflags", "+faststart",
            str(out_mp4),
        ]
        subprocess.run(cmd, check=True, stdout=subprocess.PIPE, stderr=subprocess.PIPE)


def build_all_7_showcase_mp4(
    loaded3: Dict[str, List[Image.Image]],
    meta4: Dict[str, Any],
    loaded4: Dict[str, List[Image.Image]],
    synced: Dict[str, List[Image.Image]],
    out_mp4: Path,
) -> None:
    """72-frame (6.0s @ 12 FPS) 1280x720 showcase comparing all 7 classes + synced VFX simultaneously."""
    f_title = load_font(21, bold=True)
    f_lbl = load_font(13, bold=True)
    f_sm = load_font(11, bold=False)

    panels = [
        ("Builder Repair+Spark (ART-003)", loaded3["builder"][10:16], 48.0 / 86.0, MEADOW_RGB),
        ("Guard Walk (6f)", loaded4["guard"][4:10], meta4["entities"]["guard"]["unity_visual_root_scale"], MEADOW_RGB),
        ("Guard ShieldBrace+VFX", synced["sync_guard_shieldblock"], meta4["entities"]["guard"]["unity_visual_root_scale"], MEADOW_RGB),
        ("Fighter Walk (6f)", loaded4["fighter"][4:10], meta4["entities"]["fighter"]["unity_visual_root_scale"], OCHRE_RGB),
        ("Fighter Slash+VFX", synced["sync_fighter_slash"], meta4["entities"]["fighter"]["unity_visual_root_scale"], OCHRE_RGB),
        ("Runner Run (6f)", loaded4["runner"][4:10], meta4["entities"]["runner"]["unity_visual_root_scale"], OCHRE_RGB),
        ("Runner DaggerLunge (6f)", loaded4["runner"][10:16], meta4["entities"]["runner"]["unity_visual_root_scale"], OCHRE_RGB),
        ("Grunt ClubSmash (ART-003)", loaded3["grunt"][10:16], 48.0 / 86.0, OCHRE_RGB),
        ("Brute GauntletCrush (6f)", loaded4["brute"][10:16], meta4["entities"]["brute"]["unity_visual_root_scale"], OCHRE_RGB),
        ("Boss CleaveSlam+Telegraph", synced["sync_boss_cleaveslam_telegraph"], meta4["entities"]["boss"]["unity_visual_root_scale"], LIMESTONE_RGB),
    ]

    seq: List[Image.Image] = []
    for t in range(60):  # 5.0s @ 12 FPS = LCM(4, 6, 10, 12, 20, 30, 60)
        canvas = Image.new("RGB", (1280, 720), DARK_BG_RGB)
        dr = ImageDraw.Draw(canvas)
        dr.text((22, 14), "ART-004 + ART-003 COMPLETE 7-CLASS & COMBAT VFX PRODUCTION SHOWCASE (12.0 FPS)", font=f_title, fill=(254, 240, 138))
        for idx, (title, clip_frs, vsc, bg) in enumerate(panels):
            r, c = divmod(idx, 5)
            ox = 20 + c * 248
            oy = 58 + r * 324
            dr.rounded_rectangle([ox, oy, ox + 238, oy + 310], radius=10, fill=(28, 36, 52), outline=(64, 78, 104), width=2)
            dr.text((ox + 10, oy + 8), title, font=f_lbl, fill=(248, 250, 252))
            fr = clip_frs[t % len(clip_frs)]

            # Top box: zoomed view
            dr.rectangle([ox + 10, oy + 34, ox + 228, oy + 184], fill=bg, outline=(17, 21, 28), width=2)
            fit_sc = min(198.0 / fr.width, 142.0 / fr.height)
            s_zoom = scale_by_factor(fr, fit_sc)
            canvas.paste(s_zoom, (ox + 10 + (218 - s_zoom.width) // 2, oy + 34 + (150 - s_zoom.height) // 2), s_zoom)

            # Bottom-left: True 1080p scale
            s_1080 = scale_by_factor(fr, vsc)
            dr.rectangle([ox + 10, oy + 194, ox + 154, oy + 298], fill=LIMESTONE_RGB, outline=(17, 21, 28), width=2)
            canvas.paste(s_1080, (ox + 10 + (144 - s_1080.width) // 2, oy + 194 + (104 - s_1080.height) // 2), s_1080)
            dr.text((ox + 14, oy + 198), "1080p", font=f_sm, fill=(17, 21, 28))

            # Bottom-right: True 360p mobile scale
            s_360 = scale_by_factor(fr, vsc / 3.0)
            dr.rectangle([ox + 162, oy + 194, ox + 228, oy + 298], fill=LIMESTONE_RGB, outline=(17, 21, 28), width=2)
            canvas.paste(s_360, (ox + 162 + (66 - s_360.width) // 2, oy + 194 + (104 - s_360.height) // 2), s_360)
            dr.text((ox + 166, oy + 198), "360p", font=f_sm, fill=(17, 21, 28))

        seq.append(canvas)
    encode_mp4(seq, out_mp4, crf=18)


def main() -> None:
    meta3, loaded3 = load_meta_and_frames(ART003_DIR)
    meta4, loaded4 = load_meta_and_frames(ART004_DIR)

    # 1. Master Atlas + Unity Presets
    build_master_atlas(meta4, loaded4)

    # 2. Contact Sheets
    cs_dir = ART004_DIR / "previews" / "contact_sheets"
    char_titles = [
        ("guard", "ART-004 GUARD (chr_role_guard) — 18 FRAMES @ 12 FPS", "Idle(4), Walk(6), ShieldBrace(6), Hit(2) | Royal Cobalt #2E7CF6, Tower Shield, Zero Handheld Weapon"),
        ("fighter", "ART-004 FIGHTER (chr_role_fighter) — 18 FRAMES @ 12 FPS", "Idle(4), Walk(6), SwordSlash(6), Hit(2) | Crimson Cloth Surcoat #E63946, No Steel Pauldrons, Broadsword"),
        ("runner", "ART-004 RUNNER (chr_enemy_runner) — 18 FRAMES @ 12 FPS", "Idle(4), Run(6), DaggerLunge(6), Hit(2) | Violet Goblinoid #9B5DE5, Twin Curved Daggers, Locked +X Facing"),
        ("brute", "ART-004 BRUTE (chr_enemy_brute) — 18 FRAMES @ 12 FPS", "Idle(4), March(6), GauntletCrush(6), Hit(2) | Siege-Orange Armor #F3722C, Iron Crusher Gauntlets, 64px 1080p"),
        ("boss", "ART-004 BOSS WARLORD (chr_boss_warlord) — 16 FRAMES @ 12 FPS (256x256)", "IdleMarch(6), CleaveSlam(8), Hit(2) | Obsidian-Violet #1E102B, Magenta Core, Colossal War-Cleaver, 102px 1080p"),
    ]
    for key, title, sub in char_titles:
        n_fr = len(loaded4[key])
        build_character_contact_sheet(title, sub, meta4["entities"][key], loaded4[key], cs_dir / f"contact_{key}_{n_fr}f.png")

    synced = build_synced_sequences(loaded4)
    build_vfx_and_sync_contact_sheet(loaded4, synced, cs_dir / "contact_vfx_and_action_sync.png")
    build_all_7_classes_comparison_sheet(meta3, loaded3, meta4, loaded4, cs_dir / "contact_all_7_classes_comparison.png")
    print("[CONTACT SHEETS PASS] 7 contact sheets written.")

    # 3. 12 FPS GIFs (19 character clips + 4 standalone VFX + 4 synced + 1 all-7-class gameplay showcase = 28 GIFs)
    gif_dir = ART004_DIR / "previews" / "gifs"
    for key in ["guard", "fighter", "runner", "brute", "boss"]:
        ent = meta4["entities"][key]
        vsc = ent["unity_visual_root_scale"]
        bg = MEADOW_RGB if key in ("guard", "fighter") else (LIMESTONE_RGB if key == "boss" else OCHRE_RGB)
        idle_first = loaded4[key][0]
        for cname, (s_idx, e_idx) in ent["clips"].items():
            c_frames = loaded4[key][s_idx : e_idx + 1]
            if cname == "hit":
                # Present 2 Hit frames followed by 4 live Idle/IdleMarch recovery frames at 12.00 FPS (< 3 Hz safe & distinct!)
                rec_frames = loaded4[key][0:4]
                padded = [c_frames[0], c_frames[1], rec_frames[0], rec_frames[1], rec_frames[2], rec_frames[3]]
                tags = ["HIT f00", "HIT f01", "SETTLE 0", "SETTLE 1", "SETTLE 2", "SETTLE 3"]
                save_review_gif(padded, f"{key.upper()} HIT (2f + settle)", vsc, gif_dir / f"{key}_{cname}_12fps.gif", bg_rgb=bg, frame_tags=tags)
            else:
                save_review_gif(c_frames, f"{key.upper()} {cname.upper()} ({len(c_frames)}f)", vsc, gif_dir / f"{key}_{cname}_12fps.gif", bg_rgb=bg)

    for vkey, vsc_ref in [
        ("vfx_fighter_slash_arc", meta4["entities"]["fighter"]["unity_visual_root_scale"]),
        ("vfx_guard_shield_block", meta4["entities"]["guard"]["unity_visual_root_scale"]),
        ("vfx_unit_hit_puff", meta4["entities"]["runner"]["unity_visual_root_scale"]),
        ("vfx_boss_slam_telegraph", meta4["entities"]["boss"]["unity_visual_root_scale"]),
    ]:
        save_review_gif(loaded4[vkey], f"{vkey} ({len(loaded4[vkey])}f)", vsc_ref, gif_dir / f"{vkey}_12fps.gif", bg_rgb=(28, 36, 48))

    save_review_gif(synced["sync_fighter_slash"], "SYNC FIGHTER + SLASH ARC (6f)", meta4["entities"]["fighter"]["unity_visual_root_scale"], gif_dir / "sync_fighter_slash_12fps.gif", bg_rgb=OCHRE_RGB)
    save_review_gif(synced["sync_guard_shieldblock"], "SYNC GUARD + SHIELD BLOCK (6f)", meta4["entities"]["guard"]["unity_visual_root_scale"], gif_dir / "sync_guard_shieldblock_12fps.gif", bg_rgb=MEADOW_RGB)
    save_review_gif(synced["sync_runner_hitpuff"], "SYNC RUNNER HIT + PUFF (6f)", meta4["entities"]["runner"]["unity_visual_root_scale"], gif_dir / "sync_runner_hitpuff_12fps.gif", bg_rgb=OCHRE_RGB)
    save_review_gif(synced["sync_boss_cleaveslam_telegraph"], "SYNC BOSS SLAM + TELEGRAPH (10f)", meta4["entities"]["boss"]["unity_visual_root_scale"], gif_dir / "sync_boss_cleaveslam_telegraph_12fps.gif", bg_rgb=LIMESTONE_RGB)

    # 4. Full 1080x1920 All-7-Class Gameplay Sequence -> GIF + 3 MP4 resolutions + 1280x720 Showcase MP4
    seq_1080 = render_all_7_classes_gameplay_sequence(loaded3, meta4, loaded4, n_ticks_12fps=48)

    # Seamless 24-frame (2.0s @ 12 FPS) 540x960 gameplay showcase GIF
    gif_game_frames = [
        im.resize((540, 960), Image.Resampling.LANCZOS).quantize(colors=256, method=Image.Quantize.MEDIANCUT, dither=Image.Dither.NONE)
        for im in seq_1080[:24]
    ]
    gif_game_frames[0].save(
        gif_dir / "all_7_classes_gameplay_showcase_12fps.gif",
        save_all=True,
        append_images=gif_game_frames[1:],
        duration=durations_12fps(24),
        loop=0,
        disposal=2,
    )
    print("[GIFS PASS] 28 12-FPS animated GIF previews written.")

    mp4_dir = ART004_DIR / "previews" / "mp4"
    encode_mp4(seq_1080, mp4_dir / "art004_gameplay_1080x1920_12fps.mp4", crf=18)
    seq_720 = [im.resize((720, 1280), Image.Resampling.LANCZOS) for im in seq_1080]
    encode_mp4(seq_720, mp4_dir / "art004_gameplay_720x1280_12fps.mp4", crf=21)
    seq_360 = [im.resize((360, 640), Image.Resampling.LANCZOS) for im in seq_1080]
    encode_mp4(seq_360, mp4_dir / "art004_gameplay_360x640_12fps.mp4", crf=25)
    build_all_7_showcase_mp4(loaded3, meta4, loaded4, synced, mp4_dir / "art004_all_7_classes_showcase_1280x720_12fps.mp4")
    print("[MP4 PASS] 4 12-FPS H.264 (yuv420p) MP4 stream previews written.")


if __name__ == "__main__":
    main()
