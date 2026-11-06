# ART-009 — Canonical Gameplay, HUD & Performance Remediation Report

## 1. Test-First Defect Reproduction Summary

Before modifying any `ART-008` generator or C# source file, `art/production/ART-009/tools/validate_canonical_fixtures.py` was written against `docs/plan/02_GAME_RULES.md`, `03_SIMULATION.md`, `05_API_WEBSOCKET.md`, `07_UNITY_ART_UX.md`, and `tooling/refeval/model.go` (`ttg-rules-v1`).

- **Pre-Remediation Baseline (`ART-008` unmodified):** **`9/25` rules `PASS`, `16/25` rules `FAIL`** (`15/15` negative mutations rejected).
- **Post-Remediation Verification (`ART-009` complete):** **`25/25` rules `PASS`, `0` failures** (`15/15` negative mutations rejected).

| Rule ID | Pre-Fix (`ART-008`) | Post-Fix (`ART-009`) | Confirmed Defect & Exact Remediation |
|---|---|---|---|
| `R01_TOWNHALL_HP_BOUNDS` | `FAIL` (30 violations) | `PASS` | `townhall_hp` used `1850..2000`; clamped and rule-stepped within `0..500` (`250` post-breach recovery floor). |
| `R02_TOWNHALL_MAX_500` | `FAIL` (30 violations) | `PASS` | `townhall_max = 2000` & hardcoded `THÀNH 1850/2000`; fixed to `500` (`CHÍNH 500/500`, `KEEP 500/500`). |
| `R03_WALL_MAX_FROM_TIER` | `FAIL` (29 violations) | `PASS` | Tier `6/7/8` used `1000/1100/1200`; fixed to `1000 + 200*(tier-1)` (`Tier 6 -> 2000`, `Tier 7 -> 2200`). |
| `R08_WAITING_TICK_MINUS_ONE` | `FAIL` (2 violations) | `PASS` | Nested `WAITING` snapshots used `wave_tick = 0`; unified to `wave_tick = -1` and `P = 0`. |
| `R09_PHASE_AT_EVERY_TICK` | `FAIL` (2 violations) | `PASS` | `WAITING` tick/phase mismatch resolved across `B00_WAITING` and `failure_07_no_viewers_waiting`. |
| `R11_VOTE_ABC_COSTS_AND_EFFECTS` | `FAIL` (8 violations) | `PASS` | Vote B falsely stated `SỬA TƯỜNG +25%`; fixed to `-50 Food` (`-50 Lương`) & `+15% Fighter (+Militia) attack`, charged once at tick `1200`. |
| `R12_CANONICAL_LANE_SPLIT` | `FAIL` (2 violations) | `PASS` | Replaced arbitrary `420/560/440` with `P=204, d=6 => base_hp=620 => 210 / 198 / 212` + `25%/55%/rem` subpools. |
| `R13_BOSS_MAX_FROM_BASE_HP` | `FAIL` (12 violations) | `PASS` | Replaced `boss_max_hp = 1800` with `ceil(base_hp / 5) = 124` (`109` in `failure_04` for `P=168 => base_hp=544`). |
| `R15_BOSS_ONLY_FIFTH_WAVE` | `FAIL` (2 violations) | `PASS` | Cleared non-zero `boss_max_hp` on non-Boss waves (`wave_no % 5 != 0`). |
| `R17_MONOTONIC_ENEMY_HP` | `FAIL` (1 violation) | `PASS` | Eliminated ordinary lane HP increases after tick `1600` spawn; damage strictly decreases valid pools. |
| `R18_SINGLE_SETTLEMENT_AND_REWARDS` | `FAIL` (1 violation) | `PASS` | Verified exact single settlement at tick `2800` (`+100 Wood, +60 Stone, +50 Food` on win; `250 HP` floor on breach). |
| `R21_WALL_UPGRADE_RULES` | `FAIL` (2 violations) | `PASS` | Enforced `construction_work >= 180` and `stone >= 600` (`-600 Stone`, `+200 wall_hp` & `wall_max`) at boundary ticks. |
| `R22_GAMEPLAY_STATUS_TEXT` | `FAIL` (1 violation) | `PASS` | Removed `THÀNH 1850/2000` and `[B: SỬA TƯỜNG +25%]`; distinguished `ĐỢT CHẴN` pre-vote PREP (`<800`) from `ĐỢT LẺ`. |
| `R23_HUD_MATCHES_FIXTURE` | `FAIL` (1 violation) | `PASS` | Replaced hardcoded C# strings with `SyntheticCommittedView` + read-back verification across all `12` states + `en-US` Vote B. |
| `R24_REBUILD_HASHES_AND_PERF_SINGLE_SOURCE` | `FAIL` (5 violations) | `PASS` | Linked `canonical_wave10_fixture_sha256`, true 1% low FPS, `GC.GetAllocatedBytesForCurrentThread()`, `Direct3D12` naming, and single-source README metrics. |
| `R25_NO_FAKE_M05_SCHEMA` | `FAIL` (1 violation) | `PASS` | Kept `synthetic_fixture: true` & `schema_version: ws-public-v1-synthetic` without fabricating unimplemented `M05` fields. |

---

## 2. Owning Files Repaired vs New Remediation Package Files

- **Repaired in Owning `ART-008` Files (Zero Asset Duplication):**
  - `art/production/ART-008/tools/showcase_contracts.py`
  - `art/production/ART-008/tools/build_showcase.py`
  - `art/production/ART-008/tools/run_unity_showcase.py`
  - `art/production/ART-008/tools/validate_art008.py`
  - `art/production/ART-008/showcase/scene_source/HudInPlacePresenter.cs`
  - `art/production/ART-008/showcase/scene_source/ART008UnityShowcaseVerifier.cs`
  - Regenerated `ART-008` fixtures, `presentation_timeline.json`, `audio_dispatch_policy.json`, `qa/performance_report.json`, `qa/unity_isolated_verification.json`, `Direct3D12` GPU renders (`unity_d3d12_*.png`, replacing deleted `unity_d3d11_*.png`), proof boards, and H.264/AAC `.mp4` videos.
- **Added in `art/production/ART-009/`:**
  - Independent `ttg-rules-v1` validator (`tools/validate_canonical_fixtures.py`) + package validator (`tools/validate_art009.py`) + separately reported hotfix countdown/audio regression suite (`3/3` positive + `6/6` negative mutations) + CI gate in `.github/workflows/ci.yml`.
  - `12` GPU-rendered `Unity 6000.6.7f1` states at `1080x1920`, `720x1280`, and `360x640` + English Vote B `40s` `1080p` proof + PREP `60s`/`15s` GPU renders (`46` GPU PNGs), dynamic countdown & before/after boards, compressed `360p` H.264/AAC stream proof, `36.00s` ambient loop handoff parity, and formal handoff audit records.
