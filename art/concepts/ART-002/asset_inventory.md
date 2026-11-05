# ART-002 Deliverable G — Production Asset Inventory & Manifest Guide

- **Work Package:** `ART-002`
- **Machine-Readable Manifest:** [asset_manifest.json](asset_manifest.json)
- **Style Version:** `bastion-dawn-v1`
- **Rights & Provenance Status:** All generated files carry **`"rights_status": "PENDING_REVIEW"`** (`GATE-LEGAL` remains pending owner/legal sign-off; zero third-party copyrighted material used).

---

## 1. Summary of Verified Existing `ART-002` Assets on Disk

Every entry in [asset_manifest.json](asset_manifest.json) points to an **actual file verified on disk** with exact pixel dimensions, byte size, and SHA-256 digest (`119` verified image files total; zero phantom/non-existent `source_file` entries):

| Subdirectory / Group | Count | Format | Status in Manifest | Purpose |
|---|---:|---|---|---|
| `characters/sprites/` | `7` | `PNG RGBA` (`128×128` ×6, `256×256` Boss) | `APPROVAL_PENDING` | Canonical single-frame `3/4` gameplay reference cutouts with `(0.50, 0.08)` pivot & hard outer alpha |
| `characters/*.png` | `22` | `PNG RGB` | `APPROVAL_PENDING` | `7` 4-view model sheets, `7` equipment/point-filter detail sheets, `7` multi-scale grayscale sheets, `1` 7-class `48 px` matrix |
| `characters/source/*.jpg` | `9` | `JPEG RGB` | `CONCEPT` | Unedited raw AI turnaround (`7`) and weapon/headgear detail (`2`) provenance plates |
| `environment/*.png` | `7` | `PNG RGB` | `APPROVAL_PENDING` | Overview (`12` elements), Castle (`4` bands), Wall (`5` states), 3-Lane layout, Tilesheet (`6` tiles + `4` palettes), Buildings (`1+6+1+3`), Scale guide |
| `environment/source/*.jpg` | `7` | `JPEG RGB` | `CONCEPT` | Unedited raw AI village, siege, recovery, castle, wall, building, and tile provenance plates |
| `ui/components/ui_icon_*.png` | `20` | `PNG RGBA` (`128×128`) | `READY_FOR_PRODUCTION` | Individual text-free `3.0 px`-inked gameplay & system icons |
| `ui/components/ui_comp_*.png` | `15` | `PNG RGBA` | `READY_FOR_PRODUCTION` | Individual text-free 9-sliceable HUD containers, HP fills, role plaques, pills, CTA, and toast |
| `ui/components/*_sheet.png` | `2` | `PNG RGB` | `APPROVAL_PENDING` | Master `icon_sheet.png` and `component_sheet.png` proof boards |
| `ui/*.png` | `15` | `PNG RGB` (`1080×1920`, `360×640`, boards) | `APPROVAL_PENDING` | `12` full-screen HUD state compositions, `1` `360×640` mobile export, `1` safe-area guide, `1` `16`-state board |
| `consistency/` & `consistency/source/` | `9` | `3 PNG` + `6 JPEG` | `CONCEPT` | `21`-variation (`7×3`) cross-seed repeatability test boards & raw runs |
| `qa/` + Root Overviews | `6` | `PNG RGB` | `CONCEPT` / `APPROVAL_PENDING` | `4` automated Visual QA diagnostic sheets + `direction_refinement_comparison.png` + `owner_approval_overview.png` *(plus `qa_metrics.json`)* |
| **Total Verified Image Assets** | **`119`** | **97 PNG + 22 JPEG** | **100% On-Disk & SHA-256 Verified** | Complete `ART-002` blueprint & provenance archive |

---

## 2. Next-Stage Production Asset Queue (`ART-003` -> `M06`, Gated on Owner Approval)

Per `ART-002 §15`, bulk animation frames and final packed Unity `.spriteatlasv2` binaries are **intentionally not mass-produced prior to owner approval**. Once the repository owner signs off [OWNER_APPROVAL.md](OWNER_APPROVAL.md), the exact `M06` production asset queue is:

### 2.1 Character Sprite Sheets (`124` Total Frames @ `12 FPS`)

