# ART-003 Programmatic & Visual QA Report

## 1. Executive QA Summary

- **Overall `ART-003` Gate Status:** **`PASS`** (pilot scope: `Builder` + `Grunt` + `vfx_builder_repair_spark` + `Characters_Pilot_Atlas`; `M06` full milestone remains `NOT_STARTED`).
- **Machine-Readable Metrics File:** `art/production/ART-003/qa/qa_metrics.json`
- **Isolated Unity `6000.6.7f1` Batchmode Proof:** `art/production/ART-003/qa/unity_isolated_verification.json` (`42/42` frames + `1024×1024` atlas `PASS`).

| QA Dimension | Requirement | Verified Result | Status |
|---|---|---|---|
| Frame Count & Dimensions | `42` PNGs (`18 + 18 + 6`) @ `128×128` RGBA | `42/42` exact `128×128` RGBA PNGs | **PASS** |
| Packed Sheets & Pilot Atlas | `3` sheets + `<= 2048×2048` atlas | `768×384` (×2), `768×128` (×1), `1024×1024` atlas | **PASS** |
| Binary Alpha & Halo Cleanup | `0` semi-transparent edge halo pixels | `0 px` semi-transparent across all `42/42` frames | **PASS** |
| Cell Clipping & Margin | `0` clipped action poses | All `42` bboxes inside `[22..122, 6..123]` (`>= 4 px` margin) | **PASS** |
| Palette Stability (`<= 16`) | Controlled `<= 16`-color character palettes | `Builder = 16`, `Grunt = 15`, `VFX = 8` exact colors | **PASS** |
| Outer Ink Contour (`#11151C`) | Crisp dark ink contour on all character frames | `36/36` character frames verified with `#11151C` contour | **PASS** |
| VFX Dissipation | `0` residual bright pixels at end frame | `0` bright pixels on `f04` and `f05`; `86.1%` Builder unobscured | **PASS** |
| Non-Static / Non-Duplicate | Every adjacent pair visibly animates | Min pixel delta = `206 px` (`VFX`), `1,697–3,014 px` (`Chars`) | **PASS** |
| Foot-Pivot & Planted Stance | Pivot `(0.50, 0.08)` -> `(64, 117)`, `0` Idle slip, monotonic Walk tread | `Idle` front/back drift = `0.0/0.0 px`; `Walk` stance tread = `10.5/7.0 px` (`Builder`), `12.5/9.5 px` (`Grunt`) | **PASS** |
| Cyclic Loop Seam (`12 FPS`) | `0.75 <= seam_ratio <= 1.65` on all cyclic loops (`Idle` & `Walk`) | `Idle` = `0.962` / `0.901`; `Walk` = `0.955` / `1.015` (all strictly inside `[0.75, 1.65]`) | **PASS** |
| Multi-Scale & Grayscale | Readable at `1080p` (`48 px`), `720p` (`32 px`), `360p` (`16 px`) & Gray | Verified in `qa_at_scale_gameplay_board.png` & `qa_360x640_and_grayscale_proof.png` | **PASS** |
| H.264 Stream Proxy | Representative `libx264 yuv420p` proxy test | `1080p CRF 20 = 35.49 dB`, `CRF 33 = 32.84 dB`, `360p = 33.50 dB` | **PASS (Proxy)** |
| Negative Self-Tests (`7/7`) | Validator must reject deliberately injected broken assets | `7/7` synthetic defects (`<0.75` seam, `>1.65` seam, Idle slip, Walk skate, fake foot, alpha halo, border clip) caught | **PASS** |

---

## 2. All 10 Mandatory Failure Tests (`ART-003 §5`)

