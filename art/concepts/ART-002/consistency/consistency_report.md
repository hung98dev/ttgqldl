# ART-002 Deliverable H — AI Art Generation Consistency Test & Prompt Engineering Report

- **Work Package:** `ART-002`
- **Sample Size:** **`21` independent character variations** (`7` canonical classes × `3` independent generation runs; zero cherry-picking or hidden rerolls)
- **Visual Comparison Sheets:**
  - `consistency/consistency_heroes_comparison.png` (`Builder`, `Guard`, `Fighter` × `3` runs + `48 px` color/grayscale stamps)
  - `consistency/consistency_enemies_comparison.png` (`Runner`, `Grunt`, `Brute`, `Boss` × `3` runs + `48 px` color/grayscale stamps)
  - `consistency/consistency_summary_matrix.png` (`8`-criterion scoring scorecard across all `7` classes)
- **Unedited Raw Run Archives:** `consistency/source/consistency_heroes_run1.jpg`, `consistency_heroes_run2.jpg`, `consistency_heroes_run3.jpg`, `consistency_enemies_run1.jpg`, `consistency_enemies_run2.jpg`, `consistency_enemies_run3.jpg`

---

## 1. Test Methodology & Honesty Policy

To test whether `bastion-dawn-v1` character designs can be reproduced reliably without turning `M06` into an inconsistent collage, three **completely independent cross-seed generation runs (`Run 1`, `Run 2`, `Run 3`)** were executed for all seven classes using the same class-level prompt specification and style lock, without image-to-image pose locking.

Every generated variation was extracted, measured programmatically (`width/height` aspect ratio, `#11151C` dark-ink pixel share, mean BT.709 luminance `Y`), downsampled to `48 px` in color and grayscale, and audited across all **8 mandatory consistency criteria**:
1. Silhouette
2. Proportions
3. Equipment
4. Main color identity
5. Camera orientation
6. Outline weight
7. Lighting direction
8. Rendering style

---

## 2. Quantitative Cross-Run Measurements (`21 / 21` Variations)

| Class | Run 1 `W/H` · `Ink%` · `Y` | Run 2 `W/H` · `Ink%` · `Y` | Run 3 `W/H` · `Ink%` · `Y` | Core Identity Survival | Cross-Seed Drift Severity |
|---|---|---|---|---|---|
| **Builder (`XÂY`)** | `0.855` · `16.6%` · `0.298` | `0.871` · `15.7%` · `0.278` | `0.781` · `18.6%` · `0.278` | `3 / 3` PASS | **Low-Moderate** (hammer grip angle & pocket height) |
| **Guard (`THỦ`)** | `0.627` · `14.9%` · `0.181` | `0.687` · `12.4%` · `0.169` | `0.658` · `15.8%` · `0.171` | `3 / 3` PASS | **Very Low** (`< 0.06` aspect variance; rock-solid shield wall) |
| **Fighter (`ĐÁNH`)** | `0.564` · `21.7%` · `0.246` | `0.741` · `18.5%` · `0.152` | `0.546` · `18.5%` · `0.171` | `3 / 3` PASS | **Moderate** (steel pauldrons/T-visor in Run 1 vs cloth sleeves in Runs 2–3) |
| **Runner** | `0.813` · `15.1%` · `0.185` | `0.775` · `15.8%` · `0.169` | `0.867` · `15.9%` · `0.192` | `3 / 3` Class ID | **High Orientation Drift** (flipped `SW` -> `SE` & dropped loincloth in Runs 2–3) |
| **Grunt** | `0.725` · `17.5%` · `0.181` | `0.890` · `16.8%` · `0.200` | `0.702` · `15.7%` · `0.194` | `3 / 3` PASS | **Very Low** (identical pot-belly, hide wrap, and spiked club across all 3 runs) |
| **Brute** | `0.816` · `22.8%` · `0.116` | `1.007` · `17.1%` · `0.144` | `0.823` · `19.7%` · `0.142` | `3 / 3` PASS | **Low-Moderate** (curved vs angular pauldrons; warm helm tint in Run 2) |
| **Boss** | `0.594` · `46.8%` · `0.126` | `0.600` · `46.2%` · `0.124` | `0.576` · `46.1%` · `0.148` | `3 / 3` PASS | **Very Low** (`< 0.025` aspect variance; identical horns, chest core & ring) |

