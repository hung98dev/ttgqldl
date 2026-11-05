# ART-002 Deliverable C — Village-Defense Environment Design System

- **Work Package:** `ART-002`
- **Style Version:** `bastion-dawn-v1`
- **Environment Directory:** `environment/`
- **Binding Authority:** [07_UNITY_ART_UX.md](../../../docs/plan/07_UNITY_ART_UX.md) (`UX001`, `ART001`, `ART004`), [02_GAME_RULES.md](../../../docs/plan/02_GAME_RULES.md), [03_SIMULATION.md](../../../docs/plan/03_SIMULATION.md)

---

## 1. Delivered Visual Environment Sheets

All seven required environment design plates exist in `environment/` (with unedited raw generation plates archived in `environment/source/`):

| Sheet File | Dimensions | Purpose & Covered Elements |
|---|---:|---|
| `environment/environment_overview.png` | `2400 × 2000` | Full 9:16 village-defense composition with numbered callouts for all **12 canonical environment elements**, plus active Wave 10 Boss Siege and Post-Loss Breached Recovery comparisons |
| `environment/castle_model_sheet.png` | `2300 × 1560` | Central Castle / Townhall landmark design across **4 cosmetic progression bands** (`Lv 1–9`, `Lv 10–24`, `Lv 25–39`, `Lv 40–50`; `townhall_hp = 500` unchanged) |
| `environment/wall_model_sheet.png` | `2300 × 1520` | Main Defensive Curtain Wall, South Gate, and Builder Repair Scaffolding across **5 structural/damage states** (`100%–60%`, `59%–25%`, `<25%`, `0 HP Breached`, `Post-Loss 250 HP Recovery`) |
| `environment/three_lane_layout.png` | `2300 × 1960` | Tactical lane geometry (`West 34%`, `Center 32% + Boss`, `East 34%`), unit muster anchors, Boss spawn & telegraph ring, and safe-zone compliance (`x <= 82%`, `y <= 88%`) |
| `environment/environment_tilesheet.png` | `2200 × 1500` | **6 low-frequency H.264-safe ground/lane tiles** (`256×256` source spec) + **4 visual-only wave lighting palettes** (`Dawn`, `Midday`, `Dusk`, `Violet Boss Eclipse`) |
| `environment/building_variations.png` | `2300 × 1520` | **1 defensive stone-and-timber watchtower**, **6 non-blocking courtyard village houses**, **1 modular Builder scaffold**, and **3 low-profile courtyard props** (cart, stone crate, sun banner) |
| `environment/environment_scale_guide.png` | `2360 × 1980` | Exact pixel footprints on the `1080 × 1920` reference canvas (`477 px = 44.2%` bastion width) + live side-by-side unit scale verification (`48 px` roles/enemies, `64 px` Brute, `102 px` Boss) |

---

## 2. Complete Specification of the 12 Canonical Environment Elements

| # | Element | Screen Coordinates (`1080×1920`) | Palette & Material Rules | Gameplay & Readability Role |
|---:|---|---|---|---|
| 1 | **Central Castle / Townhall** | `x = 390..522`, `y = 635..805` (`~132 × 170 px`) | Cobalt-slate roof (`#3B6EA5`), warm limestone (`#E8DFC8`), oak beams (`#8C5A32`), golden sun crest (`#F4C430`) | Unmistakable village heart; visually communicates progression (`townhall_level 1..50`) while authoritative `townhall_hp = 500` stays fixed |
| 2 | **Main Defensive Curtain Wall** | `x = 215..692`, `y = 680..1040` (**`477 px` wide = `44.2%`**) | Cut limestone ashlar (`#9BA5B0`), parapet walk (`#CBD5E1`), `3.0 px` `#11151C` outer silhouette | Primary damage sponge (`wall_tier 1..20`, `wall_max_hp = 1000 + 200*(tier-1)`); dominates central composition |
| 3 | **West Battle Lane** | `(0, 0)` -> `(335, 675)` (`~74 px` road width) | Smooth packed ochre earth (`#CFA96E`) with `#9E7E4C` verge | Carries `round(enemy_hp_pool * 0.34)` threat; angled `32°` NW->SE |
| 4 | **Center Battle Lane** | `x = 411..497`, `y = 0..660` (`86 px` wide) | Pale cut limestone causeway (`#B8C0C8`) with `#4E5662` stone curbs | Carries `enemy_hp_pool - W - E` (`~32%`) + **exclusive Boss approach lane** |
| 5 | **East Battle Lane** | `(876, 288)` -> `(575, 675)` (`~74 px` road width) | Smooth packed ochre earth (`#CFA96E`) terminating strictly left of `x = 885 px` | Carries `round(enemy_hp_pool * 0.34)` threat; `100%` clear of right-hand TikTok icon rail (`x > 82%`) |
| 6 | **Main South Gate** | `x = 414..494`, `y = 955..1042` (`80 × 87 px`) | Iron-banded oak double doors (`#8C5A32` + `#475569`) + cobalt arch roof | Visual muster anchor for **Guard (`THỦ`)** shield-wall formation |
| 7 | **Builder Repair Scaffolding** | 4 wall-flank attachment slots (`NW`, `NE`, `SW`, `SE`) | Modular pegged timber frame (`#9C6644`) + golden `4-point` repair sparks during `20 s` Builder cadence | Appears cosmetically during `PREP` repairs and post-damage reconstruction; never blocks enemy lanes |
| 8 | **Defensive Watchtower** | `x = 308..378`, `y = 632..778` (`70 × 146 px`) | Cylindrical/square limestone tower + cobalt conical slate cap (`1` canonical landmark tower) | Reinforces stronghold silhouette on upper-left bastion rim without implying non-canonical auto-turret rules |
| 9 | **Village Houses (`6` Variations)** | Inner courtyard ring (`x = 248..652, y = 735..965`) | Compact timber-and-plaster cottages (`<= 82 px` height) with muted slate roofs (`#4B6B94`) | Conveys living community ownership while staying strictly inside the curtain wall so zero lane units are occluded |
| 10 | **Environmental Decorations (`3` Props)** | Courtyard plaza (`x = 380..530, y = 815..890`) | Low-profile timber supply cart, cut-stone crate stack, village sun banner (`<= 36 px` height) | Adds scale cues for Wood/Stone/Food economy with zero tall trees or bushes in combat lanes |
| 11 | **Ground & Lane Tile System (`6` Tiles)** | Full `1080 × 1920` world plane (`256×256` tile masters) | Smooth sunlit meadow (`#68A347`), subtly shaded turf, ochre flank road, ochre curb, limestone Boss slab, plaza earth | Measured **`50.0%` lower Laplacian high-frequency noise** than `ART-001` (`10.87` vs `21.76`), preventing H.264 grass mosquito static |
| 12 | **Boss Spawn & Telegraph Area** | Center causeway `x = 368..540`, `y = 440..595` | Concentric `#E879F9` / `#D946EF` runic ground ring (`168 px` diameter) + `Violet Boss Eclipse` world tint | Activates during `FINALE` (`t = 240..260s` warning) and `BOSS` (`tick 2600` on `wave % 5 == 0`) |

