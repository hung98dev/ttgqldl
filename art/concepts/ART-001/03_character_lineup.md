# ART-001 — Deliverable D: Character Concept Lineup (*Proposed Direction A: "Bastion Dawn"*)

**Project:** TTGQLDL (`hung98dev/ttgqldl`)  
**Visual Authority:** [`docs/plan/07_UNITY_ART_UX.md`](../../../docs/plan/07_UNITY_ART_UX.md) (`ART001`, `ART002`)  
**Visual Deliverables:**
- Primary 7-Character Lineup Sheet (with `48 px` color & BT.709 grayscale proof strips): `images/character_lineup.png`
- High-Resolution Role & Enemy Close-Up Sheet: `images/character_closeups.png`
- Raw Concept Sources (zero text): `images/source/character_lineup_raw.jpg`, `images/source/heroes_closeup_raw.jpg`, `images/source/enemies_closeup_raw.jpg`

---

## 1. Unified Cast Architecture & Shared Style Rules

All seven original character designs inhabit the same **Bright Stylized Fantasy (*Bastion Dawn*)** world and obey a single visual contract:

- **Proportions:** Compact, sturdy **`3.2`-head-tall stylized 2D strategy proportions** for standard humanoids (`Builder`, `Guard`, `Fighter`, `Grunt`), **`2.8`-head hunched forward lean** for `Runner`, **`3.6`-head wide-shouldered bulk** (`1.35×` height, `1.6×` width) for `Brute`, and **`2.1×` towering scale** for `Boss`. Oversized weapons, shields, helmets, and pauldrons ensure readability when displayed at `48 px` height on a `1080 × 1920` / `360 × 640` broadcast.
- **3/4 Pseudo-Isometric Orientation:** Consistent `3/4` top-down camera angle oriented toward the village center / active clash point (mirrored cleanly on X only where symmetrical silhouette grammar is preserved; asymmetric hero shields/weapons authored in canonical `3/4` facing).
- **Line, Lighting & Shadow:** Uniform `3.0 px` dark-ink outer silhouette (`#11151C`) + `1.5 px` internal form break at `128 × 128` source; `45°` top-left warm sunlight (`#FFF6E0`), 2-step cel shadow (`-28%` value), and soft elliptical ground contact shadow (`#0B0F19`, `45%` alpha) anchored at `pivot = (0.50, 0.08)`.

---

## 2. Character-by-Character Breakdown

### 2.1 Player Role 1 — Builder (`XÂY` / `BUILD`)
- **Color Identity:** Primary golden-yellow `#F4C430` (`BT.709 Y = 0.59`), cream rolled-sleeve shirt `#EDE4CF` (`Y = 0.77`), forged iron mallet `#6E7885` (`Y = 0.18`), warm leather gloves/boots `#8E5B34`.
- **Equipment:** Heavy bib leather work apron with front tool pocket, yellow artisan cap, thick work gauntlets, and an oversized two-handed oak-and-iron rectangular mallet.
- **Silhouette Grammar:** **Asymmetric `T`-Shape** — wide blocky hammerhead extending clearly outside the body contour plus straight vertical apron hem.
- **Visual Hierarchy & Gameplay Readability:** Highest diffuse torso luminance (`Y = 0.59`) among player roles; paired with golden repair sparks along wall scaffolds so viewers immediately connect `XÂY` comments to visible wall HP recovery every `20 s` boundary.

### 2.2 Player Role 2 — Guard (`THỦ` / `DEFEND`)
- **Color Identity:** Primary royal cobalt-blue `#2E7CF6` (`Y = 0.22`), bright steel plate & shield rim `#DCE6F2` (`Y = 0.78`), deep navy shadow `#1A478F` (`Y = 0.06`).
- **Equipment:** Brushed steel nasal/kettle helmet with blue crest, flowing cobalt shoulder cape, steel breastplate/greaves, and an extra-wide full-height rectangular tower shield braced forward.
- **Silhouette Grammar:** **Fortress Block / Vertical Wall** — flat-topped rectangular tower shield (`~45%` of total character width) paired with a sweeping triangular cape counterweight behind.
- **Visual Hierarchy & Gameplay Readability:** Even in pure grayscale, the bright `#DCE6F2` rectangular shield rim (`Y = 0.78`) enclosing a mid-dark `#2E7CF6` core (`Y = 0.22`) creates an unmistakable bordered slab around castle gates.

