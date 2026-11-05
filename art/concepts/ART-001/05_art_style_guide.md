# ART-001 — Deliverable F: Reusable Production Art Style Guide (*Proposed Direction A: "Bastion Dawn"*)

**Project:** TTGQLDL (`hung98dev/ttgqldl`)  
**Style Version:** `bastion-dawn-v0.1-proposed` (pending `ART-002` owner lock)  
**Target Engine:** Unity `6000.6.7f1` (Built-in 2D Pipeline, `1080 × 1920` portrait reference, `<= 150` pooled entities)

---

## 1. Character Proportions, Perspective & Frame Geometry

### 1.1 Proportions & `3/4` Camera Rule
- **Camera Projection:** Fixed orthographic `3/4` pseudo-isometric top-down angle (`~35°` vertical elevation). Ground contact ellipses use a `2 : 1` width-to-height ratio.
- **Head-to-Body Ratios:**
  - **Player Roles (`Builder`, `Guard`, `Fighter`) & `Grunt`:** **`3.2` heads tall** (`head + helmet = ~31%` of character height; torso/apron `~37%`; sturdy legs/boots `~32%`).
  - **`Runner`:** **`2.8` heads effective vertical height** due to `45°` forward dash lean (`0.90×` hero vertical height).
  - **`Brute`:** **`3.6` heads tall (`1.35×` hero height)** and **`1.60×` hero shoulder width** (square siege-plate silhouette).
  - **`Boss`:** **`2.10×` hero height** (`~96–104 px` on-screen height) with upward-sweeping crescent horns.
- **Facing Convention:** Author canonical frames facing **center-right (`+X` / `SE–NE` `3/4` toward lane clash)**. Symmetrical silhouettes (`Grunt`, `Brute`, `Runner`, `Builder`) may flip `SpriteRenderer.flipX` across West vs East lanes; `Guard` and `Fighter` keep shield/weapon readability clear on both flanks because silhouette shape (tall rectangle vs curved plume + angled blade) is view-invariant.

### 1.2 Source Frame Sizing, Pivot & Safe Margins
- **Standard Unit Frame (`ART001` Roles & `ART002` Normal Enemies):** **`128 × 128` px RGBA PNG** (`8`-bit per channel, straight/unpremultiplied RGB with binary-stepped outer alpha for `FilterMode.Point`).
- **Boss Frame (`ART002` Boss):** Authored at **`256 × 256` px RGBA PNG** (`2×` standard cell inside the `<= 2048 × 2048` character atlas, same `Pixels Per Unit`) so `2.1×` height never clips weapon swings while maintaining identical pixel/contour density to `128 × 128` units; if a strict uniform `128 × 128` sheet is required by importer tooling, pack at `128 × 128` with `PPU = 64` (`0.5×` standard `PPU = 128`)—both approaches are documented in [`06_unity_handoff.md`](06_unity_handoff.md) for `ART-002` sign-off.
- **Normalized Pivot (`07_UNITY_ART_UX.md` line 23):** **`pivot = (0.50, 0.08)`** (`x = 64 px`, `y = 10 px` from bottom edge of `128 × 128` frame). Every character's front/back foot ground-contact midpoint MUST sit exactly on `(64, 10)` across all `Idle`, `Walk`, `Action/Attack`, and `Hit` frames so sprites never slide or jitter during state transitions.
- **Safe Margins (`128 × 128` Canvas):**
  - Foot baseline at `y = 10 px` from bottom (`8%`), reserving `10 px` below feet for contact shadow & hammer/club ground impact.
  - Standing `1.0×` hero body occupies `y = 10..96 px` (`86 px` tall in source -> scaled by orthographic camera to **`~48 px` displayed height** on `1080 × 1920`), leaving `y = 97..124 px` (`28 px` top margin) and `12 px` left/right margins for overhead hammer arcs, plumes, and sword slashes.

---

## 2. Line Weight, Shape Grammar, Shading & Edge Discipline

- **Outer Silhouette Contour:** **`3.0 px` solid `#11151C`** at `128 × 128` source (`~1.1–1.3 px` at `48 px` display height). Every unit MUST read as a recognizable solid black-filled stamp if filled `100%` `#000000`.
- **Internal Detail Lines:** **`1.5 px` `#18202C`** maximum; omit belt-stitching, nostrils, individual fingers (draw chunky `2`-segment mitts), and chainmail rings smaller than `3 px` so nothing turns to noisy static under H.264 compression.
- **Shape Grammar Summary:**
  - `Builder` = **Trapezoid Apron + Orthogonal `T`-Mallet**
  - `Guard` = **Vertical Rectangle Shield + Swept Triangle Cape**
  - `Fighter` = **Curved Plume Arc + `40°` Diagonal Wedge Blade**
  - `Runner` = **Acute Forward Chevron + Swept Fin Ears**
  - `Grunt` = **Sphere Belly + Diagonal Spiked Club**
  - `Brute` = **Wide Horizontal Cube Shelf + Pillar Legs**
  - `Boss` = **Inverted Horned Triangle + Glowing Rib Fissures + Ground Circle**
