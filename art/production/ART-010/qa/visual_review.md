# ART-010A — Independent Visual QA Review (Revised Modular Village Defense)

- **Package:** `ART-010` (Phase A Revision — Clash-of-Clans-Inspired Modular Fantasy Village Defense)
- **Target Engine:** Unity `6000.6.7f1` — Built-in 2D (`1080×1920` portrait + genuine `360×640` mobile previews)
- **Unified Art Direction:** `Suncrest Meadow Modular Village v1` (elevated `3/4` isometric, colorful stylized 3D-like 2D game sprites, warm sunlit grass & stone clarity)
- **Inspected Deliverables:** `29` PNG images (`24` variant PNGs + `2` modular building/wall/scale sheets + `3` comparison boards) + `6` Markdown handoff docs + `2` QA JSON manifests
- **QA Verdict:** **`PASS_FOR_OWNER_LAYOUT_SELECTION`** *(Phase A exploration complete; awaiting owner village-layout choice before `ART-010B` modular asset production)*

---

## 1. Truthfulness & Scope Disclosure (Mandatory)

1. **Static Concept & Layout Exploration Only:** All `29` PNG files in `art/production/ART-010/` are **static Phase A art-direction, modular-building, and HUD layout mockups** created for owner approval. They are **NOT** Unity runtime playmode renders and make zero claim of runtime FPS, draw-call batching, or animated sprite playback.
2. **Production Masters Untouched:** Existing `ART-003`..`ART-009` character, VFX, environment, UI, audio, and Unity showcase assets remain 100% intact and unedited on `main`.
3. **100% Original Fantasy IP (`GATE-LEGAL = PENDING_REVIEW`):** High-level inspiration is taken strictly from the **spatial base-building clarity, modular footprints, and protective settlement appeal** of *Clash of Clans*. Zero proprietary Supercell meshes, textures, characters, logos, icons, fonts, or exact building silhouettes are copied. Per `07_COMPLIANCE_LEGAL_RISK.md`, formal legal sign-off (`GATE-LEGAL`) remains `PENDING_REVIEW` by counsel/owner.

---

## 2. Ten-Point Mandatory Visual QA Checklist (Section 10)

Every generated `1080×1920` plate, `360×640` native downscale, close-up sheet, and comparison board was inspected directly via image viewer (`read_file`) prior to sign-off:

