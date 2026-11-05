# ART-001 — Deliverable E: Livestream UX & Anchored HUD System

**Project:** TTGQLDL (`hung98dev/ttgqldl`)  
**Visual Authority:** [`docs/plan/07_UNITY_ART_UX.md`](../../../docs/plan/07_UNITY_ART_UX.md), [`docs/plan/02_GAME_RULES.md`](../../../docs/plan/02_GAME_RULES.md), [`docs/plan/09_QA_ACCEPTANCE.md`](../../../docs/plan/09_QA_ACCEPTANCE.md) (`QA-CLI-02`, `QA-CLI-03`, `QA-UX-01..03`, `QA-PSY-01`)  
**HUD Blueprint Sheet:** `images/hud_layout.png`

---

## 1. First-Three-Second Comprehension Contract (`QA-PSY-01` / `07_UNITY_ART_UX.md`)

An uncoached smartphone viewer scrolling onto a muted `360 × 640` stream must understand three facts within **3 seconds**:

1. **Castle / Wall Condition (Top-Left Anchor):** `TƯỜNG 920/1200` + `CHÍNH 500/500` paired with crenellated-wall and pitched-keep vector icons plus segmented fill bars (`green >= 60%`, `amber 25%..59%`, `hatched coral < 25%`).
2. **Incoming Threat & Remaining Time (Top-Right + Threat Strip):** `00:38` (`ĐỢT 10`) high-contrast countdown box + `BOSS 88HP · TÂY 115 · GIỮA 106 · ĐÔNG 117` directly above the converging three-lane battlefront.
3. **Exactly ONE Plain-Language Primary CTA (Bottom Safe Anchor):** **`Bình luận XÂY / THỦ / ĐÁNH`** (`48 px` bold @ `1080p` = `16.0 px` @ `360p`, `18.39:1` WCAG contrast), directly above the three accepted role count cards (`[Hammer] XÂY 28`, `[Shield] THỦ 56`, `[Sword] ĐÁNH 22`). **Zero competing primary CTAs** (no gift prompts, no like-goals, no chat-flood meters).

---

## 2. Responsive Coordinate Grid & Important Technical Audit Finding

### 2.1 Authoritative Screen Regions (`1080 × 1920` Reference, `CanvasScaler.matchWidthOrHeight = 0.5`)

`07_UNITY_ART_UX.md` expresses vertical layout bands in **top-down screen percentages** (`Y_top = 0%` at top edge, `100%` at bottom edge), whereas Unity `RectTransform` normalized anchors (`anchorMin`, `anchorMax`) use **bottom-left origin** (`Y_unity = 1.0 - Y_top`).