| Class | Priority | Source Cell | Clips & Frame Counts (`12 FPS`) | Total Frames | Target Unity Path |
|---|---|---:|---|---:|---|
| **Builder (`XÂY`)** | `P0` | `128 × 128` | `Idle (4) + Walk (6) + Repair (6) + Hit (2)` | `18` | `unity/Assets/Art/Characters/Builder/builder_sheet.png` |
| **Guard (`THỦ`)** | `P0` | `128 × 128` | `Idle (4) + Walk (6) + ShieldBrace (6) + Hit (2)` | `18` | `unity/Assets/Art/Characters/Guard/guard_sheet.png` |
| **Fighter (`ĐÁNH`)** | `P0` | `128 × 128` | `Idle (4) + Walk (6) + SwordSlash (6) + Hit (2)` | `18` | `unity/Assets/Art/Characters/Fighter/fighter_sheet.png` |
| **Runner** | `P0` | `128 × 128` | `Idle (4) + Run (6) + DaggerLunge (6) + Hit (2)` | `18` | `unity/Assets/Art/Characters/Runner/runner_sheet.png` |
| **Grunt** | `P0` | `128 × 128` | `Idle (4) + Walk (6) + ClubSmash (6) + Hit (2)` | `18` | `unity/Assets/Art/Characters/Grunt/grunt_sheet.png` |
| **Brute** | `P0` | `128 × 128` | `Idle (4) + March (6) + GauntletCrush (6) + Hit (2)` | `18` | `unity/Assets/Art/Characters/Brute/brute_sheet.png` |
| **Dread Bastion Warlord** | `P0` | `256 × 256` | `IdleMarch (6) + CleaveSlam (8) + Hit (2)` | `16` | `unity/Assets/Art/Characters/Boss/boss_sheet.png` |
| **Total Character Frames** | — | Max `2048×2048` atlas | — | **`124`** | `Characters_P0.spriteatlasv2` (`<= 150` pooled entities) |

### 2.2 Sliced Environment Prefab Sprites (`26` Static Sprites, `P0` / `P1`)
- **Castle / Townhall (`4` sprites):** Band 1 (`P0` launch mandatory) + Bands 2–4 (`P1` multi-wave cosmetic progression) sliced from `environment/castle_model_sheet.png` (`512 × 512` RGBA cells).
- **Curtain Wall, South Gate & Scaffolds (`6` sprites, `P0`):** `5` normalized HP ratio states (`100–60%`, `59–25%`, `<25%`, `0 HP`, `250 HP Recovery`) + `1` modular Builder scaffold overlay (`environment/wall_model_sheet.png`).
- **Village Buildings & Courtyard Props (`10` sprites, `P0`/`P1`):** `1` watchtower + `6` non-blocking courtyard houses + `3` low-profile courtyard props (`environment/building_variations.png`).
- **Seamless Terrain & Road Tiles (`6` tiles, `P0`):** `256 × 256` low-frequency seamless tiles (`environment/environment_tilesheet.png`).

### 2.3 Pooled VFX Flipbooks (`5` Archetypes, `30` Frames Total, Cap `<= 20` Simultaneous)
1. `vfx_builder_repair_spark` (`6f @ 12 FPS`, `128×128` RGBA, `#FEF08A` 4-point star + timber chip — `P0`).
2. `vfx_fighter_slash_arc` (`4f @ 12 FPS`, `128×128` RGBA, `#FFFFFF` / `#E63946` crescent blade sweep — `P0`).
3. `vfx_guard_shield_block` (`4f @ 12 FPS`, `128×128` RGBA, `#DCE6F2` hexagonal steel impact ring — `P0`).
4. `vfx_unit_hit_puff` (`6f @ 12 FPS`, `128×128` RGBA, low-opacity cel dust puff on enemy defeat — `P0`).
5. `vfx_boss_slam_telegraph` (`10f @ 12 FPS`, `256×256` RGBA, concentric `#D946EF` ground rune pulse & tick `2600/2800` shockwave — `P0`).

### 2.4 Original / CC0 SFX & Audio Cues (`8` Cues for `M06-W0605` `SFX001`, Zero Licensed Music)
- `sfx_builder_hammer`, `sfx_fighter_slash`, `sfx_enemy_breach_thud`, `sfx_prep_horn_15s`, `sfx_vote_open_chime`, `sfx_boss_horn_2600`, `sfx_wave_win_fanfare`, `sfx_wave_loss_drum` — to be synthesized or sourced with documented `CC0` / original provenance under `-14 LUFS` / `-1 dBTP` loudness budget in `M06`.
