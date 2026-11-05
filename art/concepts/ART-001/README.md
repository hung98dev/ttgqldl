# TTGQLDL — `ART-001`: Visual Identity, Art Direction & Livestream UX Concept Package

**Repository:** `hung98dev/ttgqldl` · **Base HEAD Inspected:** `0dc74549802ba42c215bcfe6c94ced67da599ba9`  
**Binding Visual Authority:** [`docs/plan/07_UNITY_ART_UX.md`](../../../docs/plan/07_UNITY_ART_UX.md) · **Rules Contract:** [`docs/plan/02_GAME_RULES.md`](../../../docs/plan/02_GAME_RULES.md) (`ttg-rules-v1`)  
**Decision Status:** **`PROPOSED` — Owner Visual Approval Required (`ART-002`)**  
**M06 Implementation Status:** **`NOT_STARTED`** (no backend, DB, protocol, Unity runtime, or canonical plan files modified)

---

## 1. Executive Summary

`ART-001` delivers a complete, inspectable visual-direction and livestream UX concept package for **TTGQLDL**, a portrait `9:16` (`1080 × 1920` reference, `360 × 640` small-screen floor) Unity `6000.6.7f1` 2D pseudo-isometric cooperative village-defense game driven by livestream comments (`XÂY / THỦ / ĐÁNH`).

- **Three Distinct Directions Explored & Rendered (`1080 × 1920`, `360 × 640`, and BT.709 Grayscale):**
  1. **Direction A — Bright Stylized Fantasy (*"Bastion Dawn" / Bình Minh Thành Lũy*)** — **Weighted Score: `8.95 / 10` (`PROPOSED`)**
  2. **Direction B — Premium Chibi Fantasy (*"Hearthguard Chibi" / Làng Chibi Hộ Vệ*)** — **Weighted Score: `7.27 / 10`**
  3. **Direction C — Arcade Silhouette Fantasy (*"Tactical Crisp Bastion" / Chiến Tuyến Sắc Nét*)** — **Weighted Score: `8.44 / 10`**
- **Original 7-Character Cast (`Builder`, `Guard`, `Fighter`, `Runner`, `Grunt`, `Brute`, `Boss`):** Coherent `3.2`-head-tall stylized lineup with `48 px` color and BT.709 grayscale proof strips confirming silhouette and value survival.
- **Anchored UX & Safe-Zone Architecture:** Preserves right `x > 82%` and bottom `y > 88%` for TikTok LIVE / OBS chrome, achieves `12.57:1 – 18.39:1` WCAG text contrast (`>= 4.5:1` target), enforces `>= 16 px / 14 px / 12 px` effective text sizes at `360 × 640`, displays one unmistakable primary CTA (`Bình luận XÂY / THỦ / ĐÁNH`), keeps even-wave `A / B / C` voting visually subordinate, and labels non-live capture honestly as `MOCK / REPLAY`.

---

## 2. Package Contents

