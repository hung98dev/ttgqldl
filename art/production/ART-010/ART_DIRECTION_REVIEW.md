# ART-010 — Honest Art Direction Critique & Three-Direction Visual Design Review

**Inspected Baseline:** `ART-003`..`ART-009` (`art/production/ART-009/previews/corrected_hud/*.png` & `mobile/*.png`)  
**Target Resolution:** Portrait `1080×1920` (`9:16`) master & `360×640` mobile verification  
**Perspective:** 2D Pseudo-Isometric / `3/4` Elevated Side-Scroll View

---

## 1. Phase 0 — Honest Visual Diagnosis of `ART-003`..`ART-009`

Although `ART-003`..`ART-009` pass every automated manifest, canonical-simulation, and performance-metric validator, a candid visual inspection of the actual `1080×1920` and `360×640` renders reveals ten structural reasons why the current prototype **fails the commercial TikTok LIVE scroll-stop test**:

1. **Empty Flat Olive-Green Void (`59.3%–60.6%` of `1080×1920`):**  
   Measured in `qa/composition_metrics.json`, `59.29%` of `05_assault_1080p.png` (`60.61%` of `07_boss_active_1080p.png`) consists of low-detail flat `32×32` blocks (`std < 4.5`). Specifically, the entire `y = 980..1380` band (`400` vertical pixels) and both left/right flanks (`x < 220`, `x > 660`) are bare `#74A14E` green fill with faint blurry dots—zero cliffs, trees, rooftops, elevation tiers, or cast shadows.
2. **Stair-Stepped Grid Lane Tiles:**  
   In `05_assault_1080p.png`, the West and East enemy lanes are assembled from unmasked `64×64` square sand tiles placed diagonally, creating harsh pixel-staircase edges against the green ground instead of curved, grounded medieval roads or bridges.
3. **Underscaled Geometric Ring Castle (`< 9%` of canvas):**  
   The circular wall and inner cluster (`~470×360 px`) read like flat isometric diagram boxes with `1 px` black outlines, uniform light-gray fills, zero stone bevels, zero warm interior window glow, and zero ambient occlusion where walls meet the dirt.
4. **Micro-Scale Characters Unreadable at `360×640`:**  
   Playable and enemy units are `~44–48 px` tall on the `1080×1920` canvas, shrinking to **`15–16 px`** in `05_assault_360p.png`. Blue Guards and Red Fighters stand directly over blue Townhall roofs and gray battlements of similar luminance, collapsing into unreadable specks on smartphone screens.
5. **Lack of Lighting Hierarchy & World Depth:**  
   Every pixel of the battlefield shares identical flat midday ambient brightness (`luma_entropy_bits = 5.338`). Nothing separates distant enemy spawn threats (cool/ominous) from the sanctuary the viewers are defending (warm/inviting).
6. **Weak Combat Impact & Thin VFX:**  
   `vfx_unit_hit_puff` renders as small white circles and `vfx_boss_slam_ring` is a thin purple ellipse on flat sand. At `360p`, sword swings, shield blocks, and wall repairs fail to communicate kinetic force.
7. **Generic Box Building Silhouettes:**  
   Watchtowers, barracks, and resource huts inside the ring lack memorable roof gables, timber framing, banners, lanterns, or smoke plumes.
8. **Disconnected Debug-Dashboard HUD:**  
   `ART-006`/`ART-009` stacks four dark `#1E2430` rectangular rows at top (`y = 20..308`) and four rows at bottom (`y = 1320..1662`), consuming `~630` vertical pixels (`33%` of screen height) with flat geometric icons that feel bolted over a test map rather than crafted into a fantasy world.
9. **Zero Environmental Storytelling:**  
   There is no visual sense of *where* enemies emerge from (dark forest, canyon, siege bridge) or *what* lives inside the village worth saving.
10. **Low Boss Finale Spectacle (`07_boss_active_1080p.png`):**  
    When the Wave 10 Boss spawns at tick `2600`, it appears as an `~85 px` purple sprite (`~28 px` at `360p`) surrounded by empty green grass, with no atmospheric sky shift or dramatic ground telegraph.

---

## 2. Phase 1 & 2 — Three Distinct Commercial-Quality Directions

All three `ART-010` directions preserve `100%` of the `ttg-rules-v1` gameplay contract (Central Townhall, Defensive Wall, 3 lanes West/Center/East, Builder/Guard/Fighter vs Grunt/Runner/Brute + Finale Boss, single CTA `Bình luận XÂY / THỦ / ĐÁNH`, `82%×88%` TikTok safe zone) while solving `ART-009`'s visual deficits through **three genuinely different spatial, lighting, and camera philosophies**:

