# ART-001 — Deliverables B & C: Gameplay Mockup Comparison & Recommended Direction

**Project:** TTGQLDL (`hung98dev/ttgqldl`)  
**Decision State:** **`PROPOSED` — Pending Repository Owner Approval (`ART-002`)**  
**M06 Milestone Status:** `NOT_STARTED` (this package is concept & visual direction only; zero mass asset production started)

---

## 1. Shared Unbiased Gameplay Scenario (`S-ART001`)

To evaluate Directions A, B, and C without bias, all three `1080 × 1920` concept images depict the **exact same deterministic `ttg-rules-v1` gameplay state** (`Wave 10` — both divisible by `5` for Boss spawn and even-numbered for A/B/C voting):

- **Simulation Timestamp:** `t = 262.0 s` (`wave_tick = 2620`, `00:38` remaining, phase `FINALE / QUYẾT CHIẾN`).
- **Why `t = 262.0 s` on Wave 10 is Rule-Exact (`02_GAME_RULES.md`):**
  - Normal enemies spawn at `tick 1600` (`t = 160 s`); the Boss does **not** exist in authoritative state until `tick 2600` (`t = 260 s`). Showing `t = 262 s` legally puts Builder, Guard, Fighter, Runner, Grunt, Brute, **and** the Center-lane Boss (`88/88 HP`) on screen simultaneously without faking early Boss presence or premature `VICTORY`/`CLEAR` banners.
  - Even-wave voting is open during `[t = 80 s, t = 120 s)` (`ticks 800..1199`) and closes deterministically at `tick 1200`. At `t = 262 s`, the locked wave tactic (`VOTE ĐỢT 10: A:28 · [B:64 +15% ĐÁNH] · C:19`) displays cleanly above the command panel **without competing** with the single primary CTA (`Bình luận XÂY / THỦ / ĐÁNH`).
- **Identical HUD Values Across A, B, and C:**
  - `TƯỜNG 920/1200` (`77%`, Tier 2) · `CHÍNH 500/500` (`100%`) · `00:38` (`ĐỢT 10`)
  - Resources: `GỖ 522` · `ĐÁ 375` · `LƯƠNG 310` · Source pill: `MOCK`
  - Threat bar: `BOSS 88HP · TÂY 115 · GIỮA 106 · ĐÔNG 117`
  - Primary CTA: `Bình luận XÂY / THỦ / ĐÁNH` (`48 px` @ 1080p = `16.0 px` @ 360p)
  - Accepted role counts: `XÂY 28` · `THỦ 56` · `ĐÁNH 22`
  - Honest watermark: `MOCK / REPLAY · MÔ PHỎNG (CHƯA LIVE)`
  - Livestream platform safe-zone simulation: right rail `x > 82%` (`886..1080 px`) and bottom chat/input zone `y > 88%` (`1690..1920 px`), using `--` placeholders rather than fabricated viewer counts.

---

## 2. Deliverable B — Inspectable Concept Image Inventory

| Direction | `1080 × 1920` Primary Mockup | `360 × 640` Mobile Preview | Grayscale Exports (`1080p` & `360p`) | Raw World Art Source (Zero UI) |
|---|---|---|---|---|
| **A: Bright Stylized (*Bastion Dawn*)** | `images/direction_a_1080x1920.png` | `images/direction_a_360x640.png` | `images/direction_a_grayscale_1080x1920.png`, `images/direction_a_grayscale_360x640.png` | `images/source/world_direction_a_raw.jpg` |
| **B: Premium Chibi (*Hearthguard*)** | `images/direction_b_1080x1920.png` | `images/direction_b_360x640.png` | `images/direction_b_grayscale_1080x1920.png`, `images/direction_b_grayscale_360x640.png` | `images/source/world_direction_b_raw.jpg` |
| **C: Arcade Silhouette (*Tactical Crisp*)** | `images/direction_c_1080x1920.png` | `images/direction_c_360x640.png` | `images/direction_c_grayscale_1080x1920.png`, `images/direction_c_grayscale_360x640.png` | `images/source/world_direction_c_raw.jpg` |

Combined 1:1 pixel diagnostic board: `images/comparison_360x640_grayscale_sheet.png`.

---

## 3. Empirical Inspection Findings (`1080×1920`, `360×640`, and BT.709 Grayscale)

### 3.1 Direction A — Bright Stylized Fantasy (*Bastion Dawn*)
- **At `1080 × 1920`:** All three lanes (West sand road, Center pale-limestone Boss causeway, East sand road) converge unambiguously on the circular stone bastion (`~44%` width). All 7 character classes stay completely clear of the top HUD (`y <= 15%`), right TikTok rail (`x > 82%`), and bottom CTA/chat zones (`y >= 73%`), including the 4 south-gate Guards (`y = 68%..74%`).
- **At `360 × 640` (`direction_a_360x640.png`):** Bold dark-ink contours (`3 px` @ `128 px` source) preserve distinct body shapes: square-shouldered orange Brutes, round green club-wielding Grunts, lean purple Runners, yellow-apron Builders on wooden scaffolds with golden repair sparks, blue rectangular-shield Guards at the south gate, red plumed Fighters on the battlements, and the `2.1×` horned Boss inside its glowing violet ring. Every essential HUD label meets minimum pixel rules (`CTA = 16 px`, `HP/Timer = 14 px`, `Secondary = 12 px`) with measured WCAG contrast `12.57:1` to `18.39:1`.
- **In Grayscale (`direction_a_grayscale_360x640.png`):** Strong multi-cue separation survives without hue: Builder aprons (`Y = 0.59`) + scaffolding sparks read bright against mid-gray masonry (`Y = 0.37`); Boss armor (`Y = 0.01`) + bright ring (`Y = 0.72`) anchor the Center lane; Guards read via tall rectangular shields + bright steel rims (`Y = 0.78`); Brutes read by `1.35×` cubic shoulder width vs spherical Grunts vs lean Runners.

