# ADR M02-CANONICAL-V2 — Complete persisted session XP in the canonical state hash

Status: **APPROVED (A1 + V-ACCEPT) — delegated owner decision, implemented in TASK 031**
Owners: repository owner (`hung98dev`)
Scope: M02 canonical serialization contract; gameplay rules unchanged (`ttg-rules-v1`)

## 0. Version identifiers (do not conflate)

| Identifier | Current value | Meaning |
|---|---|---|
| Gameplay rules version | `ttg-rules-v1` | Every number, formula, ordering and bound in 02_GAME_RULES.md. A change requires a rules-version bump. Unchanged by this ADR. |
| Canonical schema version | `2` | The canonical game-state JSON projection and its SHA-256 hash (04_DATABASE.md). Version 1 remains verifiable through the explicit compatibility path. |
| Private checkpoint envelope version | `4` | Private replay/restoration integrity payload (canonical hash + complete authoritative SessionState + pending queues). Version 4 supersedes the incremental v1/v2/v3 exception lists; it is a private format, not the canonical schema, and it did not change with canonical v2. |
| Reference fixture schema version | `2` | Shape of the independently generated golden fixtures (`tooling/refeval`). A shape change bumps it and fails the production differential test until reviewed. The v2 metadata update did not change the fixture shape. |

These four numbers move independently. The private envelope version is **not** the
canonical schema version, and the active canonical schema is 2 while the private
checkpoint envelope stays 4.

## 0a. Decision record (TASK 031)

The project requester explicitly delegated the architectural choice to the
implementing assistant and then issued the implementation instruction:

> "tự đề xuất và chọn giúp tôi phương án tốt nhất và cho tôi prompt đi"
> (delegation quoted by the requester in the TASK 031 instruction)

Under that delegation the requester selected:

```text
Selected canonical scope: A1 (targeted schema v2, complete persisted XP)
Selected versioning policy: V-ACCEPT (four-axis separation)
Expanded scope A2: NOT AUTHORIZED
Gameplay rules: ttg-rules-v1 (unchanged)
New canonical schema: 2
Private checkpoint envelope: 4 (unchanged)
Reference fixture schema: 2 (unchanged shape)
```

Provenance, recorded honestly: the decision originates from the repository
requester's explicit conversational delegation and implementation instruction, not
from the implementing agent's own choice. There is **no GitHub review approval, no
signed decision artifact, no independently verified owner identity claim and no
authorization timestamp** beyond the conversation itself; none is invented here.
The repository owner of record remains `hung98dev`. Implementation, compatibility
verification and exact-HEAD CI were performed under that instruction.

## 1. Problem

Canonical schema v1 hashed `session_xp` only inside the current-wave `members`
collection. Session XP persists across waves in the separate XP collection, so an
actor who was not a current-wave member could have that persisted value altered
without changing the canonical v1 state hash. The gap was reproduced by
`backend/internal/canonical.TestPersistedXPOutsideCurrentMembersIsNotHashedInV1`,
which shows two v1 states with identical hashes and different retained XP.

This mattered because QA-DB-05 / QA-OPS-03 rely on the canonical hash to detect
lost or duplicated XP across checkpoints and replay. The private checkpoint
envelope (version 4) already covered all persisted XP and every other authoritative
field that canonical v1 omitted, but that is a separate private integrity payload,
not the canonical game-state hash.

**Resolution (schema v2, implemented):** the active canonical projection includes
`session_progress`, so retained XP now changes the canonical hash. The historical
v1 gap is preserved intentionally as a compatibility artifact and can never be
mistaken for active coverage.

## 2. Approved schema v2 (implemented)

```json
{
  "schema_version": 2,
  "rules_version": "ttg-rules-v1",
  "session_progress": [
    {"actor_key": "<64-hex scoped actor key>", "session_xp": 100}
  ]
}
```

Requirements:

- `session_progress` contains **every** persisted per-session XP record, sorted by
  `actor_key` bytes (canonical ascending order).