---

## 3. Castle Tier Progression & Wall Damage State Rules

### 3.1 Castle / Townhall Cosmetic Progression (`castle_model_sheet.png`)
Per `docs/plan/02_GAME_RULES.md`, `townhall_level` (`1..50`) is cosmetic village progression while **`townhall_hp = 500` remains fixed** across waves. To keep asset cost controlled while rewarding multi-wave viewers, `50` levels map cleanly to **4 visual landmark bands**:
1. **Band 1 (`Lv 1–9` — Timber Village Hall):** Fieldstone plinth + exposed oak half-timbering + slate gable roof + small sun pennant (P0 initial default).
2. **Band 2 (`Lv 10–24` — Reinforced Stone Keep):** Full cut-limestone facade + twin slate dormers + iron-bound arched doors.
3. **Band 3 (`Lv 25–39` — Bastion Citadel):** Flanking round corner turrets + gilded roof ridge + sculpted golden sun-shield crest over main arch.
4. **Band 4 (`Lv 40–50` — Sovereign Dawn Keep):** Tall central belfry spire + multi-tier cobalt-and-gold roofs + celebratory banners (purely visual prestige; zero mechanical buff).

### 3.2 Curtain Wall & Gate Damage States (`wall_model_sheet.png`)
Regardless of `wall_tier` (`1..20`, `wall_max_hp = 1000..4800`), the wall sprite swaps cleanly across **5 normalized HP ratio states**:
1. **State 1 — Healthy (`100%–60%` HP):** Crisp intact limestone merlons, closed iron-bound gate, clean mortar.
2. **State 2 — Chipped & Scuffed (`59%–25%` HP):** Stepped masonry cracks, chipped crenelations, scorched gate planks, active Builder scaffolds on flanks.
3. **State 3 — Critical (`24%–1%` HP):** Deep structural fissures, partially buckled gate leaves, knocked-out merlons, warning dust puffs.
4. **State 4 — Breached (`0` HP):** Collapsed central gate arch and breached parapet allowing enemies to strike Townhall HP directly.
5. **State 5 — Post-Loss Recovery (`250` HP Floor):** Weathered stone braced with fresh oak Builder scaffolding (`wall_hp = max(wall_hp, 250)`, `townhall_hp = max(townhall_hp, 250)` per `ttg-rules-v1`).

---

## 4. Unity 2D Sorting Layer & Occlusion Contract

To guarantee combat units are never hidden behind decorative roofs or props (`§7.1`), Unity `01_BroadcastWorld` must enforce six deterministic sorting layers:

| Order | Unity `SortingLayer` | Assigned Assets | `Order in Layer` Rule |
|---:|---|---|---|
| `0` | `Ground` | Smooth meadow base, low-frequency sunlit cel zones | `0..10` |
| `1` | `LanesAndDecals` | West/East ochre lanes, Center limestone road, Boss concentric ground telegraph ring | `0..20` |
| `2` | `BastionBack` | North wall arc, courtyard earth, inner houses (`6`), courtyard props (`3`), castle keep | Dynamic `-round(worldY * 100)` |
| `3` | `Units` | Pooled Builders, Guards, Fighters, Runners, Grunts, Brutes, Boss (`<= 150` total pooled entities) | Dynamic `-round(pivotY * 100)` |
| `4` | `BastionFront` | South wall rim & gate arch lip (kept `<= 32 px` parapet height so South Guards remain `100%` visible in front of the gate) | Fixed local offset |
| `5` | `VFX` | Builder repair sparks, sword slash arcs, hit flashes, Boss slam shockwave (`<= 20` simultaneous bursts) | `0..50` |
