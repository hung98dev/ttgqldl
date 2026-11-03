# TTGQLDL / LIVE VILLAGE — NORMATIVE IMPLEMENTATION CONTRACT V2.0
Date: 2026-11-03. Scope: complete engineering-ready simulator product, with authorized TikTok LIVE adapter as externally gated integration. Authoritative document set: docs/v2/*. Older docs/00..09 are superseded after migration. MUST/SHALL = mandatory, SHOULD = non-blocking recommendation, MAY = optional. No unstated default.

## Source precedence and change control
1. docs/v2/00_CONTRACT.md (scope/governance and source precedence)
2. docs/v2/01_GAME_RULES.md and 02_SIMULATION.md (domain/game mathematics)
3. docs/v2/03_PERSISTENCE.md and 04_PROTOCOL.md (backend boundaries)
4. docs/v2/05_PLATFORM_SECURITY.md, 06_UNITY_CONTENT.md and 07_OPERATIONS.md (platform/client/ops)
5. docs/v2/08_ACCEPTANCE.md and 09_EXECUTION_MATRIX.md (delivery evidence/gates)
6. docs/v2/10_REVIEW.md (historical findings; not a new rules source)
On discrepancy: agent MUST stop the affected requirement, file a docs discrepancy ticket, fix with explicit design decision and tests; may continue independent tasks. No agent may quietly choose between conflicting requirements. All docs and fixtures changes are versioned together. This contract governs M00. Approval cannot be self-certified where legally or explicitly human-controlled.

## Product definition and explicit non-goals
A desktop-origin 2D Unity portrait (9:16) video game whose authoritative Go service ingests validated text events and updates a persistent cooperative village. Comments XAY/THU/DANH (or exact English equivalent) register one contribution role per wave. Thousands of logical viewers are aggregated into bounded simulation and <=150 onscreen representative units. Roles persist for the entire active wave without repeated messages. Every wave lasts 300 simulation seconds; new viewers may join any non-RESULTS phase; a vote is a separate optional action.
P0 delivery: offline simulator/mock/replay, Go service, PostgreSQL database, Unity broadcast client, authenticated operator console, artifact assets, 10k load evidence, security/privacy and recoverability. P1 separately: authorized LIVE adapter, policy verification, live session rehearsal. Do NOT describe P0 as 'TikTok LIVE production working' until P1 gate passes. Out of scope for P0: real-money mechanics, microtransactions, donation-to-win, multiplayer mobile installs, global game economy, public user login, voicebot, gift-based combat changes, automated account signup, scraping.

## Fixed stack / approved extensions
Go 1.27.3 (official stable as of 2026-11-03), PostgreSQL 18.6 (19 RC excluded until GA), Unity Editor 6000.6.7f1. Windows 11 development/OBS, Linux amd64 backend. Go net/http, pgx/v5, sqlc and goose; a maintained WebSocket implementation selected and pinned with security review in M01. Operator React + TypeScript + Vite with exact lockfile in M01. All versions/digests pinned at scaffold, dependency security checks mandatory. Unity 2D Built-in render pipeline by default for predictable 2D; migration to URP requires ADR and measured benefit. Docker Compose for development and single-host initial deployment; PostgreSQL owns durable source of truth. No Kafka/Redis/Kubernetes before a measured bottleneck ADR. All timestamps UTC. No cross-architecture promise without testing.
Version sources: https://go.dev/dl/ ; https://www.postgresql.org/support/versioning/ ; https://www.postgresql.org/about/news/postgresql-19-rc-1-released-3409/ ; https://unity.com/releases/editor/archive .

## Operability and autonomous engineering boundary
Engineering AGENT MAY implement, test, simulate, revise non-product code and deploy development environments without additional human input once this fixed baseline is accepted. AGENT MUST NOT claim permission to connect to TikTok without real approved credentials, consent and terms. AGENT MUST NOT self-grant legal approval, stream moderator coverage, live credentials, copyright licenses or real-device evidence. On unavailable external access, produce functioning offline production-quality mock, exact blocker report and ready-to-enable adapter interface. Unknown decision must have safe deterministic fallback documented in docs/v2/11_DECISION_REGISTER.md; do not invent one.

## CI/quality
main is the SINGLE code branch. No new branch or PR unless owner explicitly changes policy. Before each commit: reproducible automated checks and evidence. After push: verify CI; immediately repair or revert regressions. No falsified test logs. Migration rollback and release safety are required even on main. No 'DONE' claim without evidence links, SHA and acceptance mapping.
## Source layout
backend/cmd/{server,simulator}, backend/internal/{domain,sim,commands,source,identity,storage,stream,operator,telemetry}, backend/db/{migrations,queries}, backend/api, backend/test/{unit,integration,golden,load}, unity/Assets/{Scenes,Scripts,Prefabs,Art,Tests}, operator/src, contracts/{openapi,events,schemas}, tooling, deploy, docs/v2, .github/workflows, docs/evidence.
## Feature flags
SIM_SOURCE=mock|replay|authorized_live; authorized_live MUST fail closed unless separately configured and approved. Sim/test UIs display MOCK or REPLAY watermark. No hidden fallback to mock in LIVE mode. Environment variables must have example, type, default, validation, owner, secret/PII classification and recovery behavior in 07_OPERATIONS.
