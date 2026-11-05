# ART-004 — Independent Mechanical, Adversarial & Visual QA Report

- **Work Package:** `ART-004` (*Bastion Dawn v1 — Complete Character Animation & Combat VFX Production*)
- **Note on Identifier Disambiguation:** `ART-004` is an art-production work package under `art/production/ART-004/` and does **not** alter or claim acceptance for normative requirement `ART004` in `docs/plan/07_UNITY_ART_UX.md` or milestone `M06` (`NOT_STARTED`).
- **QA Verdict:** **PASS** (Art-Production Package Scope; Legal `PENDING_REVIEW`, `M06 = NOT_STARTED`)
- **Reproducible Validator:** `python -B art/production/ART-004/tools/validate_art004_qa_and_manifest.py` (`exit 0`)
- **ART-003 Hardened Regression Validator:** `python -B art/production/ART-003/tools/validate_art003_qa_and_manifest.py` (`exit 0`)

---

## 1. ART-003 Pre-Flight Audit & Hardening Summary

Before generating any `ART-004` frame, `ART-003` (`Builder`, `Grunt`, `vfx_repair_spark`) was audited against its own reported metrics:

1. **Idle Loop-Seam Discrepancy (`seam_ratio = 2.000` vs Claimed `[0.80, 1.35]`):**
   - **Root Cause:** `ART-003` originally animated 4-frame `Idle` clips with a symmetric `[0.0, 0.5, 1.0, 0.5]` triangle wave (`f01 == f03` torso/head height, differing only slightly in tool/club tilt), creating a wrap jump `2.0x` larger than internal half-steps while `validate_art003_qa_and_manifest.py` previously asserted `seam_ratio <= 2.15`.
   - **Fix Applied:** Rebuilt `Builder` and `Grunt` 4-frame `Idle` clips using genuine `90°` phase-shifted quadrature vertical/rotational breathing (`0°, 90°, 180°, 270°`) and tightened both `ART-003` and `ART-004` validators to enforce `0.75 <= seam_ratio <= 1.65` with zero duplicate frames.
   - **Post-Hardening Measured `ART-003` Seams:** `Builder Idle = 0.962`, `Builder Walk = 0.955`, `Grunt Idle = 0.901`, `Grunt Walk = 1.015` (all strictly inside `[0.75, 1.65]`).
2. **Planted-Foot Contact Honesty:**
   - Replaced silhouette bounding-box `max_y` proxies with explicit per-foot sole tracking (`front_foot_xy`, `back_foot_xy`) cross-checked directly against opaque PNG pixels (`alpha == 255`) within `2 px` of each reported sole anchor, verifying monotonic `-X` stance tread (`0.0 px` forward skate during stance) and real swing-foot ground clearance (`>= 2 px`).
3. **Documentation Alignment:**
   - Updated `art/production/ART-003/APPROVAL_RECORD.md` to state `"4 cosmetic Townhall level bands"` (matching canonical `docs/plan/02_GAME_RULES.md`) and added `7/7` negative self-tests + regenerated all `86` SHA-256 digests in `art/production/ART-003/asset_manifest.json`.

---

## 2. Mechanical Validation Matrix (15 Automated Checks)

Measured from `art/production/ART-004/qa/qa_metrics.json`:

| Check # | Verification Rule | Target / Threshold | Measured Result | Verdict |
|---|---|---|---|---|
| 01 | Remaining character production frame count | `88` frames (`18+18+18+18+16`) | `88` PNGs verified on disk | **PASS** |
| 02 | Remaining combat VFX production frame count | `24` frames (`4+4+6+10`) | `24` PNGs verified on disk | **PASS** |
| 03 | Total new real production frames | `112` frames (`RGBA`) | `112` PNGs verified | **PASS** |
| 04 | Sprite dimensions | `128x128` (`Guard`, `Fighter`, `Runner`, `Brute`, `3 VFX`), `256x256` (`Boss`, `Boss Slam VFX`) | `112/112` exact dimensions | **PASS** |
| 05 | Binary alpha integrity (`0` or `255` only) | `0` semi-transparent halo pixels (`0 < alpha < 255`) | `0` halo pixels across `112` frames | **PASS** |
| 06 | Border safety & overscan (`x=0, W-1`, `y=0, H-1`) | `0` border-clipped weapon/plume/horn/VFX frames | `0` clipped frames (`2 px` min margin) | **PASS** |
| 07 | Normalized ground/center pivots | Characters `[0.50, 0.08]`, radial/ground VFX `[0.50, 0.50]` | `112/112` verified | **PASS** |
| 08 | Character color discipline (`Bastion Dawn v1`) | `<= 16` quantized colors per character frame + `#11151C` contour | `<= 16` colors on all `88` character frames | **PASS** |
| 09 | Class silhouette & identity invariants | Fighter `0` steel shoulder pixels; Boss `0` baked ground ring pixels; Guard `0` handheld weapon; Brute `0` handheld weapon; Runner twin daggers | All invariants verified across `88` frames | **PASS** |
| 10 | Anatomical `3/4` `+X` facing lock | Zero accidental horizontal flips across all clips | `88/88` frames verified `+X` | **PASS** |
| 11 | Cyclic loop continuity (`9` cyclic loops) | `0.75 <= seam_ratio <= 1.65`, `0` repeated frames | Range `0.761` – `0.957` (`9/9` inside bounds) | **PASS** |
| 12 | Planted-foot anchoring & sole pixel cross-check | Idle slip `0.0 px`; stance monotonic `-X`; swing lift `>= 2 px` | `20/20` character clips grounded & verified | **PASS** |
| 13 | Sprite sheets (`9` sheets) & Master Atlas (`2048x2048`) | Byte-exact cell/UV match + `0` `4 px` gutter alpha bleed | `9/9` sheets + `112/112` slices exact, `0` bleed px | **PASS** |
| 14 | Animated previews (`28` GIFs + `4` H.264 MP4s) | True `12.0 FPS` (`83.33 ms` mean GIF delay; `r_frame_rate = 12/1` `yuv420p`) | `28/28` GIFs + `4/4` MP4s verified (`PSNR = 34.46 dB`) | **PASS** |
| 15 | Full package SHA-256 manifest integrity | `100%` file coverage + byte-size + SHA-256 match | Verified on every run | **PASS** |

