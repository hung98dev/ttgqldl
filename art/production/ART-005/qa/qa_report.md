# ART-005 — Independent Mechanical, Adversarial & Visual QA Report

## 1. Executive Summary

- **Package:** `ART-005` (`art/production/ART-005/`)
- **Validator Status:** `PASS` (`python -B art/production/ART-005/tools/validate_art005.py`)
- **Isolated Engine Verification:** `PASS` (`Unity 6000.6.7f1 -batchmode -nographics -quit`, recorded in `qa/unity_isolated_verification.json`; production `unity/` directory untouched)
- **Adversarial Negative Suite:** `12 / 12` corrupted inputs deterministically rejected (`qa/adversarial_validator_results.json`)
- **Milestone & Gate Honesty:** `M06` remains `NOT_STARTED` in [10_IMPLEMENTATION_MATRIX.md](../../../../docs/plan/10_IMPLEMENTATION_MATRIX.md); `GATE-LEGAL`, `GATE-OBS`, live Unity `QA-VIS-01` FPS/GC soak, and human `QA-PSY-01` tests remain open.

---

## 2. Mechanical & Visual QA Results

| Verification Domain | Measured Result | Threshold / Contract | Status |
|---|---|---|---|
| Canonical Sprite Inventory | `26` canonical + `14` companion sprites (`40` total) | Exact dimensions & RGBA mode | `PASS` |
| Cutout Alpha Integrity | `0` semi-transparent halo pixels across all cutouts | Binary `0 / 255` alpha | `PASS` |
| Border Clearance | `>= 4 px` (`>= 10 px` on all 20 canonical cutouts) | `>= 2 px` non-wrapping border margin | `PASS` |
| Castle Progression Swap Drift | `0.0 px` (`y_max = 461` across `Bands 1–4`, `0.0000 u` in Unity) | `0.0 px` jump; `townhall_hp = 500` invariant | `PASS` |
| Wall Outer Span & Transparency | `477 px` (`44.17%` of `1080p`), transparent interior/exterior | `~44%` bastion span; split `BastionBack` + `BastionFront` | `PASS` |
| Wall HP State Mapping | Verified across all `20` wall tiers (`1000..4800` HP) + recovery | Exact `60% / 25% / >0% / 0 / recovery` priority | `PASS` |
| Terrain Calmness & Zero Ground Ink | `0` `#11151C` ground pixels; `HSV S <= 46.3%`; `Laplacian SD < 16.0` | `HSV S <= 48%`, low H.264 noise | `PASS` |
| Seamless Tile & Verge Transitions | `0.0–2.0` RGB edge diff (`tile_01 <-> tile_02`, `tile_01 -> tile_04 -> tile_03`) | `<= 14.0` max boundary diff | `PASS` |
| Unity `2048×2048` Atlases (`2×`) | `20 + 20 = 40` byte-exact slices, `0` gutter bleed, `32.0 MiB` total | `<= 2048×2048`, `Point`, `mipmaps = false` | `PASS` |
| Safe Zones (`x <= 82%`, `y <= 88%`) | East lane max `874 px <= 885.6 px`; max `y < 1689.6 px` | Zero intrusion into TikTok chrome/comment zones | `PASS` |
| H.264 `yuv420p` Stream Readability | Real `ffmpeg libx264` `2.5 Mbps` encode/decode (`PSNR ~ 38.4 dB`) | Crisp `48 px` units & `102 px` Boss at `1080p/720p/360p` | `PASS` |
| SHA-256 `asset_manifest.json` | 100% tracked files verified (`bytes` + `sha256`) | Exact cryptographic match | `PASS` |

---

## 3. Iterative Defects Caught & Repaired During Production

1. **Townhall `Band 4` Spire & Watchtower Roof Top Margin:** Initial pass clipped `env_castle_band4_lv40_50.png` and `env_bldg_watchtower.png` at `y = 0`. Calibrated belfry/spire and watchtower upper shaft heights so both sprites maintain `17 px` top border clearance while preserving vertical progression hierarchy.
2. **Circular Wall Equatorial Flank Seam & Front-Slice Crop:** Replaced piecewise North/South rampart heights with continuous `wall_h(theta) = 160 - 52 * sin(theta)` and expanded `env_wall_state*_front.png` to `512×256` (`crop y = 208..464`), eliminating flank notches and ensuring `>= 4 px` slice clearance.
3. **Courtyard Horseshoe Visibility & Castle Roof Deconfliction:** Repositioned all 6 courtyard cottages inside the inner wall ellipse (`y = 804..898`) so all 6 cottages + Watchtower + Townhall + 3 Props are 100% visible, and moved Center Causeway defenders to `y <= 594` and South Gate (`y = 1034`) so zero units overlap the Townhall roof.
4. **HSV Saturation & Limestone Mortar Laplacian Tuning:** Calibrated meadow/ochre tokens to strict `HSV S <= 46.3%` (`< 48%`) and softened limestone causeway mortar contrast (`CAUSEWAY_SEAM = (154, 163, 174)`) so `validate_art005.py` passes `Laplacian SD < 16.0` without relaxing any QA threshold.

---

## 4. Twelve Adversarial Negative Tests (`qa/adversarial_validator_results.json`)

All 12 injected corruptions were verified to raise `ValidationFailure`:
1. `01_missing_castle_sprite`
2. `02_incorrect_wall_damage_mapping`
3. `03_townhall_level_band_mismatch`
4. `04_corrupted_terrain_tile_seam`
5. `05_non_transparent_building_background`
6. `06_wrong_pivot`
7. `07_atlas_uv_corruption`
8. `08_sprite_clipped_at_border`
9. `09_incorrect_sha256_hash`
10. `10_sorting_layer_hides_boss`
11. `11_safe_zone_violation_x_gt_82pct`
12. `12_broken_road_to_meadow_transition`
