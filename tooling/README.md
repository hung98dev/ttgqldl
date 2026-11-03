# tooling/ — developer entry points and baseline validators

Portable entry points for the TTGQLDL repository plus two dependency-free Go
programs. Every entry point exists as a PowerShell (`.ps1`) and a Bash (`.sh`)
script; both do the same thing and propagate their exit code.

```
tooling/
  _common.ps1|.sh          shared helpers (repo root, logging, pinned Go toolchain)
  doctor.ps1|.sh           environment verification against the pinned baseline
  dev-up.ps1|.sh           start the PostgreSQL 18.6 development dependencies
  dev-down.ps1|.sh         stop them without deleting data
  backend-run.ps1|.sh      run the Go backend from source
  backend-test.ps1|.sh     gofmt + go vet + unit tests (+ -Race)
  tooling-test.ps1|.sh     plan validator + golden vectors + tooling tests
  migrate-up.ps1|.sh       apply all pending PostgreSQL migrations
  migrate-down.ps1|.sh     roll back one migration (or -To N / --to N)
  migrate-status.ps1|.sh   show applied and pending migrations
  schema-dump.ps1|.sh      regenerate backend/db/schema.sql for sqlc
  unity-test.ps1|.sh       run Unity EditMode/PlayMode tests in the pinned editor
  unity-build.ps1|.sh      build the Windows standalone player
  unitycheck/              licence-free Unity project validation (CI gate)
  unity-results.ps1        shared NUnit result evaluation (rejects zero-test runs)
  unity-results.test.ps1   regression tests for that evaluation (no Unity needed)
  operator-run.ps1|.sh     start the operator console (Vite) against the Go backend
  operator-test.ps1|.sh    npm ci, typecheck, lint, tests and production build
  security-scan.ps1|.sh    govulncheck, gosec, npm audit, gitleaks, actionlint, SBOM
  sbom/                    deterministic CycloneDX inventory + container pin check
  unity-test.wrapper.test.ps1  wrapper-level Unity result acceptance tests
  cmd/unitygate/           acceptance preflight (exact main HEAD, owner switches)
  sanitize-unity-log.ps1   redacts secrets/identities from Unity logs before upload
  sanitize-unity-log.test.ps1  regression tests for that sanitizer
  cmd/unityfake/           test-only stand-in editor for those tests
  validate-plan.ps1|.sh    (M00) plan-validator wrapper
  reference-eval.ps1|.sh   (M00) golden-vector wrapper
  planvalidate/            canonical-plan consistency checker (Go, stdlib only)
  refeval/                 independent gameplay reference evaluator (Go, stdlib only)
```

## Developer entry points (M01 Phase A)

Run them from anywhere; each resolves the repository root itself.

