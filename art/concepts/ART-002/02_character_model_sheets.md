# ART-002 Deliverable B — Character Model Sheets & Scale Verification

- **Work Package:** `ART-002`
- **Style Version:** `bastion-dawn-v1`
- **Character Directory:** `characters/`
- **Canonical Reference Sprites:** `characters/sprites/` (`7` static `3/4` RGBA PNGs with hard-thresholded outer alpha)
- **Master 48 px Color & Grayscale Matrix:** `characters/all_characters_48px_matrix.png`
- **Binding Authority:** [07_UNITY_ART_UX.md](../../../docs/plan/07_UNITY_ART_UX.md) (`ART002`), [02_GAME_RULES.md](../../../docs/plan/02_GAME_RULES.md)

---

## 1. Universal Character Production Specifications

All seven canonical character classes share identical rendering, perspective, and Unity import rules so they read as a single commercial cast rather than disparate illustrations:

| Property | Standard Classes (`Builder`, `Guard`, `Fighter`, `Runner`, `Grunt`, `Brute`) | Center Boss (`Dread Bastion Warlord`) |
|---|---|---|
| **Authoritative Gameplay View** | `3/4` isometric gameplay view (defenders face outward/laneward; enemies march inward toward bastion) | `3/4` front-isometric downward march on Center causeway |
| **Source Frame Canvas** | **`128 × 128` RGBA PNG** | **`256 × 256` RGBA PNG** (documented overscan cell at identical `PPU = 192` so `~102 px` height + crescent horns + runic cleaver never clip `128 px` bounds) |
| **Normalized Ground Pivot** | **`(0.50, 0.08)`** -> pixel `(64, 10)` from bottom | **`(0.50, 0.08)`** -> pixel `(128, 20)` from bottom |
| **Unity Import Settings** | `Texture Type: Sprite (2D and UI)`, `Filter Mode: Point (no filter)`, `Compression: None` (master) / `RGBA32` atlas, `Max Atlas: 2048 × 2048` | Same (`PPU = 192`, `Filter Mode: Point`, `Pivot: Custom (0.50, 0.08)`) |
| **Outer Alpha Rule** | Straight RGB + binary hard outer alpha (`alpha >= 160..192 -> 255`, else `0`) over `3.0 px` `#11151C` ink | Same (`6.0 px` source ink -> `~3.0 px` on-screen weight) |
| **Future `M06` Animation Rate** | **`12 FPS`** (`Idle 4f`, `Walk 6f`, `Attack/Action 6f`, `Hit 2f` = `18` frames/class; cosmetic only, decoupled from `10 Hz` backend tick) | **`12 FPS`** (`Idle/March 6f`, `Heavy Cleave/Slam 8f`, `Hit 2f` = `16` frames) |

---

## 2. Deliverable Matrix Across All Seven Classes

Every required sheet exists on disk inside `characters/` and was verified across all four orthographic turnaround angles (`Front`, authoritative `3/4 Gameplay`, `Side Profile`, `Back`) plus isolated equipment callouts and multi-scale color/grayscale terrain strips:

| # | Class | Category | Model Sheet | Detail Sheet | Scale & Grayscale Sheet | Static `3/4` Sprite |
|---:|---|---|---|---|---|---|
| 1 | **Builder (`XÂY`)** | Playable Role | `characters/builder_model_sheet.png` | `characters/builder_detail_sheet.png` | `characters/builder_scale_comparison.png` | `characters/sprites/builder_ref_q34.png` (`128×128`) |
| 2 | **Guard (`THỦ`)** | Playable Role | `characters/guard_model_sheet.png` | `characters/guard_detail_sheet.png` | `characters/guard_scale_comparison.png` | `characters/sprites/guard_ref_q34.png` (`128×128`) |
| 3 | **Fighter (`ĐÁNH`)** | Playable Role | `characters/fighter_model_sheet.png` | `characters/fighter_detail_sheet.png` | `characters/fighter_scale_comparison.png` | `characters/sprites/fighter_ref_q34.png` (`128×128`) |
| 4 | **Runner** | Enemy (Priority `1`) | `characters/runner_model_sheet.png` | `characters/runner_detail_sheet.png` | `characters/runner_scale_comparison.png` | `characters/sprites/runner_ref_q34.png` (`128×128`) |
| 5 | **Grunt** | Enemy (Priority `2`) | `characters/grunt_model_sheet.png` | `characters/grunt_detail_sheet.png` | `characters/grunt_scale_comparison.png` | `characters/sprites/grunt_ref_q34.png` (`128×128`) |
| 6 | **Brute** | Enemy (Priority `3`) | `characters/brute_model_sheet.png` | `characters/brute_detail_sheet.png` | `characters/brute_scale_comparison.png` | `characters/sprites/brute_ref_q34.png` (`128×128`) |
| 7 | **Dread Bastion Warlord** | Center Boss | `characters/boss_model_sheet.png` | `characters/boss_detail_sheet.png` | `characters/boss_scale_comparison.png` | `characters/sprites/boss_ref_q34.png` (`256×256`) |

 Raw unedited turnaround and prop-sheet sources are preserved in `characters/source/` (`builder_turnaround_raw.jpg`, `guard_turnaround_raw.jpg`, `fighter_turnaround_raw.jpg`, `runner_turnaround_raw.jpg`, `grunt_turnaround_raw.jpg`, `brute_turnaround_raw.jpg`, `boss_turnaround_raw.jpg`, `hero_props_detail_raw.jpg`, `enemy_props_detail_raw.jpg`).