1. **Builder hammer changing shape between frames:** **PASS (0% mutation).** The oversized iron T-hammer is driven by a locked rigid cutout layer with deterministic FK angles; mallet geometry is invariant across all `18` frames.
2. **Grunt club changing size:** **PASS (0% mutation).** The `68 px` spiked wooden club is driven by a single locked rigid layer; it never morphs into an axe or sword or changes shaft length.
3. **Idle animation creating artificial horizontal sliding:** **PASS (`0.0 px` per-foot drift).** Individual sole audits verify `Builder Idle` front/back feet remain anchored at `[52.5, 117.0]` and `[73.0, 117.0]` (`0.0 px` slip), and `Grunt Idle` front/back feet remain anchored at `[47.0, 117.0]` and `[78.0, 117.0]` (`0.0 px` slip), cross-checked against opaque PNG alpha pixels.
4. **Walk feet skating:** **PASS (Genuine Per-Foot Tread Verified).** `verify_planted_foot_motion()` tracks `front_foot_xy` and `back_foot_xy` separately: during front stance (`f00–f02`), the grounded boot treads monotonically backward (`60.0 -> 56.0 -> 49.5 px` on `Builder`; `55.5 -> 51.0 -> 43.0 px` on `Grunt`) while the swing boot lifts (`y = 114`) and advances; during back stance (`f03–f05`), the rear boot treads monotonically backward (`79.0 -> 76.0 -> 72.0 px` on `Builder`; `84.5 -> 82.0 -> 75.0 px` on `Grunt`) with zero forward skating.
5. **Character head shrinking or expanding:** **PASS (`0%` scale change).** Head sprites use uniform `1.0` scale across all `18` frames (`Builder` `3.2`-head ratio, `Grunt` `3.0`-head ratio).
6. **Shadow / light direction changing:** **PASS.** `45°` top-left warm sunlight cel bands remain fixed across all poses, paired with a consistent `#11151C` ground contact shadow ellipse.
7. **Action poses leaving the `128×128` cell:** **PASS.** Highest overhead weapon reach is `y = 13` (`Builder Repair f02`) and `y = 6` (`Grunt ClubSmash f01`); rightmost strike reach is `x = 108` (`Builder`) and `x = 122` (`Grunt`). Zero border pixels touched across all `42` frames.
8. **Loop seam between final and first frames:** **PASS (`0.901–1.015`, strictly inside `[0.75, 1.65]`).** `Builder Idle` = `0.962` (`1,662 / 1,728.0 px`), `Builder Walk` = `0.955` (`2,421 / 2,533.8 px`), `Grunt Idle` = `0.901` (`2,043 / 2,268.0 px`), and `Grunt Walk` = `1.015` (`3,410 / 3,361.2 px`).
9. **Repeated or near-identical frames falsely presented as animation:** **PASS.** Zero duplicate frames exist anywhere in the `42` production PNGs (`1,697–4,046` changed pixels between every adjacent character frame pair; `206–1,511` changed pixels across `VFX` dissipation).
10. **VFX obscuring character silhouette:** **PASS.** At peak impact (`Builder Repair f04` + `vfx_builder_repair_spark f01` offset `(+34, +26)`), **`86.12%`** of Builder's opaque silhouette remains completely uncovered.

---

## 3. ART-004 Phase 0 Audit & Validator Hardening

During the `ART-004` Phase 0 audit, three narrow QA/animation gaps in `ART-003` were identified and repaired:

1. **Double-Reversal `Idle` Curves & Loop-Seam Enforcement (`0.420` / `0.445` -> `0.962` / `0.901`):** Initial `ART-003` `Idle` curves (`pelvis_dy = [0.0, +0.8, -0.6, +0.3]`) reversed vertical velocity twice inside 4 frames (`f01 -> f02` jumped from full exhale to inhale while `f03 -> f00` barely moved), while `validate_art003_qa_and_manifest.py` only enforced the upper bound `seam_ratio <= 1.65`. Both `Idle` clips now use smooth single-cycle quadrature breathing (`sin` torso breath + `cos` head/arm overlap), and the validator strictly enforces `0.75 <= seam_ratio <= 1.65`.
2. **Genuine Planted-Foot Movement Tracking:** Replaced static combined ground-band `Y` checks with `verify_planted_foot_motion()`, measuring individual `front_foot_xy` and `back_foot_xy` sole coordinates cross-checked against rendered PNG opaque pixels, `<= 0.5 px` `Idle` slip, and monotonic backward stance tread + swing lift on `Walk`.
3. **Approval Record Terminology & Built-in Negative Self-Tests:** Corrected `"4 cosmetic Townhall damage states"` in [APPROVAL_RECORD.md](../APPROVAL_RECORD.md) to `"4 cosmetic Townhall level bands (Lv 1–9, 10–24, 25–39, 40–50)"` matching `ART-002`, and added `7` in-memory negative self-tests in `validate_art003_qa_and_manifest.py`.