- **2-Step Cel Shading & `FilterMode.Point` Edge Rule:**
  - `1` flat base color (`~65%` area) + `1` cel shadow (`~28%` area, `-28%` value shifted `+8°` toward cool indigo) + `1` crisp rim highlight (`~7%` top-left metal/stone edge).
  - **Hard Outer Alpha Threshold:** Threshold outer silhouette alpha at `alpha >= 224 → 255`, `< 224 → 0` on export so `FilterMode.Point` in Unity produces crisp cel silhouettes with zero translucent halo artifacts.

---

## 3. Canonical Palette Tokens & Grayscale Separation Contract

All production sprites and tiles must sample from the locked token table in [`01_visual_directions.md §2.2`](01_visual_directions.md) and pass two automated checks before M06 commit:

1. **Terrain Saturation Ceiling:** Environment turf and lane tiles MUST keep HSV Saturation **`S <= 48%`** and Value **`V` between `35%` and `78%`** so saturated unit colors (`S = 68%..88%`) and `3 px` `#11151C` (`V = 11%`) contours always pop.
2. **BT.709 Grayscale Delta (`|ΔY| >= 0.12` or Distinct Silhouette Class):**
   - Player roles separate in grayscale via **luminance + prop geometry**: `Builder` bright apron (`Y = 0.59` + T-mallet) vs `Guard` bright-rimmed rectangle shield (`rim Y = 0.78` framing `Y = 0.22` slab) vs `Fighter` dark crimson surcoat (`Y = 0.18` + `Y = 0.80` diagonal steel sword + tall plume).
   - Enemies separate via **luminance + body mass**: `Boss` (`Y = 0.01`, `2.1×` horned) vs `Runner` (`Y = 0.19`, `0.9×` chevron) vs `Brute` (`Y = 0.29`, `1.6×` wide cube) vs `Grunt` (`Y = 0.42`, `1.0×` sphere).

---

## 4. Environment Scale, Lanes, Castle/Wall States & Sorting Layers

- **World Footprint (`ART003`, `ART004`):**
  - Central Townhall + circular/polygonal curtain wall = **`~44%` of screen width (`475 px` on `1080 px`)**, centered horizontally inside the safe viewport (`center X ≈ 454 px`, well clear of `x > 885 px` right reserve).
  - **Three Unmistakable Lanes:** West (`TÂY`, `#CFA96E` ochre road entering `NW` at `35°`), Center (`GIỮA`, `#B8C0C8` wide paved stone Boss causeway entering straight `N` at `90°`), East (`ĐÔNG`, `#CFA96E` ochre road entering `NE`, outer curb strictly `< 870 px`).
  - **6 Environmental Tile Variants + 4 Visual-Only Lighting Palettes (`ART004`):** Sunlit Dawn (Waves `1–3`), Golden Midday (`4`, `6–8`), Dusk Siege (`9`, `11–14`), Violet Eclipse (Boss Waves `5, 10, 15, 20...` — shifts ambient shadow tint only, never reduces HUD contrast or role hue readability).
- **5 Visual Wall Damage / Tier States (`ART003` — Mapped Across `20` Authoritative Tiers):**
  - **Tier Progression (`4` visual material bands × `5` HP damage states, or `5` consolidated visual states per `07_UNITY_ART_UX.md` line 18):**
    1. `State 1 (100%..80% HP / Reinforced):` Intact crenellations, crisp mortar, clean bannerets.
    2. `State 2 (79%..55% HP / Chipped):` Chipped merlons, hairline masonry cracks, active wooden Builder scaffolds.
    3. `State 3 (54%..30% HP / Breached Parapet):` Missing merlon teeth, exposed timber bracing, rubble chips at base.
    4. `State 4 (29%..1% HP / Critical Wall):` Deep structural fissures, low jagged wall profile, orange warning dust.
    5. `State 5 (0 HP / Collapsed Wall — Overflow to Townhall):` Flattened rubble ring; Townhall gates exposed directly to enemy hits.
  - Additional village props (`ART003`): `1` Townhall, `1` Watchtower, `6` cottages/houses, `3` non-blocking village decorations.
