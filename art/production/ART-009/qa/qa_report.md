# ART-009 — Canonical Remediation & Final Unity Handoff QA Report

- **Overall Status:** `PASS`
- **Package Acceptance Checks (`C01..C07`):** `7/7 PASS`
- **Canonical `ttg-rules-v1` Validator (`R01..R25`):** `25/25 PASS` (cross-checked against `25` M02 golden vectors)
- **Adversarial Negative Mutation Tests (`N01..N15`):** `15/15 REJECTED`
- **Open External Gates:** `M04 = IN_PROGRESS`, `M05 = NOT_STARTED`, `M06 = NOT_STARTED`, `GATE-LEGAL = PENDING_REVIEW` (`Music` muted `-80 dB`), `GATE-OBS = NOT EXECUTED`, Human UX/Listening = `NOT EXECUTED`

## 1. ART-009 Package Checks (`C01..C07`)

| Check ID | Status | Evidence |
|---|---|---|
| `C01_canonical_25_rules_and_15_negatives` | `PASS` | Canonical rules=25/25 PASS; Negative=15/15 rejected |
| `C02_required_art009_deliverables_present` | `PASS` | Verified 18 required docs/boards/QA files; missing=[] |
| `C03_unity_12_states_1080p_720p_360p_verified` | `PASS` | Verified 37 GPU state PNGs across 1080p/720p/360p + en-US Vote B; errs=[] |
| `C04_single_source_performance_and_d3d12_consistency` | `PASS` | 150-entity FPS=1222.02 matched across performance_report.json, ART-008 README & ART-009 policy; stale d3d11 files=0 |
| `C05_audio_gain_and_dedupe_preserved` | `PASS` | PREP=2000ms(5), ASSAULT=3500ms(3) @ -5.5dB cue + -1.5dB bus = -7.0dB (0.4467 lin), 12 voices, Music -80dB, Two-Tier dedupe verified |
| `C06_status_truthfulness_privacy_and_no_asset_duplication` | `PASS` | Gates truthful; leaks=[]; ART-009 PNG count=43 (no 244-asset duplication) |
| `C07_compressed_360p_h264_aac_stream_verified` | `PASS` | 360x640 H.264 + 48 kHz AAC stream verified (size=272981 bytes) |

## 2. Canonical `ttg-rules-v1` Regression Checks (`R01..R25`)

| Rule ID | Status | Title |
|---|---|---|
| `R01_TOWNHALL_HP_BOUNDS` | `PASS` | Townhall current HP is always within 0..500 |
| `R02_TOWNHALL_MAX_500` | `PASS` | Townhall maximum HP is always 500 across fixtures and Unity HUD |
| `R03_WALL_MAX_FROM_TIER` | `PASS` | wall_max strictly equals 1000 + 200 * (wall_tier - 1) |
| `R04_WALL_HP_BOUNDS` | `PASS` | Wall current HP is always within 0..wall_max |
| `R05_VILLAGE_LEVEL_BOUNDS` | `PASS` | Village level is within 1..50 and matches total_wins formula |
| `R06_WALL_TIER_BOUNDS` | `PASS` | Wall tier is always within 1..20 |
| `R07_RESOURCE_BOUNDS` | `PASS` | Resources (wood, stone, food) remain within 0..1,000,000 |
| `R08_WAITING_TICK_MINUS_ONE` | `PASS` | WAITING state consistently uses wave_tick = -1 and P = 0 |
| `R09_PHASE_AT_EVERY_TICK` | `PASS` | Phase derivation matches [-1=waiting, 0..1599=prep, 1600..2599=assault, 2600..2799=finale, 2800..2999=results] |
| `R10_EVEN_WAVE_ONLY_VOTE` | `PASS` | Voting opens only on even waves during ticks 800..1199 and locks at 1200 |
| `R11_VOTE_ABC_COSTS_AND_EFFECTS` | `PASS` | Vote A/B/C costs, effects, localization, and one-time tick-1200 deduction match ttg-rules-v1 |
| `R12_CANONICAL_LANE_SPLIT_AT_SPAWN` | `PASS` | Initial three-lane HP (34%/32%/rem) and runner/grunt/brute subpools (25%/55%/rem) follow ttg-rules-v1 |
| `R13_BOSS_MAX_FROM_BASE_HP` | `PASS` | Boss max HP strictly equals ceil(base_hp / 5) and never uses cinematic 1800 pools |
| `R14_BOSS_ABSENT_BEFORE_2600` | `PASS` | Boss does not exist and has 0 authoritative HP prior to tick 2600 |
| `R15_BOSS_EVERY_FIFTH_WAVE_ONLY` | `PASS` | Boss spawns/incoming occurs exclusively on every fifth wave (wave_no % 5 == 0) |
| `R16_NO_PREMATURE_RESULTS` | `PASS` | Wave settlement result is never emitted before tick 2800 and always present in 2800..2999 |
| `R17_MONOTONIC_ENEMY_HP_AND_DECORATIVE_UNITS` | `PASS` | Ordinary and Boss HP decrease monotonically after spawn; visual units are decorative proxies |
| `R18_SINGLE_SETTLEMENT_AND_REWARDS` | `PASS` | Settlement executes once at tick 2800 with exact +100W/+60S/+50F rewards and zero RESULTS repair |
| `R19_ROLE_COUNTS_AND_MILITIA_MATH` | `PASS` | Role counts sum to unique_joined and militia bonus equals max(0, 40 - 4*(P-1)) for 1<=P<=10 |
| `R20_RESULTS_QUEUED_NEXT_WAVE_ISOLATION` | `PASS` | Registrations during RESULTS stay queued for Wave+1 and promote cleanly at tick 3000 rollover |
| `R21_WALL_UPGRADE_RULES` | `PASS` | Wall tier upgrades obey 200*tier work and 100*tier stone thresholds at 200-tick boundaries |
| `R22_TRUTHFUL_STATUS_AND_WATERMARK_TEXT` | `PASS` | Backpressure vs drops, synthetic watermarks, and Music PENDING_REVIEW mute (-80 dB) remain truthful |
| `R23_HUD_MATCHES_UNDERLYING_FIXTURE` | `PASS` | All 12 GPU-rendered Unity HUD states derive 100% of counters from SyntheticCommittedView fixtures |
| `R24_REBUILD_HASHES_AND_PREVIEW_PROVENANCE` | `PASS` | Timeline durations, SHA-256 provenance, 1% low FPS math, GC profiler counters, and README figures agree |
| `R25_NO_UNSUPPORTED_M05_WS_SCHEMA_CLAIM` | `PASS` | Fixtures explicitly mark M05 WS schema as NOT_STARTED synthetic views and enforce <= 300 session XP cap |