---

## 3. Defects Caught & Repaired During Production Iteration (`v1 -> v2`)

During visual inspection of Pass 1 (`v1`) sheets, five concrete visual defects were identified and repaired in `build_art003_rigs_and_frames.py` before final export:

1. **Inverted FK Strike Sign & Left-Border Clipping (`v1` Defect):** Pass 1 wrist/upper-arm rotation angles swung Grunt's club `2 px` past the left border (`x = 0`) on `ClubSmash f02` and angled downward strikes toward bottom-left while the motion smear sat on top-right. **Fix:** Corrected FK shoulder/elbow/wrist kinematics so `f01–f02` raise overhead (`y >= 6`) and `f03–f04` strike forward-right (`x <= 122`), backed by an automated `assert not border_clipped` gate.
2. **Enclosed Studio-Grey Hole & Cross-Material Cutout Leaks (`v1` Defect):** Border flood-fill missed an enclosed low-chroma grey pocket between Grunt's inner thigh and club, plus a yellow apron sliver on Builder's hammer top. **Fix:** Added interior neutral-hole stripping (`diff < 24` and `chroma < 16`) and material-specific color leak filters.
3. **Apron Patch Seam & Straight Chest Line (`v1` Defect):** Reconstructed apron behind Builder's lifted hammer was slightly darker than the lower skirt, and Grunt's chest crease looked overly straight. **Fix:** Sampled the exact unobstructed right-apron RGB median and replaced Grunt's chest line with a curved 5-point goblin pectoral contour.
4. **Blocky Arc Smear (`v1` Defect):** Polygon motion smear looked thick at the tail. **Fix:** Implemented `draw_tapered_cel_smear()` with a sleek tapered circular crescent leading directly into the striking mallet/club head.
5. **Grunt Face Occlusion on `ClubSmash f01–f02` (`v1` Defect):** Overhead raise crossed in front of Grunt's left eye. **Fix:** Adjusted upper-arm raise (`-88°..-92°`) so the club lifts cleanly beside the right shoulder, keeping Grunt's snarling face, eyes, and tusks `100%` visible.

---

## 4. Visual QA Plates & Limitations

All static diagnostic boards are located in `art/production/ART-003/qa/` and can be inspected without opening Unity:

- `qa_at_scale_gameplay_board.png` — `1080×1920` (`48 px`), `720×1280` (`32 px`), and `360×640` (`16 px`) 1:1 native pixel crops + Meadow/Ochre/Limestone terrain contrast matrix.
- `qa_360x640_and_grayscale_proof.png` — Side-by-side `360×640` RGB & BT.709 Grayscale + 6-frame `Repair` and `ClubSmash` `16 px` silhouette breakdown.
- `qa_h264_compression_proof.png` — Real `ffmpeg libx264 yuv420p` stream compression comparison (`CRF 20` vs congested `CRF 33`).
- `qa_temporal_flicker_and_onion_skin.png` — Multi-frame onion-skin weapon/gait arcs + per-clip frame-delta & loop-seam bar charts.
- `qa_foot_pivot_and_alpha_audit.png` — Ground pivot `(64, 117)` alignment plot + binary alpha audit + extracted `16`/`15`-color swatches.

**Honest Limitations:**
- H.264 `libx264 yuv420p` tests are software stream proxies, **not** substitutes for real OBS hardware encode + mobile livestream capture (`GATE-OBS` in [09_QA_ACCEPTANCE.md](../../../../docs/plan/09_QA_ACCEPTANCE.md)).
- Only `Builder` (`18f`), `Grunt` (`18f`), and `vfx_builder_repair_spark` (`6f`) are animated in this vertical slice; `Guard`, `Fighter`, `Runner`, `Brute`, `Boss`, environment sprites, and `M06` runtime scene wiring remain future work.