| HUD Zone | Top-Down Screen `%` (`07_UNITY_ART_UX.md`) | `1080 × 1920` Pixel Rect (`x0, y0 → x1, y1`) | Exact Unity `RectTransform` Anchors (`anchorMin → anchorMax`, Direct Child of Full-Canvas Root) | Purpose |
|---|---|---|---|---|
| **Safe Area Envelope** | `x: 0%..82%`, `y: 0%..88%` | `(0, 0) → (885, 1689)` | `(0.000, 0.120) → (0.820, 1.000)` | Usable HUD envelope excluding TikTok right & bottom chrome |
| **Right Platform Reserve** | `x: 82%..100%`, `y: 0%..100%` | `(886, 0) → (1080, 1920)` | `(0.820, 0.000) → (1.000, 1.000)` | Reserved for TikTok avatar, like/comment/share rail |
| **Bottom Platform Reserve** | `x: 0%..100%`, `y: 88%..100%` | `(0, 1690) → (1080, 1920)` | `(0.000, 0.000) → (1.000, 0.120)` | Reserved for TikTok comment input & scrolling chat overlay |
| **Top HUD (`TopHud`)** | `x: 3.9%..81.1%`, `y: 1.4%..11.4%` *(spec band `<=14%`)* | `(42, 26) → (876, 218)` | `(0.039, 0.886) → (0.811, 0.986)` | Wall HP, Townhall HP, Wave/Timer, 3 Resources, Source status |
| **Threat / System Banner** | `x: 3.9%..81.1%`, `y: 11.8%..15.0%` | `(42, 226) → (876, 288)` | `(0.039, 0.850) → (0.811, 0.882)` | Boss incoming/HP alert, 3-lane HP summary, Pause/Disconnect/DB alerts |
| **World Viewport (`World`)** | `x: 5.0%..82.0%`, `y: 15.6%..72.5%` *(spec `20..74%`)* | `(54, 300) → (885, 1392)` | *Orthographic Camera World (`y: 0.275..0.844`)* | Central castle (`~44%` width = `475 px`), West/Center/East lanes, `<=150` pooled sprites |
| **Subordinate Vote Strip** | `x: 3.9%..81.1%`, `y: 72.9%..75.7%` | `(42, 1400) → (876, 1454)` | `(0.039, 0.243) → (0.811, 0.271)` | Even-wave `A / B / C` vote tally (`[80,120s)` open; `[120,280s)` locked buff) |
| **Primary Commands (`Commands`)** | `x: 3.9%..81.1%`, `y: 76.1%..85.1%` *(spec `76..86%`)* | `(42, 1462) → (876, 1634)` | `(0.039, 0.149) → (0.811, 0.239)` | Single CTA `Bình luận XÂY / THỦ / ĐÁNH` + `XÂY / THỦ / ĐÁNH` role counts |
| **MOCK / REPLAY Badge** | `x: 3.9%..81.1%`, `y: 85.4%..87.8%` | `(42, 1640) → (876, 1686)` | `(0.039, 0.122) → (0.811, 0.146)` | Mandatory honest simulation watermark outside authorized `P1` LIVE |

### 2.2 Pre-Existing M01 Skeleton Layout Discrepancies Reported for M06 Handoff (Zero Runtime Files Modified)

During our mandatory inspection of binding sources and the M01 Unity skeleton (left untouched per parallel `TASK 036` workspace rules), three concrete discrepancies between [`07_UNITY_ART_UX.md`](../../../docs/plan/07_UNITY_ART_UX.md) and `../../../unity/Assets/_TTGQLDL/` were identified for remediation in **`M06-W0601` / `M06-W0604`**:

1. **Y-Axis Coordinate Inversion in `HudLayoutConfig.cs` & `SafeAreaPanel.cs`:** `HudLayoutConfig.cs` declares `origin bottom-left`, yet assigns `TopHud = [0.05, 0.05 → 0.82, 0.14]`, `Commands = [0.08, 0.76 → 0.75, 0.86]`, and `BottomReserve = [0, 0.88 → 1, 1]`. In Unity's bottom-up `RectTransform` space, `y = 0.05..0.14` places `TopHud` at the **bottom** of the screen, `Commands` near the **top**, and `SafeAreaPanel.ComputeAnchors` (`yMax = Min(yMax, 0.88f)`) reserves the **top 12%** instead of the **bottom 12%**! Converting top-down percentages via `anchorMinY = 1f - yMaxTopDown` and `anchorMaxY = 1f - yMinTopDown` resolves this cleanly in M06.
2. **Parent-Child Double-Scaling in `SceneFactory.cs`:** `SceneFactory.CreateBroadcastScene()` nests `TopHud` and `Commands` inside `SafeArea` (`0..0.82, 0..0.88`), compounding the `0.82 / 0.88` factor twice (`0.82 × 0.82 = 67.2%` width). Child panels inside `SafeArea` should either normalize against `SafeArea` (`x / 0.82`) or anchor directly to full-Canvas root using the table in §2.1.
3. **Sub-Minimum Font Sizes & Missing Primary CTA Copy in `SceneFactory.cs`:** `SceneFactory.cs` currently authors several labels (`BannerText = 20`, `Watermark = 22`, `SpotlightText = 22`, `ResourcesText = 24`) on the `1080 × 1920` reference canvas, which scale down to **`6.7–8.0 px` at `360 × 640`** (below `MinimumSecondaryFontPx = 12`, which requires **`>= 36 px` at `1080 × 1920`**), and omits the literal headline string `Bình luận XÂY / THỦ / ĐÁNH`.