- Duplicate actor keys are rejected; `session_xp` is an integer in `0..300`.
- `members[].session_xp` remains for current-wave members (compatibility with the
  documented member contract); it must agree with the corresponding
  `session_progress` entry.
- No raw provider IDs, aliases, comments, wall times or cosmetic randomness.
- `rules_version` stays `ttg-rules-v1`; this is a serialization change, not a
  gameplay change.

## 3. Hash compatibility (as implemented)

- v1 hashes remain unchanged and continue to verify as v1 through an explicit
  revision-selected projection path. Two historical fixtures are frozen with their
  exact v1 canonical JSON and SHA-256 in
  `backend/internal/canonical/schema_v2_test.go`:
  base `438cc8924df9e067bd75ef1cb02e96ec5e9a26d6cb6bbf5826fe174a328a2357`,
  rich `b0a5bad26b0548810b003359e774462c3ddddc715e363760ad8bd56e2196b86c`.
- v2 introduces new hash values for every active state; v1 fixtures are **not**
  rewritten and remain historical v1 vectors.
- The schema version is inside the hashed payload, so a v1 and a v2 payload can
  never collide semantically. Unknown schema versions are rejected.
- The state revision, not a caller flag, selects the projection version. Schema-v1
  states are structurally valid and hashable, but the simulation, input admission
  and checkpoint layer reject them as inactive: they are never advanced, queued
  into or restored as live sessions.
- Historical verification therefore does not "pretend" a legacy state is current;
  it validates the state under its recorded revision and hashes the exact v1 field
  set.

## 4. Checkpoints and replay

- The private checkpoint envelope is at **version 4**. It carries the active
  canonical `state_hash` (v2 for new checkpoints, v1 only for historical v1
  artifacts) plus a `private_digest` computed over the complete authoritative
  restoration payload: the entire `SessionState` (revision, session and village
  ids, source mode, lifecycle and pause reasons, village resources and progression,
  wave pools/boss/vote/ballots/defeat/result/boss-incoming, members, pending
  next-wave registrations, persisted XP, all difficulty internals, streaks, privacy
  epoch and input sequence) **and** every accepted but not yet applied role and
  ballot command. This is independent of the canonical schema version, so envelope
  version 4 did not change with canonical v2.
- Version 4 was introduced because the earlier incremental exception list (v1
  queues, v2 persisted XP, v3 pause reasons) still left valid canonical-hash
  collisions on fields that change continuation: `Mode`, `InputSequence`, applied
  ballot identity/sequence, `Difficulty.MilitiaAttack`, `Difficulty.BossMaxHP`,
  `Difficulty.ActualPopulation`, `Difficulty.RecoveryPercent`, `Wave.BossIncoming`,
  `Wave.Result` and `Village.Level`. All ten collisions are reproduced and
  separated in `backend/internal/simulation.TestPrivateStateCollisions` (under
  canonical v2; A1 does not cover those fields).
- Unsigned envelope versions 1, 2 and 3 are rejected, never silently migrated.
- The private digest is an integrity check against corruption and partial edits,
  **not** authentication: an attacker who can rewrite both payload and digest can
  recompute it. A trusted external anchor/signature is out of M02 scope.
- Checkpoints are created and verified only for the active revision. A legacy v1
  checkpoint/state fails closed (`ErrInvalidState`) rather than being migrated or
  advanced; its historical hash remains verifiable through the canonical
  compatibility path.
- Replay comparison uses v2 hashes and complete state equality; existing v1 replay
  traces remain historical artifacts.

## 4a. Canonical omission inventory after A1 (measured, not assumed)

Each row is a pair of independently valid states that share the active **canonical
v2 (A1)** hash and differ in exactly one authoritative field. Canonical equality and
private digest separation are both executed by `TestPrivateStateCollisions`.

