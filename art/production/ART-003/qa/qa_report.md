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
| Non-Static / Non-Duplicate | Every adjacent pair visibly animates | Min pixel delta = `206 px` (`VFX`), `1,702–3,014 px` (`Chars`) | **PASS** |
| Foot-Pivot & Zero Sliding | Pivot `(0.50, 0.08)` -> `(64, 117)`, `0` Idle slip | `Idle` `foot_y = 117` constant, `X` slip = `0.0 px` (`67.0` / `71.0`) | **PASS** |
| Cyclic Loop Seam (`12 FPS`) | Smooth `last -> first` transition on `Idle` & `Walk` | `Walk` seam ratio = `0.955` (`Builder`), `1.015` (`Grunt`) | **PASS** |
| Multi-Scale & Grayscale | Readable at `1080p` (`48 px`), `720p` (`32 px`), `360p` (`16 px`) & Gray | Verified in `qa_at_scale_gameplay_board.png` & `qa_360x640_and_grayscale_proof.png` | **PASS** |
| H.264 Stream Proxy | Representative `libx264 yuv420p` proxy test | `1080p CRF 20 = 35.48 dB`, `CRF 33 = 32.81 dB`, `360p = 33.51 dB` | **PASS (Proxy)** |

---

## 2. All 10 Mandatory Failure Tests (`ART-003 §5`)

1. **Builder hammer changing shape between frames:** **PASS (0% mutation).** The oversized iron T-hammer is driven by a locked rigid cutout layer with deterministic FK angles; mallet geometry is invariant across all `18` frames.
2. **Grunt club changing size:** **PASS (0% mutation).** The `68 px` spiked wooden club is driven by a single locked rigid layer; it never morphs into an axe or sword or changes shaft length.
3. **Idle animation creating artificial horizontal sliding:** **PASS (`0.0 px` drift).** `Builder Idle` foot-band median `X` stays at `[67.0, 67.0, 67.0, 67.0]` and `foot_y` at `[117, 117, 117, 117]`; `Grunt Idle` stays at `[71.0, 71.0, 71.0, 71.0]` and `[117, 117, 117, 117]`.
4. **Walk feet skating:** **PASS.** Alternating stance contacts (`f00–f01` and `f03–f04`) plant cleanly along the `y = 117..119` ground plane while the passing leg lifts (`2.5–3.2 px`) with clear heel-toe alternation.
5. **Character head shrinking or expanding:** **PASS (`0%` scale change).** Head sprites use uniform `1.0` scale across all `18` frames (`Builder` `3.2`-head ratio, `Grunt` `3.0`-head ratio).
6. **Shadow / light direction changing:** **PASS.** `45°` top-left warm sunlight cel bands remain fixed across all poses, paired with a consistent `#11151C` ground contact shadow ellipse.
7. **Action poses leaving the `128×128` cell:** **PASS.** Highest overhead weapon reach is `y = 13` (`Builder Repair f02`) and `y = 6` (`Grunt ClubSmash f01`); rightmost strike reach is `x = 108` (`Builder`) and `x = 122` (`Grunt`). Zero border pixels touched across all `42` frames.
8. **Loop seam between final and first frames:** **PASS.** `Builder Walk` `f05 -> f00` changes `2,421 px` vs intra-clip mean `2,533.8 px` (`ratio = 0.955`); `Grunt Walk` `f05 -> f00` changes `3,413 px` vs intra-clip mean `3,361.2 px` (`ratio = 1.015`).
9. **Repeated or near-identical frames falsely presented as animation:** **PASS.** Zero duplicate frames exist anywhere in the `42` production PNGs (`1,702–4,042` changed pixels between every adjacent character frame pair; `206–1,511` changed pixels across `VFX` dissipation).
10. **VFX obscuring character silhouette:** **PASS.** At peak impact (`Builder Repair f04` + `vfx_builder_repair_spark f01` offset `(+34, +26)`), **`86.12%`** of Builder's opaque silhouette remains completely uncovered.

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
