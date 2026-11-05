# TTGQLDL — ART-002 Repository Owner Visual Approval Package

- **Work Package:** `ART-002` (*Bastion Dawn Visual Direction Refinement, Character Model Sheets & Production Design Blueprint*)
- **Package Status:** **`PASS — OWNER APPROVED`**
- **Visual Direction Decision State:** **`APPROVED`** *(explicitly approved by repository owner on `2026-11-05T15:15:37+07:00`; `rights_status` remains `PENDING_REVIEW` pending `GATE-LEGAL`)*
- **Implementation Milestone `M06` State:** **`NOT_STARTED`** *(`ART-003` vertical-slice pilot authorized; full `M06` runtime & `124`-frame completion remains `NOT_STARTED`)*
- **Single-Board Executive Visual Summary:** `owner_approval_overview.png`

---

## 1. Executive Summary for Repository Owner

`ART-002` refines the recommended `ART-001` visual direction — **Direction A: Bastion Dawn (`bastion-dawn-v1`)** — into a complete, Unity-actionable production design blueprint without touching gameplay rules, backend code, schemas, or Unity runtime files.

You can inspect the entire visual design at a glance without reading any source code by opening these key visual plates inside `art/concepts/ART-002/`:

1. **Executive All-in-One Overview:** `owner_approval_overview.png`
2. **Before / After Refinement (`ART-001` vs `ART-002`):** `direction_refinement_comparison.png`
3. **Refined Master 9:16 Gameplay & Livestream Safe Overlay:** `ui/hud_master.png`, `ui/hud_master_livestream_overlay.png`, `ui/hud_safe_area_guide.png`
4. **Mobile `360 × 640` & BT.709 Grayscale Proofs:** `ui/hud_mobile_360x640.png`, `qa/qa_resolutions_sheet.png`, `qa/qa_grayscale_and_contrast_audit.png`
5. **All 7 Character Classes (4-View Model Sheets & `48 px` Matrix):** `characters/all_characters_48px_matrix.png` and `characters/<class>_model_sheet.png` (`builder`, `guard`, `fighter`, `runner`, `grunt`, `brute`, `boss`)
6. **Castle, Curtain Wall & 3-Lane Village World:** `environment/environment_overview.png`, `environment/castle_model_sheet.png`, `environment/wall_model_sheet.png`, `environment/three_lane_layout.png`
7. **Complete 16-State UX Board & 35-Piece UI Kit:** `ui/state_board.png`, `ui/components/icon_sheet.png`, `ui/components/component_sheet.png`

---

## 2. What You Are Reviewing & Approving

### 2.1 Final Visual Direction (`bastion-dawn-v1` — [01_visual_direction.md](01_visual_direction.md))
- **Warm, bright stylized fantasy** with uniform **`3.0 px` `#11151C` dark-ink outlines**, **two-step cel shading** under fixed **`45°` top-left sunlight**, and calm low-frequency terrain (`HSV S <= 48%`, **`50.0%` lower high-frequency Laplacian noise** than `ART-001`) engineered to survive mobile H.264 stream compression (`qa/qa_h264_compression_simulation.png`).

### 2.2 Playable Character Lineup & Enemy Cast ([02_character_model_sheets.md](02_character_model_sheets.md))
- **3 Playable Viewer Roles (`128×128` source, `48 px` 1080p height, `(0.50, 0.08)` foot pivot):**
  - **Builder (`XÂY`):** Golden-yellow bib apron (`#F4C430`) + oversized orthogonal iron/oak T-mallet (`characters/builder_model_sheet.png`).
  - **Guard (`THỦ`):** Royal cobalt-blue cape (`#2E7CF6`) + full-height steel-rimmed rectangular tower shield, zero sword (`characters/guard_model_sheet.png`).
  - **Fighter (`ĐÁNH`):** Crimson-red surcoat & helmet plume (`#E63946`) + `40°` diagonal steel broadsword (`characters/fighter_model_sheet.png`).
- **3 Normal Enemies + Center Boss:**
  - **Runner (Priority `1`):** Lean `44 px` violet sprinter (`#9B5DE5`) + swept ears + twin daggers (`characters/runner_model_sheet.png`).
  - **Grunt (Priority `2`):** Round `48 px` moss-green goblinoid (`#4CC964`) + diagonal oak club (`characters/grunt_model_sheet.png`).
  - **Brute (Priority `3`):** Wide `64 px` cubic siege-orange hulk (`#F3722C`) + iron helm + crusher gauntlets (`characters/brute_model_sheet.png`).
  - **Dread Bastion Warlord (Boss):** Towering `102 px` (`256×256` overscan canvas at identical `PPU = 192`) obsidian-violet warlord (`#1E102B` + `#D946EF` core) + crescent horns + concentric ground telegraph ring (`characters/boss_model_sheet.png`).

### 2.3 Castle, Wall & Three-Lane Environment ([03_environment_design.md](03_environment_design.md))
- Central circular curtain wall + cobalt-slate castle keep occupy **`477 px` (`44.2%` of `1080 px` width)** centered at `x = 454 px`, keeping `100%` of the wall, South gate, and East lane strictly clear of the `x > 82%` livestream icon rail.
- Includes **4 cosmetic Townhall level bands** (`Lv 1–9`, `10–24`, `25–39`, `40–50`; `townhall_hp = 500` unchanged), **5 Wall/Gate damage & scaffold states**, **6 low-frequency tiles**, **4 wave lighting palettes**, **6 non-blocking courtyard cottages**, and **3 courtyard props**.

