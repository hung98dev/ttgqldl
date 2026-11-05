# ART-002 Deliverable A — Final Art Direction Refinement (`bastion-dawn-v1`)

- **Work Package:** `ART-002`
- **Selected Visual Foundation:** Direction A — *Bastion Dawn* (`ART-001` score `8.95 / 10.00`)
- **Style Version:** `bastion-dawn-v1`
- **Approval Status:** `PROPOSED` (pending explicit repository-owner visual sign-off in [OWNER_APPROVAL.md](OWNER_APPROVAL.md))
- **Binding Authority:** [07_UNITY_ART_UX.md](../../../docs/plan/07_UNITY_ART_UX.md), [02_GAME_RULES.md](../../../docs/plan/02_GAME_RULES.md), [03_SIMULATION.md](../../../docs/plan/03_SIMULATION.md)
- **Visual Comparison Plate:** `direction_refinement_comparison.png`

---

## 1. Final Visual Identity Statement

**Bastion Dawn (`bastion-dawn-v1`)** is a **bright, sunlit stylized fantasy village-defense aesthetic** engineered specifically for **9:16 portrait mobile livestreaming** under H.264 bitrate compression.

The world presents a warm, inviting medieval-fantasy stronghold at golden morning light: a cobalt-slate castle keep and circular limestone curtain wall anchored at the heart of a calm, low-frequency meadow, approached from the top by three unmistakable battle lanes (West ochre road, Center pale-limestone Boss causeway, East ochre road). Every gameplay actor is locked into a crisp **dark-ink outer silhouette (`3.0 px` `#11151C`)** filled with **two-step cel shading** and **role-coded primary hues** so first-time viewers scrolling onto a `360 × 640` phone preview understand within three seconds:

1. **What we are defending** — the central castle and wall (`~44%` screen width) with live segmented HP bars.
2. **Where the danger is** — color-and-shape-distinct enemies marching down West, Center, and East lanes.
3. **How to join immediately** — one unmistakable primary command plaque: **`Bình luận XÂY / THỦ / ĐÁNH`**.

### Explicit Non-Goals (Guardrails)

`bastion-dawn-v1` strictly rejects:
- Dark gritty/grimdark realistic fantasy or muddy desaturated brown palettes.
- Retro mosaic pixel-art tiles or noisy sub-pixel dithering.
- Abstract geometric board-game pawns lacking character charm.
- Flat enterprise dashboard tables or multi-CTA HUD clutter.
- Painterly micro-rendered grass blades, cobblestone noise, or soft feathered airbrush alpha edges that disintegrate into H.264 macroblock ringing.

---

## 2. Master Color System & Semantic Token Architecture

All production shaders, sprite palettes, terrain tiles, and Unity UI prefabs must bind to the following locked HEX tokens and **BT.709 relative luminance (`Y`)** targets (`Y = 0.2126 R_lin + 0.7152 G_lin + 0.0722 B_lin`).

### 2.1 Primary Role & Castle Tokens (Defenders)

| Token ID | Semantic Role | Primary HEX | Cel Shadow HEX | Rim / Accent HEX | BT.709 `Y` (Primary) | Shape Redundancy |
|---|---|---|---|---|---:|---|
| `color.role.builder` | Builder (`XÂY`) | `#F4C430` | `#C28E14` | `#FEF08A` | `0.59` | T-shaped iron/oak mallet + flat leather bib apron |
| `color.role.guard` | Guard (`THỦ`) | `#2E7CF6` | `#1D4ED8` | `#DCE6F2` | `0.21` | Full-height vertical steel-rimmed tower shield + cape |
| `color.role.fighter` | Fighter (`ĐÁNH`) | `#E63946` | `#9F1239` | `#F8FAFC` | `0.16` | Crest plume + `40°` diagonal steel broadsword |
| `color.landmark.castle` | Central Castle Keep | `#3B6EA5` | `#24476E` | `#F4C430` | `0.15` | Steep cobalt-slate roof + warm `#E8DFC8` ashlar walls |
| `color.landmark.wall` | Curtain Wall & Parapet | `#9BA5B0` | `#64748B` | `#CBD5E1` | `0.37` | Circular crenelated masonry ring + timber-iron South gate |

