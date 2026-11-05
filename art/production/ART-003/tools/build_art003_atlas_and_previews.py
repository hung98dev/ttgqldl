#!/usr/bin/env python3
"""
ART-003 Deterministic Atlas Packer, Contact Sheets & 12 FPS Preview Generator (GIF + H.264 MP4).

Generates:
  1. Packed Pilot Sprite Atlas (1024x1024 RGBA32, 4px cell padding, FilterMode.Point safe)
     + JSON slice metadata + Unity 6000.6.7f1 Preset/SpriteAtlas YAML references.
  2. Static Contact Sheets for Builder (18f), Grunt (18f), and Synchronized Repair + VFX (6f).
  3. True 12 FPS Animated GIF loops (83/83/84 ms = 12.0 FPS) for all required clips,
     synchronized Builder + Spark, and composite Builder + Grunt gameplay.
  4. Real 12 FPS H.264 (yuv420p) MP4 videos via ffmpeg at 1080x1920, 720x1280, and 360x640,
     plus a master showcase MP4.
"""

from __future__ import annotations

import json
import subprocess
import tempfile
from pathlib import Path
from typing import Dict, List, Tuple

import numpy as np
from PIL import Image, ImageDraw, ImageFont

REPO_ROOT = Path(__file__).resolve().parents[4]
ART002_DIR = REPO_ROOT / "art" / "concepts" / "ART-002"
ART003_DIR = REPO_ROOT / "art" / "production" / "ART-003"

MEADOW_RGB = (104, 163, 71)     # #68A347
OCHRE_RGB = (207, 169, 110)     # #CFA96E
LIMESTONE_RGB = (184, 192, 200) # #B8C0C8
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


def load_frames_from_meta() -> Tuple[Dict, Dict[str, List[Image.Image]]]:
    with open(ART003_DIR / "clip_frame_metadata.json", "r", encoding="utf-8") as fp:
        meta = json.load(fp)
    loaded: Dict[str, List[Image.Image]] = {}
    for ent_key, ent_val in meta["entities"].items():
        loaded[ent_key] = [Image.open(REPO_ROOT / rec["file"]).convert("RGBA") for rec in ent_val["frames"]]
    return meta, loaded


# ============================================================================
# 1. DETERMINISTIC 1024x1024 PILOT SPRITE ATLAS + UNITY PRESETS
# ============================================================================

def build_pilot_atlas(meta: Dict, loaded: Dict[str, List[Image.Image]]) -> None:
    atlas_dir = ART003_DIR / "atlas"
    presets_dir = atlas_dir / "unity_presets"
    presets_dir.mkdir(parents=True, exist_ok=True)

    # Pack 42 sprites (128x128 each) into 1024x1024 RGBA32 with 4px gutter on every side (pitch = 136px, 7 cols x 6 rows = 952x816)
    atlas_w, atlas_h = 1024, 1024
    pad = 4
    pitch = 128 + pad * 2  # 136 px
    cols = 7
    atlas_im = Image.new("RGBA", (atlas_w, atlas_h), (0, 0, 0, 0))

    ordered_entries = []
    for ent_key in ["builder", "grunt", "vfx_builder_repair_spark"]:
        for rec, im in zip(meta["entities"][ent_key]["frames"], loaded[ent_key]):
            ordered_entries.append((ent_key, rec, im))

    slices = []
    for idx, (ent_key, rec, im) in enumerate(ordered_entries):
        r, c = divmod(idx, cols)
        x = pad + c * pitch
        y = pad + r * pitch
        atlas_im.paste(im, (x, y))
        # Unity texture coordinates have y=0 at BOTTOM; record both top-down and Unity bottom-up rects
        unity_y = atlas_h - (y + 128)
        slices.append(
            {
                "atlas_index": idx,
                "name": Path(rec["file"]).stem,
                "entity": ent_key,
                "clip": rec["clip"],
                "frame_index": rec["frame_index"],
                "source_file": rec["file"],
                "rect_top_down_px": [x, y, 128, 128],
                "rect_unity_bottom_up_px": [x, unity_y, 128, 128],
                "uv_rect_unity": [
                    round(x / atlas_w, 6),
                    round(unity_y / atlas_h, 6),
                    round(128 / atlas_w, 6),
                    round(128 / atlas_h, 6),
                ],
                "pivot_normalized": [0.50, 0.08],
                "pixels_per_unit": 192,
                "tight_bbox_in_cell_px": rec["bbox"],
            }
        )

    atlas_im.save(atlas_dir / "Characters_Pilot_Atlas.png", optimize=False)

    atlas_json = {
        "atlas_name": "Characters_Pilot_Atlas",
        "style_version": "bastion-dawn-v1",
        "dimensions": [atlas_w, atlas_h],
        "max_allowed_dimensions": [2048, 2048],
        "format": "RGBA32 (Uncompressed PNG)",
        "filter_mode": "Point",
        "mipmaps": False,
        "padding_px": pad,
        "allow_rotation": False,
        "tight_packing": False,
        "pixels_per_unit": 192,
        "unity_world_scale_for_48px_at_ortho5": {
            "camera_orthographic_size": 5.0,
            "reference_screen_px": [1080, 1920],
            "screen_px_per_world_unit": 192.0,
            "source_standing_body_height_px": 86,
            "displayed_height_at_scale_1_0_px": 86.0,
            "recommended_visual_root_scale_for_48px": round(48.0 / 86.0, 5),
            "alternative_ppu_for_scale_1_0": 344,
            "displayed_height_on_360x640_px": 16.0,
        },
        "total_packed_sprites": len(slices),
        "occupancy_ratio": round((42 * 128 * 128) / (atlas_w * atlas_h), 4),
        "slices": slices,
    }
    with open(atlas_dir / "Characters_Pilot_Atlas.json", "w", encoding="utf-8") as fp:
        json.dump(atlas_json, fp, indent=2)

    # Write Unity 6000.6.7f1 TextureImporter Preset YAML reference
    preset_yaml = """%YAML 1.1
%TAG !u! tag:unity3d.com,2011:
--- !u!181963792 &2655988077585873504
Preset:
  m_ObjectHideFlags: 0
  m_CorrespondingSourceObject: {fileID: 0}
  m_PrefabInstance: {fileID: 0}
  m_PrefabAsset: {fileID: 0}
  m_Name: TTG_CharacterSprite_Point192
  m_TargetType:
    m_NativeTypeID: 1006
    m_ManagedTypePPtr: {fileID: 0}
    m_ManagedTypeFallback: 
  m_Properties:
  - target: {fileID: 0}
    propertyPath: m_TextureType
    value: 8
    objectReference: {fileID: 0}
  - target: {fileID: 0}
    propertyPath: m_SpriteMode
    value: 1
    objectReference: {fileID: 0}
  - target: {fileID: 0}
    propertyPath: m_SpritePixelsToUnits
    value: 192
    objectReference: {fileID: 0}
  - target: {fileID: 0}
    propertyPath: m_Alignment
    value: 9
    objectReference: {fileID: 0}
  - target: {fileID: 0}
    propertyPath: m_SpritePivot.x
    value: 0.5
    objectReference: {fileID: 0}
  - target: {fileID: 0}
    propertyPath: m_SpritePivot.y
    value: 0.08
    objectReference: {fileID: 0}
  - target: {fileID: 0}
    propertyPath: m_EnableMipMap
    value: 0
    objectReference: {fileID: 0}
  - target: {fileID: 0}
    propertyPath: m_FilterMode
    value: 0
    objectReference: {fileID: 0}
  - target: {fileID: 0}
    propertyPath: m_AlphaIsTransparency
    value: 1
    objectReference: {fileID: 0}
  - target: {fileID: 0}
    propertyPath: m_TextureCompression
    value: 0
    objectReference: {fileID: 0}
  - target: {fileID: 0}
    propertyPath: m_sRGBTexture
    value: 1
    objectReference: {fileID: 0}
"""
    (presets_dir / "TTG_CharacterSprite_Point192.preset.yaml").write_text(preset_yaml, encoding="utf-8")

    atlas_yaml = """%YAML 1.1
%TAG !u! tag:unity3d.com,2011:
--- !u!687078895 &4343727234628468602
SpriteAtlas:
  m_ObjectHideFlags: 0
  m_Name: Characters_Pilot
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
    (presets_dir / "Characters_Pilot.spriteatlasv2.yaml").write_text(atlas_yaml, encoding="utf-8")

    verifier_cs = """// Reference Editor script for M06 integration under unity/Assets/_TTGQLDL/Editor/