### 2.4 HUD Master, Mobile Preview, Grayscale & 16 States ([04_ux_ui_and_states.md](04_ux_ui_and_states.md), [05_ui_component_kit.md](05_ui_component_kit.md))
- Default **Vietnamese (`vi-VN`)** interface (with English localization table) communicating Castle/Wall HP, 3-lane threat/timer, and the **single primary CTA `Bình luận XÂY / THỦ / ĐÁNH`** (`48 px` Bold = `16.0 px` @ `360×640`, `17.85:1` WCAG contrast).
- Zero HUD pixels enter `x > 82%` (`886..1079 px`) or `y > 88%` (`1690..1919 px`); minimum WCAG text contrast across all 12 HUD pairs is **`11.74 : 1`** (`>= 4.5 : 1` required).
- Covers all **16 authoritative states** (`ui/state_board.png`) with honest `MOCK / REPLAY` watermarks, non-viewer `P <= 10` militia disclosure, and safe halt banners for `SOURCE_RECONNECTING` and `STORAGE_UNAVAILABLE`.

---

## 3. Asset Inventory Summary, Production Risks & Rights Questions

### 3.1 Delivered Blueprint vs Future Production Queue ([asset_inventory.md](asset_inventory.md), [asset_manifest.json](asset_manifest.json))
- **Delivered & Verified on Disk in `ART-002`:** `119` real image files (`97` production-blueprint PNGs + `22` archived raw provenance JPGs), every one recorded with dimensions and SHA-256 checksums in [asset_manifest.json](asset_manifest.json).
- **Queued for Next Stage (`ART-003` -> `M06`, Gated on Approval):** `124` animated character sprite frames (`6 × 18f + 1 × 16f` @ `12 FPS`), sliced environment sprite/tile prefabs, `5` pooled VFX flipbooks (`<= 20` simultaneous bursts), and `SFX001` original audio cues.

### 3.2 Key Production Risks Identified in Consistency & QA Testing
1. **Cross-Seed AI Animation Drift ([consistency/consistency_report.md](consistency/consistency_report.md)):** Unconstrained text-to-image runs preserved color and class identity `21/21` times, **but** flipped `Runner` facing (`SW -> SE`) in `2/3` runs and varied `Fighter` shoulder pauldrons and `Builder` hammer grip. **Mitigation:** `ART-003` must animate from locked `Deliverable B` turnaround rigs / pose-controlled passes + fixed `16-color` LUT cleanup, never raw unconstrained text seeds.
2. **External Cold-Viewer & Broadcast Gates:** `QA-PSY-01` (`>= 5` cold human viewers) and real `OBS 32.1.x` capture (`GATE-OBS`) remain required before `M06` can claim final acceptance.

### 3.3 Outstanding Rights & Provenance Questions (`GATE-LEGAL`)
- All `ART-002` visuals are **100% original concepts** generated via Google Antigravity (`image-generator`) and Python (`Pillow`/`NumPy`) with zero third-party franchise characters, logos, or ripped game assets.
- Per repository governance (`09_QA_ACCEPTANCE.md` `GATE-LEGAL`), every generated asset in [asset_manifest.json](asset_manifest.json) is recorded as **`"rights_status": "PENDING_REVIEW"`** until the repository owner confirms commercial/livestream usage policy for AI-assisted assets and selects final `OFL-1.1` font files (`Noto Sans` / `Be Vietnam Pro`) for Unity bundling.

---

## 4. Explicit Decisions Requiring Repository-Owner Sign-Off

1. **Confirm `Bastion Dawn (bastion-dawn-v1)`** as the locked visual direction (retiring `ART-001` Directions B and C).
2. **Confirm the `256 × 256` Boss source frame exception** at identical `PPU = 192` (`128 × 128` for all 6 standard classes; `256 × 256` for Dread Bastion Warlord so `102 px` horns and cleaver never clip).
3. **Confirm the 4 cosmetic Townhall visual bands (`Lv 1–9`, `10–24`, `25–39`, `40–50`)** while keeping authoritative `townhall_hp = 500` unchanged.
4. **Confirm AI-assisted + 2D rig/cleanup workflow (`GATE-LEGAL` review)** before authorizing `ART-003` sprite animation pilot.

---

## 5. Repository Owner Approval Checklist (Signed Off by Repository Owner)

- **Approver:** Repository Owner (`hungdev98`) via explicit interactive approval gate prior to `ART-003` execution
- **Recorded Timestamp:** `2026-11-05T15:15:37+07:00` (`2026-11-05T08:15:37Z`)
- **Inspected Base Commit:** `32b963e47ee490979c3ff370cf6a5d3cddcdcc1e`
- **Scope & Governance Note:** Approves all 8 visual/technical blueprint items below (locking `Bastion Dawn v1`, `256 × 256` Boss cell at `PPU = 192`, and 4 cosmetic Townhall level bands), authorizes `ART-003` (`Builder` + `Grunt` + `vfx_builder_repair_spark` production pilot), and via explicit follow-up sign-off on `2026-11-05T16:51:17+07:00` (`2026-11-05T09:51:17Z`, inspected HEAD `24966d6a99dd868729ea5ee1ad99c4b7b1e89675`) authorizes `ART-004` (`Guard`, `Fighter` with crimson cloth shoulders / no steel pauldrons, `Runner`, `Brute`, `Boss`, and 4 remaining combat VFX = `112` frames) plus narrow `ART-003` `Idle`/validator hardening. Asset commercial/livestream legal clearance (`GATE-LEGAL`, `rights_status = PENDING_REVIEW`), `QA-PSY-01`, `GATE-OBS`, and `GATE-LIVE` remain separate unexecuted gates.

```text
[x] Approve Bastion Dawn visual direction
[x] Approve playable character designs
[x] Approve enemy and Boss designs
[x] Approve castle and environment
[x] Approve HUD and livestream layout
[x] Approve color and rendering rules
[x] Approve production asset specifications
[x] Approve transition to ART-003
```