| Field | Canonical v2 (A1) | Private digest v4 | Continuation impact |
|---|---|---|---|
| Persisted XP outside current members | **covered** (`session_progress`) | covered | retained-XP integrity (QA-DB-05/QA-OPS-03) |
| `SessionState.InputSequence` | omitted (collision) | covered | accepted-input sequence namespace (M03 ingest dedupe, durable replay ordering) |
| `SessionState.Mode` | omitted (collision) | covered | source admission and audit provenance above the reducer |
| `WaveState.Ballots` identity | omitted (counts/selected only; collision) | covered | same-tick latest-wins re-vote accounting |
| `Difficulty.MilitiaAttack` | omitted (collision) | covered | added to fighter attack at every combat boundary |
| `WaveState.BossIncoming` | omitted (collision) | covered | suppresses duplicate `boss_incoming` events |
| `WaveState.Result` | omitted (collision) | covered | rollover fails closed while pending |
| `VillageState.Level` | omitted (collision) | covered | persistent progression value |
| `Difficulty.ActualPopulation` | omitted; the anchor is a capped proxy (collision above 10,000) | covered | truthful participant count |
| `Difficulty.RecoveryPercent` | omitted (collision) | covered | derived recovery modifier |
| `Difficulty.BossMaxHP` | omitted (collision) | covered | tick-2600 boss spawn hit points |

Retained XP is now covered canonically. Every remaining A2-scope field is
explicitly **not authorized** and stays covered by the private payload; canonical v1
hashes for those fields also remain verifiable as historical artifacts.

## 5. M04 persistence implications

The database already stores `session_player_progress(session_id, actor_key,
session_xp)`, so v2 adds no new storage. M04 must ensure the projection reads a
consistent snapshot of that table when hashing, and that privacy erasure removes
or rekeys rows before the next canonical hash boundary.

## 6. Privacy and erasure

`actor_key` is the scoped private identifier already used in the member list. The
canonical projection is private and never published; public snapshots must not
expose `session_progress`. Erasure removes the actor's progress row and advances
`privacy_epoch`, so the post-erasure hash necessarily differs.

## 7. Rollback

Reverting to v1 means reverting the serializer and the schema constant; v2 states
are then unreadable and must be rejected, not silently coerced. No data migration
is required because the underlying table is unchanged.

## 8. Alternatives

1. Keep v1 and rely on the private checkpoint digest only — rejected: replay
   integrity and QA-DB-05 require canonical coverage of authoritative XP.
2. Hash a single aggregate XP total — rejected: it cannot detect which actor's XP
   was lost or duplicated.
3. Include `session_progress` in the public snapshot — rejected: privacy leak.

## 9. Acceptance tests (executed)

- Every persisted XP record appears exactly once, sorted by `actor_key`.
- Duplicate keys, out-of-range XP and unsorted input fail closed.
- Changing a retired actor's XP changes the v2 hash.
- v1 and v2 payloads cannot be confused; foreign versions are rejected.
- Canonical v1 fixtures still verify under v1.
- Full canonical, checkpoint, replay and M02 regression suite passes; exact-HEAD
  CI green.

## 9a. Private checkpoint layer (version 4, implemented)

The private checkpoint format is **version 4** and covers every authoritative field
plus the pending queues. It complements canonical v2: a checkpoint whose state or
queues were altered — including the retained-XP field that canonical v1 missed —
fails verification before any simulation step runs. The private format did not
change with canonical v2 (its payload and digest are unaffected by the canonical
schema tag), so it stays version 4.

## 9b. Versioning policy (ACCEPTED — V-ACCEPT)

`docs/plan/03_SIMULATION.md` previously said that any change to "tick ordering,
rounding, hashing fields or boundary handling" required a rules-version bump. The
approved policy separates four independent axes:

| Change class | Version axis | Examples |
|---|---|---|
| Gameplay mathematics, event ordering, reducer semantics, boundary handling | gameplay rules version (`ttg-rules-v1` → `ttg-rules-v2`) | fighter coefficient, militia formula, vote priority, tick order |
| Canonical serialization only: which authoritative fields are projected/hashed, encoding rules, schema tag | canonical schema version (`1` → `2`) | adding `session_progress`, hashing applied ballot identities |
| Private checkpoint format or digest payload | private envelope version (`4` → `5`) | changing `privateRestorationPayload`, digest encoding |
| Independent reference fixture shape | fixture schema version (`2` → `3`) | typed input model changes in `tooling/refeval` |