### 2.2 Secondary & Environment Background Tokens (Low-Saturation Canvas)

Environment surfaces are intentionally capped at **HSV Saturation `S <= 48%`** and mid-luminance (`Y = 0.30..0.52`) so high-chroma units (`S = 68%..88%`) and dark `#11151C` (`Y = 0.01`) outlines pop cleanly in both color and grayscale:

| Token ID | Surface Element | Base HEX | Shadow / Curb HEX | Highlight HEX | BT.709 `Y` | Max HSV `S` |
|---|---|---|---|---|---:|---:|
| `color.env.meadow` | Smooth Meadow Turf | `#68A347` | `#528536` | `#7CB356` | `0.31` | `48%` |
| `color.env.lane_flank` | West & East Ochre Lanes | `#CFA96E` | `#9E7E4C` | `#DFBE88` | `0.43` | `46%` |
| `color.env.lane_center` | Center Limestone Boss Road | `#B8C0C8` | `#4E5662` | `#D8DEE4` | `0.51` | `8%` |
| `color.env.courtyard` | Inner Packed-Earth Plaza | `#D8B384` | `#A68256` | `#E6C89E` | `0.49` | `39%` |
| `color.env.timber` | Beams, Gate & Scaffolds | `#8C5A32` | `#5C381C` | `#B07848` | `0.13` | `64%` |

### 2.3 Enemy, Boss & Danger Tokens (Threat Hierarchy)

| Token ID | Threat Archetype | Primary HEX | Cel Shadow HEX | Accent / Glow HEX | BT.709 `Y` | Priority / Silhouette |
|---|---|---|---|---|---:|---|
| `color.enemy.runner` | Runner (Priority `1`) | `#9B5DE5` | `#6B21A8` | `#E9D5FF` | `0.16` | Lean `45°` sprinter + swept ears + twin daggers |
| `color.enemy.grunt` | Grunt (Priority `2`) | `#4CC964` | `#27853B` | `#8E5B34` | `0.43` | Round pot-bellied goblinoid + diagonal spiked club |
| `color.enemy.brute` | Brute (Priority `3`) | `#F3722C` | `#B44110` | `#4A525D` | `0.26` | `1.35x` wide cubic siege hulk + iron helm + gauntlets |
| `color.enemy.boss` | Dread Bastion Warlord | `#1E102B` | `#0E0716` | `#D946EF` / `#FCE7F3` | `0.01` / `0.25` | `2.10x` obsidian colossus + crescent horns + ground ring |
| `color.alert.warn` | Prep 60s/15s & Reconnect | `#FACC15` | `#CA8A04` | `#FEF08A` | `0.79` | Solid/slow-pulse amber border (`<= 2 Hz`, never `> 3 Hz`) |
| `color.alert.crit` | Critical HP / DB Halt | `#EF4444` | `#991B1B` | `#FEE2E2` | `0.21` | Coral-red fill + diagonal dark hatch / octagon `[X]` |

### 2.4 HUD Surface & Typography Tokens

| Token ID | UI Usage | HEX / RGBA | Text Foreground | Measured WCAG Contrast |
|---|---|---|---|---:|
| `color.ui.panel_primary` | Top HUD & Vote Card | `#101726` (`alpha 96%`) | `#FFFFFF` / `#F1F5F9` | `17.91 : 1` / `16.35 : 1` |
| `color.ui.cta_plaque` | Primary Command Banner | `#0F172A` (`alpha 98%`) + `#EAB308` rim | `#FFFFFF` (`48 px` Bold) | `17.85 : 1` |
| `color.ui.timer_pill` | Wave Countdown Pill | `#1E293B` + `#FACC15` rim | `#FEF08A` (`43 px` Bold) | `12.57 : 1` |
| `color.ui.banner_boss` | Boss Encounter Strip | `#260C31` + `#E879F9` rim | `#FFFFFF` (`36 px` SemiBold) | `17.81 : 1` |
| `color.ui.mock_badge` | Honest Mock/Replay Footer | `#1C160A` + `#FACC15` rim | `#FACC15` (`36 px` SemiBold) | `11.74 : 1` |

