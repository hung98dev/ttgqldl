# ART-002 Deliverables D & E — Vertical Livestream UX/UI Master Design & 16-State System

- **Work Package:** `ART-002`
- **Default Interface Language:** Vietnamese (`vi-VN`), with complete English (`en-US`) localization table in §5
- **UI Directory:** `ui/`
- **State Board:** `ui/state_board.png` (`16 / 16` canonical states)
- **Binding Authority:** [07_UNITY_ART_UX.md](../../../docs/plan/07_UNITY_ART_UX.md) (`UX001`, `UX002`, `ART003`), [02_GAME_RULES.md](../../../docs/plan/02_GAME_RULES.md), [09_QA_ACCEPTANCE.md](../../../docs/plan/09_QA_ACCEPTANCE.md)

---

## 1. Delivered High-Fidelity HUD Screens & Guides

All 14 HUD master screens, mobile exports, safe-area guides, and the 16-state board exist in `ui/`:

| File | Resolution | Represented State / Purpose |
|---|---:|---|
| `ui/hud_master.png` | `1080 × 1920` | Clean broadcast master (`Wave 10` Boss climax, segmented HP bars, join toast, locked Vote B, primary CTA, honest `MOCK / REPLAY` footer) |
| `ui/hud_master_livestream_overlay.png` | `1080 × 1920` | Master composition with realistic TikTok LIVE / OBS platform chrome (`x > 82%` icon rail + `y > 88%` comment drawer) proving **zero occlusion** |
| `ui/hud_waiting.png` | `1080 × 1920` | `WAITING_FOR_VIEWER` (`wave_tick = -1`, `P = 0`, clock frozen at `05:00`, zero fake viewers/units) |
| `ui/hud_preparation.png` | `1080 × 1920` | `PREP` (`t = 0..160s`, odd-wave objective strip, Builder wall repairs, honest `Tự động hỗ trợ (Hệ thống · P≤10)` militia tag) |
| `ui/hud_assault.png` | `1080 × 1920` | `ASSAULT` (`t = 160..260s`, 3-lane normal enemy combat with West/Center/East HP readout + non-occluding join toast) |
| `ui/hud_boss.png` | `1080 × 1920` | `BOSS` (`tick >= 2600` on every 5th wave, Center Dread Bastion Warlord + `Violet Boss Eclipse` + Boss HP banner) |
| `ui/hud_voting.png` | `1080 × 1920` | `VOTE_OPEN` (even-wave `[80..120s)` subordinate non-button `A / B / C` tally strip with `-50 Gỗ` / `-50 Lương` resource costs) |
| `ui/hud_results_win.png` | `1080 × 1920` | `RESULTS_WIN` (`t = 280..300s` settled victory reward `+100 Gỗ · 60 Đá · 50 Lương` + next-wave comment registration notice) |
| `ui/hud_results_loss.png` | `1080 × 1920` | `RESULTS_LOSS` (`250 HP` & `100 Gỗ` recovery floor + `-15%` next-wave enemy HP relief + weathered wall with scaffolds; zero viewer-blaming text) |
| `ui/hud_source_reconnecting.png` | `1080 × 1920` | `SOURCE_RECONNECTING` (`PROVIDER_DISCONNECTED` -> dimmed world, frozen clock, orange `STALE` source pill, zero fabricated chat) |
| `ui/hud_storage_unavailable.png` | `1080 × 1920` | `STORAGE_UNAVAILABLE` (PostgreSQL write failure -> frozen clock, red `LỖI DB` octagon `[X]`, honest progress-halt banner) |
| `ui/hud_degraded_drops.png` | `1080 × 1920` | `DEGRADED_DROPS` (valid ingress queue overflow `dropped > 0` -> amber `DEGRADED` warning banner while combat continues) |
| `ui/hud_mobile_360x640.png` | `360 × 640` | True `1:1` pixel mobile livestream preview verifying `>= 16 px` CTA, `>= 14 px` HP/Timer, and `>= 12 px` secondary text |
| `ui/hud_safe_area_guide.png` | `2460 × 2080` | Side-by-side Clean Broadcast vs Livestream Platform Overlay with cyan/red safe-envelope geometry (`[0..885, 0..1689]`) |
| `ui/state_board.png` | `2640 × 2480` | Complete `4 × 4` **16-State Visual Design System** covering every state in `07_UNITY_ART_UX.md` |

---

## 2. Three-Second Information Hierarchy & Single-CTA Discipline