---

## 3. Typography, Contrast, Redundancy & Accessibility Contract

### 3.1 Font Hierarchy & Minimum Type Sizes

| Hierarchy Tier | Vietnamese Sample Copy | `1080 × 1920` Size | `360 × 640` Equivalent | Spec Floor (`07_UNITY_ART_UX.md`) | Measured WCAG Contrast |
|---|---|---:|---:|---:|---:|
| **Tier 1 — Primary Command CTA** | `Bình luận XÂY / THỦ / ĐÁNH` | **`48 px` Bold** (`+4 px` dark stroke) | **`16.0 px`** | `>= 16 px` | **`18.39 : 1`** (`#FFFFFF` on `#0E1422`) |
| **Tier 2 — Countdown Timer** | `00:38` | **`42–44 px` Bold** | **`14.0–14.7 px`** | `>= 14 px` | **`12.57 : 1`** (`#FEF08A` on `#1E293B`) |
| **Tier 3 — Castle & Wall HP** | `TƯỜNG 920/1200` · `CHÍNH 500/500` | **`42 px` Bold SemiCondensed** | **`14.0 px`** | `>= 14 px` | **`17.91 : 1`** (`#FFFFFF` on `#101726`) |
| **Tier 4 — Role Counts** | `XÂY 28` · `THỦ 56` · `ĐÁNH 22` | **`38 px` Bold** | **`12.7 px`** | `>= 12 px` | **`16.95 : 1`** (`#FFFFFF` on `#161E2E`) |
| **Tier 5 — Secondary / Resources / Boss / Vote / MOCK** | `GỖ 522` · `BOSS 88HP` · `VOTE ĐỢT 10` · `MOCK / REPLAY` | **`36–37 px` Bold/SemiCondensed** | **`12.0–12.3 px`** | `>= 12 px` | **`15.44 : 1` – `17.91 : 1`** |
| **Tier 6 — Rotating Alias Spotlight** | `minh_tuan… (XÂY)` *(max `12` labels, `<= 12` chars + `…`)* | **`36 px` SemiBold** | **`12.0 px`** | `>= 12 px` | **`>= 9.50 : 1`** on dark pill scrim |

- **Recommended Open-Licensed Production Fonts for M06:** **Be Vietnam Pro** (Bold / ExtraBold — native Vietnamese diacritic design, SIL OFL 1.1) paired with **Barlow Semi Condensed** (Bold — tabular numerals & compact HUD labels, SIL OFL 1.1).

### 3.2 Icon + Shape + Text Redundancy (Zero Color-Alone Information)
- **Roles:** Builder = `[T-Mallet Icon] + Left Gold Bar + "XÂY"`; Guard = `[Tower-Shield Icon] + Left Blue Bar + "THỦ"`; Fighter = `[Angled Sword Icon] + Left Red Bar + "ĐÁNH"`.
- **Structures & Resources:** Wall = `[Crenellation Icon] + "TƯỜNG"`; Townhall = `[Pitched-Keep Icon] + "CHÍNH"`; Wood = `[Log Cylinder] + "GỖ"`; Stone = `[Faceted Hexagon] + "ĐÁ"`; Food = `[Scored Loaf] + "LƯƠNG"`.
- **Alerts:** Boss/Source warnings = `[Triangle Hazard `!`]` + solid double border; Pause = `[Double Vertical Bar `II`]`; Storage Failure = `[Octagon/Cross `X`]` + striped warning border.

