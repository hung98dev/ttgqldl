# EXECUTABLE WORK PACKAGES — ALL MANDATORY
Each row requires implementation, automated verification and evidence before marking PASS. Dependencies are strict. Work packages cannot be skipped because a demo appears to work.

## M00 — Specification approval
W0001 Review all docs for contradictory rules, produce ADR registry.
W0002 Produce final component diagram, data-flow and threat boundary diagrams.
W0003 Fix deliverables to P0 and create public issue labels/milestones.
W0004 Record official version source links and dated hashes/installer verification.
W0005 Record TikTok permissions gap with authorized owner decision.
Gate: architecture and game constants signed off by human product/tech owner.

## M01 — Reproducible scaffolding
W0101 Generate backend Go module/go.work and executable /healthz endpoint.
W0102 Set PostgreSQL 18.6 image digest and least-privileged local roles.
W0103 Bootstrap Unity project with Editor 6000.6.7f1 and exact packages.
W0104 Bootstrap TypeScript operator project with lockfile.
W0105 Create lint/test/format scripts for Windows and Linux.
W0106 GitHub Actions Go test/race, database integration, Unity edit/playmode, TS lint/test.
W0107 Docker Compose dev stack with no public DB binding, secret templates.
Gate: clean Windows and Linux checkout reproducible and CI PASS.

## M02 — Gameplay domain
W0201 Entity/state types with explicit integer ranges and validation.
W0202 Pure deterministic tick 10 Hz, phase state and pause/resume.
W0203 Deterministic integer square root and combat resolution ordering.
W0204 Deterministic monster seeding, three lanes, stable targeting and boss.
W0205 Resource consumption, construction tier upgrades and reward caps.
W0206 Defeat/recovery, XP caps, victory detection.
W0207 Votes and selected buffs / tie-breaking rules.
W0208 Snapshot canonical JSON and SHA256 golden fixtures at 100 seeds.
Gate: AC005, AC006, AC008 and deterministic golden fixtures PASS.

## M03 — Ingestion and commands
W0301 Validated typed ingest event shape and message size bounds.
W0302 Unicode Vietnamese normalizer/exact command grammar with fuzz testing.
W0303 Stable scoped identity/HMAC and ephemeral fallback.
W0304 Provider event deduplication with honest missing-ID fallback.
W0305 Last-valid role-per-window SQL upsert behavior and concurrent test.
W0306 Vote window and last-vote-wins semantics.
W0307 Mock event generator seeded for 10/100/1k/10k viewers.
W0308 NDJSON replay fixtures and deterministic sequence logs.
Gate: AC001-004, AC012, queue/drop accuracy.

## M04 — Durable state
W0401 SQL migrations for village/session/player.
W0402 SQL migrations for receipts/actions/votes/events.
W0403 Transactional server sequence and outbox atomicity.
W0404 Snapshots every 60 simulated seconds and on wave end.
W0405 Crash replay algorithm and checksum verification.
W0406 Database backup/restore and schema rollback integration tests.
W0407 Key rotation design and data erasure endpoint.
Gate: AC007, AC015, AC018 PASS.

## M05 — Network contract
W0501 Versioned OpenAPI 3.1 covering admin endpoints, statuses/errors.
W0502 JSON Schema for every WebSocket message with schema_version.
W0503 Bounded publish diff, resync and slow subscriber eviction.
W0504 Authentication/authorization and short-lived stream tokens.
W0505 Reconnect compatibility and schema mismatch tests.
Gate: on reconnect, state identical and no missing authoritative events.

## M06 — Unity renderer
W0601 9:16 fixed village composition, 3 lanes and castle HUD.
W0602 3 roles (idle/walk/action/hit), pooled active <=150.
W0603 3 enemies+boss, pooled animations/telegraphs/hit effects.
W0604 Top HUD+command legend+poll+results+status messages.
W0605 On-screen alias moderation, rotation, streamer overlay safe guides.
W0606 WebSocket client snapshot/reconnect/sequence handling.
W0607 Content creation pipeline, licensing, sprite atlas and animation import tests.
W0608 OBS capture tests 720p/1080p/1440p incl chat overlay.
Gate: AC009, AC016-017 PASS on specified hardware with recordings.

## M07 — Operator and moderation
W0701 Authenticated session start/pause/resume/stop UI and endpoints.
W0702 Live metrics, provider health, source lag, dropped/accepted counts.
W0703 Alias mute/hide and removal, audit history and retention controls.
W0704 Emergency stop, fallback to mock only visibly and only in dev.
W0705 RBAC/CSRF tests, rate limits and admin action integration tests.
Gate: AC014-015 PASS.

## M08 — Hardening
W0801 10k identities/1k eps 10min+5k burst perf harness.
W0802 Profile Go allocations, DB queries, WS throughput and Unity FPS.
W0803 Queue full and partial event loss diagnostics.
W0804 Kill backend midwave, PG disconnect, broken WS, restart recover.
W0805 Fuzz command/parser, SQL injection tests, auth security tests.
W0806 CVE+SBOM+license+secret scans with CI gates.
W0807 Balance Monte Carlo across 10/100/1k/10k roles, tune under approved ADR.
Gate: release SLO actual measured evidence, no high/critical security bugs.

## M09 — TikTok authorized connection
W0901 Identify and document permitted official integration pathway.
W0902 Verify account eligibility, permission scope, event schema, cursor and rate.
W0903 Implement provider adapter only after explicit green light.
W0904 Integration real session with consent and disconnection scenario.
W0905 Platform policy/privacy/monetization review and signed approval.
Gate: AC019 PASS. If pathway unavailable, BLOCKED. Never use unauthorized workarounds.

## M10 — Production and recovery
W1001 Deployment images pinned by SHA, TLS and secret provisioning.
W1002 Postgres PITR+daily backups and clean restore rehearsal.
W1003 Observability dashboards, alert policies and operator runbook.
W1004 2h staged LIVE test with overlay/sound, operator and moderation.
W1005 Rollback drill and migration compatibility.
Gate: RPO <=5m and RTO <=30m demonstrated; production rehearsal PASS.

## M11 — Final acceptance
W1101 20 acceptance IDs PASS and artifact links.
W1102 Security, privacy and license sign-off.
W1103 Release changelog, known limitations, next upgrades.
W1104 Approvals and release tagging.
Gate: no unchecked P0 requirements; no human approval forged by AI.

## Evidence formatting
File docs/evidence/Mxx-Wxxxx.md includes: immutable commit SHA; machine; exact command; logs/metrics; testcase IDs; reviewer; UTC; PASS/FAIL/BLOCKED; link to build artifacts. Every work package ID has one evidence file. Never copy PASS from one environment to another.
