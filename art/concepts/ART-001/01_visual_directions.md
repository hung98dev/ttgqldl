# ART-001 — Deliverable A: Three Distinct Visual Directions

**Project:** TTGQLDL (`hung98dev/ttgqldl`)  
**Authority:** [`docs/plan/07_UNITY_ART_UX.md`](../../../docs/plan/07_UNITY_ART_UX.md) (primary visual authority), [`docs/plan/02_GAME_RULES.md`](../../../docs/plan/02_GAME_RULES.md) (`ttg-rules-v1`)  
**Package Status:** `PROPOSED` concept exploration (M06 remains `NOT_STARTED`; owner visual approval pending)

---

## 1. Shared Non-Negotiable Baseline Across All Three Directions

Every direction evaluated in this document obeys the locked broadcast and engineering constraints:

- **Format & Camera:** Vertical `9:16` (`1080 × 1920` primary reference, `360 × 640` minimum legibility baseline), **Unity 6000.6.7f1 Built-in 2D** orthographic pseudo-isometric (`3/4` top-down) fixed camera; never 3D, never side-scrolling, no manual camera pan.
- **Spatial Grammar:** Central Townhall + defensive wall occupying approximately **44% of screen width** (`~475 px` on `1080 px` width) inside the safe viewport (`x = 5%..82%`, `y = 16%..74%`), approached by three unmistakable lanes (**West / TÂY**, **Center / GIỮA**, **East / ĐÔNG**).
- **Livestream Safe Zones:** Right `x > 82%` (`886..1080 px`) and bottom `y > 88%` (`1690..1920 px`) reserved for TikTok LIVE / OBS platform overlays; zero essential HUD or gameplay silhouettes placed in reserved chrome.
- **Anchored UI Only:** Zero essential text baked into world artwork; minimum effective text sizes at `360 × 640` are **`>= 16 px` (`>= 48 px` @ 1080p)** for the single primary command CTA (`Bình luận XÂY / THỦ / ĐÁNH`), **`>= 14 px` (`>= 42 px` @ 1080p)** for castle/wall HP and timer, and **`>= 12 px` (`>= 36 px` @ 1080p)** for secondary status labels, with **WCAG contrast `>= 4.5:1`** and icon + shape + text redundancy.
- **Entity Pooling & Sprite Targets:** `<= 150` pooled on-screen representatives (`40` Builders, `40` Guards, `40` Fighters, `30` Enemies default quota; Boss always visible when active); `128 × 128` RGBA PNG source frames (`pivot = (0.5, 0.08)`) legible at **`48 px` displayed height** and across **ITU-R BT.709 grayscale**.

---

## 2. Direction A — Bright Stylized Fantasy (*"Bastion Dawn" / Bình Minh Thành Lũy*) — **[PROPOSED RECOMMENDED]**

### 2.1 Philosophy, Tone & Original Visual Lineage
- **Concept Name:** *Bastion Dawn* (*Bình Minh Thành Lũy*)
- **Art Philosophy:** High-clarity 2D cel-illustrated tabletop siege fantasy. Combines bold, uniform dark-ink outer contours and 2-step cel value blocks with compact `3.2`-head-tall heroic proportions and oversized silhouette props (chunky iron mallet, tall rectangular tower shield, sweeping plumed helmet + broadsword).
- **Emotional Tone:** Energetic, heroic, welcoming, and briskly readable—sunny enough to feel inviting during `160 s` preparation phases, yet dramatic when arcane violet boss telegraphs cut across the central stone causeway.
- **Original Visual Lineage (Described in Formal Craft Terms):** Clean European/mobile 2D strategy-defense illustration featuring inked contour hierarchy, stepped gouache-cel flats, desaturated mid-value turf planes, and simplified architectural masonry blocks with zero photographic noise.

### 2.2 Master HEX Palette & Grayscale Luminance (`Y` BT.709)

