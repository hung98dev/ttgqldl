# ART-003 Animation Pipeline, Palette Lock & Cleanup Rules

## 1. Why Controlled 2D Rig/FK Cutout Was Chosen Over Frame-by-Frame Text-to-Image

Generating `18` independent text-to-image frames per character produces fatal production defects at `48 px` (`1080p`) and `16 px` (`360p`) gameplay scale: mutating weapon shapes, drifting head-to-body ratios, flickering cel-shading boundaries, extra fingers/straps, and horizontal foot-sliding.

To guarantee **zero identity drift**, `ART-003` implements a deterministic 4-stage pipeline in `art/production/ART-003/tools/build_art003_rigs_and_frames.py`:

1. **Canonical Reference Extraction:** Segments the approved `ART-002` `3/4` gameplay references (`art/concepts/ART-002/characters/sprites/builder_ref_q34.png` and `art/concepts/ART-002/characters/sprites/grunt_ref_q34.png`, cross-checked against `builder_model_sheet.png` and `grunt_model_sheet.png`) into clean anatomical layers (`head`, `torso`, `back_arm`, `front_upper_arm`, `front_forearm_and_weapon`, `left_leg`, `right_leg`).
2. **Interior Occlusion & Material Cleanup:** Strips studio neutral background AND enclosed low-chroma background holes (`diff < 24`, `chroma < 16`), removes cross-layer color leaks (e.g., apron yellow pixels on iron hammer top, goblin chest green pixels on wooden club), and reconstructs occluded torso/apron/pelvis regions using exact sampled material medians so swinging limbs reveal zero holes or seams.
3. **Pivot-Locked Forward Kinematics (FK):** Anchors every `128×128` cell to normalized pivot **`(0.50, 0.08)`** -> pixel **`(64, 117)`** (`11 px` bottom margin, `32 px` overhead weapon arc clearance). Poses are driven by explicit joint angles, cyclic sinusoidal breathing/gait curves, and tapered cel-shaded motion smears (`#FEF08A` core + `#FFFFFF` leading edge + `#11151C` ink border) on impact frames (`f03`).
4. **Post-Composite Bastion Dawn v1 Cleanup Pass:** Quantizes interior colors via Lab-space nearest-centroid mapping onto a **locked `<= 16`-color character palette**, enforces binary alpha (`alpha = 0` or `255`, `0` semi-transparent halo pixels), and reconstructs a crisp external **`#11151C`** dark-ink silhouette border across `100%` of character frames.

---

## 2. Locked Character Palettes (`<= 16` Colors Including `#11151C` Ink)

### 2.1 Builder (`chr_role_builder` — `16` Exact Extracted Colors)

| Material Role | Extracted Centroid Hex Swatches | Gameplay Readability Function |
|---|---|---|
| Outer Dark Ink Contour | `#11151C` | Survives H.264 4:2:0 compression & `16 px` `360p` downscale |
| Golden Apron & Artisan Cap | `#FEF08A`, `#F4C430`, `#D49417` | Instant `XÂY` (Builder) role recognition on Meadow/Ochre/Limestone |
| Cream Work Shirt & Smear Highlight | `#FFFBEB`, `#F3EAD6`, `#D5C4A3` | High-value torso contrast & `Repair f03` leading arc edge |
| Warm Skin (`45°` Top-Left Light) | `#F4C29F`, `#DA9B74` | Clean facial clarity without noisy micro-shading |
| Leather Gloves, Belt & Boots | `#AD6E4D`, `#A66E44`, `#7E502F`, `#533420` | Grounds foot contacts at `y = 117` |
| Oversized T-Hammer Iron | `#9BA7B6`, `#6E7885`, `#47515E` | Unmistakable T-mallet head across all `18` frames |

### 2.2 Grunt (`chr_enemy_grunt` — `15` Exact Extracted Colors)

