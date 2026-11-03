# ORDERED IMPLEMENTATION MATRIX AND MANDATORY WORK PACKAGES
Allowed state NOT_STARTED, IN_PROGRESS, BLOCKED, FAIL, PASS. All P0 milestones start NOT_STARTED until implemented/tested. M10 and M11 start BLOCKED on external rights. Each package requires docs/evidence/Mxx-Wxxxx.md with SHA, test IDs, exact command, UTC, environment/hardware, actual results, PASS/FAIL/BLOCKED and reviewer. Completion is NOT inferred from this document.

| ID | Dependencies | Binding deliverable | Initial state | Completion gate |
|---|---|---|---|---|
| M00 | none | resolve plan consistency, fixture plan, traceability | NOT_STARTED | no internal contradictory requirements |
| M01 | M00 | pinned Go/PG/Unity/TS skeleton; ALL SQL schema and CI | NOT_STARTED | clean Windows/Linux build and PG migrations |
| M02 | M01 | 10Hz pure deterministic domain/game/replay | NOT_STARTED | golden math/wave/boss/vote/XP tests |
| M03 | M01,M02 | mock/replay source, parser, identity, durable input log | NOT_STARTED | spam, dedupe, role and queue tests |
| M04 | M02,M03 | fenced single writer, atomic tick+outbox, snapshots/recovery | NOT_STARTED | crash/replay/privacy/restore tests |
| M05 | M04 | contract-checked API, RBAC, WebSocket, projected state | NOT_STARTED | contract, security, resync tests |
| M06 | M05 | Unity gameplay HUD, all assets, OBS capture | NOT_STARTED | 150 object/FPS/legibility/rights test |
| M07 | M05 | operator dashboard, moderation, deletion and emergency stop | NOT_STARTED | admin security and E2E |
| M08 | M06,M07 | 10k viewer load, chaos, profiling, SAST and UX QA | NOT_STARTED | measured SLOs and proofs |
| M09 | M08 | P0 standalone MOCK/REPLAY release and recovery | NOT_STARTED | all P0 acceptance PASS |
| M10 | M09,LIVE permission | authorized TikTok comment adapter | BLOCKED | provider contract and real integration |
| M11 | M10,legal/OBS/owner | LIVE production rehearsal and release | BLOCKED | real 2h moderated broadcast + approvals |

M00 W0001 cross-section contradiction audit; W0002 test-vector catalog; W0003 verify official version releases; W0004 dependency DAG; W0005 requirement-to-test traceability; W0006 baseline engineering approval.
M01 W0101 Go executable and tests; W0102 PG18.6 image and Compose; W0103 ALL schema/migrations/indexes/sqlc (before M03); W0104 Unity pinned editor and tests; W0105 TS operator lockfiles; W0106 GitHub Actions unit/race/integration/Unity/security; W0107 Windows+Linux tooling; W0108 secrets templates.
M02 W0201 integer domain/types; W0202 phase and tick state; W0203 per-wave role and cooldown; W0204 anchor and three-lane HP pools/boss; W0205 attack/guard/builder/upgrade; W0206 vote; W0207 loss/reward/XP; W0208 100 golden seed/crash/replay fixtures.
M03 W0301 Unicode exact grammar; W0302 mock/replay event adapter; W0303 scoped HMAC/ephemeral identity; W0304 provider ID duplicate handling; W0305 durable monotonic event input; W0306 valid/invalid queue/drop metrics; W0307 10k unique mock participants; W0308 event fuzz tests.
M04 W0401 writer epoch fencing; W0402 committed state+tick+outbox; W0403 canonical snapshots and replay; W0404 SQL crash restart; W0405 PITR/backup restore; W0406 privacy erasure/backup tombstones; W0407 event retention.
M05 W0501 OpenAPI 3.1 schemas; W0502 JSON WS schemas; W0503 admin auth, CSRF and RBAC; W0504 full/patch/public snapshot and resync; W0505 expiring token handling; W0506 TS/C# typed clients and negative tests.
M06 W0601 centered castle, wall and lanes; W0602 pooled <=150 entities; W0603 builder/guard/fighter, enemies and boss animations; W0604 all source/phase screens and HUD; W0605 production original/right-cleared sprite/audio manifest; W0606 OBS 9:16 platform overlay screenshots; W0607 15min FPS+GC; W0608 human 3sec understandability.
M07 W0701 secure operator UI; W0702 start/pause/resume/stop; W0703 health/lag/queue metrics and alerts; W0704 name filtering/block/hide; W0705 erasure and evidence audit; W0706 emergency stop/recovery and admin E2E.
M08 W0801 1k events/s 10min + 5k/s burst10s with10k actors; W0802 backend crash, PG outage, reconnect; W0803 fencing split-brain attempt; W0804 SAST/SBOM/secrets/license audit; W0805 Go/DB profiling; W0806 Unity OBS real capture; W0807 full acceptance audit.
M09 W0901 build pinned images; W0902 mock/replay release visibly labeled; W0903 backup/restore and rollback; W0904 100% P0 evidence audit; W0905 changelog/known limitations.
M10 W1001 approved account event access and permission proof; W1002 authorized adapter; W1003 real reconnect/ID/delivery semantics; W1004 owner-consented live E2E; W1005 policy/privacy verification.
M11 W1101 real OBS/LIVE preview; W1102 human moderation readiness; W1103 >=2h real broadcast rehearsal; W1104 disaster rollback drill; W1105 legal/operator release signoff.

Execution loop: inspect latest main and CI; choose first unblocked NOT_STARTED/FAIL milestone with dependencies PASS; implement code, tests and evidence; do not change combat math without recorded revision; update progress only with proof; commit directly main after checks; confirm CI; fix/revert if failed. Never pretend a missing approval is PASS.