### 2.3 Player Role 3 — Fighter (`ĐÁNH` / `ATTACK`)
- **Color Identity:** Primary crimson-red `#E63946` (`Y = 0.18`), bright polished broadsword blade `#E2E8F0` (`Y = 0.80`), chainmail/steel pauldrons `#94A3B8` (`Y = 0.36`).
- **Equipment:** Steel visored barbute helmet topped with a tall sweeping crimson horsehair plume, red armored surcoat over chainmail, and a wide double-edged broadsword held at a `35°–45°` combat diagonal.
- **Silhouette Grammar:** **Sharp Crest + Diagonal Wedge** — tall curved plume adding `+22%` vertical crest silhouette above the helm plus a crisp bright diagonal sword jutting toward the enemy lane.
- **Visual Hierarchy & Gameplay Readability:** Placed forward in lane chokepoints; the bright `#E2E8F0` blade slash arc and tall red plume separate Fighters instantly from flat-headed Builders and block-shielded Guards at `48 px`.

*(Note on `P <= 10` Non-Viewer System Militia — `02_GAME_RULES.md` / `07_UNITY_ART_UX.md`): When `1 <= P <= 10`, system assistance (`Tự động hỗ trợ`) uses a desaturated slate-silver variant (`#94A3B8` tunic, `#CBD5E1` short spear, zero role hue, zero player alias) so it is never mistaken for a real viewer.)*

---

### 2.4 Enemy Archetype 1 — Runner (Fast Subpool — `25%` Lane HP, Hit Priority `1`)
- **Color Identity:** Lean violet-purple `#9B5DE5` (`Y = 0.19`), deep plum shadow `#5A2A96` (`Y = 0.05`), pale lavender rim `#D8B4FE` (`Y = 0.56`), twin curved steel daggers `#CBD5E1`.
- **Equipment & Anatomy:** Lithe, forward-lunging impish raider with long swept-back pointed ears, light dark-violet jerkin, digitigrade sprinter legs, and twin curved blades.
- **Silhouette Grammar:** **Acute `45°` Forward Chevron (`<` / `>`)** — narrowest waist/limb profile in the game (`0.9×` height) with long pointed ears projecting backward like speed fins.
- **Gameplay Readability:** Consumed first by Fighter damage (`runner -> grunt -> brute`); lean forward posture communicates speed and fragility at a glance.

### 2.5 Enemy Archetype 2 — Grunt (Core Subpool — `55%` Lane HP, Hit Priority `2`)
- **Color Identity:** Moss-emerald green `#4CC964` (`Y = 0.42`), forest shadow `#257A38` (`Y = 0.14`), weathered oak club `#784B2A` with steel spikes `#CBD5E1`, ragged hide loincloth `#6B4C35`.
- **Equipment & Anatomy:** Stocky, round pot-bellied goblinoid foot-soldier (`1.0×` height) with broad rounded shoulders, short wide stance, and a chunky spiked wooden club hoisted diagonally.
- **Silhouette Grammar:** **Circle / Sphere Mass + Diagonal Club** — round belly and curved back form a continuous circular silhouette distinct from the angular Runner and cubic Brute.
- **Gameplay Readability:** Forms the bulk (`55%`) of each lane's HP pool; mid-high luminance (`Y = 0.42`) separates clearly from dark-purple Runners (`Y = 0.19`) in grayscale.