### 3.3 Join Feedback, Rotating Spotlight & Anti-Spam UX
- **Valid First Join:** Updates aggregate role integer (`XÂY`, `THỦ`, `ĐÁNH`) on the next stream publication (`<= 200 ms` cadence) with a gentle `180 ms` scale-pop (`1.00 → 1.08 → 1.00`) on the matching role card plus a compact `2.0 s` non-occluding toast (`+1 ĐÁNH · minh_tuan…`) anchored at `y = 70.5%` (below castle gate, above Vote strip).
- **Fair Spotlight Policy:** Round-robin by `joined_seq` over non-hidden, sanitized aliases; maximum **one `2 s` spotlight per `5 s`**, maximum **`12` simultaneous alias tags**, truncated to **`<= 12` visible characters + `…`**.
- **Anti-Spam Honesty:** Duplicate comments in the same role produce **zero** toast, **zero** counter bump, and **zero** visual burst. Role switch after cooldown (`>= 600` ticks after first switch) displays subtle receipt `(Đổi sang THỦ ở nhịp 20s tới)`.

### 3.4 A/B/C Even-Wave Vote Presentation (`[t = 80 s, t = 120 s)`)
- Appears **only on even waves** (`wave_no % 2 == 0`).
- **While Open (`ticks 800..1199`):** Compact horizontal strip (`y = 72.9%..75.7%`) showing integer tallies, exact resource cost, and close countdown: `VOTE (Còn 25s): [A:28 +Thủ -50 Gỗ] [B:64 +Đánh -50 Lương] [C:19 Giảm phạt]`. Never styled as a clickable button (since viewers watch a video stream, not an interactive HTML overlay!) and kept `25%` smaller in font size (`36 px` vs `48 px`) than `Bình luận XÂY / THỦ / ĐÁNH`.
- **At Close (`tick 1200`):** Locks deterministically (`A > B > C` tie-break, affordability check at `tick 1200`, fallback `C`), highlights chosen option with a checkmark badge for `20 s` (and compact buff pill through `tick 2800`), charging resource cost **only** for the selected option.

### 3.5 Accessibility & Reduced-Motion Policy
- **Flash Limit:** Strict **`0` flashes `> 3 Hz`** anywhere in HUD or VFX (Boss warning uses a smooth `0.8 Hz` sinusoidal border glow, disabled to static `100%` opacity when `ReducedMotion = true`).
- **Reduced-Motion Toggle:** Operator/Unity config flag disables all micro-hit camera impulses, count-pop scaling, and scrolling toast slide animations (switches to instant clean fade `150 ms`).
- **Muted Comprehension:** Every phase transition (`PREP_60_WARN`, `PREP_15_WARN`, `ASSAULT`, `BOSS_INCOMING`, `BOSS`, `RESULTS_WIN`, `RESULTS_LOSS`, `PAUSED`, `SOURCE_RECONNECTING`) carries explicit Vietnamese text + icon state on Canvas so muted viewers never rely on audio cues.

---

## 4. Complete Presentation State Matrix (`01_BroadcastWorld` Canvas States — Zero Scene Reloads)

