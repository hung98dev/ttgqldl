# ART-004 — Character Animation & Combat VFX Production Pipeline (`Bastion Dawn v1`)

**Style Lock:** `Bastion Dawn v1`  
**Canonical Visual Spec:** [07_UNITY_ART_UX.md](../../../docs/plan/07_UNITY_ART_UX.md)  
**Pilot Baseline:** [ART-003 Pipeline](../ART-003/ANIMATION_PIPELINE_AND_CLEANUP_RULES.md)

---

## 1. Locked Visual Invariants Across All 7 Classes

1. **Camera & Perspective:** Fixed `3/4` top-down pseudo-isometric orthographic view (`~35°–40°` pitch). Canonical source orientation faces `+X` (`front-right`), clean horizontal mirroring (`SpriteRenderer.flipX`) is used for `-X` lanes without silhouette breakage.
2. **Contour & Lighting:** Continuous `#11151C` dark-ink outer contour (`1–2 px` at source resolution), crisp `2`-step cel shading under fixed `45°` top-left warm sunlight (`#FFF3D6`), cool slate shadows (`#2C3545`).
3. **Alpha & Palette Discipline:** Strictly binary alpha (`0` or `255` — `0` semi-transparent edge halo pixels across all `112` frames), `<= 16` quantized RGB colors per character frame, zero embedded terrain, zero baked world shadows, zero text labels inside sprite frames.
4. **Cadence & Anchoring:** `12 FPS` (`83.33 ms/frame`; GIF preview cadence `[80, 80, 90] ms` = exact `250 ms / 3 frames = 12.00 FPS`), normalized ground pivot `(0.50, 0.08)` (`(64.0, 117.76)` top-down / `(64.0, 10.24)` Unity bottom-up in `128×128`; `(128.0, 235.52)` top-down / `(128.0, 20.48)` Unity bottom-up in `256×256`).

---

## 2. Reproducible 2D Cutout Rig & FK Animation Architecture

Rather than raw diffusion frame-by-frame hallucination (which mutates weapons, proportions, and outlines), `ART-004` uses a two-module deterministic Python/Pillow/NumPy/SciPy pipeline:

1. **`art/production/ART-004/tools/art004_rig_lib.py`:**
   - Extracts clean RGBA foregrounds from the owner-approved `ART-002` `3/4` character turnarounds (`guard_turnaround_raw.jpg`, `fighter_turnaround_raw.jpg`, `runner_turnaround_raw.jpg`, `brute_turnaround_raw.jpg`, `boss_turnaround_raw.jpg`), stripping studio backdrops and floor shadow ovals.
   - Applies character-specific design locks prior to rigging:
     - **Guard (`chr_role_guard`):** Preserves rectangular steel-rimmed cobalt tower shield (`#DCE6F2` / `#2E7CF6`), rounded kettle helm, and clenched right gauntlet with **zero handheld sword/spear/axe**.
     - **Fighter (`chr_role_fighter`):** Converts shoulder steel caps into crimson `#E63946` cloth surcoat sleeves (`convert_shoulder_steel_to_crimson_cloth`) so **zero steel shoulder pauldrons** exist; isolates open-face barbute helm + tall crimson plume + single broadsword with **zero shield**.
     - **Runner (`chr_enemy_runner`):** Locks `+X` `3/4` crouched goblinoid facing across all `18` frames (eliminating the `ART-002` cross-seed left/right flip), preserving violet `#9B5DE5` skin, dark-plum `#4C286E` vest, swept-back ears, and **both curved steel daggers**.
     - **Brute (`chr_enemy_brute`):** Preserves wide square siege-orange `#F3722C` silhouette, flat dark-iron `#4A525D` bucket helm, and massive iron crusher gauntlets `#6E6A66` with **zero handheld club/axe/hammer**.
     - **Boss (`chr_boss_warlord`):** Isolates obsidian-violet `#1E102B` horned plate, glowing magenta `#D946EF` chest core, spiked shoulders, and colossal notched runic war-cleaver `#4C3B63` inside `256×256` with **zero baked ground telegraph rings**.
   - Heals occluded limb joints (`neck_plug`, `elbow_plug`, `pelvis_plug`, `skirt_patch`) so large-angle swings never open transparent holes.
2. **`art/production/ART-004/tools/build_art004_characters_and_vfx.py`:**
   - Evaluates forward-kinematic (`FK`) pose tables at `4×` supersampled resolution (`512×512` -> `128×128`, `1024×1024` -> `256×256`).
   - Downsamples with premultiplied alpha, thresholds at `alpha >= 0.48` (`binary 0/255`), removes micro-specks, quantizes to each character's locked `<= 16`-color palette, and enforces closed `#11151C` outer ink contours.
   - Synthesizes all `4` combat VFX archetypes (`24` frames) with bounded palettes, `< 3 Hz` flash safety, and `0` bright pixels on final dissipation frames.

---

## 3. Character & VFX Clip Matrix (`112` Frames)

| Entity ID | Cell Size | Sheet Size (`Grid`) | Clips & Frame Counts (`@ 12 FPS`) | Total Frames |
|---|---|---|---|---:|
| `chr_role_guard` | `128×128` | `768×384` (`6×3`) | `Idle(4)`, `Walk(6)`, `ShieldBrace(6)`, `Hit(2)` | `18` |
| `chr_role_fighter` | `128×128` | `768×384` (`6×3`) | `Idle(4)`, `Walk(6)`, `SwordSlash(6)`, `Hit(2)` | `18` |
| `chr_enemy_runner` | `128×128` | `768×384` (`6×3`) | `Idle(4)`, `Run(6)`, `DaggerLunge(6)`, `Hit(2)` | `18` |
| `chr_enemy_brute` | `128×128` | `768×384` (`6×3`) | `Idle(4)`, `March(6)`, `GauntletCrush(6)`, `Hit(2)` | `18` |
| `chr_boss_warlord` | `256×256` | `1024×1024` (`4×4`) | `IdleMarch(6)`, `CleaveSlam(8)`, `Hit(2)` | `16` |
| `vfx_fighter_slash_arc` | `128×128` | `512×128` (`4×1`) | `Burst(4)` — synced to `Fighter.SwordSlash[f02..f05]` | `4` |
| `vfx_guard_shield_block` | `128×128` | `512×128` (`4×1`) | `Burst(4)` — synced to `Guard.ShieldBrace[f02..f05]` | `4` |
| `vfx_unit_hit_puff` | `128×128` | `768×128` (`6×1`) | `Burst(6)` — synced to any unit `Hit[f00..f01]` + dissipation | `6` |
| `vfx_boss_slam_telegraph` | `256×256` | `1280×512` (`5×2`) | `Burst(10)` — `f00..f04` rune warning, `f05..f07` shockwave, `f08..f09` fade | `10` |
| **Total `ART-004`** | — | **`9` Sprite Sheets** | **`19` Character Clips + `4` VFX Clips** | **`112`** |