// Kept outside unity/ as .cs.txt in ART-003 so M01-M05 runtime/editor assemblies remain untouched.
using UnityEditor;
using UnityEngine;

namespace Ttgqldl.Editor.ArtPilot
{
    public static class PilotSpriteImportVerifier
    {
        public static void ApplyAndVerifySprite(string assetPath)
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
            settings.spritePivot = new Vector2(0.50f, 0.08f);
            settings.spriteMeshType = SpriteMeshType.FullRect;
            importer.SetTextureSettings(settings);
            importer.SaveAndReimport();

            var sprite = AssetDatabase.LoadAssetAtPath<Sprite>(assetPath);
            Debug.Log($"[ART-003] Verified {assetPath}: rect={sprite.rect}, pivot={sprite.pivot}, ppu={sprite.pixelsPerUnit}, filter={sprite.texture.filterMode}");
        }
    }
}
"""
    (presets_dir / "PilotSpriteImportVerifier.cs.txt").write_text(verifier_cs, encoding="utf-8")


# ============================================================================
# 2. STATIC CONTACT SHEETS (1:1, 48px GAMEPLAY SCALE, 2x NN + PIVOT CROSSHAIRS)
# ============================================================================

def scale_char_to_height(im_128: Image.Image, body_target_h: int = 48) -> Image.Image:
    """Scale 128x128 cell (where standing body = 86px) proportionally so standing body = body_target_h."""
    cell_sz = max(16, int(round(128 * (body_target_h / 86.0))))
    return im_128.resize((cell_sz, cell_sz), Image.Resampling.NEAREST)


def build_character_contact_sheet(
    title: str,
    subtitle: str,
    ent_meta: Dict,
    frames: List[Image.Image],
    out_path: Path,
) -> None:
    f_title = load_font(24, bold=True)
    f_sub = load_font(15, bold=False)
    f_lbl = load_font(14, bold=True)
    f_small = load_font(12, bold=False)

    cols = 6
    rows = 3
    cw, ch = 260, 290
    W = 40 + cols * cw
    H = 110 + rows * ch + 20
    board = Image.new("RGB", (W, H), DARK_BG_RGB)
    dr = ImageDraw.Draw(board)

    dr.text((24, 18), title, font=f_title, fill=(254, 240, 138))
    dr.text((24, 52), subtitle, font=f_sub, fill=(203, 213, 225))
    dr.line([(24, 84), (W - 24, 84)], fill=(51, 65, 85), width=2)

    for idx, (rec, fr) in enumerate(zip(ent_meta["frames"], frames)):
        r, c = divmod(idx, cols)
        ox = 20 + c * cw
        oy = 98 + r * ch
        dr.rounded_rectangle([ox + 6, oy + 6, ox + cw - 6, oy + ch - 6], radius=10, fill=(28, 36, 52), outline=(64, 78, 104), width=2)

        clip_tag = f"#{idx:02d}  {rec['clip'].upper()} f{rec['frame_index']:02d}"
        dr.text((ox + 16, oy + 14), clip_tag, font=f_lbl, fill=(248, 250, 252))

        # Left swatch: 128x128 source on Meadow (#68A347) with (64, 117) pivot crosshair
        sx, sy = ox + 16, oy + 44
        dr.rectangle([sx, sy, sx + 128, sy + 128], fill=MEADOW_RGB, outline=(17, 21, 28), width=2)
        dr.line([(sx, sy + 117), (sx + 128, sy + 117)], fill=(82, 133, 54), width=1)
        dr.line([(sx + 64, sy), (sx + 64, sy + 128)], fill=(82, 133, 54), width=1)
        board.paste(fr, (sx, sy), fr)
        # Red pivot tick at (64, 117)
        dr.line([(sx + 60, sy + 117), (sx + 68, sy + 117)], fill=(239, 68, 68), width=2)
        dr.line([(sx + 64, sy + 113), (sx + 64, sy + 121)], fill=(239, 68, 68), width=2)
        dr.text((sx + 4, sy + 132), "128x128 (1:1)", font=f_small, fill=(148, 163, 184))

        # Right-top swatch: Actual 1080p gameplay size (48px standing -> 71x71 cell) on Ochre road
        fr_48 = scale_char_to_height(fr, 48)
        rx, ry = ox + 158, oy + 44
        dr.rectangle([rx, ry, rx + 82, ry + 82], fill=OCHRE_RGB, outline=(17, 21, 28), width=2)
        px = rx + (82 - fr_48.width) // 2
        py = ry + (82 - fr_48.height) // 2
        board.paste(fr_48, (px, py), fr_48)
        dr.text((rx + 4, ry + 86), "1080p (48px)", font=f_small, fill=(254, 240, 138))

        # Right-bottom swatch: Actual 360x640 mobile size (16px standing -> 24x24 cell) on Limestone + Grayscale
        fr_16 = scale_char_to_height(fr, 16)
        mx, my = ox + 158, oy + 150
        dr.rectangle([mx, my, mx + 38, my + 38], fill=LIMESTONE_RGB, outline=(17, 21, 28), width=1)
        board.paste(fr_16, (mx + (38 - fr_16.width) // 2, my + (38 - fr_16.height) // 2), fr_16)

        # Grayscale 48px stamp beside it
        tmp_gray = Image.new("RGB", (38, 38), LIMESTONE_RGB)
        fr_24 = scale_char_to_height(fr, 24)
        tmp_gray.paste(fr_24, ((38 - fr_24.width) // 2, (38 - fr_24.height) // 2), fr_24)
        tmp_gray = tmp_gray.convert("L").convert("RGB")
        board.paste(tmp_gray, (mx + 44, my))
        dr.rectangle([mx + 44, my, mx + 82, my + 38], outline=(17, 21, 28), width=1)
        dr.text((mx, my + 42), "360p | Gray", font=f_small, fill=(148, 163, 184))

        bbox = rec["bbox"]
        dr.text(
            (ox + 16, oy + ch - 52),
            f"bbox: [{bbox[0]},{bbox[1]}..{bbox[2]},{bbox[3]}]",
            font=f_small,
            fill=(203, 213, 225),
        )
        dr.text(
            (ox + 16, oy + ch - 32),
            f"foot_y={rec['foot_contact_y']} | halo={rec['semi_transparent_pixels']}",
            font=f_small,
            fill=(134, 239, 172),
        )

    out_path.parent.mkdir(parents=True, exist_ok=True)
    board.save(out_path)


def compose_builder_with_spark(builder_repair_frames: List[Image.Image], vfx_frames: List[Image.Image]) -> List[Image.Image]:
    """
    Create an 8-frame 12 FPS synchronized Repair + Spark sequence (160x128 RGBA):
      t0: Builder repair f00
      t1: Builder repair f01
      t2: Builder repair f02 (overhead apex)
      t3: Builder repair f03 (downward strike) + VFX spark f00 at impact point (x+34, y+26)
      t4: Builder repair f04 (impact compression) + VFX spark f01
      t5: Builder repair f05 (recovery) + VFX spark f02
      t6: Builder idle/ready + VFX spark f03
      t7: Builder idle/ready + VFX spark f04..f05 dissipation
    """
    out = []
    # Spark offset places spark center (64,67) right onto Builder's downward hammer strike face (98, 96)
    spark_offset = (34, 26)
    seq_map = [
        (builder_repair_frames[0], None),
        (builder_repair_frames[1], None),
        (builder_repair_frames[2], None),
        (builder_repair_frames[3], vfx_frames[0]),
        (builder_repair_frames[4], vfx_frames[1]),
        (builder_repair_frames[5], vfx_frames[2]),
        (builder_repair_frames[5], vfx_frames[3]),
        (builder_repair_frames[0], vfx_frames[4]),
    ]
    for b_fr, v_fr in seq_map:
        c = Image.new("RGBA", (176, 144), (0, 0, 0, 0))
        c.paste(b_fr, (12, 8), b_fr)
        if v_fr is not None:
            v_layer = Image.new("RGBA", (176, 144), (0, 0, 0, 0))
            v_layer.paste(v_fr, (12 + spark_offset[0], 8 + spark_offset[1]), v_fr)
            c = Image.alpha_composite(c, v_layer)
        out.append(c)
    return out


def build_vfx_and_sync_contact_sheet(
    builder_repair: List[Image.Image],
    vfx_frames: List[Image.Image],
    sync_frames: List[Image.Image],
    out_path: Path,
) -> None:
    f_title = load_font(24, bold=True)
    f_lbl = load_font(14, bold=True)
    f_small = load_font(12, bold=False)

    W, H = 1580, 680
    board = Image.new("RGB", (W, H), DARK_BG_RGB)
    dr = ImageDraw.Draw(board)
    dr.text((24, 18), "ART-003 VFX PILOT — vfx_builder_repair_spark (6f @ 12 FPS) & SYNCHRONIZED HAMMER IMPACT", font=f_title, fill=(254, 240, 138))
    dr.text((24, 52), "Row 1: Isolated 128x128 RGBA VFX Frames (Golden #FEF08A 4-Point Sparks + Seasoned Oak Chips, Zero End-Frame Bright Pixels)  |  Row 2: Synchronized Builder Repair + Spark", font=f_small, fill=(203, 213, 225))

    # Row 1: 6 VFX frames on dark slate & limestone swatches
    for i, vf in enumerate(vfx_frames):
        ox = 24 + i * 254
        oy = 88
        dr.rounded_rectangle([ox, oy, ox + 242, oy + 230], radius=10, fill=(28, 36, 52), outline=(64, 78, 104), width=2)
        dr.text((ox + 12, oy + 10), f"VFX f{i:02d}  (t={i*83.3:.0f}ms)", font=f_lbl, fill=(248, 250, 252))
        dr.rectangle([ox + 12, oy + 38, ox + 140, oy + 166], fill=(32, 42, 34), outline=(17, 21, 28), width=2)
        board.paste(vf, (ox + 12, oy + 38), vf)
        vf_48 = scale_char_to_height(vf, 48)
        dr.rectangle([ox + 150, oy + 38, ox + 230, oy + 118], fill=LIMESTONE_RGB, outline=(17, 21, 28), width=2)
        board.paste(vf_48, (ox + 150 + (80 - vf_48.width) // 2, oy + 38 + (80 - vf_48.height) // 2), vf_48)
        arr = np.array(vf)
        bright_px = int(np.sum((arr[:, :, 3] > 0) & (arr[:, :, 0] > 220) & (arr[:, :, 1] > 200)))
        dr.text((ox + 12, oy + 178), f"opaque={int((arr[:,:,3]>0).sum())}px", font=f_small, fill=(203, 213, 225))
        dr.text((ox + 12, oy + 198), f"bright_spark_px={bright_px}", font=f_small, fill=(134, 239, 172))

    # Row 2: 8 synchronized frames on Bastion Meadow + Curtain Wall backdrop strip
    for i, sf in enumerate(sync_frames):
        ox = 24 + i * 192
        oy = 350
        dr.rounded_rectangle([ox, oy, ox + 182, oy + 300], radius=10, fill=(28, 36, 52), outline=(64, 78, 104), width=2)
        lbl = f"Sync t{i:02d}" + (" [IMPACT!]" if i in (3, 4) else "")
        dr.text((ox + 10, oy + 10), lbl, font=f_lbl, fill=(254, 240, 138) if i in (3, 4) else (248, 250, 252))
        # Draw wall masonry right side + meadow floor inside preview box
        bx, by = ox + 8, oy + 38
        dr.rectangle([bx, by, bx + 166, by + 144], fill=MEADOW_RGB, outline=(17, 21, 28), width=2)
        dr.rectangle([bx + 110, by + 18, bx + 164, by + 124], fill=(155, 165, 176), outline=(17, 21, 28), width=2)
        sub = sf.resize((156, 128), Image.Resampling.NEAREST)
        board.paste(sub, (bx + 4, by + 8), sub)

        # Also show actual 1080p gameplay scale (48px tall) underneath!
        sf_game = sf.resize((int(round(sf.width * 48 / 86)), int(round(sf.height * 48 / 86))), Image.Resampling.NEAREST)
        gx, gy = ox + 8, oy + 196
        dr.rectangle([gx, gy, gx + 166, gy + 92], fill=OCHRE_RGB, outline=(17, 21, 28), width=2)
        board.paste(sf_game, (gx + (166 - sf_game.width) // 2, gy + (92 - sf_game.height) // 2), sf_game)
        dr.text((gx + 6, gy + 6), "1080p (48px)", font=f_small, fill=(17, 21, 28))

    out_path.parent.mkdir(parents=True, exist_ok=True)
    board.save(out_path)


# ============================================================================
# 3. TRUE 12 FPS ANIMATED GIFS & REAL H.264 MP4 STREAM PREVIEWS
# ============================================================================

def durations_12fps(n_frames: int) -> List[int]:
    """
    GIF89a stores frame delays in 10ms centiseconds.
    [80, 80, 90] ms = 250ms per 3 frames = exact 12.00 FPS (83.33ms mean) without centisecond truncation!
    """
    if n_frames == 8:
        return [80, 80, 90, 80, 90, 80, 80, 90]  # 670ms / 8 = 83.75ms (11.94 FPS, <0.5ms error)
    base = [80, 80, 90]
    return [base[i % 3] for i in range(n_frames)]


def save_clip_gif(
    frames_rgba: List[Image.Image],
    label: str,
    out_gif: Path,
    bg_rgb: Tuple[int, int, int] = MEADOW_RGB,
) -> None:
    """
    Build a crisp 12 FPS GIF showing 3 simultaneous scales in one compact review card:
      - Left:   2x Nearest-Neighbor zoom (256px) for silhouette & cleanup inspection
      - Top-R:  1:1 Source (128x128)
      - Mid-R:  Actual 1080x1920 Gameplay Scale (48px tall) on Ochre lane
      - Bot-R:  Actual 360x640 Mobile Scale (16px tall) on Limestone + Grayscale
    """
    f_hdr = load_font(14, bold=True)
    f_sm = load_font(12, bold=False)
    rendered_frames: List[Image.Image] = []

    for idx, fr in enumerate(frames_rgba):
        W, H = 430, 320
        card = Image.new("RGB", (W, H), DARK_BG_RGB)
        dr = ImageDraw.Draw(card)
        dr.text((14, 10), f"{label}  [12 FPS | f{idx:02d}/{len(frames_rgba)-1:02d}]", font=f_hdr, fill=(254, 240, 138))

        # 2x NN view
        z2 = fr.resize((fr.width * 2 if fr.width == 128 else 256, 256), Image.Resampling.NEAREST)
        dr.rectangle([14, 38, 14 + 260, 38 + 264], fill=bg_rgb, outline=(17, 21, 28), width=2)
        card.paste(z2, (16 + (256 - z2.width) // 2, 40), z2)
        dr.text((20, 282), "2x NN Zoom", font=f_sm, fill=(17, 21, 28))

        # 1:1 Source 128x128
        rx = 288
        dr.rectangle([rx, 38, rx + 128, 38 + 128], fill=bg_rgb, outline=(17, 21, 28), width=2)
        s1 = fr if fr.width == 128 else fr.resize((128, int(round(fr.height * 128 / fr.width))), Image.Resampling.NEAREST)
        card.paste(s1, (rx, 38 + (128 - s1.height) // 2), s1)
        dr.text((rx + 4, 42), "1:1 (128px)", font=f_sm, fill=(17, 21, 28))

        # 1080p Gameplay 48px
        f48 = fr.resize((int(round(fr.width * 48 / 86)), int(round(fr.height * 48 / 86))), Image.Resampling.NEAREST)
        dr.rectangle([rx, 174, rx + 128, 248], fill=OCHRE_RGB, outline=(17, 21, 28), width=2)
        card.paste(f48, (rx + (128 - f48.width) // 2, 174 + (74 - f48.height) // 2), f48)
        dr.text((rx + 4, 178), "1080p (48px)", font=f_sm, fill=(17, 21, 28))

        # 360p Mobile 16px + Grayscale 48px
        f16 = fr.resize((max(8, int(round(fr.width * 16 / 86))), max(8, int(round(fr.height * 16 / 86)))), Image.Resampling.NEAREST)
        dr.rectangle([rx, 256, rx + 60, 302], fill=LIMESTONE_RGB, outline=(17, 21, 28), width=2)
        card.paste(f16, (rx + (60 - f16.width) // 2, 256 + (46 - f16.height) // 2), f16)
        dr.text((rx + 4, 258), "360p", font=f_sm, fill=(17, 21, 28))

        g_box = Image.new("RGB", (62, 46), LIMESTONE_RGB)
        f32 = fr.resize((int(round(fr.width * 32 / 86)), int(round(fr.height * 32 / 86))), Image.Resampling.NEAREST)
        g_box.paste(f32, ((62 - f32.width) // 2, (46 - f32.height) // 2), f32)
        g_box = g_box.convert("L").convert("RGB")
        card.paste(g_box, (rx + 66, 256))
        dr.rectangle([rx + 66, 256, rx + 128, 302], outline=(17, 21, 28), width=2)

        rendered_frames.append(card.quantize(colors=256, method=Image.Quantize.MEDIANCUT, dither=Image.Dither.NONE))

    out_gif.parent.mkdir(parents=True, exist_ok=True)
    rendered_frames[0].save(
        out_gif,
        save_all=True,
        append_images=rendered_frames[1:],
        duration=durations_12fps(len(rendered_frames)),
        loop=0,
        disposal=2,
    )


def render_gameplay_composite_sequence(
    loaded: Dict[str, List[Image.Image]],
    n_ticks_12fps: int = 48,  # 4.0 seconds @ 12 FPS = exact LCM of 4-frame, 6-frame, and 8-frame loops!
) -> List[Image.Image]:
    """
    Render a 48-frame (4.0s @ 12 FPS) actual-scale 1080x1920 Bastion Dawn gameplay scene
    with animated Builders repairing the wall (with synchronized repair sparks), walking Builders,
    idling Builders, marching Grunts on West/Center/East lanes (flipped cleanly via flipX on East!),
    and Grunts smashing the perimeter wall — ZERO baked fake viewer names, honest MOCK watermark!
    """
    b_idle = loaded["builder"][0:4]
    b_walk = loaded["builder"][4:10]
    b_rep = loaded["builder"][10:16]
    b_hit = loaded["builder"][16:18]

    g_idle = loaded["grunt"][0:4]
    g_walk = loaded["grunt"][4:10]
    g_smash = loaded["grunt"][10:16]
    g_hit = loaded["grunt"][16:18]
    vfx = loaded["vfx_builder_repair_spark"]

    f_hud_lg = load_font(40, bold=True)
    f_hud_md = load_font(30, bold=True)
    f_hud_sm = load_font(24, bold=True)

    # Pre-build static 1080x1920 Bastion Dawn world + HUD base (safe zones x<=82%, y<=88% strictly honored!)
    base_1080 = Image.new("RGB", (1080, 1920), MEADOW_RGB)
    bdr = ImageDraw.Draw(base_1080)
    # Subtle sunlit dawn meadow bands (low-frequency, S <= 48%)
    bdr.polygon([(0, 0), (1080, 0), (1080, 620), (0, 760)], fill=(114, 173, 79))
    bdr.polygon([(0, 1280), (1080, 1180), (1080, 1920), (0, 1920)], fill=(92, 148, 62))

    # Three unmistakable approach lanes converging on the central 44% bastion (x=216..692, center x=454, y=980)
    # West Ochre Lane
    bdr.polygon([(40, 260), (168, 260), (355, 760), (245, 810)], fill=OCHRE_RGB, outline=(142, 112, 68), width=4)
    # East Ochre Lane (strictly left of x=885 safe zone!)
    bdr.polygon([(720, 260), (846, 260), (660, 810), (555, 760)], fill=OCHRE_RGB, outline=(142, 112, 68), width=4)
    # Center Limestone Boss Causeway
    bdr.polygon([(392, 260), (516, 260), (526, 770), (382, 770)], fill=LIMESTONE_RGB, outline=(96, 106, 120), width=4)

    # Central Bastion Courtyard & Circular Curtain Wall (width = 476 px = 44.1% of 1080 px, centered at x=454!)
    bdr.ellipse([216, 740, 692, 1190], fill=(216, 179, 132), outline=(17, 21, 28), width=6)
    bdr.ellipse([238, 762, 670, 1168], fill=(216, 179, 132), outline=(100, 116, 139), width=18)
    # Central Cobalt-Slate Keep roof emblem inside courtyard
    bdr.rounded_rectangle([374, 835, 534, 985], radius=14, fill=(232, 223, 200), outline=(17, 21, 28), width=5)
    bdr.polygon([(360, 855), (454, 785), (548, 855)], fill=(59, 110, 165), outline=(17, 21, 28), width=5)
    # South Gate timber arch
    bdr.rounded_rectangle([408, 1136, 500, 1196], radius=8, fill=(140, 90, 50), outline=(17, 21, 28), width=5)

    # Top Anchored HUD (x=36..876 <= 82%, y=36..256)
    bdr.rounded_rectangle([36, 36, 876, 244], radius=18, fill=(16, 23, 38), outline=(234, 179, 8), width=3)
    bdr.text((60, 52), "ĐỢT 03  •  GIAO TRANH", font=f_hud_md, fill=(255, 255, 255))
    bdr.rounded_rectangle([638, 48, 856, 104], radius=12, fill=(30, 41, 59), outline=(250, 204, 21), width=2)
    bdr.text((662, 56), "GIỜ 01:40", font=f_hud_md, fill=(254, 240, 138))
    # Segmented Wall & Castle HP bars
    bdr.text((60, 116), "TƯỜNG 820/1000", font=f_hud_sm, fill=(255, 255, 255))
    bdr.rounded_rectangle([290, 118, 540, 148], radius=8, fill=(30, 41, 59), outline=(148, 163, 184), width=2)
    bdr.rounded_rectangle([293, 121, 495, 145], radius=6, fill=(76, 201, 100))
    bdr.text((60, 162), "LÂU ĐÀI 500/500", font=f_hud_sm, fill=(255, 255, 255))
    bdr.rounded_rectangle([290, 164, 540, 194], radius=8, fill=(30, 41, 59), outline=(148, 163, 184), width=2)
    bdr.rounded_rectangle([293, 167, 537, 191], radius=6, fill=(59, 130, 246))
    bdr.text((568, 124), "GỖ 340  ĐÁ 210", font=f_hud_sm, fill=(241, 245, 249))
    bdr.text((568, 164), "LƯƠNG 180", font=f_hud_sm, fill=(241, 245, 249))

    # Bottom Single Primary CTA + Role Counts + Honest MOCK Watermark (strictly x<=876, y<=1680 <= 88%!)
    bdr.rounded_rectangle([54, 1460, 864, 1546], radius=16, fill=(18, 26, 43), outline=(100, 116, 139), width=2)
    bdr.text((84, 1484), "XÂY: 12     THỦ: 09     ĐÁNH: 15", font=f_hud_md, fill=(255, 255, 255))
    bdr.rounded_rectangle([54, 1558, 864, 1632], radius=18, fill=(15, 23, 42), outline=(234, 179, 8), width=4)
    bdr.text((104, 1570), "Bình luận XÂY / THỦ / ĐÁNH", font=f_hud_lg, fill=(255, 255, 255))
    bdr.rounded_rectangle([54, 1640, 520, 1684], radius=10, fill=(28, 22, 10), outline=(250, 204, 21), width=2)
    bdr.text((72, 1646), "MOCK / ART-003 PILOT PREVIEW", font=f_hud_sm, fill=(250, 204, 21))

    # Pre-scale all sprites to exact 48px standing height on 1080x1920 (71x71 cell)!
    b_idle_48 = [scale_char_to_height(f, 48) for f in b_idle]
    b_walk_48 = [scale_char_to_height(f, 48) for f in b_walk]
    b_rep_48 = [scale_char_to_height(f, 48) for f in b_rep]
    b_hit_48 = [scale_char_to_height(f, 48) for f in b_hit]

    g_idle_48 = [scale_char_to_height(f, 48) for f in g_idle]
    g_walk_48 = [scale_char_to_height(f, 48) for f in g_walk]
    g_smash_48 = [scale_char_to_height(f, 48) for f in g_smash]
    g_hit_48 = [scale_char_to_height(f, 48) for f in g_hit]
    vfx_48 = [scale_char_to_height(f, 48) for f in vfx]

    def flip_h(im: Image.Image) -> Image.Image:
        return im.transpose(Image.Transpose.FLIP_LEFT_RIGHT)

    seq: List[Image.Image] = []
    for t in range(n_ticks_12fps):
        frame = base_1080.copy()
        u24 = (t % 24) / 24.0  # Exact 24-frame (2.0s @ 12 FPS) seamless loop period = LCM(4, 6, 8, 12)

        # Collect entities with (foot_y, sprite_rgba, center_x) for authentic Y-sorting!
        draw_list: List[Tuple[float, Image.Image, float]] = []

        # 1. West Lane Grunts patrolling diagonally with smooth periodic turnaround (zero loop-seam jump!)
        for x_mid, y_mid, dx_amp, dy_amp, ph in [
            (175.0, 445.0, 18.0, 32.0, 0.0),
            (235.0, 585.0, 20.0, 36.0, 0.5),
        ]:
            ang = 2.0 * np.pi * (u24 + ph)
            gx = x_mid + dx_amp * np.sin(ang)
            gy = y_mid + dy_amp * np.sin(ang)
            gw_sp = g_walk_48[(t + int(ph * 6)) % 6] if np.cos(ang) >= 0 else flip_h(g_walk_48[(t + int(ph * 6)) % 6])
            draw_list.append((float(gy), gw_sp, float(gx)))

        # 2. Center Limestone Causeway Grunts (1 idling scout + 2 smashing north gate!)
        draw_list.append((540.0, g_idle_48[t % 4], 454.0))
        draw_list.append((748.0, g_smash_48[t % 6], 418.0))
        draw_list.append((752.0, flip_h(g_smash_48[(t + 3) % 6]), 492.0))

        # 3. East Ochre Lane Grunts (mirrored via flipX as specified in ART-003 §3!)
        east_ang = 2.0 * np.pi * u24
        ex = 702.0 - 18.0 * np.sin(east_ang)
        ey = 575.0 + 30.0 * np.sin(east_ang)
        ew_sp = flip_h(g_walk_48[(t + 2) % 6]) if np.cos(east_ang) >= 0 else g_walk_48[(t + 2) % 6]
        draw_list.append((float(ey), ew_sp, float(ex)))
        draw_list.append((785.0, flip_h(g_smash_48[(t + 1) % 6]), 642.0))

        # 4. Builders inside/along Curtain Wall:
        #    - 2 Builders actively repairing West & East wall parapets with synchronized vfx_builder_repair_spark!
        #    - 1 Builder patrolling courtyard (Walk cycle, seamless 24f loop)
        #    - 1 Builder idling near South Gate + 1 Builder taking Hit reaction near North breach
        rep_phase_w = t % 8
        b_west_fr = b_rep_48[min(5, rep_phase_w)] if rep_phase_w < 6 else b_idle_48[rep_phase_w - 6]
        draw_list.append((825.0, b_west_fr, 272.0))

        rep_phase_e = (t + 4) % 8
        b_east_fr = b_rep_48[min(5, rep_phase_e)] if rep_phase_e < 6 else b_idle_48[rep_phase_e - 6]
        draw_list.append((835.0, b_east_fr, 588.0))

        # Courtyard walking Builder (24-frame smooth periodic patrol)
        patrol_x = 454.0 + 78.0 * np.sin(2.0 * np.pi * u24)
        facing_right = np.cos(2.0 * np.pi * u24) >= 0
        bw_sp = b_walk_48[t % 6] if facing_right else flip_h(b_walk_48[t % 6])
        draw_list.append((1045.0, bw_sp, float(patrol_x)))

        # Idling Builder by South Gate
        draw_list.append((1125, b_idle_48[(t + 1) % 4], 454))

        # Builder Near North Wall showing Hit reaction every 16 frames when Grunt smashes!
        if (t % 12) in (3, 4):
            draw_list.append((792, b_hit_48[(t % 12) - 3], 452))
        else:
            draw_list.append((792, b_idle_48[t % 4], 452))

        # Sort back-to-front by foot Y (20_CastleAndUnits_YSorted)
        draw_list.sort(key=lambda item: item[0])

        for fy, sp, cx in draw_list:
            # Ground contact shadow ellipse on 10_GroundTelegraphsAndShadows
            sh = Image.new("RGBA", (36, 14), (0, 0, 0, 0))
            ImageDraw.Draw(sh).ellipse([2, 2, 34, 12], fill=(17, 21, 28, 95))
            frame.paste(sh, (int(round(cx - 18)), int(round(fy - 7))), sh)

            # Paste sprite anchored at normalized pivot (0.50, 0.08)
            px = int(round(cx - sp.width * 0.50))
            py = int(round(fy - sp.height * 0.92))
            frame.paste(sp, (px, py), sp)

        # Sorting layer 30_CombatAndRepairVFX: synchronized repair sparks on West & East wall Builders!
        for rep_ph, bx, by in [(rep_phase_w, 272, 825), (rep_phase_e, 588, 835)]:
            if 3 <= rep_ph <= 7:
                vf = vfx_48[min(5, rep_ph - 3)]
                vx = int(round(bx + 18 - vf.width * 0.5))
                vy = int(round(by - 20 - vf.height * 0.5))
                frame.paste(vf, (vx, vy), vf)

        seq.append(frame)
    return seq


def encode_mp4_from_pil_frames(frames_rgb: List[Image.Image], out_mp4: Path, crf: int = 18) -> None:
    out_mp4.parent.mkdir(parents=True, exist_ok=True)
    with tempfile.TemporaryDirectory() as td:
        tdir = Path(td)
        for idx, im in enumerate(frames_rgb):
            im.save(tdir / f"f_{idx:04d}.png")
        cmd = [
            "ffmpeg",
            "-y",
            "-framerate",
            "12",
            "-i",
            str(tdir / "f_%04d.png"),
            "-c:v",
            "libx264",
            "-pix_fmt",
            "yuv420p",
            "-crf",
            str(crf),
            "-movflags",
            "+faststart",
            str(out_mp4),
        ]
        subprocess.run(cmd, check=True, stdout=subprocess.PIPE, stderr=subprocess.PIPE)


def build_showcase_mp4(
    loaded: Dict[str, List[Image.Image]],
    sync_frames: List[Image.Image],
    out_mp4: Path,
) -> None:
    """Build a 72-frame (6.0s @ 12 FPS) 1280x720 side-by-side showcase MP4 of all 9 clips playing simultaneously at 12 FPS."""
    f_title = load_font(24, bold=True)
    f_lbl = load_font(15, bold=True)
    f_sm = load_font(12, bold=False)

    panels = [
        ("Builder Idle (4f)", loaded["builder"][0:4], MEADOW_RGB),
        ("Builder Walk (6f)", loaded["builder"][4:10], OCHRE_RGB),
        ("Builder Repair (6f)", loaded["builder"][10:16], MEADOW_RGB),
        ("Builder Hit (2f)", loaded["builder"][16:18], OCHRE_RGB),
        ("Repair + Spark Sync (8f)", sync_frames, MEADOW_RGB),
        ("Grunt Idle (4f)", loaded["grunt"][0:4], MEADOW_RGB),
        ("Grunt Walk (6f)", loaded["grunt"][4:10], OCHRE_RGB),
        ("Grunt ClubSmash (6f)", loaded["grunt"][10:16], LIMESTONE_RGB),
        ("Grunt Hit (2f)", loaded["grunt"][16:18], OCHRE_RGB),
    ]

    seq: List[Image.Image] = []
    for t in range(72):
        canvas = Image.new("RGB", (1280, 720), DARK_BG_RGB)
        dr = ImageDraw.Draw(canvas)
        dr.text((24, 16), "ART-003 BASTION DAWN v1 — ALL ANIMATION CLIPS & VFX SHOWCASE (12.0 FPS)", font=f_title, fill=(254, 240, 138))
        for idx, (title, clip_frs, bg) in enumerate(panels):
            r, c = divmod(idx, 5) if idx < 5 else (1, idx - 5)
            ox = 24 + c * 248
            oy = 66 + r * 316
            dr.rounded_rectangle([ox, oy, ox + 236, oy + 298], radius=10, fill=(28, 36, 52), outline=(64, 78, 104), width=2)
            dr.text((ox + 10, oy + 8), title, font=f_lbl, fill=(248, 250, 252))
            # Hold Hit 2f with 4 frames of idle recovery so it doesn't strobe > 3Hz!
            if "Hit" in title:
                sub_idx = t % 8
                fr = clip_frs[sub_idx] if sub_idx < 2 else (loaded["builder"][0] if "Builder" in title else loaded["grunt"][0])
            else:
                fr = clip_frs[t % len(clip_frs)]

            # 1:1 128px box
            dr.rectangle([ox + 12, oy + 36, ox + 156, oy + 180], fill=bg, outline=(17, 21, 28), width=2)
            s_fit = fr if fr.width <= 140 else fr.resize((140, int(round(fr.height * 140 / fr.width))), Image.Resampling.NEAREST)
            canvas.paste(s_fit, (ox + 12 + (144 - s_fit.width) // 2, oy + 36 + (144 - s_fit.height) // 2), s_fit)

            # 1080p 48px box + 360p 16px box
            f48 = fr.resize((int(round(fr.width * 48 / 86)), int(round(fr.height * 48 / 86))), Image.Resampling.NEAREST)
            dr.rectangle([ox + 12, oy + 190, ox + 144, oy + 286], fill=LIMESTONE_RGB, outline=(17, 21, 28), width=2)
            canvas.paste(f48, (ox + 12 + (132 - f48.width) // 2, oy + 190 + (96 - f48.height) // 2), f48)
            dr.text((ox + 16, oy + 194), "48px", font=f_sm, fill=(17, 21, 28))

            f16 = fr.resize((max(8, int(round(fr.width * 16 / 86))), max(8, int(round(fr.height * 16 / 86)))), Image.Resampling.NEAREST)
            dr.rectangle([ox + 154, oy + 190, ox + 224, oy + 286], fill=LIMESTONE_RGB, outline=(17, 21, 28), width=2)
            canvas.paste(f16, (ox + 154 + (70 - f16.width) // 2, oy + 190 + (96 - f16.height) // 2), f16)
            dr.text((ox + 158, oy + 194), "16px", font=f_sm, fill=(17, 21, 28))

        seq.append(canvas)

    encode_mp4_from_pil_frames(seq, out_mp4, crf=18)


def main() -> None:
    meta, loaded = load_frames_from_meta()
    build_pilot_atlas(meta, loaded)

    cs_dir = ART003_DIR / "previews" / "contact_sheets"
    gif_dir = ART003_DIR / "previews" / "gifs"
    mp4_dir = ART003_DIR / "previews" / "mp4"

    build_character_contact_sheet(
        "ART-003 BUILDER (XÂY) — ALL 18 PRODUCTION FRAMES @ 12 FPS",
        "Canvas: 128x128 RGBA | Pivot: (0.50, 0.08) -> (64, 117) | Locked 16-Color Bastion Dawn v1 Palette | Idle(4) + Walk(6) + Repair(6) + Hit(2)",
        meta["entities"]["builder"],
        loaded["builder"],
        cs_dir / "contact_builder_all_18f.png",
    )
    build_character_contact_sheet(
        "ART-003 GRUNT (ENEMY PRIORITY 2) — ALL 18 PRODUCTION FRAMES @ 12 FPS",
        "Canvas: 128x128 RGBA | Pivot: (0.50, 0.08) -> (64, 117) | Locked 16-Color Bastion Dawn v1 Palette | Idle(4) + Walk(6) + ClubSmash(6) + Hit(2)",
        meta["entities"]["grunt"],
        loaded["grunt"],
        cs_dir / "contact_grunt_all_18f.png",
    )

    sync_frames = compose_builder_with_spark(loaded["builder"][10:16], loaded["vfx_builder_repair_spark"])
    build_vfx_and_sync_contact_sheet(
        loaded["builder"][10:16],
        loaded["vfx_builder_repair_spark"],
        sync_frames,
        cs_dir / "contact_vfx_and_sync_board.png",
    )

    # Individual 12 FPS GIF previews
    save_clip_gif(loaded["builder"][0:4], "Builder Idle (4f)", gif_dir / "preview_builder_idle_12fps.gif", MEADOW_RGB)
    save_clip_gif(loaded["builder"][4:10], "Builder Walk (6f)", gif_dir / "preview_builder_walk_12fps.gif", OCHRE_RGB)
    save_clip_gif(loaded["builder"][10:16], "Builder Repair (6f)", gif_dir / "preview_builder_repair_12fps.gif", MEADOW_RGB)
    # Pad Hit 2f with 4 rest frames in GIF preview so viewer sees distinct hit -> recovery without >3Hz strobe
    save_clip_gif(
        loaded["builder"][16:18] + [loaded["builder"][0]] * 4,
        "Builder Hit + Settle",
        gif_dir / "preview_builder_hit_12fps.gif",
        OCHRE_RGB,
    )

    save_clip_gif(loaded["grunt"][0:4], "Grunt Idle (4f)", gif_dir / "preview_grunt_idle_12fps.gif", MEADOW_RGB)
    save_clip_gif(loaded["grunt"][4:10], "Grunt Walk (6f)", gif_dir / "preview_grunt_walk_12fps.gif", OCHRE_RGB)
    save_clip_gif(loaded["grunt"][10:16], "Grunt ClubSmash (6f)", gif_dir / "preview_grunt_clubsmash_12fps.gif", LIMESTONE_RGB)
    save_clip_gif(
        loaded["grunt"][16:18] + [loaded["grunt"][0]] * 4,
        "Grunt Hit + Settle",
        gif_dir / "preview_grunt_hit_12fps.gif",
        OCHRE_RGB,
    )

    save_clip_gif(
        loaded["vfx_builder_repair_spark"] + [Image.new("RGBA", (128, 128), (0, 0, 0, 0))] * 2,
        "vfx_builder_repair_spark (6f)",
        gif_dir / "preview_vfx_repair_spark_12fps.gif",
        (32, 42, 34),
    )
    save_clip_gif(
        sync_frames,
        "Builder Repair + Spark Sync",
        gif_dir / "preview_builder_repair_with_spark_12fps.gif",
        MEADOW_RGB,
    )

    # Full 1080x1920, 720x1280, 360x640 composite gameplay animation sequence (48f = 4.0s @ 12 FPS)
    seq_1080 = render_gameplay_composite_sequence(loaded, n_ticks_12fps=48)
    seq_720 = [im.resize((720, 1280), Image.Resampling.BILINEAR) for im in seq_1080]
    seq_360 = [im.resize((360, 640), Image.Resampling.BILINEAR) for im in seq_1080]

    # Save 12 FPS crop GIF of the active bastion battle zone (x=180..760, y=460..1040 -> 580x580)
    crop_gifs = [
        im.crop((170, 480, 750, 1020)).quantize(colors=256, method=Image.Quantize.MEDIANCUT, dither=Image.Dither.NONE)
        for im in seq_1080[:24]
    ]
    crop_gifs[0].save(
        gif_dir / "preview_gameplay_composite_1080p_crop_12fps.gif",
        save_all=True,
        append_images=crop_gifs[1:],
        duration=durations_12fps(len(crop_gifs)),
        loop=0,
    )

    encode_mp4_from_pil_frames(seq_1080, mp4_dir / "preview_gameplay_1080x1920_h264_12fps.mp4", crf=20)
    encode_mp4_from_pil_frames(seq_720, mp4_dir / "preview_gameplay_720x1280_h264_12fps.mp4", crf=21)
    encode_mp4_from_pil_frames(seq_360, mp4_dir / "preview_gameplay_360x640_h264_12fps.mp4", crf=22)
    build_showcase_mp4(loaded, sync_frames, mp4_dir / "preview_pilot_all_clips_showcase_12fps.mp4")

    print("Atlas, Contact Sheets, 11 GIFs (12 FPS), and 4 H.264 MP4s generated successfully!")


if __name__ == "__main__":
    main()