| # | Validation Question | Result | Verified Evidence |
|---|---|---|---|
| 1 | **Does each variant immediately look like a village-defense game?** | **PASS** | All three variants (`V1`, `V2`, `V3`) present an elevated `3/4` isometric walled settlement on sunlit emerald meadow grass with individual modular footprints, cobblestone paths, active villagers/defenders, and converging enemy siege roads — completely eliminating the rejected monolithic cliff/waterfall fantasy-painting look. |
| 2 | **Do the three variants share one cohesive art style?** | **PASS** | All three variants share `Suncrest Meadow Modular Village v1`: identical `3/4` isometric camera angle, identical roof-tile palette (`#2E67C6` royal blue Townhall, `#3A7CA5` slate-blue Guard Barracks, `#B93838` crimson Fighter Barracks, `#F2B830` striped gold Builder Workshop, `#E8C15A` golden thatch Granary), identical warm limestone walls, and identical character/HUD styling. Only the **spatial layout** varies. |
| 3 | **Are the nine building types visually distinct?** | **PASS** | Verified in `buildings/modular_buildings_closeup_sheet.png` (`2160×2360`) and inside every variant's `layout_diagram_1080p.png`: `1. Central Townhall (4×4)`, `2. Builder Workshop (3×3)`, `3. Guard Barracks (3×3)`, `4. Fighter Barracks (3×3)`, `5. Wood Storage (2×3)`, `6. Stone Storage (2×3)`, `7. Food Granary (3×3)`, `8. Village Houses (2×2)`, and `9. Watchtower (2×2)` each read within `<1.5s` by silhouette, roof color, and role props. |
| 4 | **Do walls form a logical defensive perimeter?** | **PASS** | Every variant encloses its village inside **one continuous connected limestone perimeter wall** (`straight` + `corner` + `gate` + `watchtower` segments) corresponding truthfully to the single canonical `TƯỜNG` HP bar (`2120/2200` PREP -> `1760/2200` ASSAULT -> `1420/2200` BOSS). All `5` wall states (`Intact`, `Damaged`, `Critical`, `Breached`, `Repair Scaffolding`) are verified in `buildings/wall_states_and_unit_scale_sheet.png`. |
| 5 | **Do characters match the scale of buildings and gates?** | **PASS** | Verified on `buildings/wall_states_and_unit_scale_sheet.png` (`2200×1513`) and across all `PREP`/`ASSAULT`/`BOSS` plates: humanoid units (`Builder`, `Guard`, `Fighter`, `Grunt`, `Runner`) stand at `~68%–75%` of doorway height (`~52–60 px` at `1080p` -> `~17–20 px` at `360p`), `Brute` at `~1.4×`, and `Boss` at `~2.6–2.8×` towering menacingly at the breached northern gate without dwarfing the entire settlement. |
| 6 | **Are the three lanes clearly readable?** | **PASS** | Three distinct dirt-and-cobblestone approach roads (`LÀN TÂY / West`, `LÀN GIỮA / Center`, `LÀN ĐÔNG / East`) enter from the northern forest treeline and converge onto the northern gatehouse sector across `V1`, `V2`, and `V3`. |
| 7 | **Are `360×640` previews still readable?** | **PASS** | Verified directly on `V1/prep_360p.png`, `V2/assault_360p.png`, `V3/boss_360p.png`, and `comparisons/all_variants_360p_matrix.png` (`1880×2240` 1:1 native pixel matrix). All `9` `360×640` files are genuine `PIL.Image.Resampling.LANCZOS` downscales (`PSNR = 99.0 dB` in `qa/composition_metrics.json`, never redrawn). Roof colors, wall states, 3 lanes, role silhouettes, and HUD counters remain crisp at `360×640`. |
| 8 | **Is HUD text clear and rule-accurate (`ttg-rules-v1`)?** | **PASS** | Canonical HUD shows only `TƯỜNG` HP, `CHÍNH 500/500`, `GỖ`, `ĐÁ`, `LƯƠNG`, `ĐỢT 10`, `GIỜ`, `[MOCK]`, phase banner, single primary CTA `Bình luận XÂY / THỦ / ĐÁNH`, `XÂY` / `THỦ` / `ĐÁNH` counters, and `MOCK / REPLAY` footer. Zero invented currencies (gems/gold/elixir), zero auto-firing cannons/ballistas, and zero invented commands. |
| 9 | **Are TikTok right `18%` and bottom `12%` zones respected?** | **PASS** | Verified programmatically (`qa/composition_metrics.json`) and visually (`layout_diagram_1080p.png` for `V1`, `V2`, `V3`): right `18%` (`x > 885 px`) and bottom `12%` (`y > 1689 px`) contain **0 px** of HUD panels and **0 px** of walled village structures (`right_18pct_hud_or_village_overlap_px = 0`, `bottom_12pct_hud_or_village_overlap_px = 0` across all 3 variants). |
| 10 | **Can the design be implemented as modular Unity 2D assets?** | **PASS** | Documented in `UNITY_MODULARIZATION_FEASIBILITY.md`: every building (`9` prefabs), wall segment (`straight`/`corner`/`gate` × `4` damage states + `scaffolding` overlay), terrain layer, prop, animated unit, and VFX archetype decomposes cleanly into independent `SpriteRenderer` prefabs sorted via isometric `TransparencySortMode.CustomAxis (0, 1, -0.26)` in Unity `6000.6.7f1` Built-in 2D (`2` `2048×2048` environment atlases). |

---

## 3. Quantitative Composition & Readability Metrics (`qa/composition_metrics.json`)

