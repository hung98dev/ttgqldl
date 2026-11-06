# ART-010A — Unity 6000.6.7f1 Modular Sprite Reconstruction Feasibility Note

**Target Engine:** Unity `6000.6.7f1` — Built-in 2D Render Pipeline  
**Target Viewport:** Portrait `1080×1920` (`9:16`), Orthographic Camera, `60 FPS` (`<= 150` pooled dynamic entities)  
**Core Rule:** **No single flattened background wallpaper.** In `ART-010B` (post-approval), the approved village layout will be assembled from discrete, reusable 2D modular sprites on an isometric diamond placement grid.

---

## 1. Isometric Grid & Camera Specification

- **Projection:** Elevated `3/4` isometric diamond grid (`2:1` screen-space diamond tile ratio, e.g., `128×64 px` per `1×1` logical village cell at `1080×1920`, `PPU = 100`).
- **Sorting Strategy (`CustomAxisSort` + Deterministic Tie-Break):**
  - Unity Project Settings -> `Transparency Sort Mode = Custom Axis (0, 1, -0.01)` across the `VillageSortingLayer`.
  - Every modular building, wall segment, gate, watchtower, decoration, and character places its sprite `Pivot` at the **front-most ground contact point of its isometric diamond footprint** so units walking along cobblestone roads sort cleanly in front of northern walls and behind southern buildings/walls with zero Z-fighting.

---

## 2. Planned Modular Sprite Breakdown for `ART-010B` (Post-Approval Scope)

Rather than painting one monolithic canvas, the approved settlement decomposes into **`~38–46` crisp static environment sprites** packed into **two `2048×2048` RGBA32 atlases** (`Atlas_Village_Terrain_Walls` + `Atlas_Village_Buildings_Props`):

| Asset Category | Planned Modular Pieces (Post-Approval `ART-010B`) | Grid Footprint | Runtime Role in `ttg-rules-v1` |
|---|---|---:|---|
| **1. Meadow Ground & Outer Trees** | Seamless mown-meadow grass base + `4` grass variation decals + `4` pine/oak tree clusters + `3` mossy rocks/wildflower tufts | Tile / Free | Static background (`SortingOrder = -500`) + outer safe-zone framing (`x > 885`, `y > 1689`) |
| **2. Cobblestone Roads & Courtyards** | Straight road (`NE-SW`, `NW-SE`), `T`-junction, central plaza patch, dirt-road outer lane transitions (`West`, `Center`, `East`) | `1×1` / `2×2` | Static ground decals (`SortingOrder = -400`) guiding the 3 enemy lane splines |
| **3. Central Townhall** | `Townhall_Tier1..Tier3` cosmetic upgrade sprites + separate grounded contact shadow | `4×4` | Authoritative `Townhall HP` (`500/500`) + village level cosmetic progression |
| **4. Role & Resource Buildings (7)** | `Builder_Workshop`, `Guard_Barracks`, `Fighter_Barracks`, `Wood_Storage` (`empty/half/full` log overlays), `Stone_Storage`, `Food_Granary` (+ rotating windmill vane sub-sprite!), `Village_House` (`2` color/chimney trims) | `3×3` / `2×3` / `2×2` | Decorative / presentation-only visual anchors (spawn/muster rally points for `Builder`/`Guard`/`Fighter`, visual fullness cues for `Wood`/`Stone`/`Food` — **zero** invented HP or currencies) |
| **5. Connected Wall, Gates & Towers** | `Wall_Straight_NE`, `Wall_Straight_NW`, `Wall_Corner_N/S/E/W`, `Gate_Double_Oak`, `Watchtower_Brazier` — each northern combat-facing piece authored in **`4` damage sprites (`Intact`, `Damaged`, `Critical`, `Breached`)** + attachable **`Repair_Scaffolding` overlay** | `1×1` / `2×1` / `2×2` | All segments read from the **single authoritative `Wall HP` ratio** (`100%..75%` Intact, `75%..35%` Damaged, `<35%` Critical, `0%` Breached) + active `Builder` repair state |
| **6. Independent Characters (7)** | `Builder`, `Guard`, `Fighter`, `Grunt`, `Runner`, `Brute`, `Boss` (`128×128` unit cells / `256×256` Boss cell, doorway-matched `~72 px` height at `1080p`) + soft multiply foot-shadow oval | Dynamic | Pooled `SpriteRenderer` entities (`<= 150` max) following canonical lane/muster waypoints |
| **7. Layered Combat & Upgrade VFX** | `Hammer_Repair_Spark`, `Guard_Shield_Flash`, `Guard_Finale_Dome`, `Fighter_Sword_Arc`, `Wall_Rubble_Burst`, `Boss_Arcane_Rune_Ring`, gentle chimney smoke loop | Dynamic | Pooled alpha/additive `SpriteRenderer` clips (`6–8` frames @ `12 FPS`) |

---

## 3. Why All Three Variants Are 100% Reconstructible Without Engine Risk

1. **Zero Perspective Distortion Across the Map:** Orthographic isometric projection means the exact same `Village_House` (`2×2`) or `Wall_Straight` (`1×1`) sprite can be stamped anywhere on the grid without perspective warping.
2. **Low Draw-Call & Memory Footprint:**
   - `2` Environment/Building Atlases (`2048×2048`) + `1` Character/VFX Atlas (`2048×2048`) + `1` Sculpted Timber/Gold UI Atlas (`1024×1024`) = **`~11 MB` compressed VRAM (ASTC 6×6 / DXT5)**.
   - Unity's `SpriteAtlas` + Built-in 2D Dynamic Batching renders the entire village, walls, `<= 150` units, and VFX in **`~16–24` draw calls** (`>60 FPS` headroom on host PC and mobile preview).
3. **Zero Simulation or Backend Changes Required:** Supporting buildings (`Barracks`, `Workshop`, `Storages`, `Houses`, `Watchtowers`) bind purely to existing read-only `CommittedView` fields (e.g., spawning decorative `Builder` walkers from `Builder_Workshop` toward wall scaffolding, or toggling `Wood_Storage` log piles at `wood >= 300`), preserving `ttg-rules-v1` 100% intact.