- **Unity 2D Sorting Layer & Y-Sort Order:**
  1. `00_TerrainAndLanes` (static tilemap/background)
  2. `10_GroundTelegraphsAndShadows` (unit contact shadows + Boss concentric telegraph ring — **always visible beneath feet, never covering units**)
  3. `20_CastleAndUnits_YSorted` (`SortingGroup` / `CustomAxisSort (0, 1, 0)` driven by foot `pivot = (0.50, 0.08)` world `Y`; Boss gets `+1` tie-break priority over overlapping normal enemies at identical `Y`)
  4. `30_CombatAndRepairVFX` (capped strictly at **`<= 20` simultaneous pooled VFX bursts**)
  5. `40_WorldLabels` (max `12` pooled alias tags, never inside `ScreenSpaceOverlay` critical bars)
  6. `50_AnchoredBroadcastHUD` (`ScreenSpaceOverlay` Canvas)

---

## 5. VFX Readability & Hard Budget (`ART005`)

- **Hard Simultaneous Cap:** **`<= 20` active VFX bursts total** (`ParticleSystem` avoided where pooled `4–6` frame sprite flipbooks suffice for zero GC allocation).
- **5 Required VFX Types (`07_UNITY_ART_UX.md` line 20):**
  1. `vfx_repair_spark`: Golden `4`-point star + wood chip (`Builder` boundary/action, `6` frames @ `12 fps`).
  2. `vfx_guard_shield`: Crisp azure hexagonal ripple on wall/shield (`Guard` block at combat boundary, `5` frames).
  3. `vfx_fighter_slash`: High-contrast white-crimson arc (`Fighter` melee hit, `4` frames).
  4. `vfx_projectile_hit`: Compact impact ring (`4` frames, max `48 × 48` px so crowds stay legible).
  5. `vfx_boss_telegraph`: Concentric `#E879F9` / `#FFFFFF` ground ring (`<= 0.8 Hz` gentle pulse, **never `> 3 Hz`**).

---

## 6. Animation Loop Discipline, Naming, Folder Organization & Rights

- **Consistent Frame Alignment:** Every animation strip exports horizontal PNG cells (`N × 128` wide by `128` tall) or indexed individual frames `..._00.png`..`..._05.png` with locked `(0.50, 0.08)` foot registration; `Hit` (`2` frames) snaps `3 px` back on frame `0` and recovers on frame `1` without moving the transform origin.
- **Asset Naming Convention (`snake_case`, deterministic prefix):**
  - Characters: `chr_<group>_<unit>_<anim>_f<NN>.png` (e.g., `chr_role_builder_idle_f00.png`, `chr_enemy_boss_attack_f07.png`)
  - Environment: `env_<zone>_<name>_v<NN>.png` (e.g., `env_wall_tier01_dmg02.png`, `env_lane_center_tile_v01.png`)
  - VFX: `vfx_<effect>_f<NN>.png`
  - UI: `ui_icon_<token>.png`, `ui_frame_<panel>.png` (9-sliced)
  - Audio (`SFX001`): `sfx_<event>_v<NN>.wav`, `mus_loop_<mood>_v<NN>.ogg` (muted by default until stream rights confirmed)
- **Target M06 Unity Folder Hierarchy (inside `unity/Assets/_TTGQLDL/Art/` — untouched in `ART-001`):**
  ```text
  unity/Assets/_TTGQLDL/Art/
    Atlases/          # <= 2048x2048 SpriteAtlas assets (.spriteatlasv2)
    Characters/       # Roles/ (Builder, Guard, Fighter, Militia) & Enemies/ (Runner, Grunt, Brute, Boss)
    Environment/      # CastleWall/, Lanes/, Tiles/, LightingPalettes/
    VFX/              # RepairSpark/, ShieldBlock/, MeleeHit/, BossTelegraph/
    UI/               # Icons/, Panels9Slice/, Fonts/
    Audio/            # SFX/ (12 short original/cleared), Ambient/ (1), Music/ (2, default muted)
  ```
- **Version Control, Provenance & Originality Gate:** Every committed file MUST have a Git-tracked `.meta` file (`QA-CLI-01`) and a matching row in the asset manifest (`QA-ASSET-01`) recording exact tool/author provenance, creation UTC, license, and human IP review status. Third-party copyrighted franchises, scraped sprites, unverified asset-store packs, and unmarked AI outputs without provenance records are strictly **FORBIDDEN (`FAIL`)**.
