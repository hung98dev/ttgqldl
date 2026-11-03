# PRODUCT SCOPE AND TECHNOLOGY — MANDATORY
**Canonical modular plan, product scope** · Updated 2026-11-03 · Branch main · Not implemented or tested.

## 0. GOALS, SCOPE, GOVERNANCE AND FIXED TECHNOLOGY
One canonical plan lives in docs/plan/ with one topic-authority per file. 00_INDEX.md is the mandatory entry point. AGENTS.md is only implementation guidance. Code, schemas, migrations, assets, fixtures and evidence are deliverables, not new plans. Do not create parallel versions, branches or pull requests without explicit request.

P0 = full offline mock/replay game with Go authoritative engine, PostgreSQL, Unity broadcast app, operator UI, verified original/right-cleared assets, test/load/recovery and CI; explicitly label simulation MOCK/REPLAY. P1 = separately gated authorized TikTok LIVE production integration with real credentials, permissions, broadcaster rights, preview, human review, moderation and real end-to-end rehearsal. AI must never claim external permission, legal compliance or real testing from an unverified document.

Locked stable baseline: Go 1.27.3, PostgreSQL 18.6, Unity Editor 6000.6.7f1. Windows 11 desktop Unity/OBS, Linux amd64 backend; Go net/http + pgx/v5 + sqlc + goose; React/TypeScript/Vite operator; Docker Compose; PostgreSQL is sole durable source of truth. Do not add Redis, Kafka, Kubernetes, microservices or an external paid service without measured evidence and explicit approved change. Pin exact third-party versions, Unity package versions, lockfiles, Docker image digests at first executable implementation. Verify vendor stable releases and license:
https://go.dev/dl/ ; https://www.postgresql.org/support/versioning/ ; https://unity.com/releases/editor/archive .

Product: 9:16 cooperative chibi village survival. Three input commands XAY/THU/DANH (and exact English aliases). One valid join per wave grants automatic 20s-boundary role contribution throughout the wave; repetitive comments do not add power or XP. 300s fixed wave, 10Hz authoritative tick, three lanes, boss every fifth wave. Logical population 10k; <=150 visual entities. Do not auto simulate empty-room losses. No pay-to-win, gambling, spam incentives, gifts influencing combat or falsified viewer counts.

Hard requirements say MUST/SHALL. If contradictory: block the affected requirement, repair the owning module and dependent fixtures/acceptance with an explicit decision; independent tasks may continue. No unstated default or fabricated PASS. Work solely on main; run pre-commit tests, verify CI post-commit and revert/repair regressions. Evidence must contain code SHA, exact commands, OS/hardware, test inputs, expected/actual output, CI artifact, review provenance.

**All authoritative links and AI read order:** [00_INDEX.md](00_INDEX.md).

Verified official patch references (as of 2026-11-03): https://go.dev/dl/ ; https://www.postgresql.org/docs/release/18.6/ ; https://www.postgresql.org/about/news/postgresql-19-rc-1-released-3409/ (19 remains RC) ; https://unity.com/releases/editor/whats-new/6000.6.7f1 (known issues must be tested). Version selection is pinned for reproducibility; any future GA upgrade requires CI + migration approval, not unreviewed automatic upgrade.