1. **Second 1 — Village Condition (Top HUD Card `y = 22..242 px`):** Segmented Wall HP (`TƯỜNG 920/1200`) and Castle HP (`CHÍNH 500/500`) at `42 px` Bold (`14.0 px` @ `360p`, `17.91:1` contrast) with colorblind-redundant bars (`>=60%` solid emerald, `25%–59%` amber, `<25%` coral-red with diagonal dark hatching), above `ĐỢT 10`, `GIỜ 00:38` (`43 px` Bold = `14.3 px` @ `360p`), source pill (`MOCK / OK / STALE / LỖI DB`), and `GỖ / ĐÁ / LƯƠNG`.
2. **Second 2 — Incoming Threat (World Center + Alert Strip `y = 250..312 px`):** Immediate 3-lane combat picture framed above the `44.2%` circular bastion, reinforced by a single contextual threat/phase banner (`BOSS 88/88 · TÂY 115 · GIỮA 106 · ĐÔNG 117`).
3. **Second 3 — Single Primary Action (`y = 1462..1550 px`):** One dominant gold-framed command plaque (`840 × 88 px`, `48 px` Bold = `16.0 px` @ `360p`, `17.85:1` contrast):
   - **`Bình luận XÂY / THỦ / ĐÁNH`**
   - Directly supported by the three role-coded counters (`XÂY 28`, `THỦ 56`, `ĐÁNH 22`) so viewers immediately map each keyword to its color, icon, and live defender count.
   - **Strict Anti-Competition Rule:** Even-wave voting (`y = 1396..1454 px`, `58 px` tall) is `< 66%` of the primary CTA height, uses non-button scoreboard framing (never fake touch buttons), and disappears outside `[80..120s)` / locked buff summary.

---

## 3. Safe Areas & Responsive Unity `RectTransform` Layout Spec

### 3.1 Safe-Zone Reservation (`hud_safe_area_guide.png`)
- **Right Platform Safe Zone (`x > 82%` = `x > 885 px`):** Reserved for TikTok LIVE host avatar, Like, Comment, Share, and disabled-gift rail. Every active HUD panel in `ART-002` ends at **`x = 876 px` (`81.11%`)**, leaving `0` HUD pixels in `x = 886..1079 px`.
- **Bottom Platform Safe Zone (`y > 88%` = `y > 1689 px`):** Reserved for TikTok scrolling chat feed and text-input bar. The lowest element (`MOCK / REPLAY` watermark) ends at **`y = 1686 px` (`87.81%`)**, leaving `0` HUD pixels in `y = 1690..1919 px`.
- **Adaptive Safe Margin Config:** Unity Canvas binds to `safeRightPct` (default `0.82`, adjustable `0.78..0.86`) and `safeBottomPct` (default `0.88`, adjustable `0.84..0.90`) so operators can tighten margins if a platform UI update widens overlays.

### 3.2 Authoritative Unity `CanvasScaler` & `RectTransform` Anchors
- **Canvas:** `Screen Space - Overlay`, `CanvasScaler = Scale With Screen Size`, `Reference Resolution = (1080, 1920)`, `Match Width Or Height = 0.5`.
- **Important Note for Future `M06` Runtime Engineer (No Runtime Files Modified in `ART-002`):** Unity `RectTransform` normalized anchors use **bottom-left `(0, 0)` to top-right `(1, 1)`** (`anchorY = 1.0 - topDownY`), whereas top-down screen specs list `y = 0%` at top and `y = 100%` at bottom:

| HUD Region | Top-Down Pixel Box (`1080×1920`) | Unity `anchorMin` `(x, y)` | Unity `anchorMax` `(x, y)` |
|---|---|---|---|
| **Top Status Card** | `[36, 22] -> [876, 242]` | `(0.033, 0.874)` | `(0.811, 0.989)` |
| **Threat / System Banner** | `[36, 250] -> [876, 312]` | `(0.033, 0.838)` | `(0.811, 0.870)` |
| **Militia Notice Pill (`P<=10`)** | `[36, 320] -> [676, 372]` | `(0.033, 0.806)` | `(0.626, 0.833)` |
| **Join Toast (`2.0 s` TTL, max `1`)** | `[76, 1330] -> [836, 1388]` | `(0.070, 0.277)` | `(0.774, 0.307)` |
| **Subordinate Vote / Objective Strip** | `[36, 1396] -> [876, 1454]` | `(0.033, 0.243)` | `(0.811, 0.273)` |
| **Primary Command CTA** | `[36, 1462] -> [876, 1550]` | `(0.033, 0.193)` | `(0.811, 0.239)` |
| **3 Role Counters Row** | `[36, 1556] -> [876, 1630]` | `(0.033, 0.151)` | `(0.811, 0.190)` |
| **Mock / Replay Honest Footer** | `[36, 1638] -> [876, 1686]` | `(0.033, 0.122)` | `(0.811, 0.147)` |