| Token Group | Token Name | HEX | BT.709 `Y` | Role / Usage |
|---|---|---|---:|---|
| **World** | `world.turf.base` | `#68A347` | `0.31` | Controlled mid-value sunlit meadow turf (never neon green) |
| **World** | `world.turf.shadow` | `#4A7A34` | `0.17` | Pine/cliff cast shadow & outer border framing |
| **World** | `world.lane.west_east` | `#CFA96E` | `0.43` | Compacted ochre-sand flank roads (West & East) |
| **World** | `world.lane.center` | `#B8C0C8` | `0.51` | Paved pale-limestone Center Boss causeway |
| **World** | `world.wall.stone` | `#9BA5B0` | `0.37` | Defensive curtain wall & Townhall masonry |
| **World** | `world.roof.timber` | `#3B6EA5` | `0.15` | Townhall slate roof (`#3B6EA5`) + warm oak framing (`#8C5A32`) |
| **Role — Builder** | `role.builder.primary` | `#F4C430` | `0.59` | Bright golden-yellow leather work apron & cap (`high Y`) |
| **Role — Builder** | `role.builder.secondary` | `#6E7885` | `0.18` | Forged iron mallet head + oak shaft (`#8E5B34`) |
| **Role — Guard** | `role.guard.primary` | `#2E7CF6` | `0.22` | Royal cobalt-blue shoulder cape & tower-shield face (`mid-low Y`) |
| **Role — Guard** | `role.guard.trim` | `#DCE6F2` | `0.78` | High-luminance steel shield rim & kettle-helm trim |
| **Role — Fighter** | `role.fighter.primary` | `#E63946` | `0.18` | Crimson-red helmet plume & armored surcoat |
| **Role — Fighter** | `role.fighter.blade` | `#E2E8F0` | `0.80` | Bright broadsword steel edge for melee readability |
| **System Militia** | `role.militia.neutral` | `#94A3B8` | `0.36` | `P <= 10` system assistance (`Tự động hỗ trợ`, non-viewer) |
| **Enemy — Runner** | `enemy.runner.primary` | `#9B5DE5` | `0.19` | Lean violet-purple forward-lunging raider |
| **Enemy — Grunt** | `enemy.grunt.primary` | `#4CC964` | `0.42` | Round moss-green brawler + dark oak spiked club (`#6B4423`) |
| **Enemy — Brute** | `enemy.brute.primary` | `#F3722C` | `0.29` | Wide cubic burnt-orange siege plate + iron cuffs (`#5C534E`) |
| **Enemy — Boss** | `enemy.boss.obsidian` | `#1E102B` | `0.01` | Oversized obsidian armor (`near-black Y = 0.01` anchor) |
| **Enemy — Boss** | `enemy.boss.fissure` | `#D946EF` | `0.27` | Glowing magenta-violet chest fissures & ground telegraph ring |
| **UI & Alerts** | `ui.scrim.primary` | `#101726` | `0.01` | Anchored HUD panel fill (`93%` alpha; `17.9:1` white text contrast) |
| **UI & Alerts** | `ui.border.gold` | `#EAB308` | `0.48` | Frame border & primary CTA highlight |
| **UI & Alerts** | `ui.warn.boss` | `#260C31` | `0.01` | Boss banner fill with `#E879F9` border (`16.8:1` text contrast) |
| **UI & Alerts** | `ui.warn.mock` | `#FACC15` | `0.64` | Honest `MOCK / REPLAY` badge accent on `#1C160A` (`15.4:1`) |

### 2.3 Rendering Craft Rules
- **Line Weight:** `3.0 px` outer silhouette ink (`#11151C`) and `1.5 px` major internal form break at `128 × 128` source resolution (`~1.1 px` outer contour at `48 px` display height). Zero sub-pixel hairline crosshatching.
- **Shading & Materials:** Crisp 2-step cel shading (`1` flat base + `1` shadow block at `-28%` value + `1` top-left rim highlight on metal/stone edges).
- **Lighting & Edge Treatment:** Fixed `45°` top-left warm key light (`#FFF6E0`), cool slate shadow tint, stepped hard-alpha outer sprite edge (`alpha >= 224` opaque, `< 224` transparent) so sprites render cleanly under Unity `FilterMode.Point` without semi-transparent fringe halos.
- **Texture Density:** Low-frequency broad shapes on terrain (`<= 2` subtle stone seam tones per tile); high silhouette clarity reserved strictly for characters, wall damage notches, and lane borders.

### 2.4 Silhouette, World, Icon & HUD Rules
- **Character Silhouette Grammar:** `3.2`-head-tall compact proportions; Builder = **T-Mallet + Apron**; Guard = **Vertical Rectangle Shield + Cape**; Fighter = **Tall Curved Plume + Angled Greatsword**; Runner = **45° Forward Chevron + Twin Daggers**; Grunt = **Sphere Torso + Diagonal Club**; Brute = **1.35× Wide Cube Pauldrons**; Boss = **2.1× Towering Horned Obsidian Silhouette + Ground Ring**.
- **HUD Treatment:** Deep slate-iron anchored cards (`#101726`) with crisp `2–4 px` warm-gold borders, geometric vector icons paired with every number, and zero decorative flourishes encroaching on text margins.

