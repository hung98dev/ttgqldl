# ART-002 Deliverable F — Icon & UI Component Kit Specification

- **Work Package:** `ART-002`
- **Style Version:** `bastion-dawn-v1`
- **Component Directory:** `ui/components/` (`37` PNG files: `20` individual `128×128` RGBA icons, `15` individual 9-sliceable RGBA UI skins, `icon_sheet.png`, `component_sheet.png`)
- **Master Sheets:** `ui/components/icon_sheet.png` and `ui/components/component_sheet.png`
- **Binding Rule:** **Zero text embedded inside any icon or UI skin.** Every label, counter, timer, and CTA string remains editable Unity TextMeshPro (`TMP_Text`) geometry for localization and crisp sub-pixel font scaling.

---

## 1. Naming & Unity Import Convention

- **Icons:** `ui_icon_<semantic_slug>.png` — `128 × 128` RGBA PNG, centered within a `16 px` outer padding frame (`96 × 96 px` active glyph area), `3.0 px` `#11151C` outer silhouette ink, `45°` top-left two-step cel shading, binary hard outer alpha (`alpha >= 224 -> 255`, else `0`) for `FilterMode.Point` or `Bilinear` UI atlas packing without dark/gray fringe halos.
- **9-Sliceable Component Skins:** `ui_comp_<component_slug>.png` — RGBA PNG with uniform rounded corner radius and flat interior fill/border runs so Unity `Image (Type = Sliced)` stretches cleanly to arbitrary text lengths without distorting corner bevels.

---

## 2. Complete Catalog of 20 Text-Free RGBA Icons (`128 × 128`)

| # | File Path | Semantic Role | Dominant HEX | Shape / Colorblind Redundancy |
|---:|---|---|---|---|
| 1 | `ui/components/ui_icon_wood.png` | Wood resource (`GỖ`) | `#9E6234` | Cut oak log cylinder + pale `#E6C79C` heartwood end-grain |
| 2 | `ui/components/ui_icon_stone.png` | Stone resource (`ĐÁ`) | `#9BA5B0` | Beveled cut-limestone ashlar block |
| 3 | `ui/components/ui_icon_food.png` | Food resource (`LƯƠNG`) | `#D9822B` | Scored golden-crust artisan bread loaf |
| 4 | `ui/components/ui_icon_castle.png` | Central Castle / Townhall HP | `#3B6EA5` | Steep cobalt gable roof + golden sun-crest doorway |
| 5 | `ui/components/ui_icon_wall.png` | Defensive Curtain Wall HP | `#9BA5B0` | Three-merlon crenelated stone battlement |
| 6 | `ui/components/ui_icon_boss.png` | Dread Bastion Warlord alert | `#D946EF` | Crescent-horned obsidian war-helm + glowing magenta visor |
| 7 | `ui/components/ui_icon_danger.png` | Incoming threat / queue warning | `#FACC15` | Equilateral warning triangle + bold exclamation bar (geometric symbol, non-linguistic) |
| 8 | `ui/components/ui_icon_timer.png` | Wave phase countdown clock | `#EAB308` | Brass-rimmed stopwatch + `45°` remaining-time sector |
| 9 | `ui/components/ui_icon_builder.png` | Builder role (`XÂY`) | `#F4C430` | Orthogonal forged-iron & oak T-mallet |
| 10 | `ui/components/ui_icon_guard.png` | Guard role (`THỦ`) | `#2E7CF6` | Steel-rimmed vertical cobalt tower shield |
| 11 | `ui/components/ui_icon_fighter.png` | Fighter role (`ĐÁNH`) | `#E63946` | Diagonal broadsword + crimson pommel tassel |
| 12 | `ui/components/ui_icon_vote_a.png` | Even-wave Vote Option A (`DEF`) | `#2E7CF6` | Fortified cobalt shield with single upward chevron crest (zero `"A"` glyph baked in) |
| 13 | `ui/components/ui_icon_vote_b.png` | Even-wave Vote Option B (`ATK`) | `#E63946` | Crossed crimson-hilted steel broadswords (zero `"B"` glyph baked in) |
| 14 | `ui/components/ui_icon_vote_c.png` | Even-wave Vote Option C (`Resource`) | `#22C55E` | Iron-bound emerald resource safeguard chest (zero `"C"` glyph baked in) |
| 15 | `ui/components/ui_icon_source_health.png` | Live source connected (`OK`) | `#22C55E` | Concentric broadcast signal arcs + central node |
| 16 | `ui/components/ui_icon_pause.png` | Operator pause (`PAUSED`) | `#FACC15` | Twin vertical amber pause bars |
| 17 | `ui/components/ui_icon_resume.png` | Resume / next-wave registration | `#4ADE80` | Right-pointing emerald play wedge |
| 18 | `ui/components/ui_icon_reconnect.png` | Source reconnecting / `MOCK` loop | `#FACC15` | Twin circular sync arrows |
| 19 | `ui/components/ui_icon_error.png` | `STORAGE_UNAVAILABLE` / DB halt | `#DC2626` | Crimson octagon + bold white cross (`[X]`) |
| 20 | `ui/components/ui_icon_militia.png` | System militia (`1 <= P <= 10`) | `#94A3B8` | Neutral slate buckler + wooden spear (`non-viewer` indicator) |