### 3.2 Direction B — Premium Chibi Fantasy (*Hearthguard Chibi*)
- **At `1080 × 1920`:** Warm, charming storybook atmosphere; rounded castle and blossoming right-margin trees feel immediately cozy.
- **At `360 × 640` (`direction_b_360x640.png`):** High-frequency cobblestone joints on West/South-East roads introduce visual busyness directly beneath combat units. Warm sandstone cobbles (`#C8A878`) reduce chromatic pop under yellow Builders (`#F5C842`) and orange Brutes (`#EB7323`). Furthermore, several South-East units drift close to the `x = 82%` right-rail boundary.
- **In Grayscale (`direction_b_grayscale_360x640.png`):** Because chibi units allocate `~50%` of height to rounded heads/hoods, Runner, Guard, Fighter, and Grunt silhouettes compress toward similar rounded blobs when color is stripped at `360 × 640`. Soft anti-aliased painterly edges also conflict with `07_UNITY_ART_UX.md`'s `FilterMode.Point` requirement.

### 3.3 Direction C — Arcade Silhouette Fantasy (*Tactical Crisp Bastion*)
- **At `1080 × 1920`:** Razor-sharp architectural geometry; the concentric neon-magenta Boss telegraph ring and bright colored shields/maces pop dramatically against smooth `#3E524B` slate-moss ground.
- **At `360 × 640` (`direction_c_360x640.png`):** Zero ground texture means zero H.264 mosquito noise on lanes; colored geometric emblems (yellow aprons, blue rectangle shields, green spheres, orange cubes) remain hyper-distinct.
- **In Grayscale (`direction_c_grayscale_360x640.png`) & Broadcast Appeal Critique:** Two real weaknesses emerge on inspection:
  1. Humanoid units share pitch-black body cores (`#0B0E13`, `Y = 0.01`) that sink into the dark `#3E524B` slate turf (`Y = 0.08`) whenever off-lane, reducing Builders and Fighters to floating colored aprons/crests.
  2. Visually, pawn-like faceless black bodies skate uncomfortably close to `07_UNITY_ART_UX.md` line 39's prohibition (*"avoid generic featureless colored boxes as final characters"*) and create a cold, clinical mood poorly suited to `160 s` cooperative village-building segments on TikTok LIVE.

---

## 4. Deliverable C — Weighted Scoring & Proposed Recommendation

| Criterion | Weight | Dir A: Bright Stylized | Dir B: Premium Chibi | Dir C: Arcade Silhouette | Evidence Summary |
|---|---:|---:|---:|---:|---|
| **Small-screen legibility** | **30%** | **`9.0`** (`2.70`) | `6.8` (`2.04`) | `8.8` (`2.64`) | Dir A preserves full-body + prop silhouettes in both `360p` color and grayscale; Dir B loses silhouette separation over cobbles; Dir C loses black limbs against dark turf in grayscale. |
| **Livestream appeal** | **25%** | **`9.2`** (`2.30`) | `9.0` (`2.25`) | `6.8` (`1.70`) | Dir A balances warm heroic fantasy invitation (`PREP`) with high-drama siege action (`ASSAULT`/`BOSS`); Dir C feels cold/clinical for casual TikTok scroll-stop. |
| **Production consistency** | **20%** | **`8.8`** (`1.76`) | `6.5` (`1.30`) | `9.0` (`1.80`) | Dir A's bold `3 px` hard-edge ink + 2-step cel flats scale cleanly across `124` sprite frames and support stepped `FilterMode.Point` alpha without Dir B's gradient boiling or Dir C's placeholder look. |
| **Gameplay clarity** | **15%** | **`9.0`** (`1.35`) | `7.2` (`1.08`) | `9.2` (`1.38`) | Dir A & C both deliver instant `< 3 s` lane, wall-scaffold, Boss-causeway, and role-task comprehension; Dir B's cobbles and decorative village roofs compete with unit reads. |
| **Asset cost / complexity** | **10%** | **`8.4`** (`0.84`) | `6.0` (`0.60`) | `9.2` (`0.92`) | Dir A requires `~35%` less cleanup/repaint effort per frame than Dir B while still looking like a complete commercial 2D game rather than minimal pawns. |
| **WEIGHTED SCORE** | **100%** | **`8.95 / 10`** | **`7.27 / 10`** | **`8.44 / 10`** | **Winner: Direction A (`+0.51` over C, `+1.68` over B)** |

### Proposed Decision (`PROPOSED — AWAITING OWNER LOCK`)
**Recommend Direction A: Bright Stylized Fantasy (*"Bastion Dawn" / Bình Minh Thành Lũy*)**, reinforced with two targeted discipline rules borrowed from Direction C:
1. **Smooth Low-Noise Lane & Turf Surfaces:** Keep flank lanes (`#CFA96E`) and meadow turf (`#68A347`) free of high-frequency pebble/grass noise (avoiding Direction B's H.264 compression artifacts).
2. **Concentric High-Contrast Boss Telegraph Ring:** Adopt a crisp dual-ring violet-white ground telegraph beneath the Center-lane Boss so `BOSS` presence is unmistakable even on muted `360 × 640` mobile playback.

> **Governance Gate:** This recommendation is **`PROPOSED`**, **not** owner-approved or locked. Bulk production of the `124` final animation frames, environment tile atlases, and audio loops in `M06` MUST NOT begin until the repository owner reviews these visual deliverables and locks the direction in **`ART-002`**.