Approved rules:

- A change may touch more than one axis, but each axis is bumped only for its own
  reason; none is a substitute for another.
- A canonical schema bump must never change gameplay. If a newly hashed field is
  itself changing semantics, the gameplay rules version must be bumped as well.
- A canonical field addition changes every hash. v1 hashes remain verifiable as
  historical v1 vectors, and `schema_version` is inside the hashed payload, so v1
  and v2 payloads cannot be confused semantically.
- Every axis bump lands with same-commit updates to the affected binding documents,
  regenerated reference vectors and a reviewed replay/fixture impact statement.
- Historical replay traces keep their original version tag; the active comparison
  uses the active canonical schema.

Status: **ACCEPTED (V-ACCEPT)** under the TASK 031 delegated owner decision recorded
in section 0a. `03_SIMULATION.md` and `04_DATABASE.md` were updated in the same
commit that implemented canonical v2.

## 9c. Canonical v2 scope: A1 versus A2

Canonical v1 projects exactly: `schema_version`, `rules_version`, `session_id`,
`village_id`, `wave_no`, `wave_tick`, `lifecycle`, `defeated`, `wall_tier`,
`wall_hp`, `townhall_hp`, `wood`, `stone`, `food`, `construction_work`,
`total_wins`, `consecutive_wins`, `consecutive_losses`, `anchor_population`,
`difficulty`, `base_hp`, `lane_pools` (nine subpools), `boss_spawned`, `boss_hp`,
`vote{open,closed,selected,counts}`, `members[7 fields incl session_xp]`,
`queued_next_wave`, `privacy_epoch`.

### Option A1 — targeted canonical v2 (complete persisted XP only)

Adds exactly one structure:

```json
{"session_progress": [{"actor_key": "<scoped actor key>", "session_xp": 100}]}
```

All persisted XP rows, sorted by `actor_key`, including actors who are no longer
current-wave members. It deliberately does not add the other canonical omissions;
the private checkpoint envelope v4 remains the complete restoration-integrity
contract. Compatibility cost: one new canonical array, every v2 hash differs from
v1, v1 fixtures stay as historical vectors.

### Option A2 — expanded canonical v2

Adds A1 plus every remaining continuation-affecting authoritative field identified
by the audit:

| Added field | Why it matters | Canonical collision reproduced today |
|---|---|---|
| `source_mode` | source admission / audit provenance | yes |
| `input_sequence` | accepted-input namespace, replay ordering | yes |
| `applied_ballots[{actor_key,option,tick,input_seq}]` | same-tick latest-wins re-vote accounting | yes (counts/selected only today) |
| `wave_result` | settlement state; rollover fails while pending | yes |
| `boss_incoming` | suppresses duplicate `boss_incoming` events | yes |
| `village_level` | persistent progression | yes |
| `difficulty.actual_population` | truthful participant count | yes (capped anchor is only a proxy) |
| `difficulty.recovery_percent` | derived recovery modifier | yes |
| `difficulty.militia_attack` | added to combat attack | yes |
| `difficulty.boss_max_hp` | tick-2600 boss pool size | yes |
| `difficulty.locked` | population-lock state | implied by anchor today |

Optional, only if the owner explicitly includes it: `pause_reasons[]` (the
independent operator/source/storage blocker set; already covered by private v4).

Trade-offs. A1 has the smallest compatibility and review surface and keeps the
canonical hash an XP-integrity contract, while private v4 carries the rest. A2 makes
the canonical hash a near-complete authoritative-state contract, which reduces
reliance on the private digest for replay integrity, but it changes every hash, adds
ordering/determinism obligations for ballot and difficulty projections, and expands
the privacy review of the canonical projection (private data, never published).