---

## 3. Complete Catalog of 15 9-Sliceable UI Component Skins

| # | File Path | Source Size | 9-Slice Insets `(L, R, T, B)` | Usage & Visual Treatment |
|---:|---|---:|---|---|
| 1 | `ui/components/ui_comp_resource_bar.png` | `360 × 68` | `(20, 20, 20, 20)` | Dark `#101726` resource/wave container + `#64748B` border |
| 2 | `ui/components/ui_comp_hp_bar_frame.png` | `380 × 44` | `(16, 16, 14, 14)` | Recessed `#0B0F19` HP track frame with `25% / 50% / 75%` notch ticks |
| 3 | `ui/components/ui_comp_hp_bar_fill_healthy.png` | `368 × 32` | `(12, 12, 10, 10)` | Solid emerald `#22C55E` fill + top cel highlight (`>= 60%` HP) |
| 4 | `ui/components/ui_comp_hp_bar_fill_warn.png` | `368 × 32` | `(12, 12, 10, 10)` | Amber `#F59E0B` fill + subtle vertical caution ticks (`25%–59%` HP) |
| 5 | `ui/components/ui_comp_hp_bar_fill_crit.png` | `368 × 32` | `(12, 12, 10, 10)` | Coral-red `#EF4444` fill + diagonal `#7F1D1D` hatching (`< 25%` HP) |
| 6 | `ui/components/ui_comp_role_counter_builder.png` | `276 × 74` | `(24, 20, 20, 20)` | `#121A2B` plaque + left `#F4C430` Builder accent tab + gold rim |
| 7 | `ui/components/ui_comp_role_counter_guard.png` | `276 × 74` | `(24, 20, 20, 20)` | `#121A2B` plaque + left `#2E7CF6` Guard accent tab + cobalt rim |
| 8 | `ui/components/ui_comp_role_counter_fighter.png` | `276 × 74` | `(24, 20, 20, 20)` | `#121A2B` plaque + left `#E63946` Fighter accent tab + crimson rim |
| 9 | `ui/components/ui_comp_vote_card_open.png` | `272 × 64` | `(18, 18, 18, 18)` | Subordinate slate `#101726` non-button vote tally card |
| 10 | `ui/components/ui_comp_vote_card_locked.png` | `272 × 64` | `(18, 18, 18, 18)` | Gold-highlighted `#1C170C` winning vote buff card |
| 11 | `ui/components/ui_comp_status_pill_ok.png` | `188 × 56` | `(26, 26, 20, 20)` | Deep emerald `#0E2A1C` pill + `#4ADE80` border |
| 12 | `ui/components/ui_comp_status_pill_warn.png` | `188 × 56` | `(26, 26, 20, 20)` | Deep amber `#3A1C0C` pill + `#FB923C` border (`STALE` / `MOCK`) |
| 13 | `ui/components/ui_comp_status_pill_err.png` | `188 × 56` | `(26, 26, 20, 20)` | Deep crimson `#3A0C12` pill + `#F87171` border (`LỖI DB`) |
| 14 | `ui/components/ui_comp_cta_banner.png` | `840 × 88` | `(28, 28, 24, 24)` | Primary CTA royal-navy `#0F172A` plaque + double `#EAB308` gold bevel |
| 15 | `ui/components/ui_comp_toast_join.png` | `640 × 58` | `(28, 28, 22, 22)` | Rounded `#10192A` join toast capsule + `#38BDF8` sky border |

---

## 4. Spacing, Padding & Typography Layout Tokens

| Token ID | Value (`1080×1920`) | Value (`360×640`) | Application |
|---|---:|---:|---|
| `space.hud.outer_margin_x` | `36 px` | `12 px` | Left padding from screen edge (`x = 3.33%`) |
| `space.hud.safe_gap_right` | `9 px` min (`876 <= 885`) | `3 px` | Safety buffer left of `x = 82%` livestream icon rail |
| `space.hud.card_gap_y` | `8 px` | `2.7 px` | Vertical rhythm between stacked HUD cards |
| `space.hud.inner_pad_x` | `14 px` | `4.7 px` | Horizontal padding inside plaques and pills |
| `space.hud.icon_text_gap` | `8–10 px` | `3 px` | Gap between `38–44 px` icon and `TMP_Text` label |
| `font.size.cta_primary` | `48 px` Bold | `16.0 px` | Primary command CTA (`>= 16 px` mobile requirement) |
| `font.size.hp_and_timer` | `42–43 px` Bold | `14.0–14.3 px` | Wall HP, Castle HP, Countdown Timer (`>= 14 px` requirement) |
| `font.size.secondary` | `36–38 px` Bold/Semi | `12.0–12.7 px` | Role counters, resources, banners, vote, mock tag (`>= 12 px` requirement) |
