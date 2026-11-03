# tooling/ — M00 baseline validators

Two dependency-free Go programs plus thin wrappers. Neither imports production
code and neither writes to the repository. They are the executable part of the
M00 baseline and the intended input to the M01 CI pipeline.

```
tooling/
  go.mod                  module ttgqldl/tooling (stdlib only)
  planvalidate/           canonical-plan consistency checker
  refeval/                independent gameplay reference evaluator + golden vectors
  validate-plan.ps1|.sh   wrapper: run the plan validator
  reference-eval.ps1|.sh  wrapper: verify the golden vectors
```

## Requirements

- Go 1.27.x. The module declares `go 1.27.0`, so it builds on any 1.27 patch.
  The product baseline pins **Go 1.27.3**; M01's `tooling/doctor` must enforce
  the exact patch. Running these tools on an older patch does not weaken the pin.
- No database, no network, no Unity, no Node.

## Commands

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

PowerShell / Bash equivalents:

```powershell
./validate-plan.ps1
./reference-eval.ps1
go test ./...
```

```bash
./validate-plan.sh
./reference-eval.sh
go test ./...
```

## Exit codes

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

## CI wiring (M01)

```yaml
- run: cd tooling && go vet ./... && go test ./...
- run: cd tooling && go run ./planvalidate -root .. -json > plan-report.json
```

A red run must block the merge to `main`; the validator is intentionally strict
so that a status claim cannot outrun its evidence.