**Neither option is implemented.** Option B (reject/defer and supply an alternative)
remains available for both scopes.

## 9d. Final recommendation and evaluation (task 030; A1 selected in task 031)

The recommendation below is engineering advice, **not approval**. It weighs the two
canonical scopes against the documented acceptance criteria and the evidence
actually produced by W0208.

| Criterion | A1 (targeted) | A2 (expanded) |
|---|---|---|
| QA-DET-01 (same inputs → same hash at ≥100 restores) | satisfied | satisfied; no additional benefit for M02, whose replay proof already compares complete states |
| QA-DET-02 (no RNG, map iteration or timestamps) | neutral | neutral, but adds ordered arrays (applied ballots) whose canonical ordering must be maintained forever |
| QA-DB-05 and QA-OPS-03 (retained XP integrity) | **closes the documented gap** | closes it too |
| Canonical-state hash completeness | XP-complete only, by design | near-complete for continuation-affecting fields |
| Replay integrity | private envelope v4 remains the full restoration contract | canonical hash becomes a second, largely duplicative detector |
| Persistence compatibility | serializer only, no schema/data migration | serializer only, but every hash changes with a wider review surface |
| Public/private separation | canonical projection stays private | unchanged, but a wider private projection to review |
| Hash/snapshot performance | one added array | larger payload and more per-hash work (ballots, difficulty internals) |
| Migration and rollback | small, bounded | larger; more v1→v2 fixture churn and review |

**Recommendation: A1.**

1. The only documented *canonical* integrity gap is persisted XP outside current
   membership (QA-DB-05/QA-OPS-03). A1 closes exactly that with the smallest
   compatibility surface.
2. Private checkpoint envelope v4 already covers every authoritative field the audit
   identified and is **required** for restoration; A2's extra canonical fields would
   duplicate that guarantee, not create it.
3. A2 changes every hash and adds permanent ordering obligations without a
   demonstrated M02 or accepted-plan requirement. It remains a valid future upgrade
   if M04 decides the canonical hash must be self-sufficient.

**If the owner prefers A2**, the frozen field list is the 9c table
(`source_mode`, `input_sequence`, applied ballots, `wave_result`, `boss_incoming`,
`village_level`, `difficulty.actual_population`, `difficulty.recovery_percent`,
`difficulty.militia_attack`, `difficulty.boss_max_hp`, `difficulty.locked`) plus
A1's `session_progress`. Explicit decision for the optional field:
**`pause_reasons` = NO** — `lifecycle` is already canonical, the blocker set is
transient operational state, and private v4 already covers it.

**Versioning policy: ACCEPTED (V-ACCEPT).** Keeping `ttg-rules-v1` while introducing
canonical schema v2 is correct because gameplay mathematics, event ordering and
reducer semantics do not change; only which authoritative fields the serialization
contract hashes. Without the split, the binding `03_SIMULATION.md` wording would have
demanded a gameplay-rules bump for a purely serialization change, conflating two
contracts that are versioned independently.

Option **B** was not selected.

## 10. Owner decision (recorded)

```text
Decision status: APPROVED — delegated owner decision (TASK 031)
Provenance: explicit project-requester delegation and implementation instruction;
            no signed artifact, GitHub review approval or verified external identity claim.

Scope decision: A1 — targeted canonical schema v2 (complete persisted session_progress).
Versioning policy: V-ACCEPT — four-axis separation accepted.
A2 (expanded scope): NOT AUTHORIZED.
B: not selected.

Gameplay rules: ttg-rules-v1 (unchanged)
Canonical schema: 2 (implemented)
Private checkpoint envelope: 4 (unchanged)
Reference fixture schema: 2 (unchanged shape; version metadata updated to 2)
```

Implementing commit and exact-HEAD CI are recorded in
`docs/evidence/M02-W0208.md`. No schema constant, fixture or hash is changed by this
ADR alone; the change landed in the TASK 031 code commit with v1 compatibility tests
and exact-HEAD CI verification. Historical v1 artifacts remain verifiable and were
not rewritten.

