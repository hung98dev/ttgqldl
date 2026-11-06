# ART-010 — Modular Village Defense Visual Overhaul (Phase A Revision)

**Package:** `ART-010` (`Phase A Revision — Clash-of-Clans-Inspired Modular Fantasy Village Defense`)  
**Status:** `AWAITING_OWNER_LAYOUT_APPROVAL` (Hard Stop before `ART-010B`)  
**Engine Target:** Unity `6000.6.7f1` — Built-in 2D (`SpriteRenderer` isometric Y-sorting + uGUI/TextMeshPro)  
**Resolution:** `1080×1920` portrait masters + genuine `1:1` LANCZOS `360×640` mobile previews  
**Canonical Gameplay Contract:** `ttg-rules-v1` (Wood / Stone / Food, Builder / Guard / Fighter, single connected Wall HP + Townhall `500` HP, 3 lanes West / Center / East, Wave 10 `PREP` -> `ASSAULT` -> `BOSS FINALE`, single CTA `Bình luận XÂY / THỦ / ĐÁNH`)

---

## 1. Owner Feedback & Direction Pivot Summary

The initial `ART-010` exploration (`d2aeed9`) was **rejected by the repository owner** because its cliffside fortresses and monolithic castles looked like standalone fantasy paintings rather than a **modular village-building defense game**.

This `ART-010A` revision replaces the rejected concepts completely:
- **The Village Is Now the Protagonist:** Elevated `3/4` isometric diamond-grid perspective over a sunlit emerald meadow clearing (`Suncrest Meadow Modular Village v1`), inspired at a high level by the base-building clarity, modular footprints, and protective attachment of *Clash of Clans* while using **100% original medieval-fantasy architecture** (zero Supercell copies).
- **9 Distinct Modular Building Archetypes:** `Central Townhall` (`500 HP`), `Builder Workshop`, `Guard Barracks`, `Fighter Barracks`, `Wood Storage`, `Stone Storage`, `Food Granary`, `Village Houses`, and `Watchtowers` (purely visual defense architecture — **zero** invented cannons, gold/elixir currencies, or extra commands).
- **One Single Connected Perimeter Wall:** Continuous pale-granite & oak perimeter wall with corner bastions, straight segments, three northern gates facing the `West`, `Center`, and `East` roads, and 5 visual condition states (`Intact`, `Damaged`, `Critical`, `Breached`, `Repair Scaffolding`) mapped to the single canonical `Wall HP` pool.
- **Three Layout Variations in One Unified Art Style:** All three candidates share the exact same rendering style, palette, building designs, character scale, and HUD so the owner is choosing a **village layout**, not unrelated illustrations:
  1. **Variant 1 — Compact Walled Village (`variants/V1_compact_walled_village/`)**
  2. **Variant 2 — Layered Defensive Village (`variants/V2_layered_defensive_village/`)**
  3. **Variant 3 — Living Fantasy Settlement (`variants/V3_living_fantasy_settlement/`)**

---

## 2. Complete Deliverable Inventory (`art/production/ART-010/`)

```text
art/production/ART-010/
  README.md                                 # Package overview, owner-feedback pivot & inventory
  ART_DIRECTION_REVIEW.md                   # Critique of ART-009 + 9-building modular design & 3 layout variants
  COMPARISON_REPORT.md                      # Variant 1 vs 2 vs 3 pros/cons, 10-question validation & recommendation
  UNITY_MODULARIZATION_FEASIBILITY.md       # Concrete Unity 6000.6.7f1 Built-in 2D sprite/grid reconstruction plan
  ASSET_REUSE_ANALYSIS.md                   # Explicit ART-003..ART-009 reuse vs replacement matrix
  OWNER_APPROVAL_REQUIRED.md                # Phase 11 STOP gate before ART-010B production

  variants/
    V1_compact_walled_village/
      prep_1080p.png                        # 1080x1920 Village in PREP + integrated HUD
      assault_1080p.png                     # 1080x1920 Village during 3-lane ASSAULT + integrated HUD
      boss_1080p.png                        # 1080x1920 Boss attacking the village (FINALE) + integrated HUD
      prep_360p.png                         # True 360x640 direct LANCZOS downscale of PREP
      assault_360p.png                      # True 360x640 direct LANCZOS downscale of ASSAULT
      boss_360p.png                         # True 360x640 direct LANCZOS downscale of BOSS
      village_clean_1080p.png               # 1080x1920 HUD-free clean modular village view
      layout_diagram_1080p.png              # 1080x1920 labeled building-layout & TikTok 82%x88% safe-area diagram

    V2_layered_defensive_village/
      prep_1080p.png | assault_1080p.png | boss_1080p.png
      prep_360p.png  | assault_360p.png  | boss_360p.png
      village_clean_1080p.png | layout_diagram_1080p.png

    V3_living_fantasy_settlement/
      prep_1080p.png | assault_1080p.png | boss_1080p.png
      prep_360p.png  | assault_360p.png  | boss_360p.png
      village_clean_1080p.png | layout_diagram_1080p.png

  buildings/
    modular_buildings_closeup_sheet.png     # Close-up showcase of all 9 individual modular buildings
    wall_states_and_unit_scale_sheet.png    # 5 connected wall states + 7-class doorway-relative scale lineup

  comparisons/
    three_variants_comparison.png           # Side-by-side Variant 1 vs 2 vs 3 across PREP, ASSAULT & BOSS
    art009_vs_revised_village.png           # Current ART-009 prototype vs revised modular village comparison
    all_variants_360p_matrix.png            # 1:1 native 360x640 mobile matrix (ART-009 vs V1 vs V2 vs V3)

  qa/
    visual_review.md                        # Brutally honest 10-question visual audit at 1080p & 360p
    composition_metrics.json                # Resolution, PSNR downscale proof & safe-area measurements
    asset_provenance.json                   # Originality, IP cleanliness & GATE-LEGAL pending disclosures
```

---

## 3. Truthfulness & Approval Gate

- **Concept Stage Only (`STOP` before `ART-010B`):** All `29` PNGs are Phase A visual mockups, **not** captures from a newly built Unity runtime scene. All `ART-003`..`ART-009` production assets, backend/SQL code, and `M06` status remain untouched.
- **100% Original Fantasy IP:** High-level base-building composition and readability take inspiration from mobile village-builders (*Clash of Clans*), while all building designs, roofs, emblems, characters, and HUD frames are original fantasy creations (`GATE-LEGAL = PENDING_REVIEW`).
