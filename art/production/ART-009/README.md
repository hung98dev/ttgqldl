# ART-009 — Canonical Gameplay Fixture Repair, HUD Truthfulness & Final Unity Handoff Validation

- **Package ID:** `ART-009`
- **Target Engine:** `Unity 6000.6.7f1` — Built-in 2D (`1080x1920` portrait reference, `Orthographic Size = 5.0`, `PPU = 192` world / `100` UI)
- **Art Direction:** `Bastion Dawn v1` (zero changes to the `244` canonical `ART-003..ART-007` master assets or `6` packed atlases)
- **Package Status:** **`PASS` (Canonical Fixture, HUD Truthfulness & Metric Definitions Remediated and Verified)**
- **Rights Status:** `PENDING_REVIEW` (`GATE-LEGAL` open; `MUS01` & `MUS02` remain **MUTED BY DEFAULT** at `-80.0 dB`, `mute = true`)

---

## 1. Mission & Summary of Repairs

`ART-009` resolves all confirmed gameplay-truthfulness defects in `ART-008`'s synthetic showcase fixtures, eliminates hardcoded Unity HUD strings via a single-revision `SyntheticCommittedView` presentation adapter, corrects isolated batchmode performance-metric definitions, regenerates all affected GPU renders/videos/boards, and enforces `25` canonical `ttg-rules-v1` rules + `15` negative mutation tests in GitHub Actions CI:

1. **Test-First Defect Reproduction (`Phases 1 & 7`):**
   - `tools/validate_canonical_fixtures.py` reproduced all confirmed `ART-008` defects before remediation (`9/25` rules passed, `16/25` failed) and now passes **`25/25` positive canonical rules** (cross-checked against all `25` independent `M02` golden vectors in `tooling/refeval/testdata/golden_vectors.json`) + **`15/15` negative mutation rejections**.
2. **Canonical `ttg-rules-v1` Fixture Repair (`Defects A–F` & `Phase 3`):**
   - **Townhall HP (`Defect A`):** Corrected from `2000/2000` (`1850/2000` hardcoded) to bounded `0..500` (`townhall_max = 500`, post-breach defeat floor `250/500`).
   - **Wall HP by Tier (`Defect B`):** Derived strictly via `wall_max = 1000 + 200 * (wall_tier - 1)` (`Tier 1 -> 1000`, `Tier 6 -> 2000`, `Tier 7 -> 2200`, `Tier 20 -> 4800`), stepping Tier `6 -> 7` only when `construction_work >= 180` and `stone >= 600` (`-600 Stone`).
   - **Three-Lane & Subpool Split (`Defect C`):** Derived strictly via `West = floor(base_hp * 34 / 100)`, `Center = floor(base_hp * 32 / 100)`, `East = base_hp - West - Center` (`P = 204`, `consecutive_wins = 5` -> `base_hp = 620` -> `210 / 198 / 212`, with `25% / 55% / rem` `runner/grunt/brute` subpools and monotonically non-increasing ordinary HP after tick `1600`).
   - **Boss Max HP & Timing (`Defect D`):** Derived strictly via `boss_max_hp = ceil(base_hp / 5)` (`124` for `base_hp = 620`), `0` HP before tick `2600`, spawning only on `wave_no % 5 == 0` at tick `>= 2600`.
   - **Even-Wave Vote B (`Defect E`):** Corrected all fixtures and localized UI (`vi-VN` & `en-US`) from the false `[B: SỬA TƯỜNG +25%]` wall-repair text to canonical `Vote B: -50 Food, +15% Fighter (+Militia) attack` (`CHỐT [B: -50 Lương, Đánh+15%]` / `LOCKED [B: -50 Food, Atk+15%]`), charged once at tick `1200`.
   - **WAITING Tick (`Defect F`):** Unified `wave_tick = -1` and `P = 0` across beat metadata and nested snapshots.
3. **Truthful In-Place Unity 6000.6.7f1 HUD & Dynamic PREP Countdown (`Phases 2, 8 & Hotfix`):**
   - `HudInPlacePresenter.cs` + `ART008UnityShowcaseVerifier.cs` consume `SyntheticCommittedView` with zero hardcoded gameplay counters, derive `PREP` threat-banner countdowns strictly from authoritative `wave_tick` (`ceil((1600 - wave_tick) / 10)` -> `60s` at `1000`, `40s` at `1200`, `15s` at `1450`, `10s` at `1500`, `1s` at `1599`, `ASSAULT` at `1600`; zero wall-clock catch-up while paused), and export **12 canonical states × 3 resolutions (`1080p`, `720p`, `360p`)** + English Vote B (`40s`) + PREP `60s`/`15s` GPU proofs (`46` hardware `Direct3D12` GPU PNGs total, `0` scene reloads).
4. **Honest Performance & Audio Accounting (`Phases 4, 5 & Hotfix`):**
   - True 1% low FPS (`1000.0 / mean(slowest ceil(N * 0.01) frameTimesMs)`) reported separately from `p99FrameTimeMs` (`1222.02 FPS` avg, `653.98 FPS` 1% low, `1.315 ms` p99 at `150` entities in `cpu_camera_render_submission_editor_batchmode`; preserves the `50`-entity tier's unexplained `~506.650 ms` 2-frame stall unedited without claiming a definitive root cause).
   - Cumulative thread managed allocations measured via `GC.GetAllocatedBytesForCurrentThread()` (`0.0 B/frame`), `drawCalls = -1` (`NOT_MEASURED`), `gpuFrameTimingStatus = NOT_MEASURED_IN_EDITOR_BATCHMODE`, and standalone 15-min / OBS capture explicitly `NOT_EXECUTED`.
   - Verified `1` original seamless `36.00s` ambient loop (`amb_village_dawn_loop.wav` per `ART-007/asset_manifest.json`) and `ASSAULT` role-join gain `0.4467` linear (`-5.5 dB` / `0.5309` cue attenuation + `-1.5 dB` / `0.8414` `SFX` bus = `-7.0 dB` combined).

---

## 2. Reproducible Commands

```powershell
# 1. Run isolated Unity 6000.6.7f1 Direct3D12 showcase & export 12x3 GPU HUD frames
python -B art/production/ART-008/tools/run_unity_showcase.py

# 2. Rebuild all affected ART-008 contracts, fixtures, boards, and H.264+AAC MP4s
python -B art/production/ART-008/tools/build_showcase.py

# 3. Build ART-009 visual comparison boards, 360p H.264/AAC proof, and performance audit
python -B art/production/ART-009/tools/build_art009_package.py

# 4. Run independent canonical ttg-rules-v1 fixture validator (25 rules + 15 negative mutations)
python -B art/production/ART-009/tools/validate_canonical_fixtures.py --write-reports

# 5. Run full ART-008 + ART-009 package validators (full & CI-fast modes)
python -B art/production/ART-008/tools/validate_art008.py --write-reports
python -B art/production/ART-009/tools/validate_art009.py --write-reports
python -B art/production/ART-009/tools/validate_canonical_fixtures.py --ci-fast
python -B art/production/ART-009/tools/validate_art009.py --ci-fast
```