| Command | What it does | Exit codes |
|---|---|---|
| `./doctor.ps1` / `./doctor.sh` | Prints one row per dependency and distinguishes correct version, missing dependency, incorrect version, unusable install and permission/licence block. Required now: Go 1.27.3, Docker + Compose plugin, pinned PostgreSQL image digest, ≥5 GiB disk. Optional (reported, not fatal): Node (W0105), Unity (M06), `.env`. Add `-Strict` / `--strict` to fail on optional gaps too. | 0 ok, 1 required failure |
| `./dev-up.ps1` / `./dev-up.sh` | Creates `.env` from `.env.example` when missing, then `docker compose --env-file .env -f compose.dev.yaml up -d --wait` and waits for the health check. Never deletes data, never edits an existing `.env`. | 0 ok, non-zero on compose failure |
| `./dev-down.ps1` / `./dev-down.sh` | `docker compose down`; the named volume is preserved. `-WipeData` / `--wipe-data` deletes it after an interactive confirmation and refuses in non-interactive shells. | 0 ok, non-zero on failure |
| `./backend-run.ps1` / `./backend-run.sh` | Points `TTG_ENV_FILE` at the repository `.env` and runs `go run ./cmd/server`. Refuses to start without `.env`. | server exit code |
| `./backend-test.ps1` / `./backend-test.sh` | gofmt check, `go vet`, `go test`. `-Race`/`--race` adds `go test -race` (needs cgo; the Linux CI job is the authoritative race gate). `-Short`/`-short` passes `-short`. | 0 ok, first failing step's code |
| `./tooling-test.ps1` / `./tooling-test.sh` | `go vet`, `go test`, then the plan validator and the golden vectors. | 0 ok, first failing step's code |
| `./migrate-up.ps1` / `./migrate-up.sh` | `backend/cmd/migrate -direction up` using `TTG_MIGRATION_DATABASE_URL` (schema-owner role). The server's app role cannot run DDL. | 0 ok, migrate exit code |
| `./migrate-down.ps1` / `./migrate-down.sh` | Rolls back one migration; `-To N` / `--to N` rolls back to version N. `0` (full rollback) asks for an interactive confirmation. | 0 ok, migrate exit code |
| `./migrate-status.ps1` / `./migrate-status.sh` | Applied and pending migrations plus the current schema version. | 0 ok |
| `./schema-dump.ps1` / `./schema-dump.sh` | Runs `pg_dump` inside the compose container and rewrites `backend/db/schema.sql` (the sqlc input), then writes `backend/db/schema.fingerprint.json` (tables, columns, constraints, indexes). Both are byte-reproducible and CI fails on drift. | 0 ok |
| `./unity-test.ps1` / `./unity-test.sh` | Finds the pinned Unity 6000.6.7f1 editor (`TTG_UNITY_EDITOR`, the Hub install or `Unity` on PATH), verifies the exact version, runs EditMode and/or PlayMode tests and reports from the NUnit XML. A run without an XML result is never reported as success. | 0 pass, 1 failure, 2 blocked (editor/version/licence), 3 harness error |
| `./unity-build.ps1` / `./unity-build.sh` | Verifies the editor, the pinned version and the Windows standalone module, builds `unity/Build/Windows/TTGQLDL.exe` and fails when the player is missing. | 0 built, 1 build failed, 2 blocked |
| `./operator-run.ps1` / `./operator-run.sh` | Checks Node 24, installs from the lockfile when needed, and starts Vite on 127.0.0.1:5173 with `/healthz`, `/readyz` and `/admin/v1` proxied to the Go backend. `--demo` / `-Demo` enables the explicit fixture preview. | 0 ok, 2 toolchain blocked |
| `./operator-test.ps1` / `./operator-test.sh` | `npm ci`, `npm run typecheck`, `npm run lint`, `npm run test -- --run`, `npm run build`, failing fast with the step's exit code. | 0 ok, 1 gate failed, 2 toolchain blocked |
| `./security-scan.ps1` / `./security-scan.sh` | Runs the pinned security gates (govulncheck, gosec, npm audit high/critical, gitleaks, actionlint) and `go run ./sbom` for the CycloneDX inventory and container digest pinning. A missing scanner exits 2 (BLOCKED), never green. Reports land in `security-reports/` (gitignored; CI uploads them). | 0 ok, 1 finding, 2 blocked |
| `go run ./cmd/unitygate ...` | The licensed-Unity acceptance preflight: requires runner + environment owner switches, `refs/heads/main`, the expected repository, and requested == checked out == freshly fetched `origin/main` tip. Ancestors are rejected. | 0 accepted, 1 rejected |
| `./sanitize-unity-log.ps1` / `.test.ps1` | Rewrites a Unity log with secrets, licence serials, OS account names, licence-channel ids, home paths and machine ids removed, then re-scans its own output. Exit 3 means a residual pattern survived and the log must not be uploaded. | 0 clean, 2 unreadable, 3 residual |
| `./unity-test.wrapper.test.ps1` | Executes the real Unity wrapper against a fake editor and asserts non-zero exits for zero-test, all-skipped, inconsistent-counter, corrupt, missing, stale and failed results. | 0 ok, 1 failure |
| `./unity-results.test.ps1` | Regression tests for the Unity result evaluation: a result file with zero tests or a fully skipped suite must fail. Runs without Unity. | 0 ok, 1 failure |
| `go run ./unitycheck -root ..` | Validates the Unity project without a licence: pinned editor version, package manifest vs lockfile, `.meta` coverage for files and folders, duplicate guids, scene guid resolution, null `m_Script`, assembly boundaries, test-assembly gating, ignore rules. Runs in CI; it is **not** a substitute for executing the editor. | 0 ok, 1 findings |

Typical loop:

```bash
cd tooling
./doctor.sh            # environment is correct?
./dev-up.sh            # PostgreSQL 18.6 on 127.0.0.1:5432
./backend-test.sh      # gofmt, vet, unit tests
cd ../backend
TTG_TEST_DATABASE_URL='postgres://ttg_app:...@127.0.0.1:5432/ttgqldl?sslmode=disable' go test ./test/integration/...
cd ../tooling
./tooling-test.sh      # M00 guarantees still hold
./dev-down.sh          # stop, keep the data volume
```

### Go toolchain note

`backend/go.mod` requires **Go 1.27.3**. If the host pins an older toolchain
(`go env -w GOTOOLCHAIN=go1.27.1`), these scripts pass `GOTOOLCHAIN=auto` to the
Go commands they run and print a note, so the module-selected toolchain is used
instead of failing. They never edit the user's global Go configuration; the
permanent fix is `go env -w GOTOOLCHAIN=auto`.

## Requirements for the validators

- Go 1.27.x. The tooling module declares `go 1.27.0`; the product baseline pins
  **Go 1.27.3** and `doctor` verifies the effective toolchain in `backend/`.
- No database, no network, no Unity, no Node.

## Validator commands

Always run from `tooling/` (the wrappers handle this for you).

```bash
# 1. Plan consistency: links, DAG, QA ids, constants, status honesty, traceability
go run ./planvalidate -root ..

# machine-readable report for CI
go run ./planvalidate -root .. -json

# 2. Gameplay reference vectors: assertions + golden-drift check
go run ./refeval -check

# print the population/role sweep table
go run ./refeval -sweep

# regenerate golden vectors after a DOCUMENTED rules_version change
go run ./refeval -write

# 3. Self-tests for both programs
go test ./...
```

