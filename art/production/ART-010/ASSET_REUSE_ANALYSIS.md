# ART-010 — Package-by-Package Asset Reuse Analysis & Unity Built-in 2D Feasibility

**Engine Target:** Unity `6000.6.7f1` — Built-in 2D Render Pipeline (Orthographic Camera, `SpriteRenderer` sorting layers, uGUI + TextMeshPro)  
**Performance Guardrails (`07_UNITY_ART_UX.md`):** `60 FPS` target, `<= 150` pooled active sprites, `<= 2048×2048` RGBA32 atlases, zero heavy post-processing shaders required.

---

## 1. Guiding Principle: Minimum Replacement Scope for Commercial Visual Quality

`ART-010` exists because visual quality—not pipeline plumbing—is currently the bottleneck. Rather than throwing away working engineering or keeping ugly placeholder art out of convenience, we separate **data/audio/code systems (`100%` reusable)** from **visual surface sprites (targeted replacement/upgrade)**.

| Existing Package | Current Deliverable | Reuse in Concept A | Reuse in Concept B | Reuse in Concept C | Recommended Action for `ART-010B` |
|---|---|---|---|---|---|
| **`ART-003` & `ART-004`** (7 Character Classes, 106 frames @ `48 px` gameplay scale) | `Builder`, `Guard`, `Fighter`, `Grunt`, `Runner`, `Brute`, `Boss` sprite sheets + `12 FPS` timing JSONs | **Timing & state contract 100% reused**; sprite art needs **repaint/reshade at `1.8×` scale** (`~84 px` screen height) | **Timing & state contract 100% reused**; sprite art **must be repainted at `2.4×` scale** (`~115 px` screen height) — upscaling `48 px` sprites `2.4×` looks blurry/blocky | **Timing & state contract 100% reused**; sprite art **repainted at `2.0×` scale** (`~96 px` screen height) to match crisp inked-painterly stone world | **Keep** exact 7 class IDs, pivot rules, `idle`/`move`/`work`/`attack`/`block` frame counts (`16–18f @ 12 FPS`), and hit-frame indices; **repaint** `256×256` master frames in cohesive painted 2D style (`1` character atlas `2048×2048`). |
| **`ART-003` & `ART-004`** (5 Combat/Repair VFX, 30 frames) | `repair_spark`, `unit_hit_puff`, `guard_block_spark`, `wall_hit_dust`, `boss_slam_ring` | **Keep** `repair_spark` & `wall_hit_dust` with warmer glow; **redesign** `unit_hit_puff`, `guard_block_spark`, `boss_slam_ring` | **Redesign all 5 VFX** into bold directional sword arcs, azure shield-dome bursts, anvil sparks & rune slam sigils | **Keep/upgrade** `repair_spark` & `wall_hit_dust`; **redesign** 3 combat/Boss VFX for high contrast against slate stone | **Replace/upgrade** 5 VFX strips (`30` frames total — small, high-leverage scope!) with directional slash arcs, crisp shield-block ripples, and glowing Boss rune telegraphs (`additive/alpha` blend sprites). |
| **`ART-005`** (Environment, Ring Castle, 3 Lanes, `2048` Atlases) | Flat green ground, stair-stepped `64×64` lane tiles, small gray ring wall, geometric huts | **0% visual sprite reuse** (`100%` layout/slot contract reuse: `Wall Lv1–7`, `Townhall Lv1–5`, `3` lane splines) | **0% visual sprite reuse** (`100%` state machine reuse for intact/damaged/critical wall & gate states) | **0% visual sprite reuse** (`100%` state machine & 3-lane pathing waypoint contract reuse) | **Full visual replacement (highest priority of `ART-010B`):** Replace flat tile grid with **layered painted 2D environment slices** (Sky/Backdrop, Canyon/Forest Midground, Curved 3-Lane Terrain Base, Modular Castle/Wall state sprites, Baked Contact Shadows, Foreground Cliff/Tree props). |
| **`ART-006`** (16-State HUD, 20 Icons, 15 Nine-Slice Skins) | Safe-area layout JSONs, 16-state rules, Vietnamese copy, flat `#1E2430` 9-slices & geometric icons | **100% layout/state/copy reuse**; **reskin** 15 nine-slice frames (oak & gold) + **repaint** core HUD icons | **100% layout/state/copy reuse**; **reskin** 15 nine-slice frames (obsidian iron & bronze) + **repaint** core HUD icons | **100% layout/state/copy reuse**; **reskin** 15 nine-slice frames (slate & antique gold) + **repaint** core HUD icons | **Keep** all 16-state visibility rules, Vietnamese localization strings, and `82%×88%` anchor rectangles; **reskin** `ui_atlas_01_icons` & `ui_atlas_02_components` with sculpted fantasy bevels (`~28` static UI sprites). |
| **`ART-007`** (12 SFX, 1 Ambient, 2 Music Loops) | `15` original WAV/OGG cues, loudness/loop contracts, `cue_manifest.json` | **100% direct reuse (zero changes needed)** | **100% direct reuse (zero changes needed)** | **100% direct reuse (zero changes needed)** | **100% Keep As-Is.** All 12 combat/UI/Boss SFX, village ambient (`36.00 s`), and calm/battle music loops fit all three fantasy directions seamlessly. |
| **`ART-008` & `ART-009`** (Canonical Fixtures & Unity C# Presenters) | `HudInPlacePresenter.cs`, `SpriteTransitionController.cs`, `BroadcastAudioDispatcher.cs`, `ttg-rules-v1` fixtures | **100% C# logic & canonical fixture reuse** (update lane world-space waypoints & camera ortho size) | **100% C# logic & canonical fixture reuse** (update lane world-space waypoints & closer ortho framing) | **100% C# logic & canonical fixture reuse** (update lane world-space waypoints & vertical sorting order) | **Keep 100% of C# presentation/audio/HUD code**, updating only serialized sprite references, `SortingOrder` Y-map, and 3-lane spline waypoints! |

---

## 2. Critical Character & VFX Answers (`PHASE 5` Checklist)

1. **Can existing `ART-003`/`ART-004` character sprites be reused directly?**  
   - **Animation timing, frame counts (`16–18f`), state names, and pivot contracts: YES (`100%`).**  
   - **Raw pixel bitmaps: NO for final commercial ship.** Existing characters were authored for `48 px` screen height (`16 px` at `360p`) with simplified pixel-art shading. Scaling them `2×–2.5×` over richly painted `1080×1920` environments creates an immediate "pasted low-res sprite on HD painting" mismatch. Repainting the `106` frames at `256×256` master resolution (`96–120 px` on-screen height) with matching directional rim light and grounded drop-shadow ovals makes characters belong natively inside the world.
2. **Are all 7 classes distinguishable at `360×640`?**  
   - Verified in `comparisons/all_three_360p.png`:
     - **Guard (`THỦ`):** Broad azure cape + tall tower shield + cyan phalanx aura (`~32–40 px` at `360p`).
     - **Fighter (`ĐÁNH`):** Crimson plume/cape + two-handed greatsword + warm orange/gold cleave arc.
     - **Builder (`XÂY`):** Bright yellow mason apron + raised war-hammer + golden repair sparks along wall battlements.
     - **Grunt / Runner / Brute:** Olive-green club infantry (`Grunt`), hunched violet twin-dagger flankers (`Runner`), and hulking rust-armored siege juggernauts (`Brute`, `1.45×` taller than Grunt).
     - **Boss (`FINALE`):** Towering dark-amethyst horned dreadlord (`2.6×–3.2×` unit height) with glowing rune-blade and magenta ground telegraph circle.
3. **Does the `3/4` perspective remain compatible?**  
   - **Yes.** All three concepts preserve a unified `3/4` elevated camera angle (`~35°–42°` pitch) where horizontal/diagonal facing sprites (`flipX` for West vs East lanes, front-three-quarter facing for Center lane) work without needing 8-directional sprite sets.

---

## 3. Concrete Unity `6000.6.7f1` Built-in 2D Architecture (How Static Art Becomes Real 60 FPS Gameplay)

None of the three concepts rely on 3D meshes, URP ShaderGraph, or expensive real-time lights. In `ART-010B`, the chosen direction translates cleanly into **Unity Built-in 2D** using deterministic layered sprites:

1. **5-Layer `SpriteRenderer` World Stack (`1` static `2048×2048` Environment Base + `1` `2048×2048` Modular Structures Atlas):**
   - `Layer 0 — Sky & Far Horizon` (`SortingOrder = -500`): Painted sky/mountain gorge plate + subtle scrolling mist alpha strip (`2` quads).
   - `Layer 1 — Integrated 3-Lane Battlefield Terrain` (`SortingOrder = -400`): Painted ground/bridges/cliffs with smooth organic roads baked into the art—eliminating `ART-005`'s blocky `64×64` tile grid and cutting terrain draw calls from `>300` tiles to **`1–2` sprites**.
   - `Layer 2 — Modular Castle & Wall States` (`SortingOrder = -200..+100`, Y-sorted): Separate transparent sprites for `Townhall_Lv1..Lv5` and `Wall_Gate_Intact / Damaged / Critical`, swapped deterministically by `SpriteTransitionController`.
   - `Layer 3 — Pooled Gameplay Units & Grounded Blob Shadows` (`SortingOrder = 100..800` by `-Mathf.RoundToInt(y * 10)`): Up to `<= 150` pooled unit `SpriteRenderer`s following 3 Bezier/polyline lane paths (`West`, `Center`, `East`), each with a soft `35%` multiply shadow oval at its pivot so units feel physically planted on stone/dirt.
   - `Layer 4 — Directional Combat VFX & Foreground Framing` (`SortingOrder = 850..950`): Pooled 6-frame `SpriteRenderer` slash arcs, shield domes, repair sparks, and subtle bottom/right cliff/tree foreground framing outside the combat lane.
2. **Diegetic uGUI + TextMeshPro Canvas (`1` `1024×1024` UI Atlas):**
   - Top Fortress Crest (`anchorMin/Max` pinned inside `x = 30..872, y = 20..202`) and Bottom Command Dock (`x = 30..872, y = 1458..1679`) using 9-sliced sculpted plaques, radial/linear filled glossy HP bars, and crisp SDF TextMeshPro Vietnamese fonts.
3. **Estimated Draw Calls & Memory Budget:**
   - **Atlases:** `4` textures (`Env_Base`, `Env_Structures_Props`, `Chars_VFX_2048`, `UI_Fantasy_1024`) = `~44 MB` uncompressed RGBA32 (`~11 MB` ASTC/DXT5).
   - **Batched Draw Calls:** `~14–22` draw calls total at peak Wave 10 (`60 FPS` trivial on mid-range mobile and livestream host PC).
