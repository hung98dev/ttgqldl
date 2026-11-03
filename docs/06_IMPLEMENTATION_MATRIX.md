# IMPLEMENTATION MATRIX — SOURCE OF TRUTH
Statuses allowed: NOT_STARTED, IN_PROGRESS, BLOCKED, PASS, FAIL. ALL ARE NOT_STARTED on initial publication except external TikTok access BLOCKED.

| ID | Mandatory delivery | Dependency | Status | Required evidence |
|---|---|---|---|---|
| M00 | Baseline docs + ADR + version pins + scope | none | IN_PROGRESS | human review and approved spec |
| M01 | Monorepo skeleton, Go1.27.3, PG18.6, Unity6000.6.7f1, compose, CI | M00 | NOT_STARTED | clean build from fresh checkout |
| M02 | deterministic simulation, math, wave state and golden tests | M01 | NOT_STARTED | replay same hash 100 seeds |
| M03 | parser, mock/replay adapter, 20s dedupe + votes | M02 | NOT_STARTED | AC001-004, AC012 |
| M04 | migrations, event log, atomic ordering, snapshots/recovery | M03 | NOT_STARTED | AC007, AC018 |
| M05 | WebSocket protocol, stream resync + typed models | M04 | NOT_STARTED | reconnect tests |
| M06 | Unity 9:16 visual complete, animation pool, readable HUD | M05 | NOT_STARTED | AC009, AC016-017 recordings |
| M07 | secure operator dashboard, RBAC, moderation, emergency stop | M05 | NOT_STARTED | AC014-015 + abuse tests |
| M08 | load, chaos, security, profiling and docs correction | M06,M07 | NOT_STARTED | AC005-011, AC013, soak report |
| M09 | approved real TikTok adapter + rights verification | M08 | BLOCKED | AC019 official approval + end-to-end stream |
| M10 | backup/restore, deployment, OBS production rehearsals | M08,M09 | NOT_STARTED | AC018-020, operator signoff |
| M11 | release approval and single-branch safeguards | M10 | NOT_STARTED | all P0 PASS, no severe issues |

M00 itself is NOT PASS until product/technical owner approves open risks and checks consistency. Coding agents MUST NOT skip M00 approval; can prepare a local scaffold patch while approval pending but cannot declare production ready. If TikTok rights unavailable, ship simulator-only preview separately with clear non-TikTok claims; M09 remains BLOCKED.

## Agent execution prompt contract
Read all /docs, inspect repo and latest CI. Select first unblocked non-PASS matrix row. Write plan of concrete code/test/doc changes; implement fully; run tests; capture evidence; update row only when objective pass; commit to main with linked evidence after required local checks, then verify CI. Do not create PRs. Never jump ahead, guess authorization or change hard rules silently. Report unresolved limitations and exact next row.

## Single-branch safeguards
Only main is canonical; never create branches or PRs unless the user explicitly requests them. Require lint, tests and security scans before pushing; verify CI after commits and immediately fix/revert failures. Human review/explicit approval for docs baseline, LIVE adapter/security configuration and production release remains mandatory, regardless of direct commits. CI version and migration changes need recorded compatibility testing.