| Material Role | Extracted Centroid Hex Swatches | Gameplay Readability Function |
|---|---|---|
| Outer Dark Ink Contour | `#11151C` | Separates moss-green skin from `#68A347` Sunlit Meadow |
| Moss-Green Goblinoid Skin | `#8EC66C`, `#5E9155`, `#3E693A`, `#284726` | Two-step cel volume + deep recessed pectoral/limb shadow |
| Rough Hide Loincloth & Wooden Club | `#B0784C`, `#96663F`, `#835331`, `#6F482B`, `#59361E`, `#492E1B` | Constant `68 px` spiked wooden club & loincloth across all `18` frames |
| Tusks, Spikes & Smear Arc | `#FEF08A`, `#F5EAB5`, `#DCE4EE`, `#8A96A6` | Preserves snarling goblin face & `ClubSmash f03` strike arc |

### 2.3 VFX Repair Spark (`vfx_builder_repair_spark` — `8` Exact Extracted Colors)

- **Core White-Hot Star & Gold Burst:** `#FFFFFF`, `#FEF08A`, `#FACC15`, `#EAB308`
- **Seasoned Oak Timber Fragments:** `#C88A4B`, `#9A5E2E`, `#6B3E1C`
- **Outer Crisp Contour:** `#11151C` (`1 px` crisp cel border so sparks read on bright `#B8C0C8` Limestone as well as dark masonry)
- **Strict Dissipation Rule:** Frame `f04` and `f05` contain **`0` bright spark pixels (`R > 200, G > 200`)** — only tiny cooling timber chips settling downward before full transparency.

---

## 3. Canvas, Pivot & Clip Frame Order Specification

- **Source Cell Canvas:** `128×128 px` RGBA PNG (`8-bit/channel`, unpremultiplied clean RGB, binary alpha `0` or `255`).
- **Normalized Pivot:** `(0.50, 0.08)` -> Top-down pixel `(64, 117)` / Unity bottom-up pixel `(64.00, 10.24)`.
- **Standing Body Height in Cell:** `86 px` (`y = 32..117`), reserving `y = 0..31` (`32 px`) overhead for raised T-hammer (`y_min = 13`) and raised Grunt club (`y_min = 6`).
- **Playback Rate:** `12.0 FPS` (`83.33 ms/frame`) across all clips.
- **Deterministic Character Sheet Layout (`768×384 px`, `6 cols × 3 rows`):**
  - **Row 0 (`y = 0..127`):** `idle_f00`, `idle_f01`, `idle_f02`, `idle_f03`, `walk_f00`, `walk_f01`
  - **Row 1 (`y = 128..255`):** `walk_f02`, `walk_f03`, `walk_f04`, `walk_f05`, `action_f00`, `action_f01` (`repair` / `clubsmash`)
  - **Row 2 (`y = 256..383`):** `action_f02`, `action_f03`, `action_f04`, `action_f05`, `hit_f00`, `hit_f01`
- **Deterministic VFX Sheet Layout (`768×128 px`, `6 cols × 1 row`):**
  - **Row 0 (`y = 0..127`):** `vfx_builder_repair_spark_f00` .. `f05`

---

## 4. Mandatory Cleanup & Rejection Gate for Scaling to Remaining 5 Classes

Before accepting any clip for `Guard`, `Fighter`, `Runner`, `Brute`, or `Boss` (`256×256` cell), `validate_art003_qa_and_manifest.py` rules apply:

1. **Zero Semi-Transparent Halos:** `(alpha > 0) & (alpha < 255)` pixel count MUST equal `0`.
2. **Zero Border Clipping:** Tight opaque bounding box MUST stay strictly within `[1..W-2, 1..H-2]` (`>= 4 px` margin recommended).
3. **Zero Idle Foot Drift:** `Idle` clip foot-contact `Y` MUST remain locked to the pivot ground line (`0 px` vertical bounce of planted soles; `<= 1.0 px` median horizontal foot shift).
4. **Non-Static Frame Guarantee:** Every adjacent frame pair MUST change by `>= 80` pixels (`>= 250` pixels on `Walk` and primary combat/repair actions).
5. **Loop Seam Ratio:** Cyclic clips (`Idle`, `Walk`) MUST satisfy `0.75 <= (seam_delta / intra_clip_mean_delta) <= 1.65`.
