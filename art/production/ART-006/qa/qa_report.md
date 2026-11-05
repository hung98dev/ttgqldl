# ART-006 — Production UI/HUD & 16-State Screen System QA Report

## 1. Executive Summary

| Gate / Verification Suite | Status | Evidence File |
|---|---|---|
| **35 UI Asset Pixel & 9-Slice Audit** | `PASS` (`31` byte-identical reuse, `4` surgical repairs) | `previews/nine_slice/component_repair_before_after.png` |
| **15-Skin Min/Nominal/Max & Near-Zero Shrink Stress** | `PASS` (`0.0` strip drift, `0` corner collisions down to `0%` HP) | `previews/nine_slice/nine_slice_stress_matrix.png` |
| **2× `1024×1024` RGBA32 UI Atlases** | `PASS` (`20` icons @ `24px` gutter; `15` skins @ `16px` gutter; `0` rotation; `0` alpha bleed) | [`atlas/ui_atlas_01_icons.json`](../atlas/ui_atlas_01_icons.json), [`atlas/ui_atlas_02_components.json`](../atlas/ui_atlas_02_components.json) |
| **16 Canonical States + 8 Compositional Combos** | `PASS` (`PREP = 0..1599`, `W0306` drop truth, `QA-OPS-04` blockers, Boss `>=2600`) | `hud/states/state_board_16_production.png`, `hud/overlays/overlay_combinations_8_board.png` |
| **Multi-Resolution Readability (`1080p`, `720p`, `360p`)** | `PASS` (`48` exports; `0` text overflows; CTA `16.0px` @ `360p`; WCAG `>= 12.3:1`) | `hud/mobile/mobile_stream_overlay_360p_720p.png` |
| **Vietnamese (`vi-VN`) & English (`en-US`) Localization** | `PASS` (`59/59` key parity; `0` missing diacritic glyphs; `12/12` stress cases pass) | `previews/nine_slice/typography_and_localization_stress_sheet.png`, `hud/composites/hud_en_us_localization_proof.png` |
| **Real FFmpeg H.264 (`libx264 yuv420p`) Stream Test** | `PASS` (`1080p @ 2.5 Mbps` PSNR `34.64 dB`; `360p @ 800 kbps` PSNR `31.71 dB`) | `previews/compression/h264_compression_readability_proof.png` |
| **BT.709 Grayscale, Colorblind & Reduced-Motion** | `PASS` (`Solid/Dots/Hatch` HP redundancy; `0` flashes `> 3 Hz`; `0` shake) | `previews/grayscale/grayscale_and_colorblind_redundancy_board.png`, `previews/grayscale/reduced_motion_accessibility_board.png` |
| **Isolated `Unity 6000.6.7f1 -batchmode` Verification** | `PASS` (`45/45` slice size checks, `24/24` fixtures, `0` missing sprites/glyphs, Prefab saved) | [`qa/unity_isolated_verification.json`](unity_isolated_verification.json) |
| **16 Mandatory Adversarial Negative Validator Cases** | `PASS` (`16/16` deliberate defects deterministically rejected) | [`qa/adversarial_validator_results.json`](adversarial_validator_results.json) |

---

## 2. Separation of Automated Measurements vs Visual Direction Judgment

### 2.1 Objective Automated Measurements (`validate_art006.py` & `Unity 6000.6.7f1`)
- **Icon Edge & Halo Cleanliness:** `0` semi-transparent border pixels across all `20` `128×128` icons; outer `1 px` border ring alpha = `0` on `20/20` icons (`>= 10 px` clear margin).
- **9-Slice Strip Constancy & Tile Periodicity:** `0.0` RGB/alpha pixel variance across top, bottom, and center stretch zones on all `13` `Sliced` skins; exact `8 px` horizontal period (`344 = 43 × 8 px`) verified pixel-identically on `ui_comp_hp_bar_fill_warn` and `ui_comp_hp_bar_fill_crit`.
- **Atlas Memory & Packing Integrity:**
  - `ui_atlas_01_icons.png` (`1024×1024` RGBA32): `24 px` gutters, `0` gutter bleed pixels, Unity `Profiler.GetRuntimeMemorySizeLong` = `4,195,232` bytes (`4.00 MiB`).
  - `ui_atlas_02_components.png` (`1024×1024` RGBA32): `16 px` gutters, `0` gutter bleed pixels, Unity `Profiler.GetRuntimeMemorySizeLong` = `4,195,232` bytes (`4.00 MiB`).
- **Safe-Zone & Zero-Occlusion Geometry:**
  - Across all `25` rendered telemetry screens (`16` canonical + `8` combos + `1` QA mobile overlay), `0` essential HUD panels exceed `x = 885.6 px` (`82%`) or `y = 1689.6 px` (`88%`), and `0` HUD panels intersect `castle_bbox` or `boss_bbox` (`intersection_area == 0`).
- **Typography & Contrast:**
  - `0` text overflows across `25` rendered HUD screens and `12` variable-length typography stress cases.
  - Minimum measured WCAG contrast ratio across all rendered HUD slots: **`12.3:1`** (requirement: `>= 4.5:1`).
  - `TMP_FontAsset.TryAddCharacters` in `Unity 6000.6.7f1` reported **`0` missing glyphs** across the full Vietnamese diacritic & HUD symbol set.
- **H.264 Mobile Video Compression (`ffmpeg libx264 -pix_fmt yuv420p`):**
  - `1080×1920 @ 2500 kbps` (`hud_stream_1080p_2500kbps.mp4`): **`34.64 dB` PSNR**.
  - `360×640 @ 800 kbps` (`hud_stream_360p_800kbps.mp4`): **`31.71 dB` PSNR**.

### 2.2 Visual Art Direction Assessment (Inspected Directly on Output PNGs)
- **Cohesive Game Interface (Not a Dashboard):** Royal-navy cel-beveled panels, `#11151C` ink outlines, and sun-gold CTA borders match the exact stroke weight and `45°` top-left lighting of `ART-005` limestone walls and `ART-003/004` character sprites.
- **`ART-005` Vertical Balance Resolved:** Introducing the translucent Lower Command Dock console (`y=1390..1688`), single-slot Join Toast (`y=1330..1388`), and Defender Spotlight Ribbon (`y=1268..1320`) eliminates the ungrounded lower-meadow feel observed in HUD-less `ART-005` previews while keeping `146 px` of forecourt breathing room below the South Gate and `0` world-coordinate changes.

---

## 3. Explicit Open Gates & Non-Claims

1. **`M05` & `M06` Runtime Integration Not Claimed:** Backend `M04` is still active and `M05` HTTP/WebSocket broadcast endpoints are not yet implemented; `M06` remains `NOT_STARTED`. All `ART-006` screens are driven by offline synthetic fixtures (`"synthetic_fixture": true`).
2. **`GATE-OBS` Hardware/Stream Soak Open:** Local `ffmpeg libx264 yuv420p` encode/decode proves mobile bitrate legibility, not a 60-minute live `OBS` + `TikTok LIVE` broadcast capture.
3. **`QA-PSY-01` Human First-Time Viewer Comprehension Open:** Automated typography/contrast/hierarchy checks pass, while live human 3-second comprehension testing remains an explicit acceptance gate during `M06/M07`.
4. **`GATE-LEGAL` Rights Sign-Off Open:** All deliverables carry `rights_status = "PENDING_REVIEW"`.
