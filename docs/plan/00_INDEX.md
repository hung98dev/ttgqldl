# TTGQLDL — CANONICAL IMPLEMENTATION PLAN INDEX
**One plan, twelve topic-owned modules, one main branch.** Status: M00 specification baseline validated by executable tooling on 2026-11-03; the product itself is NOT IMPLEMENTED OR TESTED (M01–M09 NOT_STARTED, M10–M11 BLOCKED). Do not create alternate plans, docs/v2 or other branches.

## Module ownership (each subject has exactly one authoritative owner)
| Document | Binding scope |
|---|---|
| [01_PRODUCT_SCOPE.md](01_PRODUCT_SCOPE.md) | Product requirements, selected versions, global rules |
| [02_GAME_RULES.md](02_GAME_RULES.md) | Commands, roles, combat, boss, economy and votes |
| [03_SIMULATION.md](03_SIMULATION.md) | Fixed ticks, authoritative event order, deterministic replay |
| [04_DATABASE.md](04_DATABASE.md) | PostgreSQL schema, transactions, snapshot and privacy erasure |
| [05_API_WEBSOCKET.md](05_API_WEBSOCKET.md) | HTTP auth/admin API and versioned WebSocket contracts |
| [06_LIVE_SECURITY.md](06_LIVE_SECURITY.md) | TikTok source rights/adapter, security, fairness and privacy |
| [07_UNITY_ART_UX.md](07_UNITY_ART_UX.md) | Portrait gameplay display, sprite/audio inventory and UX |
| [08_OPERATIONS.md](08_OPERATIONS.md) | Dev setup, CI, deployment, alerts, recovery and OBS |
| [09_QA_ACCEPTANCE.md](09_QA_ACCEPTANCE.md) | Test IDs, reproducible acceptance, benchmarks and evidence |
| [10_IMPLEMENTATION_MATRIX.md](10_IMPLEMENTATION_MATRIX.md) | Only milestone statuses, dependencies and work packages |
| [11_EXTERNAL_GATES.md](11_EXTERNAL_GATES.md) | Permission, legal, human UX, real hardware release blockers |
| [12_REVIEW_FINDINGS.md](12_REVIEW_FINDINGS.md) | Historical 30-perspective simulated review; NOT normative |

## AI read and execution contract
1. ALWAYS read AGENTS.md, this INDEX, 01_PRODUCT_SCOPE.md, 10_IMPLEMENTATION_MATRIX.md, and latest main/CI status.
2. Select first unblocked milestone with completed dependencies. Read its topic-owner modules and 09_QA_ACCEPTANCE.md. Read any cross-module dependency. For M00, rule changes, architecture revisions and final release, read ALL modules.
3. Map every MUST to requirement ID, module anchor, milestone/work package, test ID, and evidence path. Unknowns and conflicting statements are BLOCKED, not AI discretion.
4. Implement code, schemas, tests, fixtures, assets, and docs/evidence/Mxx-Wxxxx.md. Run relevant local checks BEFORE committing directly main; verify CI AFTER; fix or revert failure. Do NOT create PRs or new branches.
5. Update matrix PASS only with real SHA, commands, hardware/environment, actual test outputs, review and artifacts; mock evidence cannot pass LIVE permission or human gates.
6. Change gameplay rules only in owning module with associated simulation, QA and matrix changes in SAME commit. Storage/API changes update related owner modules and QA together. Findings log is not a second plan.
7. When modules conflict, stop the affected package, document the conflict and correct the owning module(s); independent unblocked work can proceed. Ownership is strict: 01 scope, 02 gameplay, 03 time/order, 04 persistence, 05 wire, 06 live/security, 07 visuals, 08 ops, 09 QA, 10 status, 11 external gates.

## Product/release boundaries
P0 is a mock/replay standalone game only after objective tests and real production assets. M00 is PASS on plan-consistency and reference-vector evidence (docs/evidence/M00-W0001..W0006.md plus `tooling/planvalidate`); implementation tests have NOT run. P1 is genuine TikTok LIVE and remains BLOCKED until expressly permitted source access, account consent, legal/asset rights, moderation and real-stream tests. AI cannot grant permissions or forge human assessment. Generated source, schema, fixtures and evidence are work outputs, not competing plan files.

Every module should remain focused. The M00 audit pushed 02_GAME_RULES.md, 04_DATABASE.md, 09_QA_ACCEPTANCE.md and 12_REVIEW_FINDINGS.md slightly past the original ~12 KB guidance because deterministic ordering and evidence schemas needed to be explicit; that is accepted. Growth beyond ~20 KB must split along ownership lines into a new topic-owned module registered in this index, never be appended into one document, and never become a second version of the plan.
