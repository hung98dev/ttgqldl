# ART-010 — Before/After Comparison Report & Recommended Art Direction

**Baseline Evaluated:** `ART-009` (`art/production/ART-009/previews/corrected_hud/*.png` & `mobile/*.png`)  
**Candidate Directions:**
- **Concept A (`A_fantasy_diorama`):** Fantasy Diorama Kingdom
- **Concept B (`B_action_defense`):** High-Impact Action Defense
- **Concept C (`C_vertical_siege`):** Vertical Siege Battlefield

> **METHODOLOGY DISCLOSURE (HONEST REPORTING):**  
> All `1–10` ratings below are **expert heuristic art-direction and technical-UX judgments** grounded in direct visual inspection of the `1080×1920` and true `360×640` deliverables plus machine-measured composition metrics (`qa/composition_metrics.json`). They are **NOT** results from human usability testing or external livestream viewer surveys. Note that for *Unity implementation cost*, higher scores (`10`) mean **more economical / cleaner to implement** (`1` = prohibitively expensive), and for *Existing asset reuse*, higher scores (`10`) mean **greater direct reuse** (`ART-009` = `10/10` baseline since its assets already exist).

---

## 1. Phase 7 — Expert Heuristic Comparison Matrix (`1–10` Scale)

| Criterion | Old ART-009 | Concept A (`Diorama`) | Concept B (`Action`) | Concept C (`Vertical 9:16`) |
|---|---:|---:|---:|---:|
| **Overall beauty** | **3 / 10** | **9.5 / 10** | **8.8 / 10** | **9.2 / 10** |
| **Castle visual impact** | **2.5 / 10** | **9.2 / 10** | **8.7 / 10** | **9.3 / 10** |
| **World depth** | **2 / 10** | **9.3 / 10** | **8.4 / 10** | **9.6 / 10** |
| **Character readability** | **3 / 10** | **8.4 / 10** | **9.6 / 10** | **8.9 / 10** |
| **Combat readability** | **3.5 / 10** | **8.3 / 10** | **9.5 / 10** | **9.2 / 10** |
| **360p readability** | **2.5 / 10** | **8.2 / 10** | **9.4 / 10** | **9.1 / 10** |
| **Boss visual impact** | **3 / 10** | **8.9 / 10** | **9.6 / 10** | **9.3 / 10** |
| **HUD clarity** | **6.5 / 10** | **9.0 / 10** | **9.1 / 10** | **9.3 / 10** |
| **TikTok-safe composition** | **6.0 / 10** | **8.8 / 10** | **9.0 / 10** | **9.8 / 10** |
| **Unity implementation cost** *(higher = simpler/cheaper)* | **9.5 / 10** | **7.2 / 10** | **7.8 / 10** | **8.0 / 10** |
| **Existing asset reuse** *(higher = more reused)* | **10 / 10** | **5.5 / 10** | **5.0 / 10** | **5.5 / 10** |

---

## 2. Criterion-by-Criterion Score Justifications

1. **Overall beauty (`3.0` vs `9.5` / `8.8` / `9.2`):**  
   `ART-009` (`3/10`) looks like an early developer test scene (`59.3%` empty olive-green void, stair-stepped road tiles). `Concept A` (`9.5/10`) achieves the warmest, most charming golden-hour storybook kingdom illustration. `Concept C` (`9.2/10`) delivers dramatic epic-fantasy majesty with mist canyons, twin waterfalls, and amber village windows. `Concept B` (`8.8/10`) delivers gritty, high-energy dusk battle spectacle.
2. **Castle visual impact (`2.5` vs `9.2` / `8.7` / `9.3`):**  
   `ART-009`'s flat gray ring (`<9%` canvas) lacks architectural silhouette. `Concept A` (`9.2/10`) showcases a steep royal-blue gabled Townhall and circular keep. `Concept C` (`9.3/10`) crowns a sheer cliff bastion above terraced village roofs, making the fortress an unmistakable vertical anchor. `Concept B` (`8.7/10`) frames massive close-up battlements and twin spires.