---

## 3. Per-Class Identity, Proportions & Costume Invariants

### 3.1 Builder (`XÂY / BUILD`) — Playable Economy & Wall-Repair Role
- **Proportions & Scale:** `3.2` heads tall (`1.00x` hero baseline); `48 px` displayed height at `1080×1920` (`16 px` at `360×640`); packed body `60 × 86 px` inside `128 × 128` canvas.
- **Silhouette Signature:** Asymmetric **orthogonal T-shaped heavy mallet** projecting cleanly from a straight-sided work apron and soft artisan cap.
- **Locked Costume Invariants (All 4 Views):** Golden-yellow leather bib apron (`#F4C430`, `Y = 0.59`) with front tool pocket, rolled-sleeve cream linen shirt (`#EDE4CF`, `Y = 0.78`), oversized forged-iron rectangular mallet (`#6E7885`) on oak shaft (`#8E5B34`), cuffed leather work gloves and boots.
- **Gameplay Alignment (`ttg-rules-v1`):** Positioned along inner courtyard and wall scaffolds; visually communicates Wood/Stone generation and `20 s` boundary Wall repair/upgrade.

### 3.2 Guard (`THỦ / DEFEND`) — Playable Gate & Wall Defense Role
- **Proportions & Scale:** `3.2` heads tall (`1.00x` hero baseline); `48 px` displayed height at `1080×1920` (`16 px` at `360×640`); packed body `54 × 86 px` inside `128 × 128` canvas.
- **Silhouette Signature:** **Vertical rectangular fortress block** dominated by a ground-to-chin tower shield and broad flowing cape.
- **Locked Costume Invariants (All 4 Views):** Royal cobalt-blue heavy cape and tunic (`#2E7CF6`, `Y = 0.21`), bright steel-rimmed rectangular tower shield (`#DCE6F2`, `Y = 0.76`) with cobalt field, rounded steel kettle/nasal helm (`#7B8794`), heavy steel sabatons. **Zero offensive sword or spear** so Guard never blurs with Fighter.
- **Gameplay Alignment (`ttg-rules-v1`):** Anchored in tight shield-wall formation at the South Gate and lane mouths (`DEF = 3 * G`).

### 3.3 Fighter (`ĐÁNH / ATTACK`) — Playable Lane Interceptor Role
- **Proportions & Scale:** `3.2` heads tall (`1.05x` total height including crest plume -> `48–50 px` at `1080×1920`); packed body `58 × 90 px` inside `128 × 128` canvas.
- **Silhouette Signature:** **Tall curved helmet plume + `40°` upward-diagonal broadsword** creating a sharp triangular attack wedge.
- **Locked Costume Invariants (All 4 Views):** Crimson-red surcoat (`#E63946`, `Y = 0.16`) and sweeping helmet plume, steel barbute helm (`#8D99AE`), single broad two-handed steel sword (`#CBD5E1`, `Y = 0.75`) held diagonally on the right across all four views, dark leather belt and boots (`#4A3525`).
- **Gameplay Alignment (`ttg-rules-v1`):** Positioned forward on the three approach lanes meeting incoming enemies (`ATK = 4 * F`, targeting Priority `1 -> 2 -> 3`).

### 3.4 Runner — Fast Priority `1` Enemy
- **Proportions & Scale:** `3.0` heads tall crouched lean build (`0.90x` height = `44 px` at `1080×1920`, `15 px` at `360×640`); packed body `66 × 78 px` inside `128 × 128` canvas.
- **Silhouette Signature:** **`45°` forward-lunging dart** with long swept-back fin ears and twin backward-held curved daggers.
- **Locked Costume Invariants (All 4 Views):** Vivid violet-purple skin (`#9B5DE5`, `Y = 0.16`), ragged dark-plum loincloth/wraps (`#3C096C`), twin pale-steel curved daggers (`#E2E8F0`), glowing pale-lilac eyes. Lean ribcage and narrow waist make it unmistakable against the round Grunt.