---

## 3. Grayscale Luminance Hierarchy

No gameplay state, role identity, or warning relies on hue alone. Under BT.709 grayscale conversion (`qa/qa_grayscale_and_contrast_audit.png` and `characters/all_characters_48px_matrix.png`), elements separate into five deliberate value bands:

1. **Band 0 — Ink & Boss Armor Anchor (`Y = 0.01..0.06`):** `#11151C` `3.0 px` unit/prop outlines, `#0F172A` HUD plaques, and `#1E102B` Boss obsidian plate.
2. **Band 1 — Deep Martial Silhouettes (`Y = 0.13..0.22`):** Fighter crimson surcoat (`Y = 0.16`), Runner violet body (`Y = 0.16`), Guard cobalt cape (`Y = 0.21`), oak beams (`Y = 0.13`). Separated from mid-value terrain (`Y = 0.31..0.51`) by both value step and bright steel/plume accents (`Y = 0.72..0.85`).
3. **Band 2 — Calm Mid-Value Terrain (`Y = 0.31..0.51`):** Meadow turf (`Y = 0.31`), curtain wall (`Y = 0.37`), West/East ochre lanes (`Y = 0.43`), Center limestone causeway (`Y = 0.51`).
4. **Band 3 — High-Value Role & Material Highlights (`Y = 0.58..0.84`):** Builder golden apron (`Y = 0.59`) + cream shirt (`Y = 0.78`), Guard tower-shield steel rim (`Y = 0.76`), Fighter silver sword blade (`Y = 0.75`), Boss hot-magenta core fissures (`Y = 0.84`).
5. **Band 4 — Critical HUD Readout (`Y = 0.88..1.00`):** Pure white `#FFFFFF` (`Y = 1.00`) and pale gold `#FEF08A` (`Y = 0.89`) typography locked inside Band 0 dark plaques.

---

## 4. Line Weight, Lighting, Shading & Material Rules

### 4.1 Three-Tier Ink Hierarchy (`#11151C`)

| Line Tier | Source Canvas (`128×128`) | `1080×1920` Display | Usage Rule |
|---|---:|---:|---|
| **Outer Silhouette Line** | `3.0 px` (`6.0 px` on `256×256` Boss) | `~2.5–3.0 px` | Continuous closed outer contour on all characters, weapons, castle, wall, and UI icons. Never colored or omitted. |
| **Major Equipment Seam** | `1.5 px` (`3.0 px` on `256×256` Boss) | `~1.2–1.5 px` | Separates weapon from torso, shield rim from field, apron from shirt, helmet from face. |
| **Terrain / Tile Seam** | `0.0–1.0 px` muted tint | `<= 1.0 px` | **Never** use black `#11151C` ink on grass or road tiles; use subtle warm/cool value steps (`#4E5662` curb, `#9E7E4C` verge) so units own the dark ink channel exclusively. |

### 4.2 Fixed `45°` Top-Left Sunlight & Two-Step Cel Shading

- **Key Light Vector:** Fixed **`45°` top-left** across every character view, building, prop, and icon (`#FFF6E0` warm morning sun).
- **Two-Step Cel Value Split:** Every material uses **1 flat base tone + 1 hard-edged shadow tone (`-26%` to `-32%` brightness shifted `+8°` toward cool slate-violet `#2A324B`)**. Smooth gradients are forbidden on unit sprites; only metals and arcane fissures add a third crisp specular band.
- **Ground Contact Shadow:** Every standing unit and building casts a crisp `40%` alpha `#0B0F19` elliptical shadow anchored at its ground pivot (`(0.50, 0.08)` for characters), extending slightly down-right (`+4 px X, +2 px Y`).