3. **World depth (`2.0` vs `9.3` / `8.4` / `9.6`):**  
   `ART-009` (`2/10`) sits on a single flat plane (`luma_entropy_bits = 5.338`). `Concept C` (`9.6/10`) creates four distinct vertical elevation planes from distant misty peaks -> arched viaduct -> outer rampart -> terraced village & waterfalls (`luma_entropy_bits = 7.217`). `Concept A` (`9.3/10`) layers forested hills, stream bridge, castle keep, and foreground cottages (`luma_entropy_bits = 7.748`). `Concept B` (`8.4/10`) trades far-horizon vista for tight close-up depth.
4. **Character readability (`3.0` vs `8.4` / `9.6` / `8.9`):**  
   `ART-009`'s `48 px` units blend into matching roof/wall colors. `Concept B` (`9.6/10`) leads decisively with `~110–126 px` characters (`2.5×` taller), bold silhouettes, and rim lighting against dark slate ground. `Concept C` (`8.9/10`) frames `~88–100 px` units cleanly against smooth stone bridges and ramparts. `Concept A` (`8.4/10`) is clear and expressive, though rich foliage/masonry requires slightly more background value control than B or C.
5. **Combat readability (`3.5` vs `8.3` / `9.5` / `9.2`):**  
   `ART-009` (`3.5/10`) uses tiny white puff circles with weak lane separation. `Concept B` (`9.5/10`) makes every sword cleave, azure shield wall, and anvil repair spark unmistakable. `Concept C` (`9.2/10`) separates the three incoming enemy lanes across three physical stone structures (left stairs, center bridge, right ledge) converging into a single readable shield-dome killzone. `Concept A` (`8.3/10`) reads well across river/forest/cliff lanes with warm sunlit atmosphere.
6. **360p readability (`2.5` vs `8.2` / `9.4` / `9.1`):**  
   Verified on `comparisons/all_three_360p.png` (true `1:1` LANCZOS downscales, `PSNR = 99 dB`). In `ART-009` (`2.5/10`), units become `15 px` noise dots. `Concept B` (`9.4/10`) retains `37–42 px` unit silhouettes and bold VFX arcs even on a budget phone viewport. `Concept C` (`9.1/10`) preserves instant 3-lane, shield-wall, castle, and Boss legibility due to strong light/dark zone separation. `Concept A` (`8.2/10`) remains clearly understandable, with slightly busier roof/tree texture at `360p`.
7. **Boss visual impact (`3.0` vs `8.9` / `9.6` / `9.3`):**  
   `ART-009`'s Boss (`3/10`) is a small purple figure on empty grass. `Concept B` (`9.6/10`) presents a towering horned obsidian dreadlord charging a cracked arcane sigil directly before the defenders' shield wall. `Concept C` (`9.3/10`) stages the Boss striding down the central viaduct summoning violet lightning onto the bastion dome. `Concept A` (`8.9/10`) shifts the warm kingdom into ominous twilight as the Boss crosses the stone bridge.
8. **HUD clarity (`6.5` vs `9.0` / `9.1` / `9.3`):**  
   `ART-009` (`6.5/10`) has accurate data (`ART-009` hotfix) but consumes `33%` of vertical height with flat debug boxes. All three `ART-010` concepts compact the HUD into two sculpted fantasy plaques (`top 10.5%`, `bottom 11.5%`), freeing `65.3%` continuous vertical battlefield while increasing CTA font prominence (`Bình luận XÂY / THỦ / ĐÁNH`). `Concept C`'s cool slate-and-gold plaque (`9.3/10`) achieves the crispest contrast.
9. **TikTok-safe composition (`6.0` vs `8.8` / `9.0` / `9.8`):**  
   While `ART-009` (`6.0/10`) keeps HUD boxes inside `x <= 885`, its empty right flank looks unfinished under TikTok's icon column. `Concept C` (`9.8/10`) solves TikTok's asymmetric `9:16` UI natively by placing the sheer chasm and twin waterfalls inside `x = 886..1080` (`rightmost 18%`), so TikTok's avatar/like/gift buttons float over atmospheric mist/waterfalls without ever blocking a single castle tower, lane, or unit!