### 2.5 Risks, Production Cost & Evaluation
- **48 px & 360 × 640 Risk:** Low-Moderate. Grass saturation must remain capped (`S <= 48%`, `#68A347`) so `#4CC964` green Grunts never blend into turf—solved by routing enemies exclusively down `#CFA96E` ochre and `#B8C0C8` stone lanes plus `3 px` dark outer ink.
- **AI-Assisted & Artist Production Consistency Risk:** Low. Stepped 2-tone cel flats and fixed `3.2`-head turnarounds are straightforward to enforce across `124` total sprite frames using palette-indexed cleanup passes.
- **Strengths:** Best equilibrium of mainstream livestream warmth, instant 3-second readability, distinct silhouette + luminance separation, and predictable Unity 2D production cost.
- **Weaknesses:** Requires strict palette linting so environment artists do not over-saturate grass or add noisy cobblestone cracks.

---

## 3. Direction B — Premium Chibi Fantasy (*"Hearthguard Chibi" / Làng Chibi Hộ Vệ*)

### 3.1 Philosophy, Tone & Original Visual Lineage
- **Concept Name:** *Hearthguard Chibi* (*Làng Chibi Hộ Vệ*)
- **Art Philosophy:** Expressive `2.0`-head-tall storybook chibi defense with soft painterly cel transitions, oversized rounded headgear, cozy timber-and-blossom village architecture, and warm lantern highlights.
- **Emotional Tone:** Cozy, charming, whimsical, and character-centric—maximizes individual cuteness and peaceful village warmth.
- **Original Visual Lineage:** Polished East-Asian mobile storybook 2D illustration featuring warm sepia-brown linework, soft airbrushed cheek/metal gradients, rounded masonry, and decorative foliage clusters.

### 3.2 Master HEX Palette & Grayscale Luminance (`Y` BT.709)

| Token Group | Token Name | HEX | BT.709 `Y` | Role / Usage |
|---|---|---|---:|---|
| **World** | `world.turf.meadow` | `#749F43` | `0.30` | Warm olive-meadow grass with blossom scatter (`#F4A6C1`) |
| **World** | `world.lane.cobble` | `#C8A878` | `0.42` | Rounded warm sandstone cobblestones (West & East) |
| **World** | `world.lane.causeway` | `#BEB9AE` | `0.49` | Warm gray-beige north stone bridge |
| **World** | `world.wall.masonry` | `#A9A49C` | `0.38` | Rounded stone wall + warm cedar Townhall (`#8F5B38`) |
| **Role — Builder** | `role.builder.primary` | `#F5C842` | `0.61` | Warm butter-gold apron & rounded hard-hat + wooden mallet |
| **Role — Guard** | `role.guard.primary` | `#3B82F6` | `0.24` | Rounded cobalt hood-cape + curved kite shield |
| **Role — Fighter** | `role.fighter.primary` | `#EF4444` | `0.21` | Ruby-red plumed helm + stubby broadsword |
| **Enemy — Runner** | `enemy.runner.primary` | `#A855F7` | `0.19` | Hooded chibi shadow-imp |
| **Enemy — Grunt** | `enemy.grunt.primary` | `#38C958` | `0.39` | Chubby emerald forest brawler |
| **Enemy — Brute** | `enemy.brute.primary` | `#EB7323` | `0.27` | Rounded-cube terracotta golem |
| **Enemy — Boss** | `enemy.boss.monarch` | `#231438` | `0.01` | Chibi-proportioned Dread Monarch + `#C026D3` crown glow |
| **UI & Alerts** | `ui.scrim.plum` | `#1C152A` | `0.01` | Rounded `24 px` fairytale dark-plum HUD cards (`17.6:1` contrast) |
| **UI & Alerts** | `ui.border.cream` | `#F4C460` | `0.58` | Warm cream-amber ornamental border |
| **UI & Alerts** | `ui.warn.boss` | `#2E0F3A` | `0.02` | Violet alert bar with `#F065FF` border (`15.9:1` contrast) |