### 4.3 Material Rendering Contract

- **Cut Limestone (Wall, Keep Base, Center Road):** Broad ashlar blocks (`>= 18 px` wide at `1080p`), flat warm/cool stone fill, crisp top-left highlight bevel, zero noisy speckle brushes.
- **Seasoned Timber (Gate, Scaffolds, Cottages, Club/Mallet Shafts):** Warm ochre-brown cel fill (`#8C5A32`) with at most **1–2** longitudinal ink break lines per beam.
- **Forged Steel & Iron (Tower Shield Rim, Broadsword, Helms, Mallet Head):** Cool slate base (`#6E7885`), crisp `#E2E8F0` / `#FFFFFF` top-left bevel band, dark `#334155` shadow plane.
- **Dyed Cloth & Leather (Apron, Cape, Surcoat):** Pure role-coded flat cel blocks (`Builder #F4C430`, `Guard #2E7CF6`, `Fighter #E63946`); zero fabric weave texture.
- **Obsidian & Arcane Energy (Boss):** Matte near-black violet plate (`#1E102B`) with sharp `#6D28D9` edge facets and self-illuminated `#D946EF -> #FCE7F3` core fissures.

---

## 5. Contrast, Camera & Orthographic Composition Rules

- **Character-to-Environment Contrast:** Terrain saturation stays `<= 48%` and terrain Laplacian high-frequency standard deviation was reduced by **`50.0%`** from `ART-001` (`21.76 -> 10.87`, see `qa/qa_h264_compression_simulation.png`). Combined with the `3.0 px` `#11151C` outer ink moat, every character maintains high edge contrast even under YUV `4:2:0` half-resolution chroma subsampling.
- **Environment-to-HUD Contrast:** All HUD containers use `94%–98%` opaque deep navy (`#101726` / `#0F172A`) framed by `2–3 px` metallic gold/semantic borders, producing **`11.74:1` to `17.91:1`** WCAG text contrast regardless of what terrain or VFX passes behind the HUD.
- **Camera & Perspective Lock:** Fixed orthographic **2D pseudo-isometric `3/4` top-down camera** (`~35°–40°` pitch, `0°` roll, zero perspective vanishing-point convergence). Camera never pans, zooms, or tilts during gameplay; screen shake is capped at `<= 4 px` for `150 ms` on Boss slam only.
- **Bastion Spatial Budget:** Central circular wall + castle measures **`477 px` wide (`44.2%` of `1080 px`)**, centered at `x = 454 px` (`x = 215..692 px`) so `100%` of the castle, wall, South gate, and East lane combat zone sit strictly left of the `x = 82%` (`885 px`) livestream overlay boundary.

---

## 6. AI Image Generation Rules, Failure Patterns & Corrections

All AI-assisted concept and sprite-base generation in `ART-002` and subsequent `ART-003` production must follow a **controlled two-stage pipeline** rather than raw unguided text-to-image sampling:

### 6.1 Mandatory Prompt Invariants
1. Always include the **Style Lock Prefix**: `"2D stylized mobile strategy game character/asset sheet, Bastion Dawn v1 style, crisp uniform black outer ink outline (#11151C), clean 2-step cel shading, 45-degree top-left warm sunlight, flat neutral light-gray studio background (#E5E7EB), zero painterly noise, zero gradients, zero 3D render gloss, zero embedded text."`
2. Lock **explicit head-count proportions** (`3.2 heads tall` for Builder/Guard/Fighter/Grunt, `3.0 heads` crouched Runner, `3.5 heads` ultra-wide Brute, `4.2 heads` Boss).
3. Lock **handedness and weapon angle** (`Builder: horizontal/orthogonal T-mallet`, `Guard: vertical rectangular shield ONLY, zero sword`, `Fighter: single 40-degree broadsword, crest plume`, `Runner: facing down-left SW toward center castle`).

