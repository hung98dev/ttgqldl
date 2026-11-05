# ART-006 — Bastion Dawn v1 Production UI Design System

## 1. Visual Identity & 3-Second Broadcast Hierarchy

`ART-006` implements a cohesive fantasy broadcast game HUD in `Bastion Dawn v1` (`#11151C` dark-ink outlines, warm `45°` top-left cel bevels, royal-navy `#161E2E` translucent slate, limestone `#9BA5B0`, cobalt `#2E7CF6`, sun-gold `#F4C430`, and crimson `#E63946`) designed to communicate three essentials within **<= 3 seconds** on a `9:16` mobile livestream:

1. **Tình trạng Làng (Village Condition — Top Status Panel `y=22..242`):**
   - **Wall HP (`TƯỜNG {hp}/{max}`)** and **Townhall/Keep HP (`CHÍNH {hp}/500`)** rendered with dual-layer 9-slice recessed tracks + `RectMask2D` clipped fills (`>=60%` solid emerald, `25%–59%` amber + `8px` caution dots, `<25%` coral-red + `45°` dark hatching).
   - **Countdown (`GIỜ mm:ss` / `--:--`)**, **Wave Badge (`ĐỢT N`)**, **Source Health Pill (`MOCK` / `REPLAY` / `MẤT NGUỒN` / `LỖI DB` / `TẠM DỪNG`)**, and **3 Resource Cells (`GỖ`, `ĐÁ`, `LƯƠNG`)**.
2. **Mối đe dọa đang tới (Incoming Threat — Threat Banner `y=250..316`):**
   - Contextual severity banner (`navy` calm, `warn` amber, `err` crimson, `ok` emerald) displaying 3-lane enemy HP (`T:{west} · G:{center} · Đ:{east}`), Boss pre-spawn warning (`tick < 2600`), active Boss HP + embedded magenta Boss HP bar (`tick >= 2600`), or freeze/degradation alerts.
3. **Một lệnh hành động duy nhất (Single Dominant Command — Primary CTA `y=1462..1550`):**
   - `Bình luận XÂY / THỦ / ĐÁNH` (`48 px` @ `1080p` = **`16.0 px` @ `360×640`**) inside the gold-rimmed `ui_comp_cta_banner` plaque.
   - **Strict Single-CTA Discipline:** Viewers interact exclusively via chat comments. Zero fake touch buttons, zero competing CTAs, and zero gift/donation/paid-buff prompts exist anywhere in the package. Even-wave voting (`A / B / C`) is strictly a subordinate tally/instruction strip (`y=1396..1454`) above the primary CTA.

---

## 2. Reusable 35-Asset Component Catalog & 9-Slice Specification

Machine-readable source of truth: [`ui_component_manifest.json`](ui_component_manifest.json).

### 2.1 Twenty `128×128` Text-Free Icons (`icons/`, `atlas/ui_atlas_01_icons.png`)
- **Resources (3):** `ui_icon_wood`, `ui_icon_stone`, `ui_icon_food`
- **Structures (2):** `ui_icon_castle`, `ui_icon_wall`
- **Combat & Alerts (3):** `ui_icon_boss`, `ui_icon_danger`, `ui_icon_timer`
- **Player Roles (3):** `ui_icon_builder`, `ui_icon_guard`, `ui_icon_fighter`
- **Even-Wave Vote Options (3):** `ui_icon_vote_a`, `ui_icon_vote_b`, `ui_icon_vote_c`
- **System, Operator & Assist (6):** `ui_icon_source_health`, `ui_icon_pause`, `ui_icon_resume`, `ui_icon_reconnect`, `ui_icon_error`, `ui_icon_militia`

### 2.2 Fifteen 9-Slice / Tiled Component Skins (`components/`, `atlas/ui_atlas_02_components.png`)

