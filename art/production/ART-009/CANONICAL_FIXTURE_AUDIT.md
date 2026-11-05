# ART-009 — Canonical `ttg-rules-v1` Fixture Audit & Backend Boundary Note

## 1. Fixture Generation Architecture & Authority Boundary

Per `Phase 3` of `ART-009` and `AGENTS.md`:

1. **Read-Only Parity Verification Against `tooling/refeval` (`M02` Golden Reference):**
   - We inspected `tooling/refeval/model.go` and `tooling/refeval/testdata/golden_vectors.json` (`C001..C025`) read-only without modifying `backend/` or `tooling/refeval/`.
   - Because `tooling/refeval` evaluates compact regression cases rather than exporting multi-beat presentation JSON snapshots with UI overlay metadata, `ART-008/tools/showcase_contracts.py` uses a **tightly bounded boundary reducer (`step_canonical_wave`)** with explicit `make_snapshot` rule assertions, and `ART-009/tools/validate_canonical_fixtures.py` cross-checks every formula directly against all `25` `M02` golden vectors (`0` discrepancies).
2. **Explicit Synthetic Classification (Never Claimed as Executable `M02` Replay):**
   - Every showcase JSON carries `"synthetic_fixture": true`, `"schema_version": "ws-public-v1-synthetic"`, and `"is_executable_m02_replay": false`.
   - `presentation_timeline.json` explicitly distinguishes:
     - **Actual logical wave duration:** `300 seconds` (`ticks -1, 0..3000` at `10 Hz`).
     - **Encoded review montage duration:** `20.0 seconds` (`240 frames @ 12 FPS`).
     - **Selected timeline beats:** `20` beats (`B00..B19`, max logical gap `20.0s <= 30.0s`).
     - **Actual simulation boundary transitions executed:** `14` boundary steps (`ticks 200..2800` every `200` ticks + `tick 2800` settlement).
3. **Optional Future Backend Fixture-Export Note for `DeepSeek Harness` (`M05`):**
   - No backend code was modified in `ART-009`. When `DeepSeek Harness` implements `M05` (`W0502`/`W0504` public WebSocket projection), adding a read-only CLI flag (e.g. `go run ./cmd/fixtureexport -seed <n> -wave 10`) that dumps committed `/ws/v1/public` snapshots from the Go `simulation` + `projection` packages will allow `M06` PlayMode tests to swap `"ws-public-v1-synthetic"` fixtures for live `ws-public-v1` captures with zero presentation rework.

---

## 2. Mathematical Verification of Wave 10 Canonical Scenario (`synth-wave10-001`)

| Rule / Boundary | Formula (`ttg-rules-v1`) | Wave 10 Showcase Value |
|---|---|---|
| **Session / Village Init** | `total_wins = 39 => village_level = min(50, 1 + floor(39/3)) = 14`; `wall_tier = 6` (`wall_max = 2000`, `wall_hp = 1560`); `townhall_hp = 500/500` | Verified (`B00_WAITING`) |
| **WAITING -> First Join** | `P = 0 => wave_tick = -1` (frozen); first valid join (`B=1`) starts `wave_tick = 0`, `militia_bonus = 40 - 4*(1-1) = 40` | Verified (`B00 -> B01`) |
| **Resource & Repair (`200..1200`)** | Every `200` ticks: `Wood += 2*isqrt(B)`, `Stone += isqrt(B)`, `Repair = 8*isqrt(B)` clamped to `wall_max`, `Work += 3*isqrt(B)` during `PREP` while `wall_tier < 20` | `wall_hp: 1560 -> 1616 -> 1688 -> 1768 -> 1856 -> 1944 -> 2000` |
| **Even-Wave Vote (`800..1200`)** | Opens only when `wave_no % 2 == 0` (`800..1199`); locks at `1200`. **Vote B** charges `-50 Food` once (`540 -> 490`) and sets `fighter_attack_multiplier = 115%` | Verified (`B05..B07`, stable through `B18`) |
| **Wall Upgrade (`tick 1400`)** | At `tick 1400`, `Work = 187 >= 180` and `Stone = 856 >= 600` -> `Stone = 256`, `Work = 7`, `wall_tier: 6 -> 7`, `wall_max: 2000 -> 2200`, `wall_hp: 2000 -> 2200` | Verified (`B08_PREP_140S_FORTIFY`) |
| **Population Anchor & Spawn (`1600`)** | `P_anchor = max(1, B+G+F) = 48+92+64 = 204`; `d = min(10, 1+5) = 6`; `base_hp = 40 + 8*6 + 38*isqrt(204) = 88 + 38*14 = 620` | `West = 210`, `Center = 198`, `East = 212` (`sum = 620`) |
| **Enemy Subpools (`1600`)** | `25% runner`, `55% grunt`, `rem brute` per lane | `W:[52,115,43]`, `C:[49,108,41]`, `E:[53,116,43]` |
| **Combat & Mitigation (`1600..2400`)** | `Attack = floor((18*isqrt(F) + militia) * 115 / 100)`; `Block = 14*isqrt(G)`; damage targets `runner -> grunt -> brute` across lanes; overflow hits `Wall` then `Townhall` | Ordinary HP `620 -> 483 -> 346 -> 191 -> 36 -> 0` at `tick 2400` (`wall_hp = 2174/2200`) |
| **Boss Warning & Spawn (`2450..2600`)** | `wave_no = 10` (`% 5 == 0`) => `boss_max_hp = ceil(620 / 5) = 124`. `tick 2450`: `boss_incoming = true`, `boss_hp = 0`. `tick 2600`: spawns `124`, immediate tick-`2600` attack (`62`) reduces Boss to `62/124` | Verified (`B15` -> `B16`) |
| **Final Boss Kill & Settlement (`2800`)** | At `tick 2800`, `F=25` (`Attack = floor(18*5*1.15) = 103 >= 62`) slays Boss (`62 -> 0`) before settlement check -> `RESULTS_WIN`, `+100W/+60S/+50F`, `total_wins: 39 -> 40` (`village_level: 14 -> 15`), `XP <= 300` cap | Verified (`B17`, `B18`) |
| **Next-Wave Registration & Rollover** | Joins during `2800..2999` (`19` viewers) queue for Wave `11` without inflating Wave `10` active counts (`204`); at `tick 3000 -> Wave 11 tick 0`, `19 + 15 = 34` active in `PREP` | Verified (`B18 -> B19`) |
| **Supplementary Defeat Proof** | `supplementary_states.defeat_wave10_breach`: unmitigated breach sets `townhall_hp = 0`, settles as `loss`, restores post-loss recovery floor `townhall_hp = 250/500`, `wall_hp = 250/2000`, `-15%` next-wave enemy HP | Verified (`09_defeat_1080p.png`) |
