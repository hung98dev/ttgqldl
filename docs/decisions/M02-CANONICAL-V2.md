# ADR M02-CANONICAL-V2 — Complete persisted session XP in the canonical state hash

Status: **PENDING OWNER APPROVAL** (not approved, not implemented)
Owners: repository owner (`hung98dev`)
Scope: M02 canonical serialization contract; gameplay rules unchanged (`ttg-rules-v1`)

## 0. Version identifiers (do not conflate)

| Identifier | Current value | Meaning |
|---|---|---|
| Gameplay rules version | `ttg-rules-v1` | Every number, formula, ordering and bound in 02_GAME_RULES.md. A change requires a rules-version bump. |
| Canonical schema version | `1` | The canonical game-state JSON projection and its SHA-256 hash (04_DATABASE.md). This ADR proposes `2`. |
| Private checkpoint envelope version | `4` | Private replay/restoration integrity payload (canonical hash + complete authoritative SessionState + pending queues). Version 4 supersedes the incremental v1/v2/v3 exception lists; it is a private format, not the canonical schema. |
| Reference fixture schema version | `2` | Shape of the independently generated golden fixtures (`tooling/refeval`). A shape change bumps it and fails the production differential test until reviewed. |

These four numbers move independently. The private envelope version is **not** the
canonical schema version, and a private checkpoint can be version 4 while the
canonical game-state schema is still version 1.

## 1. Problem

Canonical schema v1 hashes `session_xp` only inside the current-wave `members`
collection. Session XP persists across waves in the separate XP collection, so an
actor who is not a current-wave member can have that persisted value altered
without changing the canonical v1 state hash. The gap is reproduced by
`backend/internal/canonical.TestPersistedXPOutsideCurrentMembersIsNotHashed`,
which shows two states with identical v1 hashes and different retained XP.

This matters because QA-DB-05 / QA-OPS-03 rely on the canonical hash to detect
lost or duplicated XP across checkpoints and replay. The private checkpoint
envelope (version 4) covers **all** persisted XP and, since version 4, every other
authoritative field that canonical v1 omits, but that is a separate private
integrity payload, not the canonical game-state hash.

## 2. Proposed schema v2

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

## 3. Hash compatibility

- v1 hashes remain unchanged and continue to verify as v1.
- v2 introduces new hash values for every existing state; v1 fixtures are **not**
  rewritten and remain as historical v1 vectors.
- The schema version is part of the hashed payload, so a v1 and a v2 payload can
  never collide semantically.

## 4. Checkpoints and replay

- The private checkpoint envelope is at **version 4**. It carries the canonical
  v1 `state_hash` unchanged plus a `private_digest` computed over the complete
  authoritative restoration payload: the entire `SessionState` (revision, session
  and village ids, source mode, lifecycle and pause reasons, village resources and
  progression, wave pools/boss/vote/ballots/defeat/result/boss-incoming, members,
  pending next-wave registrations, persisted XP, all difficulty internals, streaks,
  privacy epoch and input sequence) **and** every accepted but not yet applied role
  and ballot command. This is independent of the canonical schema version.
- Version 4 was introduced because the earlier incremental exception list (v1
  queues, v2 persisted XP, v3 pause reasons) still left valid canonical-hash
  collisions on fields that change continuation: `Mode`, `InputSequence`, applied
  ballot identity/sequence, `Difficulty.MilitiaAttack`, `Difficulty.BossMaxHP`,
  `Difficulty.ActualPopulation`, `Difficulty.RecoveryPercent`, `Wave.BossIncoming`,
  `Wave.Result` and `Village.Level`. All ten collisions are reproduced and
  separated in `backend/internal/simulation.TestPrivateStateCollisions`.
- Unsigned envelope versions 1, 2 and 3 are rejected, never silently migrated.
- The private digest is an integrity check against corruption and partial edits,
  **not** authentication: an attacker who can rewrite both payload and digest can
  recompute it. A trusted external anchor/signature is out of M02 scope.
- After approval, canonical v2 must be serialized in checkpoints with an explicit
  schema-version tag; restoring a v1 checkpoint under v2 must fail closed rather
  than migrate silently.
- Replay comparison switches to v2 hashes; existing v1 replay traces remain
  historical artifacts.

## 4a. Canonical v1 omission inventory (measured, not assumed)

Each row is a pair of independently valid states that share the canonical v1 hash
and differ in exactly one authoritative field. Canonical equality and private
digest separation are both executed by `TestPrivateStateCollisions`.

| Field | Canonical v1 | Private digest v4 | Continuation impact |
|---|---|---|---|
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

Every field the task brief asked to inspect is therefore covered by the private
payload, while the fields canonical v1 does hash (`session_id`, `village_id`,
lifecycle, resources, pools, boss existence/HP) remain canonical.

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

## 9. Acceptance tests after approval

- Every persisted XP record appears exactly once, sorted by `actor_key`.
- Duplicate keys, out-of-range XP and unsorted input fail closed.
- Changing a retired actor's XP changes the v2 hash.
- v1 and v2 payloads cannot be confused; foreign versions are rejected.
- Canonical v1 fixtures still verify under v1.
- Full canonical, checkpoint, replay and M02 regression suite passes; exact-HEAD
  CI green.

## 9a. Independent of this decision (already implemented)

The private checkpoint format is **version 4** and already covers every
authoritative field plus the pending queues. It is not a substitute for canonical
v2 (`docs/evidence/M02-W0208.md` records both), but it removes the practical
restore vulnerability while the owner decides: a checkpoint whose state or queues
were altered — including the persisted-XP gap in section 1 — fails verification
before any simulation step runs. `CanonicalSchemaVersion` remains 1.

## 10. Owner decision

```text
Decision status: PENDING OWNER APPROVAL

Option A: Approve canonical schema v2 with complete session_progress.
Option B: Reject/defer; provide the required alternative that restores full XP hash coverage.

No option is selected.
```

No schema version constant, fixture or hash was changed by this record. The owner
has not approved v2, and no task may self-approve it. M02 therefore stays
IN_PROGRESS until the owner selects an option and the selected implementation,
compatibility verification and exact-HEAD CI all succeed.