### 3.5 Grunt — Core Priority `2` Enemy
- **Proportions & Scale:** `3.0` heads tall stout build (`1.00x` height = `48 px` at `1080×1920`, `16 px` at `360×640`); packed body `63 × 86 px` inside `128 × 128` canvas.
- **Silhouette Signature:** **Compact rounded boulder torso** paired with a thick diagonal wooden club.
- **Locked Costume Invariants (All 4 Views):** Moss-green skin (`#4CC964`, `Y = 0.43`), round pot-belly, stitched brown hide loincloth and belt (`#7F5539`), gnarled oak club (`#9C6644`) with iron stud.

### 3.6 Brute — Heavy Priority `3` Siege Enemy
- **Proportions & Scale:** `3.5` heads tall ultra-broad hulk (`1.35x` unit scale = `64 px` displayed height at `1080×1920`, `21 px` at `360×640`); packed body `87 × 106 px` inside `128 × 128` canvas.
- **Silhouette Signature:** **Massive flat-topped trapezoid / square block** with oversized layered shoulder pauldrons and ground-scraping gauntlets (`1.6x` Grunt width).
- **Locked Costume Invariants (All 4 Views):** Siege-orange enameled heavy armor (`#F3722C`, `Y = 0.26`), flat-topped dark-iron bucket helm (`#4A525D`), riveted iron crusher gauntlets (`#343A40`) — zero handheld stick weapon so its fist-battering silhouette stays distinct from Grunt and Boss.

### 3.7 Dread Bastion Warlord — Center Lane Boss (Every 5th Wave at Tick `2600`)
- **Proportions & Scale:** `4.2` heads tall towering warlord (`2.10x` hero scale = `102 px` displayed height at `1080×1920`, `34 px` at `360×640`); packed body `123 × 204 px` inside **`256 × 256` RGBA canvas**.
- **Silhouette Signature:** **Wide upward-curving crescent horns + jagged shoulder spikes + notched greatsword + concentric arcane ground telegraph ring**.
- **Locked Costume Invariants (All 4 Views):** Matte obsidian-violet war-plate (`#1E102B`, `Y = 0.01`), glowing magenta-violet ribcage and visor fissures (`#D946EF`, `Y = 0.25` -> `#FCE7F3`, `Y = 0.84` hot core), colossal notched runic war-cleaver (`#4C3B63`), and a crisp concentric ground telegraph ring (`#E879F9`) centered on foot pivot `(0.50, 0.08)`.

---

## 4. Multi-Scale (`128px -> 360×640`) & Grayscale Verification Results

Each `<character>_scale_comparison.png` sheet and `characters/all_characters_48px_matrix.png` tests every character across **five scales** (`128×128` [or `256×256`] source canvas, `96 px`, `64 px`, mandatory **`48 px` target floor**, and `360×640` mobile scale `15–34 px`) over all three actual game terrain colors (`Meadow #68A347`, `Ochre Flank #CFA96E`, `Limestone Causeway #B8C0C8`) in both **RGB Color** and **BT.709 Grayscale**:

| Class | `48 px` Color Readability | `48 px` BT.709 Grayscale Readability | Primary Differentiator Surviving at `48 px` Grayscale | Verdict |
|---|---|---|---|---|
| **Builder** | `PASS` | `PASS` | Brightest defender torso (`Y = 0.59–0.78`) + horizontal T-mallet head | **PASS** |
| **Guard** | `PASS` | `PASS` | High-value `#DCE6F2` (`Y = 0.76`) vertical rectangular shield rectangle | **PASS** |
| **Fighter** | `PASS` | `PASS` | Dark `#E63946` (`Y = 0.16`) torso + tall plume + bright `#CBD5E1` `40°` blade | **PASS** |
| **Runner** | `PASS` | `PASS` | Leanest crouched width + swept ears + twin bright daggers | **PASS** |
| **Grunt** | `PASS` | `PASS` | Round mid-bright (`Y = 0.43`) belly + thick `3.0 px` `#11151C` ink boundary | **PASS** |
| **Brute** | `PASS` | `PASS` | `64 px` height + wide cubic pauldron silhouette + dark iron gauntlets | **PASS** |
| **Boss** | `PASS` | `PASS` | `102 px` height + `Y = 0.01` black-violet mass + `Y = 0.84` chest core + ground ring | **PASS** |