### 6.2 Observed AI Failure Patterns & Mandatory Corrections

| Failure Pattern Observed in `ART-002` | Root Cause | Mandatory Production Correction |
|---|---|---|
| **Proportion stretch (`5.0` heads instead of `3.2` heads)** — caught in initial Guard/Fighter `v1` turnarounds | Prompt terms like `"knight"` or `"warrior"` bias diffusion models toward realistic heroic anatomy | Explicitly specify `"compact 3.2-heads-tall stylized game unit matching dwarf-stout village proportions"` and reject/regenerate any sheet failing the head-grid check (`guard_turnaround_v2` & `fighter_turnaround_v2` fixed this). |
| **Hallucinated extra weapons** — Guard `v1` added a side sword; Fighter `v1` back view drew a duplicate sword tip | Symmetry/knight priors attach swords to any armored character | Add explicit negative constraints (`"NO sword, NO spear, shield ONLY"` for Guard; `"SINGLE sword held on right side across all views"` for Fighter) plus manual RGBA mask cleanup if needed. |
| **Left/Right facing flip across seeds** — caught in `Runner` consistency Runs `2–3` | Text prompts weakly bind screen-left vs character-left | Never animate multi-frame sheets from unconstrained text seeds; lock the approved Deliverable B `3/4` turnaround as structural pose/img2img reference and mirror programmatically in Unity via `SpriteRenderer.flipX` where appropriate. |
| **Armor sub-detail drift** — Fighter `Run 1` steel pauldrons vs `Runs 2–3` cloth sleeves | Ambiguous material boundary on shoulders | Lock exact costume sub-parts in Deliverable B (`crimson cloth surcoat sleeves + steel barbute helm with red plume`) before `ART-003` keyframe cleanup. |
| **Soft anti-aliased fringes on `Point` sprites** | Diffusion cutouts produce semi-transparent gray edge pixels | Run automated hard-alpha thresholding (`alpha >= 160..192 -> 255`, else `0`) over `#11151C` outer ink at export time (`build_characters_and_consistency.py`). |

---

## 7. Refinement Delta: `ART-001` Baseline vs `ART-002` (`bastion-dawn-v1`)

See `direction_refinement_comparison.png` for the side-by-side visual proof (`1080×1920` color + `360×640` BT.709 grayscale):

| Dimension | `ART-001` Direction A (`v0.1` Concept) | `ART-002` Refined `bastion-dawn-v1` (Production Blueprint) |
|---|---|---|
| **Visual Identity** | Bright Stylized Fantasy (*Bastion Dawn*) | **Preserved** — recognizably the exact same warm sunlit stylized castle-defense world, upgraded to blueprint precision |
| **Bastion Width & Safe Zone** | Wide bastion (`~72%` width); East wall approached `x = 82%` margin | **Exact `44.2%` width (`477 px`)** centered at `x = 454 px`; `100%` of wall & East lane stay strictly left of `x = 82%` (`885 px`) |
| **Terrain Compression Noise** | Visible grass tufts & pine needle chatter (`Laplacian SD = 21.76`) | **Smooth H.264-safe cel turf & roads (`Laplacian SD = 10.87`, `-50.0%` chatter)** |
| **HUD Typography & Layout** | 2-row compact top bar; synthetic vector shapes | **3-row zero-clipping top bar + 20-icon `128×128` RGBA kit + 15 9-sliceable skins** (`11.74:1–17.91:1` WCAG contrast) |
| **Colorblind HP Redundancy** | Color fill + numeric HP | **Segmented HP bars + numeric HP + critical diagonal hatch** under both Wall and Castle |
| **Character Blueprint Depth** | Single lineup + `3/4` portrait close-ups | **Full 4-view (`Front, 3/4, Side, Back`) model sheets, weapon/point-filter detail sheets, and 5-scale grayscale audits** for all 7 classes |