| Metric | `ART-009` Baseline (`02`/`05`/`07`) | `V1` Compact Walled Village | `V2` Layered Defensive Village | `V3` Living Fantasy Settlement |
|---|---|---|---|---|
| Native Resolution | `1080×1920` | `1080×1920` + `360×640` | `1080×1920` + `360×640` | `1080×1920` + `360×640` |
| Flat Empty Block Ratio (`1080p` PREP/ASSAULT) | `59.3% – 60.6%` *(empty lawn)* | **`18.1% – 19.2%`** | **`13.7% – 14.0%`** | **`18.2% – 19.7%`** |
| Luminance Entropy (`bits`, `1080p`) | `5.28 – 5.34` | **`7.42 – 7.61`** | **`7.33 – 7.54`** | **`7.24 – 7.64`** |
| Edge Energy Mean (`1080p` ASSAULT) | `8.22` | **`13.06`** | **`18.33`** | **`16.95`** |
| True `360p` Downscale Fidelity (`PSNR`) | N/A | **`99.0 dB`** (exact LANCZOS) | **`99.0 dB`** (exact LANCZOS) | **`99.0 dB`** (exact LANCZOS) |
| Village Wall Safe Bounds (`x0..x1, y0..y1`) | `308..772, 650..1120` | `56..850, 540..1455` | `52..880, 285..1472` | `12..818, 575..1498` |
| TikTok Safe Zone Overlap (`x > 885` or `y > 1689`) | `0 px` | **`0 px` (`PASS`)** | **`0 px` (`PASS`)** | **`0 px` (`PASS`)** |

---

## 4. Adversarial Defects Caught & Resolved Before Sign-Off

1. **Fighter Positioning in `V1` Assault (`v1_assault_raw` v1 -> v2):** Initial raw plate placed Red Fighters swinging sword arcs inside the empty central courtyard instead of engaging enemies. Regenerated (`v1_assault_raw_v2`) so Red Fighters sally outside the NW and NE flanks directly engaging incoming purple Runners and green Grunts while Blue Guards hold the central gate against bronze Brutes.
2. **Perimeter Framing in `V2` (`v2_village_clean` v1 -> v2 + `26 px` safe-left alignment):** Initial raw plate zoomed too close (`x = 0..100%`), clipping the east wall into the right `18%` TikTok zone. Regenerated at full `9:16` framing and shifted `26 px` left so the entire dual-terrace perimeter sits cleanly inside `x = 52..880 <= 885` and `y = 285..1472 <= 1689`.
3. **Right Safe-Zone Margin & Boss Guard Line in `V3` (`v3_village_clean` v2 -> v4; `v3_boss_raw` v1 -> v2):** Initial `V3` extended the eastern wall to `x ~ 89%`, and initial `V3` Boss plate placed Blue Guards north behind the Boss. Regenerated both plates so the eastern wall terminates at `x = 818 <= 885` (leaving a scenic meadow/fence strip under TikTok's right icons) and Blue Guards form a coherent cyan shield wall inside the breached northern gate directly between the Obsidian Boss and the illuminated Townhall.
4. **Column Header Spacing on `three_variants_comparison.png`:** Visual inspection caught the `VARIANT 2` and `VARIANT 3` single-line headers touching at `440 px` card width. Widened cards to `530×942` (`gap_x = 120 px`) on a `2160×3190` board with two-line headers, verified zero text collision.

---

## 5. Honest Per-Variant Readability Tradeoffs & Recommendation

- **Variant 1 — Compact Walled Village (`RECOMMENDED #1`):** Best `<2s` *Clash of Clans*-style base-building silhouette, longest northern 3-lane runway (`~340 px` visible enemy approach), crispest separation between all 9 buildings at `360×640`, and cleanest isometric grid mapping for Unity `6000.6.7f1`. Minor tradeoff: less cozy nighttime lantern warmth than `V3` unless `V3`'s courtyard well and warm window/lantern glow are merged into `V1` during `ART-010B`.
- **Variant 2 — Layered Defensive Village:** Strongest tactical stronghold hierarchy (outer military courtyard + stepped retaining terrace + inner Townhall sanctum), highest architectural edge density (`18.33`), but shorter northern enemy approach lanes (`~110 px`) and denser `360×640` silhouette packing.
- **Variant 3 — Living Fantasy Settlement:** Most charming, lived-in village atmosphere (curved cobblestone streets, central stone well, warm lantern glow in `BOSS FINALE`), slightly softer defensive geometry than `V1`'s crisp octagonal ring.