## Exit codes (validators)

| Code | Meaning |
|---|---|
| 0 | all checks passed |
| 1 | at least one ERROR (the report lists check, file, line, message) |
| 2 | usage or I/O error (for example the plan directory is missing) |

`planvalidate` treats a missing canonical module, an unresolvable link or
anchor, a duplicate id, a milestone cycle, an undefined work package, a
constant drift, a PASS milestone without PASS evidence for **every** work
package, or a traceability row whose evidence file does not exist as a hard
ERROR. M10/M11 may not leave BLOCKED without human-signed evidence.

## What the plan validator checks

1. `canonical-plan-files` — all 13 modules of `docs/plan/` exist.
2. `markdown-links` — every relative link and `#heading` anchor resolves.
3. `matrix-parse` / `milestone-dag` / `work-packages` — the milestone table
   parses, ids are unique, dependencies are `Mnn` or declared `GATE-*`, the graph
   is acyclic and every milestone is reachable from M00; each work package
   belongs to the milestone encoded in its id.
4. `qa-id-definitions` / `qa-id-references` — every QA id is defined once in
   09_QA_ACCEPTANCE.md, and every reference (including `A..B`, `A/0x/0y`
   shorthands) resolves.
5. `obsolete-parallel-plans` — no `MASTER_PLAN.md`, `docs/v2`, `docs/v3` or
   `PLAN.md`, and no non-prohibition reference to them.
6. `constant-contracts` — wave length, tick boundaries, fighter/guard/builder
   coefficients, militia falloff, boss divisor, lane split, wall thresholds,
   resource cap, role-change cooldown, `ttg-rules-v1` and the toolchain pins are
   present in their owner modules, and retired values (for example a 14 fighter
   coefficient) are absent.
7. `evidence-load` / `evidence-schema` — each `docs/evidence/Mxx-Wxxxx.md`
   carries a parseable `<!-- evidence ... -->` block whose milestone, work
   package, status, commit, head, tests and reviewer fields are valid.
8. `status-evidence` — a milestone marked PASS has a PASS evidence record for
   every work package, the narrative "Current truth" line agrees with the table,
   and M10/M11 cannot report PASS without human evidence.
9. `traceability` — `docs/traceability.csv` has the canonical 10 columns,
   resolvable anchors, defined work packages and QA ids, existing evidence
   paths, and maps **every** defined QA id.
10. `index-coverage` — `00_INDEX.md` still references every module.

## What the reference evaluator proves

`refeval` is a **second, independent** implementation of the equations in
02_GAME_RULES.md and 03_SIMULATION.md. It deliberately is not the production
reducer: M02 must re-implement and match it, never import it.

- 23 curated cases with boundary-by-boundary traces,
- a 1080-run population/role/difficulty sweep with a stable digest,
- hand-derived assertions for enemy HP, boss HP, attack, guard block, militia,
  repair, construction work, upgrades, votes, XP, defeat recovery, anchor
  boundaries and role-change cooldown.

`refeval/testdata/golden_vectors.json` is the artifact M02 consumes. After an
approved `rules_version` change, regenerate it with `go run ./refeval -write`
and review the diff in the same commit.

## CI wiring

`.github/workflows/ci.yml` runs, on `ubuntu-24.04`:

- backend: `gofmt` check, `go vet`, `go test`, `go test -race`, PostgreSQL 18.6
  integration tests and a Linux build;
- tooling: `go vet`, `go test`, `go run ./planvalidate -root ..`,
  `go run ./refeval -check` and `bash -n` over every shell entry point.

A red run must block the merge to `main`; the validator is intentionally strict
so that a status claim cannot outrun its evidence.

Licensed Unity execution lives in a separate manual workflow,
`.github/workflows/unity-editor-tests.yml`. It fails closed until the repository
owner registers the `[self-hosted, Windows, X64, ttgqldl-unity]` runner with a
Unity 6000.6.7f1 licence held in the protected `unity-licensed` environment and
sets the repository variable `TTG_UNITY_RUNNER_ENABLED=true`. The milestone
verdict is recorded in `docs/evidence/M01-ACCEPTANCE.md`.

Unity EditMode/PlayMode execution and the Windows player build are **not** part of
the CI pipeline: no licensed Unity runner is available and no licence secret may
be committed. The `unity` job runs the licence-free validator only, and the
editor-dependent gate stays BLOCKED in CI while it passes locally through the two
`unity-*` scripts.

The backend job starts PostgreSQL 18.6 from the pinned digest with
`docker/postgres/initdb` mounted, so CI exercises the same least-privilege role
model as the developer: migrations run as `ttg_owner`, runtime and permission
tests run as `ttg_app`, and `sqlc diff` plus the schema fingerprint fail the job
on generated-code or schema drift.
