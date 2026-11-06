# ART-010A — Modular Fantasy Village Art Direction & Building System Review

**Style Identity:** `Suncrest Meadow Modular Village v1` (100% Original Fantasy IP; High-Level Base-Building Composition Inspired by *Clash of Clans*)  
**Perspective:** Elevated `3/4` Isometric Diamond Grid (`~42°` pitch)  
**Target Viewport:** Portrait `1080×1920` (`9:16`) + Genuine `360×640` Mobile Downscales

---

## 1. Why Previous Directions Were Rejected & What Changed

1. **Why `ART-009` Failed Visually:** Over `59%` of the frame was flat olive-green void, road tiles formed jagged `64×64` staircases, buildings were tiny flat geometric boxes inside a `<9%` gray circle, and `48 px` characters collapsed into `15 px` specks at `360×640`.
2. **Why the First `ART-010` Exploration Was Rejected by the Owner:** Even though detailed, Concepts A/B/C framed monolithic castles, mountain cliffs, narrow bridges, and waterfalls—reading like standalone portrait fantasy illustrations rather than a **modular village-building defense game** that can be constructed from discrete runtime sprites.
3. **Structural Reset in `ART-010A`:**
   - **The Village Is the Protagonist:** Occupies `~45%` of the active portrait viewport (`y = 28%..75%`, `x = 4%..81%`) on a bright, sunlit emerald meadow clearing bordered by stylized pine/oak trees.
   - **Colorful Stylized 3D-Rendered-as-2D Craft:** Chunky beveled timber beams, pale-granite masonry, saturated colour-coded roofs, warm ambient occlusion, and consistent `135°` bottom-right cast shadows.
   - **100% Plausible Runtime Modularity:** Every structure sits on a discrete isometric diamond plinth (`4×4`, `3×3`, `2×3`, `2×2`) separated by cobblestone roads and grass margins—zero cliffs, zero waterfalls, zero inseparable painted megastructures.

---

## 2. Original 9-Building Modular Village System (`buildings/modular_buildings_closeup_sheet.png`)

All nine archetypes strictly respect `ttg-rules-v1` (**no** gold/elixir currencies, **no** auto-shooting cannons/archer towers, **no** building-specific HP pools beyond canonical `Townhall` and `Wall`):

1. **Central Townhall (`Nhà Chính` — `4×4` Plinth, Authoritative `500/500 HP`):** Two-story ashlar-and-oak guildhall with steep royal-cobalt-blue shingled roofs (`#2358A6`), burnished gold ridge trim (`#F5BA36`), octagonal bell cupola, glowing stained-glass windows, and sun-crest banners.
2. **Builder Workshop (`Xưởng Thợ Xây` — `3×3` Plinth, Visual Anchor for `XÂY`):** Open timber carpenter/mason yard with yellow-and-cream striped awning (`#E59A24`), wooden A-frame pulley crane, sawhorse, lumber stack, iron anvil, and oversized builder mallet.
3. **Guard Barracks (`Trại Thủ Vệ` — `3×3` Plinth, Visual Anchor for `THỦ`):** Fortified stone-and-oak hall with cerulean-blue slate roof (`#2E73B8`), bold azure kite-shield crest above the arch, rack of tower shields, and wooden training dummy.
4. **Fighter Barracks (`Trại Chiến Binh` — `3×3` Plinth, Visual Anchor for `ĐÁNH`):** Heavy timber-and-iron hall with crimson-red tiled roof (`#B83232`), crossed greatsword emblem, broadsword rack, and spinning practice pell.
5. **Wood Storage (`Kho Gỗ` — `2×3` Plinth, Visual Cue for `GỖ`):** Cedar-roofed open lumber pavilion storing strapped pyramids of cut pine logs, plank racks, and a woodcutter's chopping block.
6. **Stone Storage (`Kho Đá` — `2×3` Plinth, Visual Cue for `ĐÁ`):** Stone-curbed masonry yard holding dressed limestone and blue-granite blocks, a timber tripod rope hoist, and wheelbarrow.
7. **Food Granary (`Kho Lương` — `3×3` Plinth, Visual Cue for `LƯƠNG`):** Charming octagonal stone-and-timber harvest barn with golden-wheat thatch roof (`#E8B339`), wooden windmill sails, grain sacks, and apple/bread barrels.
8. **Village Houses (`Nhà Dân` — `2×2` Plinths, Emotional Life):** Cozy half-timbered cottages with warm terracotta roofs (`#D4653B`), smoking stone chimneys, flower boxes, and lantern glow.
9. **Watchtowers & Connected Perimeter Wall (`Tháp Canh & Tường Thành Liên Kết`):**
   - **Watchtowers (`2×2`):** Square pale-granite lookout towers with iron fire-braziers and banners (**strictly visual defense architecture — zero automatic weapon mechanics**).
   - **Connected Wall (`buildings/wall_states_and_unit_scale_sheet.png`):** One single continuous stone-and-oak perimeter loop (`corner`, `straight`, `double oak gate`) with 5 visual condition states (`Intact`, `Damaged`, `Critical`, `Breached`, `Repair Scaffolding`) driven by the single canonical `Wall HP` pool—never compartmentalized into fake multi-health grids.

---

## 3. Three Layout Variations Within One Unified Art Style

All three variants use the **exact same art style, camera pitch, building assets, doorway-matched unit scale (`~70%` of door height), and Royal Timber & Gold HUD**:

- **Variant 1 — Compact Walled Village (`V1_compact_walled_village`):** Classic symmetrical octagonal compound; Townhall dead-center on a cobblestone ring road framed symmetrically by Guard/Fighter Barracks (north), Workshop/Granary (mid), Wood/Stone Storages (flanks), and 3 Houses (south).
- **Variant 2 — Layered Defensive Village (`V2_layered_defensive_village`):** Bold chevron-angled northern wall with 3 Watchtower gates opening onto a dedicated front Military Muster Plaza (`Builder Workshop`, `Guard Barracks`, `Fighter Barracks`), separated by a stepped banner promenade (not a second wall!) from the protected southern Civic & Resource Core (`Townhall`, `Storages`, `Houses`).
- **Variant 3 — Living Fantasy Settlement (`V3_living_fantasy_settlement`):** Warm sunlit town plaza with central stone well, bunting flags, lantern posts, bustling western artisan/supply quarter (`Workshop`, `Wood Storage`, `Stone Storage`), northern gate barracks, and cozy eastern/southern harvest & residential cottages (`Food Granary` + `Houses`).
