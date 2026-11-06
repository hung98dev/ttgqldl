# ART-010A — Explicit Asset Reuse & Replacement Matrix (`ART-003`..`ART-009`)

**Principle:** Keep `100%` of validated engineering, audio, animation contracts, and canonical simulation bindings while replacing prototype flat-circle visuals with modular isometric village sprites in `ART-010B` (only after owner layout approval).

---

## 1. Package-by-Package Reuse vs Replacement Matrix

| Package | Existing Assets | Reuse % | Keep As-Is | Replace / Repaint in `ART-010B` (Post-Approval) |
|---|---|---:|---|---|
| **`ART-003`** (Pilot Sprites: `Builder`, `Grunt`, `Repair Spark`) | `36` character frames (`48 px` gameplay height) + `6` spark frames + `12 FPS` timing JSONs | **40% (Contracts & Timing)** | `18`-frame state layout (`idle`/`move`/`work_or_attack`), `12 FPS` cadence, pivot conventions, hit/spark sync frames | **Repaint** `Builder` & `Grunt` sprite sheets (`~72 px` screen height on `128×128` master cells) in chunky stylized 3D-like 2D village art style; **upgrade** `repair_spark` brightness for scaffolding repairs |
| **`ART-004`** (Remaining 5 Classes + 4 Combat VFX) | `Guard`, `Fighter`, `Runner`, `Brute`, `Boss` (`88` frames) + `24` VFX frames | **40% (Contracts & Timing)** | Exact 7-class roster, state machine timing (`16–18f`), Boss scale hierarchy (`~2.8×` unit height), VFX cue IDs | **Repaint** `5` character sheets to match vibrant village lighting & doorway-relative scale (`~24–28 px` at `360p`); **redesign** `4` combat VFX (`shield_spark`, `sword_arc`, `wall_dust`, `boss_rune_slam`) |
| **`ART-005`** (Environment, Ring Castle, Tile Lanes) | Flat `#74A14E` green void, stair-stepped `64×64` sand tiles, tiny gray circular wall (`<9%` canvas) | **10% (Slot/State Rules Only)** | Logical 3-lane topology (`West`, `Center`, `East`), `Townhall` upgrade slot (`Lv1–5`), `Wall` damage state thresholds | **100% Visual Replacement:** Build modular isometric sprite kit (`9` distinct building archetypes, connected stone-and-oak wall segments + gates + scaffolding, smooth cobblestone roads, meadow & trees) |
| **`ART-006`** (16-State HUD, 20 Icons, 15 Nine-Slices) | Safe-area layout JSONs (`82%×88%`), 16-state rules, Vietnamese strings, dark `#1E2430` debug boxes | **65% (UX, Anchors & Copy)** | All 16-state transition rules, Vietnamese/English localization tables, TikTok `x <= 885, y <= 1689` safe anchors | **Reskin** `15` nine-slice panels (`Suncrest Royal Timber & Burnished Gold`) + **repaint** `20` beveled fantasy icons (`Wood`, `Stone`, `Food`, `Wall`, `Townhall`, `Boss`, `Builder`, `Guard`, `Fighter`) |
| **`ART-007`** (Original Audio: 12 SFX, 1 Ambient, 2 Music) | `15` WAV/OGG cues (`36.00s` seamless village ambient loop, `12` combat/repair/UI SFX, `2` music loops) | **100% Direct Reuse** | **All 15 audio masters**, loudness normalization, loop points, `cue_manifest.json`, and `PENDING_REVIEW` music mute safety | **None (`0` audio files need replacement)** — warm medieval village ambience and hammer/sword/shield/Boss SFX fit the modular village direction natively |
| **`ART-008` & `ART-009`** (Showcase Fixtures & Unity C# Code) | `HudInPlacePresenter.cs`, `SpriteTransitionController.cs`, `BroadcastAudioDispatcher.cs`, `30` canonical `ttg-rules-v1` snapshots + dynamic PREP countdown | **90% Direct Reuse** | **100% of C# HUD/audio/countdown/dedupe logic** and canonical `ttg-rules-v1` fixture snapshots | Update only serialized `Sprite` references, isometric `CustomAxisSort` settings, and 3-lane/muster world-space waypoints in the isolated showcase scene |

---

## 2. Effort Concentration Summary for `ART-010B`

- **Zero Backend / Simulation / Audio Rework (`0%` churn):** `backend/`, PostgreSQL migrations, `ttg-rules-v1` math, `ART-007` audio, and `ART-009` HUD state logic stay untouched.
- **Focused High-Impact Art Production (`100%` visual value):**
  1. **Modular Village Sprite Kit (`~42` static sprites across `2` `2048×2048` atlases)** — replaces `ART-005` completely.
  2. **Cohesive Character & VFX Sheet Upgrade (`106` character frames + `30` VFX frames in `1` `2048×2048` atlas)** — upgrades `ART-003`/`ART-004` to match the vibrant stylized 3D-rendered-as-2D village buildings.
  3. **Royal Timber & Gold HUD Atlas (`1` `1024×1024` atlas)** — upgrades `ART-006` skins in-place.