Per `07_UNITY_ART_UX.md` line 42 (mitigating Unity `6000.6.7f1`'s `SceneManager.LoadSceneAsync` player-hang issue), **all states below render as Canvas state transitions inside `01_BroadcastWorld`**:

| Presentation State | Server Condition (`ttg-rules-v1`) | Top / Threat Banner | World Viewport Truthfulness | Command / Vote Strip Behavior |
|---|---|---|---|---|
| `ATTRACT` / `WAITING` (`WAITING_FOR_VIEWER`) | `lifecycle=waiting`, `wave_tick=-1`, `P=0` | `CHỜ NGƯỜI CHƠI ĐẦU TIÊN · 05:00` | Peaceful intact village, **0 fake player sprites**, **0 enemies**, **0 unattended damage** | `Bình luận XÂY / THỦ / ĐÁNH` + `XÂY 0 · THỦ 0 · ĐÁNH 0` |
| `PREP` | `tick 0..1599` (`t=0..160s`) | `CHUẨN BỊ XÂY DỰNG · 04:12` *(if `P<=10`, show `Tự động hỗ trợ` badge)* | Active Builders repair/upgrade wall at `20 s` boundaries; Guards/Fighters muster | Primary CTA active; odd waves show village objective at `t=80s` instead of fake poll |
| `PREP_60_WARN` / `PREP_15_WARN` | `t=130s` (`PREP_30`) / `t=145s` (`PREP_15`) & `t=100s` (`60s` to assault) | `[!] QUÁI SẮP TẤN CÔNG TRONG 30s / 15s` | Scout dust/ silhouettes telegraph at outer lane rims (non-damaging) | Primary CTA stays dominant |
| `VOTE_OPEN` | Even `wave_no`, `tick 800..1199` (`t=80..120s`) | Normal `PREP` header | Normal `PREP` village work | Subordinate Vote Strip opens (`A / B / C` integer counts + close timer; eligible `joined_tick<=1199`) |
| `ASSAULT` | `tick 1600..2599` (`t=160..260s`) | `TẤN CÔNG 3 ĐƯỜNG · TÂY / GIỮA / ĐÔNG` | Pooled `<=150` sprites clash; impacts resolve at `t=180,200,220,240s` | Primary CTA active (newcomers add power without raising `base_hp`!) |
| `BOSS_INCOMING` | `wave_no % 5 == 0`, `t=245..259.9s` **OR** ordinary pools `== 0` before `tick 2600` | **`[!] CẢNH BÁO BOSS ĐƯỜNG GIỮA SẮP XUẤT HIỆN`** | **NEVER displays `CLEAR` or `VICTORY`** even if all ordinary subpools hit `0` early! | Urgent Fighter/Guard reinforcement read via role counts |
| `FINALE` / `BOSS` | `tick 2600..2799` (`t=260..280s`) | `QUYẾT CHIẾN · BOSS [HP]/[MAX]` *(or `ĐỢT CUỐI` on non-boss wave)* | Boss (`wave%5==0`) spawns on Center causeway with telegraph ring; two combat checks (`2600`, `2800`) | Primary CTA active; locked even-wave vote buff shown |
| `RESULTS_WIN` | `tick 2800..2999`, `townhall_hp > 0` & all pools (incl. Boss) `== 0` | `CHIẾN THẮNG! +100 Gỗ · +60 Đá · +50 Lương` | Surviving defenders celebrate; settled **only after** `tick 2800` combat | Sub-note: `Bình luận ngay để đăng ký ĐỢT TIẾP THEO (tự chuyển ở giây 300)` |
| `RESULTS_LOSS` | `tick 2800..2999`, `townhall_hp == 0` **or** surviving enemy HP `> 0` | `THẤT THỦ / HẾT GIỜ — HỒI PHỤC ĐỢT SAU` *(shows Recovery `-15%..-45%` HP)* | Honest breached/weathered state; `250` HP / `100` Wood floors shown if `Defeat` | Same `NEXT WAVE` queue registration; cosmetic XP preserved |
| `SOURCE_RECONNECTING` | `ConnectionState.Disconnected/Stale` or `PROVIDER_DISCONNECTED` | **`[!] MẤT KẾT NỐI NGUỒN — ĐỒNG HỒ TẠM DỪNG`** | World dimmed `35%`, animations idle in place, simulation tick **frozen**, `STALE` pill visible | Never fabricates comment joins while disconnected |
| `PAUSED` | `lifecycle == Paused` | **`[II] TẠM DỪNG BỞI ĐIỀU HÀNH VIÊN`** | Simulation clock **frozen**; retains `PAUSED` across source loss until all blockers clear | Connected joins show `(Đã ghi nhận — áp dụng khi tiếp tục)` |
| `STORAGE_UNAVAILABLE` | `lifecycle == StorageUnavailable` | **`[X] LỖI LƯU TRỮ DỮ LIỆU — TẠM DỪNG AN TOÀN`** | Simulation clock **frozen**; unsaved state never disguised as active play | Input acceptance halted/flagged honestly |
| `DEGRADED_DROPS` / `MAINTENANCE` | Valid event drop `> 0` (`QA-LOAD-01`) / `Stopped` | **`[!] QUÁ TẢI HÀNG ĐỢI — CÓ LỆNH BỊ RƠI`** / `BẢO TRÌ HỆ THỐNG` | Persistent amber-crimson diagnostic badge on top-right of Safe Area | Honest operator/audience degradation notice |