### 2.6 Enemy Archetype 3 — Brute (Heavy Subpool — `20%` Lane HP, Hit Priority `3`)
- **Color Identity:** Burnt siege-orange `#F3722C` (`Y = 0.29`), rust shadow `#9E3B0E` (`Y = 0.08`), heavy slate-iron gauntlets `#6E6A66` (`Y = 0.15`), hot amber rim `#FDBA74` (`Y = 0.58`).
- **Equipment & Anatomy:** Massive `1.35×`-tall, `1.6×`-wide armored siege-hulk encased in rectangular riveted orange plate pauldrons, flat-topped iron bucket helm, and oversized stone-crusher gauntlets.
- **Silhouette Grammar:** **Wide Heavy Square / Cube** — flat horizontal shoulder shelf wider than the character is tall at the torso, zero neck gap, thick pillar legs.
- **Gameplay Readability:** Immediately communicates tanky front-line weight (`last ordinary subpool to die in each lane`) via pure width and squareness even at `360 × 640` grayscale.

### 2.7 Boss — Dread Bastion Warlord (Every `5th` Wave, `tick 2600`, Center Lane)
- **Color Identity:** Obsidian black-violet plate `#1E102B` (`Y = 0.01` — darkest silhouette anchor on screen), glowing arcane magenta-violet chest/helm fissures `#D946EF` (`Y = 0.27`) with `#FCE7F3` (`Y = 0.86`) hot core, dark-violet war-cleaver `#4C3B63`, and concentric ground telegraph ring (`#E879F9` / `#FFFFFF`).
- **Equipment & Anatomy:** Towering `2.1×`-height (`~96–104 px` displayed height) horned warlord clad in jagged obsidian plate with wide crescent horns, spiked pauldrons, glowing ribcage runes, and a massive notched cleaver.
- **Silhouette Grammar:** **Oversized Horned Apex + Ground Ring** — more than double normal unit height, wide upward-curving horn crown, internal high-contrast light fissures, and persistent ground telegraph circle.
- **Gameplay Readability:** Impossible to miss in `150`-entity crowds; always spawned on the pale `#B8C0C8` (`Y = 0.51`) Center stone causeway where its `#1E102B` (`Y = 0.01`) obsidian body achieves maximum figure-ground contrast (`> 10:1` luminance ratio).

---

## 3. Future Production Animation Specification (`M06` Target — Confirmed Against `07_UNITY_ART_UX.md`)

> **Scope Boundary:** Final sprite sheets are **not** produced in `ART-001`. Once the owner approves and locks the visual direction in `ART-002`, `M06-W0603` will produce sprite sheets strictly matching [`07_UNITY_ART_UX.md`](../../../docs/plan/07_UNITY_ART_UX.md) line 23:

| Unit Category | Entities | Idle | Walk | Action / Attack | Hit | Total Frames / Unit | Playback Rate | Source Canvas & Pivot |
|---|---|---:|---:|---:|---:|---:|---|---|
| **Player Roles** | `Builder`, `Guard`, `Fighter` *(+ neutral Militia palette/gear variant)* | `4` | `6` | `6` *(Action: hammer / brace / slash)* | `2` | **`18` frames** | **`12 fps`** | `128 × 128` RGBA PNG, `pivot = (0.50, 0.08)` |
| **Normal Enemies** | `Runner`, `Grunt`, `Brute` | `4` | `6` | `6` *(Attack)* | `2` | **`18` frames** | **`12 fps`** | `128 × 128` RGBA PNG, `pivot = (0.50, 0.08)` |
| **Boss** | `Dread Warlord` *(Wave % 5 == 0)* | `6` | `0` *(march integrated or slow step in idle/approach)* | `8` *(Heavy cleave + slam)* | `2` | **`16` frames** *(in `256 × 256` overscan or `128 × 128` scaled cell — see `06_unity_handoff.md`)* | **`12 fps`** | RGBA PNG, `pivot = (0.50, 0.08)`, Unity Atlas `<= 2048 × 2048` |

- **Decoupled Visual Timing Rule (`07_UNITY_ART_UX.md` line 25):** Sprite frame loops (`12 fps`) and VFX bursts (`<= 20` simultaneous) are strictly cosmetic and **never** drive or alter `10 Hz` authoritative backend combat math.