### 3.3 Anti-Spam Toast & Viewer Feedback Discipline
- **Single-Slot Join Toast (`y = 1330..1388 px`):** Displays at most `1` accepted command toast at a time (`1.8 s` display + `0.2 s` fade), truncating `display_name` safely at `14` characters (`+1 ĐÁNH · minh_tuan… (Đã ghi nhận)`), coalescing bursts `> 3/sec` into `+12 người vừa chọn THỦ!`, and strictly ignoring rejected/duplicate spam so trolls are never rewarded on screen.
- **No Fake Viewers or Monetization:** Never render fake viewer counts, fabricated usernames, gift animations, or pay-to-win prompts. System militia (`1 <= P <= 10`) is explicitly labeled `Tự động hỗ trợ (Hệ thống · P≤10)` and never inflates `XÂY / THỦ / ĐÁNH` viewer counters.

---

## 4. Complete 16-State Visual Design Matrix (`ui/state_board.png`)

All 16 states execute as Canvas ViewModels inside persistent scene `01_BroadcastWorld` (never reloading scenes mid-session, avoiding Unity `6000.6.7f1` `LoadSceneAsync` player hang `QA-VIS-02`):

| # | State ID | Trigger / Authoritative Rule | Primary Information | Secondary Information | Active CTA | Status & Visual Feedback |
|---:|---|---|---|---|---|---|
| 1 | `ATTRACT` | Pre-open / idle attract loop (`P = 0`) | Village landmark + opening prompt | Base resources `400 / 250 / 300` | `Bình luận XÂY / THỦ / ĐÁNH` | `MOCK` / `OK` pill; calm sunlit village |
| 2 | `WAITING_FOR_VIEWER` | `wave_tick = -1`, `P = 0` (`02_GAME_RULES`) | `CHỜ NGƯỜI CHƠI · ĐỒNG HỒ CHƯA CHẠY` | `0` roles, `0` damage, `05:00` | `Bình luận XÂY / THỦ / ĐÁNH` | Clock frozen at `05:00`; zero fake units |
| 3 | `PREP` | `tick 0..999` (`0..100s`) | `CHUẨN BỊ (0..160s) · XÂY TƯỜNG & TẬP HỢP` | Odd-wave goal + Militia pill (`P<=10`) | `Bình luận XÂY / THỦ / ĐÁNH` | Builder golden repair sparks every `20 s` |
| 4 | `PREP_60_WARN` | `tick 1000` (`60 s` to `ASSAULT`) | `CÒN 60 GIÂY TRƯỚC KHI QUÁI TẤN CÔNG!` | Distant lane dust / Vote strip | `Bình luận XÂY / THỦ / ĐÁNH` | Steady `#FACC15` amber banner (`0` `>3 Hz` flash) |
| 5 | `PREP_15_WARN` | `tick 1450` (`15 s` to `ASSAULT`) | `KHẨN CẤP: QUÁI XUẤT HIỆN TRONG 15 GIÂY!` | Locked even-wave buff / role counts | `Bình luận XÂY / THỦ / ĐÁNH` | High-priority amber alert rim + horn cue |
| 6 | `VOTE_OPEN` | Even wave `tick 800..1199` (`[80..120s)`) | `BẦU CHỌN ĐỢT CHẴN · Gõ A/B/C (Còn 25s)` | `A (-50 Gỗ) · B (-50 Lương) · C (0)` | `Bình luận XÂY / THỦ / ĐÁNH` | Subordinate non-button vote bar (`58 px` tall) |
| 7 | `ASSAULT` | `tick 1600..2599` (`[160..260s)`) | `GIAO TRANH · TÂY 142 · GIỮA 134 · ĐÔNG 144` | 3-lane combat + join toast | `Bình luận XÂY / THỦ / ĐÁNH` | Active `12 FPS` lane clashes & HP bar updates |
| 8 | `FINALE` (`BOSS_INCOMING`) | `wave % 5 == 0`, normal lanes `0` before `tick 2600` | `QUÁI THƯỜNG HẾT · BOSS RA Ở GIÂY 260!` | Center lane violet warning aura | `Bình luận XÂY / THỦ / ĐÁNH` | **Never** declares early win before Boss spawns! |
| 9 | `BOSS` | `wave % 5 == 0`, `tick >= 2600` (`[260..280s)`) | `BOSS GIỮA 88/88 · TÂY 115 · ĐÔNG 117` | Dread Warlord + concentric ring | `Bình luận XÂY / THỦ / ĐÁNH` | `Violet Boss Eclipse` palette + slam ticks `2600/2800` |
| 10 | `RESULTS_WIN` | Settled win (`tick 2800..2999`) | `THẮNG ĐỢT 10! +100 Gỗ · 60 Đá · 50 Lương` | Next-wave pre-registration strip | `Bình luận XÂY / THỦ / ĐÁNH` | Emerald victory banner + celebratory banner |
| 11 | `RESULTS_LOSS` | `townhall_hp == 0` or `tick 2800` timeout | `THẤT THỦ · Hồi 250 HP & -15% quái đợt sau` | Weathered wall (`250 HP`) + `100 Gỗ` | `Bình luận XÂY / THỦ / ĐÁNH` | Constructive recovery floor; zero viewer blame |
| 12 | `SOURCE_RECONNECTING` | `PROVIDER_DISCONNECTED` | `MẤT KẾT NỐI NGUỒN · ĐỒNG HỒ DỪNG` | Dimmed world (`60%`) + frozen timer | `Bình luận XÂY / THỦ / ĐÁNH` | Orange `STALE` pill; zero fabricated chat |
| 13 | `PAUSED` | `OPERATOR_PAUSED` | `TẠM DỪNG BỞI ĐIỀU HÀNH · GIỜ DỪNG` | Frozen tick & unit poses | `Bình luận XÂY / THỦ / ĐÁNH` | `[II]` Pause icon + persisted pause state |
| 14 | `STORAGE_UNAVAILABLE` | PostgreSQL write failure (`09_QA`) | `LỖI LƯU TRỮ DỮ LIỆU · TẠM DỪNG AN TOÀN` | Dimmed world (`55%`) + frozen tick | `Bình luận XÂY / THỦ / ĐÁNH` | Red `LỖI DB` `[X]` pill; never fakes save success |
| 15 | `DEGRADED_DROPS` | Valid event queue drop `> 0` | `QUÁ TẢI HÀNG ĐỢI · LỆNH RƠI (DEGRADED)` | Live high-traffic role counters | `Bình luận XÂY / THỦ / ĐÁNH` | Amber `STALE` diagnostic flag while game runs |
| 16 | `MAINTENANCE` | Session stopped / maintenance | `PHIÊN ĐÃ DỪNG / BẢO TRÌ · HẸN GẶP LẠI` | Timer `--:--`, idle safe village | `PHIÊN DỪNG — CHỜ MỞ LẠI` | Stays in `01_BroadcastWorld` without reload |