### Concept A — Fantasy Diorama Kingdom (`concepts/A_fantasy_diorama/`)
- **Core Vision:** *A living, warm golden-hour storybook kingdom that viewers immediately fall in love with and want to protect.*
- **Composition & World Depth:** Elevated `3/4` diorama perspective. Three natural cobblestone and dirt lanes wind through sunlit oak trees, timber palisades, a stone stream bridge (Center lane), and pine-covered rocky steps (East lane) toward the northern outer gatehouse, while the grand half-timbered blue-slate Townhall, inner circular stone keep, smithy, lantern-lit courtyard, and cozy cottages fill the middle and lower tiers (`flat_empty_block_ratio` reduced from `59.29%` to **`0.51%`**).
- **Lighting & Color Hierarchy:** Warm late-afternoon amber sunlight (`#F6B95E`) illuminates the Townhall gables and village lanterns, contrasting cleanly against olive-green Grunts, rust-iron Brutes, violet Runners, and the Finale Boss's dark amethyst armor + rune-circle telegraph.
- **HUD Theme:** *Storybook Royal Oak & Burnished Gold* — sculpted dark oak plaques with riveted brass trim and painted fantasy icons.

### Concept B — High-Impact Action Defense (`concepts/B_action_defense/`)
- **Core Vision:** *A visceral, close-quarters fantasy battle happening LIVE right in front of the viewer.*
- **Composition & Camera Framing:** `~1.6×` tighter camera crop centered directly on the fortified stone gatehouse, battlement walkway, and converging three-lane killzone (left canyon trail, central torch-lit siege bridge, right cliff path). Units stand **`~110–126 px` tall at `1080p` (`37–42 px` at `360p` — `2.5×` larger than `ART-009`)**.
- **Combat & VFX Readability:** Bold crimson-gold Fighter cleave arcs, overlapping azure Guard shield-wall shockwaves, bright golden Builder anvil-sparks on cracked battlements, and an imposing Finale Boss (`~380 px` tall) charging a cracked magenta arcane ground sigil.
- **Lighting & Color Hierarchy:** Dramatic dusk-storm slate tones (`#1E2736`) make warm torchlight, glowing cathedral windows, and saturated combat VFX pop instantaneously on OLED phone screens.
- **HUD Theme:** *Forged Obsidian Iron & Ember Bronze* — dark hammered steel plates with warm ember-bronze bevels.

### Concept C — Vertical Siege Battlefield (`concepts/C_vertical_siege/`)
- **Core Vision:** *A purpose-built `9:16` vertical cliffside citadel siege engineered natively for portrait TikTok LIVE.*
- **Portrait-Native 4-Zone Architecture:**
  1. **Upper Threat Horizon (`y = 205..560`):** Cool mist-shrouded mountain gorge with three unmistakable vertical approach routes—West timbered cliff stairs, Center arched stone viaduct bridge, and East carved cliffside ledge.
  2. **Primary Killzone & Outer Rampart (`y = 520..880`):** Semi-circular stone bastion where Blue Guards hold an arcane phalanx dome, Red Fighters counter-slash flanked enemies, and Yellow Builders hammer repair sparks onto damaged merlons.
  3. **Sanctuary Citadel & Tiered Village (`y = 880..1440`):** Steeply terraced medieval Townhall with royal blue roofs and warm amber windows cascading down stone staircases above the cliffside dock.
  4. **Natural Right-Rail TikTok Safe Margin (`x > 885`):** A dramatic sheer chasm and twin waterfalls occupy the rightmost `18%` of the frame, giving atmospheric beauty where TikTok stacks avatar/like/share buttons while guaranteeing **zero gameplay units or HUD elements are ever occluded**.
- **HUD Theme:** *Citadel Chiseled Slate & Antique Gold* — crisp architectural slate plaques with antique gold trim.

---

## 3. Gameplay Contract & Safe-Area Integrity Across All Three Concepts

- **Exact `ttg-rules-v1` Fidelity:**
  - `ASSAULT` (`gameplay_1080p.png` / `gameplay_360p.png`): Wave 10 active assault (`GIỜ 02:00`), Wall `1760/2200`, Townhall `500/500`, Wood `592` / Stone `256` / Food `490`, 3-lane threat banner `GIAO TRANH · TÂY 155 · GIỮA 143 · ĐÔNG 157`, 6 non-Boss classes visible (`Builder`, `Guard`, `Fighter`, `Grunt`, `Runner`, `Brute`), **zero premature Boss**.
  - `FINALE` (`boss_1080p.png` / `boss_360p.png`): Wave 10 Boss finale (`GIỜ 00:40`, corresponding to canonical tick `>= 2600`), Wall `1420/2200`, Boss HP bar `BOSS 62/124 · TÂY 0 · ĐÔNG 0`, prominent Center-lane Boss + arcane slam telegraph.
  - **Single Canonical Viewer CTA:** `Bình luận XÂY / THỦ / ĐÁNH` + role counts (`XÂY` / `THỦ` / `ĐÁNH`) + honest `MOCK / REPLAY · MÔ PHỎNG (CHƯA LIVE)` indicator. Zero fake viewer counters, zero invented currencies, zero extra commands.
- **TikTok Safe-Area Verification (`safe_area.png`):**
  - Top HUD crest: `x = 30..872`, `y = 20..202` (`100% <= 885 px` right safe line).
  - Bottom CTA & role dock: `x = 30..872`, `y = 1458..1679` (`100% <= 885 px` right safe line and `<= 1689 px` bottom `88%` safe line).
  - Open battlefield window between HUD bands: `y = 203..1457` (**`1,254` vertical pixels = `65.3%` unobstructed vertical gameplay view**, compared to `ART-009`'s `1,012 px` empty green middle).
