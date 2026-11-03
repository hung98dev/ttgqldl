# TTGQLDL / LIVE VILLAGE — MASTER CONTRACT v1.0
Date 2026-11-03. Status: SPECIFICATION, NOT IMPLEMENTATION. English is normative. All MUST requirements are release blockers unless explicitly marked optional.

## Technology lock
Go 1.27.3; PostgreSQL 18.6; Unity Editor 6000.6.7f1 (Unity 6.6 Update, latest verified patch on 2026-11-03). Windows 11 dev, Docker Compose PostgreSQL dev, Linux amd64 deployment. Go standard net/http, pgx/v5, sqlc, goose, WebSocket library to be pinned by security audit in M01, structured logging slog, OpenTelemetry, Prometheus. Operator: React + TypeScript + Vite, pin exact versions in M01 lockfiles. No Redis/Kafka/Kubernetes in baseline. PostgreSQL 19 is RC, forbidden until production GA and compatibility tests.
Verified vendor resources: https://go.dev/dl/ https://www.postgresql.org/docs/release/ https://www.postgresql.org/about/news/postgresql-19-rc-1-released-3409/ https://unity.com/releases/editor/archive

## Objective
A vertical 9:16 community village defense game for TikTok LIVE. Viewer can join via one text command, no install. Thousands of distinct logical viewers participate while only bounded aggregate units are simulated and rendered. Persistent village per broadcaster; clear 300-second wave loop. External real-time TikTok comment access MUST be legally authorized and documented; unverified integration is release BLOCKED, never faked.

## Definition of release
Release requires all P0 rows in docs/06_IMPLEMENTATION_MATRIX.md PASS with source commit, reproducible test logs and review. No blocker/critical or high security bugs; deterministic replay; crash/reconnect tests; OBS portrait video review; 10,000 viewer & load benchmarks; restore rehearsal; livestream approval. Mock mode may be released as a separate product without TikTok claims.

## Required repository layout
backend/cmd/server, backend/internal/{domain,simulation,commands,ingest,identity,store,stream,operator,security,telemetry}, backend/db/{migrations,queries}, backend/api/openapi.yaml, backend/test/{integration,load,golden}, unity/Assets/{Scenes,Scripts,Prefabs,Art,Tests}, operator/src, deploy, docs, .github/workflows. All code formatted and linted; dependency version lock; SBOM; license scan; secrets scan.

## Strict AI execution rules
Read docs 00–07 and any ADR before coding. Follow matrix order. Implement one acceptance group at a time. Every PR must contain code, tests, docs status, commands/evidence, migration/rollback notes and SHA. Never mark PASS based on self-description. No changing spec or formulas to make tests pass; change through ADR and new golden fixtures. Unknown external dependency = BLOCKED; never hide with mock success. Use exact stable versions until approved lock update. Never merge without CI and required review. Error paths, idempotency and instrumentation are mandatory.

## Architecture
Monolithic Go backend with independently testable modules and a single authoritative writer per village: approved LIVE adapter / mock / replay -> bounded ingest -> dedup + identity -> parser -> keyed command window aggregate -> deterministic state machine -> Postgres event log + snapshots -> versioned snapshot WebSocket -> Unity display / authenticated operator. Clock, RNG and I/O injected; domain simulation pure and deterministic. Unity calculates no authoritative stats. A single viewer may have one effective action per 20-second window and at most one effective ballot per active vote.

## Fixed baseline numbers
Wave = 300 seconds; preparation 180s, assault 90s, finale 20s, result 10s. Boss every 5th wave. Three combat roles, three lanes, three grunt archetypes + boss. Server simulation tick 10Hz; client visual target 30fps on benchmark hardware. At most 150 representative on-screen units by default. Logical population capacity target >=10,000. Load target 1,000 input events/s sustained 10min with 5,000/s bursts for 10s; internal accepted-to-published p95 <=1s at target hardware, subject to measured release evidence. Operator can pause/stop and resync. Pacing clear event at least every 30-60 seconds, without fabricating actions or forced drama.

## Product and ethical constraints
No pay-to-win, gambling, purchase-to-improve-odds, gift-based random rewards, spam incentives, deceptive fake viewers, unnecessary personal data or harassment on-screen. Explicit streamer name opt-out, moderation and data deletion policy. Fail visibly if provider unavailable.

## Document index
01_GAMEPLAY.md: all rules and balance. 02_BACKEND_DATA_API.md: service/data/API contract. 03_LIVE_SECURITY.md: adapter, security, privacy. 04_UNITY_UX.md: presentation design. 05_QA_SLO.md: tests and gates. 06_IMPLEMENTATION_MATRIX.md: work sequence. 07_OPERATIONS.md: setup and recovery.
