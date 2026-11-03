# Mandatory AI Agent Execution Contract

Project: TTGQLDL LIVE Village. This document is binding for every coding agent.

## Sources of truth
Read docs/00_MASTER_SPEC.md through docs/09_OPEN_GATES_AND_DECISIONS.md before any code change. In case of inconsistency: STOP, document a discrepancy and create a proposed ADR; do NOT silently select your favorite interpretation. The user/technical owner must approve any material game rule or technology change. Status and results belong in docs/06_IMPLEMENTATION_MATRIX.md.

## Work execution
1. Fetch latest main and inspect actual repo/CI; never assume a task is done.
2. Select first unblocked NOT_STARTED/FAIL milestone whose dependencies are PASS.
3. Convert all MUST requirements into concrete source changes and automated acceptance tests.
4. Implement full code, docs, API schemas, SQL migrations, error handling, security and metrics, not just happy-path demo.
5. Execute unit, integration, race and applicable Unity tests. Produce reproducible output, SHA and hardware/version metadata.
6. Update matrix status to PASS only when **all** checks pass; otherwise FAIL or BLOCKED with evidence.
7. Work directly on the single canonical branch `main`. Run required unit/integration/security checks BEFORE committing; commit only passing changes with exact test evidence recorded under docs/evidence. Never create feature/docs branches or PRs for this project unless the user explicitly changes the branch policy. Never force-push main or bypass safety checks.
8. Describe changed behavior, test commands/results, known limitations, rollback and next unblocked milestone.

## Product release invariants
- Go 1.27.3, PostgreSQL 18.6, Unity 6000.6.7f1 baseline; pin actual dependencies.
- Deterministic authoritative simulation in Go, never in Unity.
- One role contribution per stable viewer per 20-second window, irrespective of message count.
- 10,000 logical participants need not render as 10,000 actors.
- No prohibited TikTok scraping, unverified API claims, paid combat advantage, fraudulent viewer input, gambling or comment spam incentives.
- All TikTok production integration BLOCKED until explicit authorized access and policy compliance proven.
- Never claim performance/SLO PASS from design estimates alone.

## Forbidden shortcuts
No dummy production code, no ignored errors, no hardcoded credentials, no disabling failing tests, no fabricated test evidence, no silent schema/data loss, no UI pretending to be LIVE when offline, no late-game rules invented without approved ADR.

## Completion output format
Milestone ID; changed files; exact tests; actual outcomes; evidence references; unresolved risks; rollback notes; matrix status; next milestone.