> [!IMPORTANT]
> Documentation convention in `ART-002` lists insets as `(Left, Right, Top, Bottom)`, whereas Unity `TextureImporter.spriteBorder` and `Sprite.border` use **`Vector4(left, bottom, right, top)` (`L, B, R, T`)**. Both tuples are explicitly stored in `ui_component_manifest.json` and `atlas/ui_atlas_02_components.json`.

| Component ID | Dimensions | Insets `(L, R, T, B)` | Unity `Vector4(L, B, R, T)` | Unity `Image.Type` | Min Size | Max Size |
|---|---|---|---|---|---|---|
| `ui_comp_resource_bar` | `256×64` | `(20, 20, 20, 20)` | `(20, 20, 20, 20)` | `Sliced` | `96×44` | `848×224` |
| `ui_comp_hp_bar_frame` | `384×44` | `(16, 16, 14, 14)` | `(16, 14, 16, 14)` | `Sliced` | `80×32` | `800×56` |
| `ui_comp_hp_bar_fill_healthy` | `368×30` | `(12, 12, 10, 10)` | `(12, 10, 12, 10)` | `Sliced` + `RectMask2D` | `26×22` (`0px` clipped) | `788×42` |
| `ui_comp_hp_bar_fill_warn` | `368×30` | `(12, 12, 10, 10)` | `(12, 10, 12, 10)` | `Tiled` (`8px` period) + `RectMask2D` | `26×22` (`0px` clipped) | `788×42` |
| `ui_comp_hp_bar_fill_crit` | `368×30` | `(12, 12, 10, 10)` | `(12, 10, 12, 10)` | `Tiled` (`8px` period) + `RectMask2D` | `26×22` (`0px` clipped) | `788×42` |
| `ui_comp_role_counter_builder` | `264×72` | `(24, 20, 20, 20)` | `(24, 20, 20, 20)` | `Sliced` | `140×54` | `360×88` |
| `ui_comp_role_counter_guard` | `264×72` | `(24, 20, 20, 20)` | `(24, 20, 20, 20)` | `Sliced` | `140×54` | `360×88` |
| `ui_comp_role_counter_fighter` | `264×72` | `(24, 20, 20, 20)` | `(24, 20, 20, 20)` | `Sliced` | `140×54` | `360×88` |
| `ui_comp_vote_card_open` | `264×54` | `(18, 18, 18, 18)` | `(18, 18, 18, 18)` | `Sliced` | `120×46` | `840×72` |
| `ui_comp_vote_card_locked` | `264×54` | `(18, 18, 18, 18)` | `(18, 18, 18, 18)` | `Sliced` | `120×46` | `840×72` |
| `ui_comp_status_pill_ok` | `184×48` | `(26, 26, 20, 20)` | `(26, 20, 26, 20)` | `Sliced` | `112×44` | `840×68` |
| `ui_comp_status_pill_warn` | `184×48` | `(26, 26, 20, 20)` | `(26, 20, 26, 20)` | `Sliced` | `112×44` | `840×68` |
| `ui_comp_status_pill_err` | `184×48` | `(26, 26, 20, 20)` | `(26, 20, 26, 20)` | `Sliced` | `112×44` | `840×68` |
| `ui_comp_cta_banner` | `834×96` | `(28, 28, 24, 24)` | `(28, 24, 28, 24)` | `Sliced` | `320×72` | `848×104` |
| `ui_comp_toast_join` | `520×56` | `(28, 28, 22, 22)` | `(28, 22, 28, 22)` | `Sliced` | `220×50` | `780×64` |

---

## 3. Responsive Layout Geometry, Safe Areas & ART-005 Balance Resolution

Machine-readable tokens: [`ui_layout_tokens.json`](ui_layout_tokens.json).