---

## 3. Cyclic Loop Seam & Planted-Foot Measurements (`ART-004`)

| Clip (`12 FPS`) | Frames | `mean_step_l1` | `wrap_l1` | `seam_ratio` (`[0.75, 1.65]`) | Foot Anchoring Measurement |
|---|---:|---:|---:|---:|---|
| `guard.idle` | 4 | `5.349` | `4.185` | `0.782` | `0.0 px` `dx/dy` sole slip |
| `guard.walk` | 6 | `9.905` | `7.543` | `0.761` | Front tread `13.5 px`, Back tread `10.0 px`, Swing `y=114.0 px` |
| `fighter.idle` | 4 | `6.941` | `6.641` | `0.957` | `0.0 px` `dx/dy` sole slip |
| `fighter.walk` | 6 | `12.327` | `10.169` | `0.825` | Front tread `11.5 px`, Back tread `10.5 px`, Swing `y=114.0 px` |
| `runner.idle` | 4 | `3.133` | `2.986` | `0.953` | `0.0 px` `dx/dy` sole slip |
| `runner.run` | 6 | `10.869` | `8.554` | `0.787` | Front tread `19.5 px`, Back tread `14.0 px`, Swing `y=112.0 px` |
| `brute.idle` | 4 | `7.199` | `6.776` | `0.941` | `0.0 px` `dx/dy` sole slip |
| `brute.march` | 6 | `16.009` | `12.959` | `0.809` | Front tread `10.5 px`, Back tread `7.0 px`, Swing `y=114.0 px` |
| `boss.idlemarch` | 6 | `8.778` | `6.781` | `0.773` | Front tread `25.0 px`, Back tread `12.0 px`, Swing `y=229.0 px` |

---

## 4. Adversarial Validator Negative Tests (`11 / 11` Rejected)

Recorded in `art/production/ART-004/qa/adversarial_validator_results.json`. Every injected defect is executed in memory against the live validation functions and verified to raise a hard `AssertionError`:

1. `ADV-01_missing_frame` — Missing PNG path referenced in manifest -> **DETECTED_AND_REJECTED** (`Missing file referenced in manifest`)
2. `ADV-02_repeated_frame` — Duplicate `fighter.walk` `f01` into `f02` -> **DETECTED_AND_REJECTED** (`Repeated/duplicate consecutive frames at 1->2 (L1=0.0)`)
3. `ADV-03_malformed_alpha` — Semi-transparent `alpha=128` halo pixel injected -> **DETECTED_AND_REJECTED** (`Malformed alpha: 1 semi-transparent halo pixels`)
4. `ADV-04_border_weapon_clip` — Sword blade pixel painted at right border `x=127` -> **DETECTED_AND_REJECTED** (`Border clipping detected: bbox=(43, 23, 127, 117) inside 128x128`)
5. `ADV-05_wrong_sprite_dimension` — Sprite resized to `124x128` -> **DETECTED_AND_REJECTED** (`Wrong sprite dimension (124, 128), expected (128, 128)`)
6. `ADV-06_swapped_frame_index` — `Brute` march `f00` and `f01` swapped vs sprite sheet -> **DETECTED_AND_REJECTED** (`Sheet cell #4 (r=0, c=4) does not match frame #4`)
7. `ADV-07_corrupted_atlas_uv` — Master Atlas slice `#10` `uv_rect_unity.x` shifted by `+0.05` -> **DETECTED_AND_REJECTED** (`Corrupted atlas UV or pixel mismatch for chr_boss_warlord_cleaveslam_f04`)
8. `ADV-08_mismatched_hash` — Leading hex nibble flipped on `chr_boss_warlord_idlemarch_f00.png` SHA-256 -> **DETECTED_AND_REJECTED** (`Mismatched SHA-256`)
9. `ADV-09_broken_loop` — Non-periodic forward-drifting walk sequence that teleports on wrap -> **DETECTED_AND_REJECTED** (`[guard.walk] Broken cyclic loop seam: seam_ratio=2.963 outside [0.75, 1.65]`)
10. `ADV-10_incorrect_pivot` — Center pivot `[0.5, 0.5]` supplied for character frame -> **DETECTED_AND_REJECTED** (`Incorrect character pivot [0.5, 0.5], expected [0.5, 0.08]`)
11. `ADV-11_incompatible_clip_length` — `Fighter` `SwordSlash` truncated from `6` to `4` frames -> **DETECTED_AND_REJECTED** (`Incompatible clip length 4, expected 6`)