10. **Unity implementation cost (`9.5` vs `7.2` / `7.8` / `8.0`):**  
    `Concept C` (`8.0/10`) and `Concept B` (`7.8/10`) have clean, modular stone boundaries (cliff layers, bridge plate, circular rampart, Townhall cluster) that slice cleanly into `4–5` static Built-in 2D `SpriteRenderer` planes. `Concept A` (`7.2/10`) requires slightly more Y-sorted tree/fence/cottage prop sprites to preserve depth around winding roads.
11. **Existing asset reuse (`10.0` vs `5.5` / `5.0` / `5.5`):**  
    All three new directions reuse `100%` of `ART-007` audio (`15` cues), `100%` of `ART-008`/`ART-009` C# HUD/simulation/state controllers, `100%` of `ART-006` 16-state UX/localization contracts, and `100%` of `ART-003`/`ART-004` animation timing contracts, while intentionally replacing `ART-005` world art, `ART-006` UI skins, and repainting `ART-003`/`ART-004` character/VFX sprite sheets (`Concept B`'s `2.4×` character scale requires the largest sprite canvas).

---

## 3. Phase 8 — Objective Synthesis & Recommended Direction

| Category | Winner | Reason |
|---|---|---|
| **Best-Looking Pure World Illustration** | **Concept A** (`Fantasy Diorama Kingdom`) | Unmatched storybook warmth, architectural charm, and lush golden-hour atmosphere. |
| **Most Visceral Combat & `360p` Unit Legibility** | **Concept B** (`High-Impact Action Defense`) | Largest character silhouettes (`37–42 px` at `360p`) and punchiest sword/shield/Boss VFX. |
| **Most Readable Portrait Layout & Best Production Compromise** | **Concept C** (`Vertical Siege Battlefield`) | Purpose-built `9:16` vertical 3-bridge convergence, cool-vs-warm lighting hierarchy, natural waterfall clearance for TikTok's right-rail UI, and cleanest modular 2D layer slicing. |

### Primary Recommendation: **Concept C Spatial Architecture + Targeted Elements of B & A (Defined Hybrid)**

Because **scroll-stopping visual quality on TikTok LIVE (`9:16` portrait)** is the primary mission of `ART-010`, we recommend approving **Concept C (`Vertical Siege Battlefield`) as the master spatial & lighting foundation**, specifically combined with two concrete strengths from `B` and `A` for `ART-010B`:

1. **From `Concept C` (Master `9:16` Composition & TikTok Safe Architecture):**
   - Retain the native vertical siege layout: three elevated northern approach bridges/paths converging down into the semi-circular stone rampart, terraced citadel below, and the natural cliff/waterfall vista occupying the rightmost `18%` TikTok UI zone (`x > 885`).
   - Retain the cool blue-slate canyon lighting for incoming enemies vs warm sanctuary lighting for the village.
2. **From `Concept B` (Heroic Character Scale & High-Impact Combat VFX):**
   - Increase gameplay unit height by `+15–20%` over Concept C's current mockup (`~100–112 px` at `1080p` / `33–37 px` at `360×640`) and adopt `Concept B`'s bolder crimson sword-cleave arcs, layered azure shield-block ripples, and cracked Boss rune-slam sigil so combat pops effortlessly on small screens.
3. **From `Concept A` (Warm Storybook Sanctuary Detail):**
   - Infuse the lower-middle Townhall roofs, timbered cottages, banners, and courtyard lanterns (`y = 880..1420`) with `Concept A`'s richer golden-amber wood/slate warmth so the fortress feels like a beloved living home rather than a cold stone outpost.

*(If you prefer a single pure direction without hybridization, **Concept C** is our #1 standalone pick for TikTok LIVE `9:16`, followed closely by **Concept B** for pure action punch and **Concept A** for cozy kingdom appeal.)*