| Deliverable | File | Description |
|---|---|---|
| **Deliverable A** | [`01_visual_directions.md`](01_visual_directions.md) | Philosophy, HEX palettes + BT.709 luminance, line/shading rules, `48 px` & AI consistency risks, and rubric |
| **Deliverables B & C** | [`02_comparison_and_recommendation.md`](02_comparison_and_recommendation.md) | Shared `Wave 10` (`t = 262 s`) comparison scenario, `360p` & grayscale inspection evidence, weighted recommendation |
| **Deliverable D** | [`03_character_lineup.md`](03_character_lineup.md) | `7`-character visual breakdown (`Builder`, `Guard`, `Fighter`, `Runner`, `Grunt`, `Brute`, `Boss`) + `12 fps` animation spec |
| **Deliverable E** | [`04_hud_ux_system.md`](04_hud_ux_system.md) | Responsive coordinate grid, M01 `HudLayoutConfig.cs` Y-axis audit report, font/contrast rules, `16` presentation states |
| **Deliverable F** | [`05_art_style_guide.md`](05_art_style_guide.md) | Production style guide: `128 × 128` frames, `pivot=(0.50,0.08)`, `3 px` contours, `5` wall states, `<=20` VFX, naming |
| **Deliverable G** | [`06_unity_handoff.md`](06_unity_handoff.md) | `16`-field manifest contract, Unity `6000.6.7f1` `FilterMode.Point` & `<= 2048 × 2048` atlas targets, `<=150` pooling |
| **Provenance & Rights** | [`07_provenance_and_rights.md`](07_provenance_and_rights.md) | Original-content declaration, tool lineage, font licensing separation, and `GATE-LEGAL` human review status |
| **Asset Manifest** | [`manifest.json`](manifest.json) | Machine-readable ledger of all `22` verified existing `ART-001` concept & source image files |
| **Visuals — Dir A** | `images/direction_a_1080x1920.png` | Direction A primary concept (`+` `images/direction_a_360x640.png`, `images/direction_a_grayscale_1080x1920.png`, `images/direction_a_grayscale_360x640.png`) |
| **Visuals — Dir B** | `images/direction_b_1080x1920.png` | Direction B primary concept (`+` `images/direction_b_360x640.png`, `images/direction_b_grayscale_1080x1920.png`, `images/direction_b_grayscale_360x640.png`) |
| **Visuals — Dir C** | `images/direction_c_1080x1920.png` | Direction C primary concept (`+` `images/direction_c_360x640.png`, `images/direction_c_grayscale_1080x1920.png`, `images/direction_c_grayscale_360x640.png`) |
| **Visuals — Matrix** | `images/comparison_360x640_grayscale_sheet.png` | `1:1` pixel `360 × 640` color & BT.709 grayscale comparison matrix across A, B, and C |
| **Visuals — Cast** | `images/character_lineup.png` | Coherent `7`-character lineup + `48 px` color/grayscale thumbnails (`+` `images/character_closeups.png`) |
| **Visuals — HUD** | `images/hud_layout.png` | Annotated `1080 × 1920` safe-area layout blueprint & `10`-state presentation board |
| **Raw Sources** | `images/source/` | `6` preserved UI-free source illustrations (`world_direction_{a,b,c}_raw.jpg`, `character_lineup_raw.jpg`, close-ups) |

---

## 3. ART-001 Acceptance Checklist

- [x] Binding project requirements (`AGENTS.md`, `00_INDEX.md`, `01_PRODUCT_SCOPE.md`, `02_GAME_RULES.md`, `07_UNITY_ART_UX.md`, `09_QA_ACCEPTANCE.md`, `10_IMPLEMENTATION_MATRIX.md`, `11_EXTERNAL_GATES.md`) inspected and pre-existing `HudLayoutConfig.cs` / `SceneFactory.cs` layout discrepancies reported without touching runtime code.
- [x] Three meaningfully different visual directions documented with HEX codes, luminance, line/shading rules, risks, and weighted scoring.
- [x] Three **real** `1080 × 1920` gameplay concept images created and visually inspected.
- [x] Three `360 × 640` previews and BT.709 grayscale previews exported and checked.
- [x] Seven original character concepts (`Builder`, `Guard`, `Fighter`, `Runner`, `Grunt`, `Brute`, `Boss`) represented in a coherent unified-style lineup with close-ups and `48 px` proofs.
- [x] HUD layout includes critical statuses, single primary Vietnamese CTA, non-competing Vote UI, `MOCK / REPLAY` watermark, `16` presentation states, and `x > 82%` / `y > 88%` livestream safe areas.
- [x] One recommended direction (**Direction A — *Bastion Dawn***) selected with weighted comparison (`8.95` vs `8.44` vs `7.27`) and concrete rationale, clearly marked **`PROPOSED`**.
- [x] Actionable art style guide and Unity `6000.6.7f1` handoff specification created.
- [x] `manifest.json` contains **only** real existing deliverables (`22` verified image files) with provenance and rights notes.
- [x] Zero backend, gameplay, DB, Unity runtime, or authoritative plan files modified; parallel `TASK 036` workspace completely respected; nothing committed/pushed to `main`.
- [x] `M06` remains `NOT_STARTED`; owner approval (`ART-002`) remains pending before bulk asset production.