---

## 5. Multi-Scale Visual, Grayscale, Terrain & H.264 Stream Audit

All five diagnostic proof plates in `art/production/ART-004/qa/` were generated from decoded stream frames and live sprite arrays, then visually inspected:

- `art/production/ART-004/qa/qa_at_scale_gameplay_board.png` — Confirms exact gameplay pixel heights (`1080x1920` -> `360x640`): Heroes (`Builder`, `Guard`, `Fighter`) & `Grunt` = `48.0 px` (`16.0 px` @ `360p`), `Runner` = `44.0 px` (`14.7 px` @ `360p`), `Brute` = `64.0 px` (`21.3 px` @ `360p`), `Boss` = `102.0 px` (`34.0 px` @ `360p`). Broad pose, weapon/shield silhouette, and Boss ground rune remain legible at `16 px` mobile scale while keeping livestream safe zones (`x > 82%`, `y > 88%`) clear of gameplay obstruction.
- `art/production/ART-004/qa/qa_360x640_and_grayscale_proof.png` — Confirms `BT.709` luminance separation (`Y = 0.2126R + 0.7152G + 0.0722B`) over Meadow (`#68A347`), Ochre Lane (`#CFA96E`), and Limestone Boss Causeway (`#B8C0C8`) without relying solely on hue.
- `art/production/ART-004/qa/qa_h264_compression_proof.png` — Decoded `libx264 yuv420p` stream inspection (`1080p CRF 18` `PSNR = 34.46 dB`, `720p CRF 21`, `360p CRF 25`). `#11151C` dark-ink contours prevent `4:2:0` chroma bleed from erasing crimson, cobalt, amber, violet, or siege-orange role reads.
- `art/production/ART-004/qa/qa_temporal_flicker_and_onion_skin.png` — Cyan/magenta `f0` vs `f(N-1)` overlap plates across all `9` cyclic loops plus `< 3 Hz` VFX luminance audit (`0` full-screen strobes; `0` bright end-frame residue).
- `art/production/ART-004/qa/qa_foot_pivot_and_alpha_audit.png` — Binary alpha rim map (`0` red semi-transparent pixels) + per-frame `front_foot_xy` / `back_foot_xy` stance tread plots.

---

## 6. Isolated Unity `6000.6.7f1` Batchmode Verification & Honest Boundaries

Recorded in `art/production/ART-004/qa/unity_isolated_verification.json`:

- **Executed:** Headless `Unity 6000.6.7f1` (`-batchmode -nographics -quit`) in an isolated scratch project outside the repository working tree (`production_unity_directory_modified = false`).
- **Verified in Real Unity Engine:**
  - `154` individual sprites (`112` `ART-004` + `42` `ART-003`) imported at `PPU = 192`, `FilterMode.Point`, `mipmaps = false`, `Uncompressed`.
  - `ART004_Master_Atlas.png` (`2048x2048`) sliced into `112/112` `SpriteMetaData` rects with `0` gutter bleed pixels and `16,778,144 bytes` (`16.00 MiB`) measured Unity `Profiler.GetRuntimeMemorySizeLong(tex)` texture memory.
  - `Animator` + `AnimationClip.SampleAnimation` verified frame-exact `12 FPS` playback across all `7` classes (`0.0` `SpriteRenderer.flipX` world-space pivot drift).
  - Mock multi-class scene (`12` pooled renderers) verified `Units` Y-sort (`sortingOrder = Mathf.RoundToInt(-y * 100f)`), ground telegraph sorting behind feet (`GroundDecals`), and hit/slash/block VFX sorting above bodies (`VFXFront`).
- **Explicitly NOT Claimed (Honest Gate Status):**
  - Production `unity/Assets/_TTGQLDL/` scene wiring (`M06` remains `NOT_STARTED`).
  - Live `30 FPS` / `24 FPS 1% low` / `0 GC` soak acceptance under `150` live simulation entities (`QA-VIS-01` gated to `M06`).
  - Real `OBS` hardware stream capture (`GATE-OBS`) and external `N >= 10` human viewer comprehension testing (`QA-PSY-01`).
  - Final legal IP sign-off (`rights_status = PENDING_REVIEW` under `GATE-LEGAL`).
