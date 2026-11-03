<!-- evidence
wp: M01-ACCEPTANCE
milestone: M01
status: IN_PROGRESS
commit: WORKTREE@c69ae5549f7fcad5d17fe12e53a3dea6436fafb7
head: c69ae5549f7fcad5d17fe12e53a3dea6436fafb7
utc: 2026-11-03T17:50:00Z
environment: Windows 11 amd64 (local) + GitHub Actions ubuntu-24.04; Go 1.27.3; Node 24.21.0; PostgreSQL 18.6 (pinned digest); Unity 6000.6.7f1 (local only)
requirements: REQ-09-CI,REQ-09-SECURITY,REQ-09-SBOM,REQ-TOOL-UNITY-RESULTS
tests: QA-GOV-02,QA-MAT-01,QA-SEC-12,QA-OPS-07,QA-CLI-04,QA-CLI-01,QA-CLI-02,QA-CLI-03
reviewer: independent work-package audit against code, tests and CI (automated + manual review)
reviewer_kind: automated
-->

# M01 acceptance record

Milestone-level audit of all eight M01 work packages at revision
`dc429781da1ed2757771bfa2fd9d726eefab6892` (CI run
[41687406542](https://github.com/hung98dev/ttgqldl/actions/runs/41687406542):
5 jobs, 64 steps, 0 failures). The audit began at `c69ae55`; the committed
revision differs only by documentation, the prepared Unity workflow and the
validator allowance documented below. Each row was checked against the
code, the automated tests and the recorded evidence; a package is PASS only when
all three agree.

Machine-written evidence lives in the per-package files; this record exists so the
milestone verdict has one auditable owner instead of being inferred from a
workflow colour.

## Status

| Field | Value |
|---|---|
| M00 | PASS (regression re-checked: plan validator 15 checks, 0 errors) |
| M01 | **IN_PROGRESS** |
| W0106 | **IN_PROGRESS** - licensed Unity CI execution not yet provisioned |
| M02-M09 | NOT_STARTED (M01 dependency gate not satisfied) |
| M10-M11 | BLOCKED (external rights) |

## Work-package audit

### W0101 - Go backend foundation - **PASS**

| Item | Value |
|---|---|
| Acceptance criteria | `GET /healthz` and `GET /readyz` behave exactly as specified; graceful shutdown; structured logs; request-id propagation; error envelope |
| Implementation | `backend/cmd/server`, `backend/internal/app/{app,handlers,middleware}.go`, `backend/internal/config`, `backend/internal/telemetry` |
| Tests | `backend/internal/app/app_test.go`; whole-module `go test ./...` (5 packages) and `go test -race ./...` in CI |
| Evidence | `docs/evidence/M01-W0101.md` |
| Revision | CI run 41687406542 on `dc42978` (`backend` job) |
| Unmet prerequisite | none |

### W0102 - PostgreSQL development environment - **PASS**

| Item | Value |
|---|---|
| Acceptance criteria | PostgreSQL 18.6 pinned by digest; compose stack; role bootstrap; data persists across restart |
| Implementation | `compose.dev.yaml`, `docker/postgres/initdb/001_roles.sh`, `tooling/dev-up`, `tooling/dev-down` |
| Tests | Same image and initdb scripts started by the CI `backend` job; migration and privilege suites |
| Evidence | `docs/evidence/M01-W0102.md` |
| Revision | CI run 41687406542 on `dc42978` |
| Unmet prerequisite | none |

### W0103 - Schema, migrations, indexes, sqlc - **PASS**

| Item | Value |
|---|---|
| Acceptance criteria | Full documented schema; additive migrations; least-privilege roles; schema fingerprint drift detection; sqlc in sync |
| Implementation | `backend/db/migrations/00001..00004`, `backend/internal/schema`, `backend/db/schema.sql`, `backend/db/schema.fingerprint.json`, `backend/internal/db` |
| Tests | `backend/test/integration/{schema,identity_aliases,roles,fingerprint,zz_migration_lifecycle}_test.go`; `sqlc diff` |
| Evidence | `docs/evidence/M01-W0103.md` |
| Revision | CI run 41687406542 on `dc42978` |
| Unmet prerequisite | none |

### W0104 - Unity project foundation - **PASS** (local execution)

| Item | Value |
|---|---|
| Acceptance criteria | Pinned Unity project opens and compiles; EditMode + PlayMode suites pass; Windows standalone builds |
| Implementation | `unity/` (9 assemblies, 3 scenes), `tooling/unity-test.ps1|.sh`, `tooling/unity-build.ps1|.sh` |
| Tests | EditMode 56/56, PlayMode 7/7 and a player build on the licensed workstation; wrapper regression 21 checks; helper 13 checks |
| Evidence | `docs/evidence/M01-W0104.md` |
| Revision | local run on the `3e04e5a` tree; wrapper re-verified after the W0106 hardening |
| Unmet prerequisite | the same suites are **not** executed in CI (see W0106) |

### W0105 - React/TypeScript operator foundation - **PASS**

| Item | Value |
|---|---|
| Acceptance criteria | Strict TS build; eslint clean; tests; production bundle without demo data or secrets; only implemented endpoints called |
| Implementation | `operator/` |
| Tests | 76 Vitest/Testing-Library tests; `tsc --noEmit`; `eslint .`; `vite build`; DEMO guard build that must fail |
| Evidence | `docs/evidence/M01-W0105.md` |
| Revision | CI run 41687406542 on `dc42978` (`frontend` job) |
| Unmet prerequisite | none |

### W0106 - CI, security, milestone acceptance - **IN_PROGRESS**

| Item | Value |
|---|---|
| Acceptance criteria | CI covers unit/race/integration/Unity/security; scanners pinned and gating; SBOM; no weakened gates |
| Implementation | `.github/workflows/ci.yml`, `.github/workflows/unity-editor-tests.yml`, `tooling/security-scan.ps1|.sh`, `tooling/sbom`, `.gitleaks.toml` |
| Tests | backend/frontend/security/tooling jobs green; Unity **static** validator green; wrapper regression 21 checks |
| Evidence | `docs/evidence/M01-W0106.md` |
| Revision | CI run 41687406542 on `dc42978` |
| Unmet prerequisite | **licensed, isolated Unity runner** - real EditMode/PlayMode/build execution in CI. The workflow is prepared and fails closed while disabled; no runner is registered and no licence secret exists. |

### W0107 - Cross-platform tooling - **PASS**

| Item | Value |
|---|---|
| Acceptance criteria | PowerShell and Bash entry points behave identically; `doctor` verifies versions; scripts resolve the repository root and propagate exit codes |
| Implementation | `tooling/*.ps1`, `tooling/*.sh`, `tooling/doctor.ps1|.sh` |
| Tests | `tooling/tooling-test.*`, shell syntax gate in CI, wrapper scripts exercised in this task |
| Evidence | `docs/evidence/M01-W0107.md` |
| Revision | CI run 41687406542 on `dc42978` |
| Unmet prerequisite | none |

### W0108 - Configuration and secrets - **PASS**

| Item | Value |
|---|---|
| Acceptance criteria | `.env.example` only; `.env` gitignored; typed startup validation; stage/prod refusals; no secret in logs or bundles |
| Implementation | `backend/internal/config`, `.env.example`, `.gitignore`, operator demo guard |
| Tests | `backend/internal/config` tests; gitleaks 0 findings; production bundle secret scan |
| Evidence | `docs/evidence/M01-W0108.md` |
| Revision | CI run 41687406542 on `dc42978` |
| Unmet prerequisite | none |

## Verification performed for this record

| Check | Result |
|---|---|
| Plan validator | 15 checks, 0 errors |
| Go vet / unit / build | 0 / 5 packages ok / 0 |
| Go race | green in CI |
| PostgreSQL migrations, schema, permissions, lifecycle | green in CI |
| `sqlc diff` | 0 |
| Reference golden vectors | PASS |
| Unity static validator | PASS 7 checks |
| Unity result + wrapper regression | 13 + 21 checks |
| Unity EditMode / PlayMode (licensed workstation) | 56 / 7 passed, 0 skipped |
| Operator typecheck / lint / tests / build | 0 / 0 / 76 / built |
| govulncheck / gosec | 0 / 0 |
| npm audit (production) | 0 critical, 0 high; 11 moderate dev-only |
| gitleaks / actionlint | 0 / 0 |
| SBOM | 439 components, 1 image digest-pinned, 144 without licence metadata |

## Why M01 is not PASS

The plan requires the Unity suites to run in CI. That requires a licensed,
isolated runner with the pinned editor and the Windows module. Provisioning one is
a repository-owner action (runner registration and licence secret); it is not
performed here, and a local workstation run is explicitly not accepted as a
substitute. `docs/evidence/M01-W0106.md` section 2 states the exact requirements.

Once that runner exists, `.github/workflows/unity-editor-tests.yml` executes the
same validated wrapper, and W0106/M01 can close without further code changes.