- **Safe-Area Contract (`07_UNITY_ART_UX.md`):**
  - Default reserved platform zones: **Right `x > 82%` (`885.6 px`)** and **Bottom `y > 88%` (`1689.6 px`)**.
  - Operator-configurable runtime margin bounds supported via `SafeAreaContainer`: Right `78%..86%`, Bottom `84%..90%`.
  - Every essential HUD panel and label sits strictly inside `x in [28..882]` and `y in [22..1688]` (`0` occluded pixels).
- **Resolving the `ART-005` Open Lower-Meadow Space Without Moving World Coordinates:**
  - In `ART-005`, the South Gate & Guards end at `y ≈ 1058 px` on `1080×1920`, leaving `862 px` (`y = 1058..1920`) open when viewed without UI.
  - `ART-006` proves and resolves this vertical budget with **zero changes to approved world coordinates**:
    - `y = 22..316`: Top Status Panel + Threat Banner (ends `42 px` above the tallest Boss war-helm at `y = 358 px` -> **`0 px` Castle/Boss overlap**).
    - `y = 316..1062`: `746 px` unobstructed 3-lane combat, Boss slam telegraph, Wall, Cottages, Watchtower, and Castle Keep.
    - `y = 1062..1208`: `146 px` southern forecourt breathing space directly below the South Gate.
    - `y = 1208..1388`: Non-occluding Diagnostic/Militia Pill (`1208..1260`), Defender Spotlight Ribbon (`1268..1320`), and Single-Slot `2.0s` Join Toast (`1330..1388`).
    - `y = 1390..1688`: **Unified Lower Command Dock** (`x=28..882`, framed by a beveled translucent royal-navy console scrim uniting the Vote/Objective Strip, Primary CTA Plaque, 3 Role Counters, and Honest Watermark Footer).
    - `y = 1690..1920`: `230 px` (`12%`) reserved for mobile livestream chat/comment chrome.
  - Side-by-side comparison proof: `previews/1080p/art005_vs_art006_vertical_balance_comparison.png`.

---

## 4. Typography, Localization (`vi-VN` / `en-US`) & Accessibility

- **Mandatory Mobile Type Scale (`360×640` equivalent = `1/3` of `1080×1920`):**
  - **Primary CTA:** `48 px` @ `1080p` = **`16.0 px` @ `360p`** (minimum required: `16 px`).
  - **Wall HP & Wave Timer:** `43 px` @ `1080p` = **`14.33 px` @ `360p`** (minimum required: `14 px`).
  - **Threat Banner:** `37 px` @ `1080p` = **`12.33 px` @ `360p`** (minimum required: `12 px`).
  - **Secondary HUD (Resources, Roles, Vote, Toasts, Watermark):** `36 px` @ `1080p` = **`12.0 px` @ `360p`** (minimum required: `12 px`).
- **Contrast & Outline Protection:** Every dynamic text label renders over dark slate/pill skins (`#101828`..`#3A1218`) with `#F8FAFC` / `#FEF08A` fill + `3 px` `#11151C` stroke, achieving **WCAG contrast `>= 12.3:1`** (far exceeding the `4.5:1` requirement) and surviving `yuv420p` H.264 `800 kbps` mobile compression cleanly.
- **Colorblind & Grayscale Redundancy (`previews/grayscale/grayscale_and_colorblind_redundancy_board.png`):**
  - HP states pair color + interior texture (`Solid` vs `8px Caution Dots` vs `45° Diagonal Hatching`) + explicit numeric fraction (`1080/1200`).
  - Roles pair color tabs (`Gold` / `Cobalt` / `Crimson`) + distinct tool silhouettes (`Mallet` / `Tower Shield` / `Broadsword`) + Vietnamese/English role verbs (`XÂY` / `THỦ` / `ĐÁNH`).
- **Reduced Motion (`previews/grayscale/reduced_motion_accessibility_board.png`):**
  - `0` flashes `> 3 Hz` anywhere; `0` mandatory camera shake; `reduced_motion = true` disables all scale/alpha tweens while retaining 100% static high-contrast state clarity.