## 3. Negative Mutation Suite (`N01..N15`)

| Mutation ID | Status | Triggered Rule(s) |
|---|---|---|
| `N01_TOWNHALL_MAX_2000` | `PASS` | `R02_TOWNHALL_MAX_500` — Inject townhall_max = 2000 & townhall_hp = 1850 |
| `N02_TIER8_WALL_MAX_1200` | `PASS` | `R03_WALL_MAX_FROM_TIER` — Inject Tier 8 with wall_max = 1200 (expected 2400) |
| `N03_INVALID_LANE_420_560_440` | `PASS` | `R12_CANONICAL_LANE_SPLIT_AT_SPAWN` — Inject 420/560/440 lane split for base_hp = 1420 |
| `N04_BOSS_MAX_1800` | `PASS` | `R13_BOSS_MAX_FROM_BASE_HP` — Inject boss_max_hp = 1800 for base_hp = 1420 (expected 284) |
| `N05_WRONG_VOTE_B_TEXT` | `PASS` | `R11_VOTE_ABC_COSTS_AND_EFFECTS` — Inject Vote B wall-repair text ('SỬA TƯỜNG +25%') |
| `N06_WAITING_TICK_ZERO` | `PASS` | `R08_WAITING_TICK_MINUS_ONE` — Inject WAITING snapshot with wave_tick = 0 |
| `N07_DUPLICATE_VICTORY` | `PASS` | `R18_SINGLE_SETTLEMENT_AND_REWARDS` — Inject duplicate wave_victory settlement event |
| `N08_BOSS_BEFORE_2600` | `PASS` | `R14_BOSS_ABSENT_BEFORE_2600` — Inject active Boss before tick 2600 |
| `N09_UNSUPPORTED_PHASE_BOUNDARY` | `PASS` | `R09_PHASE_AT_EVERY_TICK` — Inject phase='assault' at tick 1500 (< 1600) |
| `N10_INVALID_RESOURCE_AWARD` | `PASS` | `R18_SINGLE_SETTLEMENT_AND_REWARDS` — Inject invalid +250 Wood victory settlement reward |
| `N11_INCORRECT_XP_CAP` | `PASS` | `R25_NO_UNSUPPORTED_M05_WS_SCHEMA_CLAIM` — Inject session XP = 340 (> 300 cap) |
| `N12_STALE_UI_COUNTER` | `PASS` | `R23_HUD_MATCHES_UNDERLYING_FIXTURE` — Inject stale HUD Castle HP text disagreeing with fixture |
| `N13_MISSING_SYNTHETIC_WATERMARK` | `PASS` | `R22_TRUTHFUL_STATUS_AND_WATERMARK_TEXT` — Inject mock source_status with watermark_visible = False |
| `N14_WRONG_MUSIC_RIGHTS_DEFAULT` | `PASS` | `R22_TRUTHFUL_STATUS_AND_WATERMARK_TEXT` — Inject gate_legal_status=APPROVED and unmuted music |
| `N15_FALSE_PERFORMANCE_METRIC_STATUS` | `PASS` | `R24_REBUILD_HASHES_AND_PREVIEW_PROVENANCE` — Inject 1000/p99 1%-low definition & draw_calls=0 MEASURED |