---

## 5. Vietnamese Default & English Localization String Table

All UI icons and 9-slice skins are `100%` text-free (`05_ui_component_kit.md`); text is rendered via Unity TextMeshPro (`Noto Sans` / `Be Vietnam Pro` Bold) using key-driven localization:

| Localization Key | Default Vietnamese (`vi-VN`) | English (`en-US`) | Max Char Budget |
|---|---|---|---:|
| `hud.cta.primary` | `Bình luận XÂY / THỦ / ĐÁNH` | `Comment BUILD / DEFEND / FIGHT` | `32` |
| `hud.cta.maintenance` | `PHIÊN DỪNG — CHỜ MỞ LẠI` | `SESSION PAUSED — BACK SOON` | `28` |
| `hud.hp.wall` | `TƯỜNG {0}/{1}` | `WALL {0}/{1}` | `16` |
| `hud.hp.castle` | `CHÍNH {0}/{1}` | `KEEP {0}/{1}` | `15` |
| `hud.wave.label` | `ĐỢT {0}` | `WAVE {0}` | `10` |
| `hud.timer.label` | `GIỜ {0}` | `TIME {0}` | `11` |
| `hud.res.wood` | `GỖ {0}` | `WOOD {0}` | `11` |
| `hud.res.stone` | `ĐÁ {0}` | `STONE {0}` | `11` |
| `hud.res.food` | `LƯƠNG {0}` | `FOOD {0}` | `12` |
| `hud.role.builder` | `XÂY {0}` | `BUILD {0}` | `11` |
| `hud.role.guard` | `THỦ {0}` | `DEFEND {0}` | `11` |
| `hud.role.fighter` | `ĐÁNH {0}` | `FIGHT {0}` | `11` |
| `hud.vote.open` | `VOTE {0}s: A:{1} (-50 Gỗ) · B:{2} · C:{3}` | `VOTE {0}s: A:{1} (-50W) · B:{2} · C:{3}` | `44` |
| `hud.militia.active` | `Tự động hỗ trợ (Hệ thống · P≤10)` | `Auto-Support (System · P≤10)` | `36` |
| `hud.watermark.mock` | `MOCK / REPLAY · MÔ PHỎNG (CHƯA LIVE)` | `MOCK / REPLAY · OFFLINE SIMULATION` | `38` |