### 3.3 Rendering Craft Rules
- **Line Weight:** `2.2 px` dark umber-brown (`#2A1E17`) tapered contour at `128 × 128` with finer `1.0 px` facial/costume detail lines.
- **Shading & Materials:** 3-step soft-edge painterly cel shading with smooth gradient roll-offs on helmets, cheeks, cobbles, and foliage.
- **Lighting & Edge Treatment:** Soft golden-hour ambient + anti-aliased alpha edges (which **conflict** with `07_UNITY_ART_UX.md`'s locked `FilterMode.Point` setting, causing stepped pixel-crawl when scaled dynamically).
- **Texture Density:** High—individual cobblestones, roof shingles, blossom petals, and grass tufts across the viewport.

### 3.4 Silhouette, World, Icon & HUD Rules
- **Character Silhouette Grammar:** Uniform `2.0`-head-tall chibi proportions (`50%` of sprite height is the round head/hood), relying heavily on facial expression and small held props rather than distinct torso aspect ratios.
- **HUD Treatment:** Soft pill-shaped plum-wood panels (`24 px` corner radius) with warm cream-gold borders.

### 3.5 Risks, Production Cost & Evaluation
- **48 px & 360 × 640 Risk:** **High.** As confirmed empirically in `images/direction_b_360x640.png` and `images/direction_b_grayscale_360x640.png`, individual cobblestone joints create high-frequency H.264 macroblock mosquito noise, warm `#C8A878` cobbles merge with `#F5C842` Builders and `#EB7323` Brutes, and `2.0`-head chibi silhouettes converge into near-identical round blobs in grayscale at `48 px`.
- **AI-Assisted & Artist Production Consistency Risk:** **High.** Maintaining soft painterly gradients, large expressive eyes, and fine trim across `16–18` frames per character causes severe frame-to-frame shimmer ("boiling") in AI-assisted workflows and doubles manual cleanup hours.
- **Strengths:** High standalone screenshot charm and cozy emotional warmth.
- **Weaknesses:** Poorest 48 px / grayscale silhouette separation, cobblestone background clutter, highest per-frame production cost, and technical friction with Unity `FilterMode.Point`.

---

## 4. Direction C — Arcade Silhouette Fantasy (*"Tactical Crisp Bastion" / Chiến Tuyến Sắc Nét*)

### 4.1 Philosophy, Tone & Original Visual Lineage
- **Concept Name:** *Tactical Crisp Bastion* (*Chiến Tuyến Sắc Nét*)
- **Art Philosophy:** Radical graphic minimalism engineered around dark desaturated slate-moss ground planes and high-chroma geometric unit emblems with stark obsidian shadow bodies.
- **Emotional Tone:** Tense, clinical, arcade-tactical, and austere—feels like a competitive night-ops command board.
- **Original Visual Lineage:** Minimalist indie tactical defense graphics featuring architectural axonometric stone massing, zero ground texture, flat 1-shadow cel planes, and high-contrast neon-rim boss telegraphs.

### 4.2 Master HEX Palette & Grayscale Luminance (`Y` BT.709)

| Token Group | Token Name | HEX | BT.709 `Y` | Role / Usage |
|---|---|---|---:|---|
| **World** | `world.ground.slate` | `#3E524B` | `0.08` | Dark desaturated slate-moss field (zero grass blades) |
| **World** | `world.lane.carved` | `#74868A` | `0.23` | Clean-edged pale slate recessed roadways + `#A8B6B9` curb |
| **World** | `world.wall.limestone` | `#C7CEBF` | `0.59` | High-key faceted limestone bastion wall |
| **Role — Body Core** | `unit.core.shadow` | `#0B0E13` | `0.01` | Shared pitch-black humanoid body silhouette |
| **Role — Builder** | `role.builder.accent` | `#FFD12A` | `0.67` | Vivid amber-yellow geometric apron & T-hammer |
| **Role — Guard** | `role.guard.accent` | `#2997FF` | `0.30` | Electric-azure rectangular shield slab |
| **Role — Fighter** | `role.fighter.accent` | `#FF3B30` | `0.23` | Scarlet angular crest & tabard + `#DDE6ED` sword |
| **Enemy — Runner** | `enemy.runner.primary` | `#B55CFF` | `0.22` | Sharp chevron violet sprinter |
| **Enemy — Grunt** | `enemy.grunt.primary` | `#34C759` | `0.42` | Spherical lime-emerald raider + mace |
| **Enemy — Brute** | `enemy.brute.primary` | `#FF7A1A` | `0.32` | Heavy cubic blaze-orange juggernaut |
| **Enemy — Boss** | `enemy.boss.void` | `#0E0617` | `0.005` | Jagged void-black colossus + `#E040FB` neon rim & target ring |
| **UI & Alerts** | `ui.scrim.obsidian` | `#0A1016` | `0.008` | Sharp `6 px` chamfered tactical HUD (`19.5:1` white text contrast) |
| **UI & Alerts** | `ui.border.cyan` | `#64D2E1` | `0.53` | Tactical cyan panel outline (`#FFD60A` on primary CTA) |

### 4.3 Rendering Craft Rules
- **Line Weight:** `3.5 px` jet-black (`#05070A`) hard-stepped contour at `128 × 128` (`100%` native `FilterMode.Point` compatibility).
- **Shading & Materials:** Flat unshaded or 1-step hard shadow (`zero` gradients); materials expressed purely by silhouette facet angles.
- **Texture Density:** Near-zero (`0` grass tufts, `0` cobbles; sparse architectural seam lines only).

### 4.4 Silhouette, World, Icon & HUD Rules
- **Character Silhouette Grammar:** Pure geometric primitives (sphere Grunt, cube Brute, chevron Runner, rectangle shield Guard, T-hammer Builder, wedge crest Fighter).
- **HUD Treatment:** Angular `6 px`-radius dark obsidian bars with crisp cyan/amber technical rules.

### 4.5 Risks, Production Cost & Evaluation
- **48 px & 360 × 640 Risk:** Low for color accents, **Moderate in Grayscale** for humanoids: as observed in `images/direction_c_grayscale_360x640.png`, the shared pitch-black humanoid bodies (`Y = 0.01`) merge into the dark `#3E524B` ground (`Y = 0.08`) whenever units stand off-road, leaving only floating shields/aprons visible.
- **Livestream & Gate Risk:** **Moderate-High.** `07_UNITY_ART_UX.md` explicitly warns: *"avoid generic featureless colored boxes as final characters."* Over-minimal pawn-like bodies risk feeling like developer placeholders and feel emotionally cold during `160 s` peaceful village-building phases on TikTok LIVE.
- **AI-Assisted & Artist Production Consistency Risk:** Very Low (simplest flat shapes to animate and verify).
- **Strengths:** Maximum H.264 compression resilience on lanes, striking Boss telegraph ring, lowest animation production cost.
- **Weaknesses:** Cold/austere livestream mood, pawn-like character read, and black-limb loss against dark turf in grayscale.

---

## 5. Summary Scoring Matrix (Rubric Defined in Deliverable A)

Scores are assigned on a `0–10` scale and multiplied by the mandatory weights (**Small-screen legibility 30%**, **Livestream appeal 25%**, **Production consistency 20%**, **Gameplay clarity 15%**, **Asset cost/complexity 10%**). Full empirical justification from the rendered `1080 × 1920`, `360 × 640`, and grayscale mockups is detailed in [`02_comparison_and_recommendation.md`](02_comparison_and_recommendation.md).

| Criterion | Weight | Dir A: Bright Stylized (*Bastion Dawn*) | Dir B: Premium Chibi (*Hearthguard*) | Dir C: Arcade Silhouette (*Tactical Crisp*) |
|---|---:|---:|---:|---:|
| Small-screen legibility (`360×640` & `48 px` & Grayscale) | **30%** | **9.0** (`2.70`) | 6.8 (`2.04`) | 8.8 (`2.64`) |
| Livestream appeal (TikTok 9:16 scroll-stop & `160s` PREP warmth) | **25%** | **9.2** (`2.30`) | 9.0 (`2.25`) | 6.8 (`1.70`) |
| Production consistency (`124` anim frames + Unity `Point` filter) | **20%** | **8.8** (`1.76`) | 6.5 (`1.30`) | 9.0 (`1.80`) |
| Gameplay clarity (3 lanes, wall damage, role-action read) | **15%** | **9.0** (`1.35`) | 7.2 (`1.08`) | 9.2 (`1.38`) |
| Asset cost / complexity (`10` = most efficient & maintainable) | **10%** | **8.4** (`0.84`) | 6.0 (`0.60`) | 9.2 (`0.92`) |
| **Weighted Total (`/ 10.00`)** | **100%** | **`8.95` (PROPOSED)** | **`7.27`** | **`8.44`** |