---

## 3. Eight-Criterion Audit & Honest Failure Analysis

### 3.1 Criteria Passing Consistently Across All `21 / 21` Variations
- **Main Color Identity (`21/21` PASS):** `100%` of runs preserved golden-yellow Builder, cobalt-blue Guard, crimson Fighter, violet Runner, moss-green Grunt, siege-orange Brute, and obsidian-magenta Boss without cross-class hue confusion.
- **Outline & Rendering Style (`21/21` PASS):** Every run maintained crisp `#11151C` cartoon ink outlines (`12.4%–22.8%` ink share on standard units) and clean two-step cel shading with zero 3D plastic gloss or pixel-art mosaic drift.
- **Proportions & Lighting (`20/21` PASS):** Explicit `3.0–3.5` head-count constraints held across all three runs (solving the `5.0`-head stretch seen in early unconstrained turnaround drafts).

### 3.2 Recorded Drift Failures (Must Not Be Ignored in `ART-003` / `M06`)

1. **Failure H-01 — `Runner` Left/Right Facing Flip & Costume Omission (`Runs 2–3`):**
   - *Observed:* `Run 1` correctly faced `3/4` down-left (`SW`) wearing a dark loincloth; `Runs 2` and `3` flipped to face `3/4` down-right (`SE`) and omitted the loincloth, rendering a bare purple torso.
   - *Impact if uncorrected:* Sprite frames generated from raw text prompts would flip facing mid-walk and flicker costume details.
   - *Production Fix:* Never generate animation frames from unconstrained text prompts. Lock `characters/sprites/runner_ref_q34.png` (`Deliverable B`) as an explicit structural pose/img2img reference (`denoise <= 0.45`) or rig/cutout-animate the `Deliverable B` turnaround parts directly, and handle `West` vs `East` lane mirroring via `SpriteRenderer.flipX` in Unity.
2. **Failure H-02 — `Fighter` Shoulder Material & Visor Drift (`Run 1` vs `Runs 2–3`):**
   - *Observed:* `Run 1` generated steel plate pauldrons (`mean Y = 0.246`) and a closed T-visor; `Runs 2–3` generated crimson cloth surcoat sleeves (`mean Y = 0.152–0.171`) and an open nasal barbute.
   - *Production Fix:* Canonicalize the `Deliverable B` turnaround (`characters/fighter_model_sheet.png`: open-face steel barbute + crimson cloth sleeves + single right-hand `40°` broadsword) and add `"crimson cloth sleeves, NO steel shoulder pauldrons"` to the Fighter prompt template.
3. **Failure H-03 — `Builder` Hammer Grip Mode (`Run 1` vs `Runs 2–3`):**
   - *Observed:* `Run 1` held the T-mallet vertically in one raised left hand with a chest bib pocket; `Runs 2–3` held the mallet diagonally across the waist with two hands and a lower apron pocket.
   - *Production Fix:* Standardize two-handed `35°` waist-ready carry for `Idle/Walk` (`characters/builder_model_sheet.png`) and reserve vertical raised mallet exclusively for frames `1–3` of the `6-frame` `Repair Action` clip.
4. **Failure H-04 — `Brute` Helm Warmth & Pauldron Curvature (`Run 2` vs `Run 3`):**
   - *Observed:* `Run 2` warm-tinted the iron bucket helm toward bronze; `Run 3` squared off the curved shoulder arches.
   - *Production Fix:* Enforce a fixed `16-color` indexed/quantized palette lut per character during `ART-003` sprite cleanup so iron stays cool `#4A525D` across every frame.

---

## 4. Locked Production Prompt Templates (`ART-003` Ready)

Every template below prepends the **Global Style Prefix (`GSP`)**:

```text
GSP = "2D stylized mobile village-defense game unit sprite reference, Bastion Dawn v1 style, fixed 3/4 top-down pseudo-isometric orthographic camera (~38-degree pitch), crisp uniform 3px #11151C black outer ink outline, clean 2-step cel shading with 45-degree top-left warm sunlight (#FFF6E0), flat neutral #E5E7EB studio background, zero ground clutter, zero embedded text, zero painterly texture, zero 3D render gloss."
```

| Class | Character-Specific Locked Prompt Body (`append to GSP`) | Mandatory Negative Constraints |
|---|---|---|
| **Builder** | `"Stout 3.2-heads-tall village Builder facing 3/4 down-right, golden-yellow leather bib apron (#F4C430) with single centered waist tool pocket over rolled-sleeve cream linen shirt (#EDE4CF), soft yellow artisan cap, brown leather work gloves and boots, gripping one oversized forged-iron rectangular T-mallet (#6E7885) on oak shaft (#8E5B34) diagonally across waist with both hands."` | `"NO sword, NO shield, NO helmet visor, NO beard drift, NO realistic tall proportions"` |
| **Guard** | `"Stout 3.2-heads-tall Bastion Guard facing 3/4 down-right, royal cobalt-blue heavy cape and tunic (#2E7CF6), rounded steel kettle helm (#7B8794), steel greaves, holding one massive ground-to-chin vertical rectangular tower shield with bright steel rim (#DCE6F2) and cobalt face planted in front."` | `"NO sword, NO spear, NO axe, NO round buckler, NO tall 5-head knight proportions"` |
| **Fighter** | `"Stout 3.2-heads-tall Vanguard Fighter facing 3/4 down-right, crimson-red cloth surcoat and sleeves (#E63946), open-face steel barbute helmet (#8D99AE) topped with tall curved crimson horsehair plume, holding one broad two-handed steel greatsword (#CBD5E1) diagonally upward at 40 degrees on right side."` | `"NO shield, NO steel shoulder pauldrons, NO dual swords, NO closed T-visor"` |
| **Runner** | `"Lean crouched 3.0-heads-tall goblinoid Runner sprinter lunging 45 degrees forward facing 3/4 down-left (screen-left), vivid violet-purple skin (#9B5DE5), long swept-back pointed fin ears, dark-plum ragged loincloth (#3C096C), gripping twin backward-swept curved steel daggers (#E2E8F0)." (Lock img2img pose from runner_ref_q34.png)` | `"NO facing right, NO bare unclothed torso, NO club, NO bulky muscles"` |
| **Grunt** | `"Short broad 3.0-heads-tall goblinoid Grunt facing 3/4 down-left, moss-green skin (#4CC964), round pot-bellied torso, stitched brown hide loincloth and belt (#7F5539), gripping one thick gnarled wooden club (#9C6644) with single iron stud diagonally across body."` | `"NO metal armor, NO shield, NO sword, NO lean tall proportions"` |
| **Brute** | `"Massive 3.5-heads-tall ultra-wide Siege Brute hulk (1.35x Grunt height, 1.6x width) facing 3/4 down-left, heavy curved siege-orange enameled shoulder pauldrons and chest plate (#F3722C), cool dark-iron flat-topped bucket helm (#4A525D) with T-slit, enormous riveted iron crusher gauntlets (#343A40)."` | `"NO handheld club/axe/sword, NO bronze-tinted helm, NO slim waist"` |
| **Boss** | `"Towering 4.2-heads-tall Dread Bastion Warlord Boss (2.1x hero height) marching 3/4 front-downward, matte obsidian-violet spiked war-plate (#1E102B), wide upward-curving crescent horns, glowing magenta-violet arcane ribcage and visor fissures (#D946EF to #FCE7F3), gripping one colossal notched runic war-cleaver (#4C3B63), standing inside a crisp concentric violet ground telegraph ring."` | `"NO flames hiding silhouette, NO horse/mount, NO muddy gradients"` |
