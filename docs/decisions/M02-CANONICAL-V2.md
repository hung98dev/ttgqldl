# ADR M02-CANONICAL-V2 — Complete persisted session XP in the canonical state hash

Status: **PENDING OWNER APPROVAL** (not approved, not implemented)
Owners: repository owner (`hung98dev`)
Scope: M02 canonical serialization contract; gameplay rules unchanged (`ttg-rules-v1`)

## 1. Problem

Canonical schema v1 hashes `session_xp` only inside the current-wave `members`
collection. Session XP persists across waves in the separate XP collection, so an
actor who is not a current-wave member can have that persisted value altered
without changing the canonical v1 state hash. The gap is reproduced by
`backend/internal/canonical.TestPersistedXPOutsideCurrentMembersIsNotHashed`,
which shows two states with identical v1 hashes and different retained XP.

This matters because QA-DB-05 / QA-OPS-03 rely on the canonical hash to detect
lost or duplicated XP across checkpoints and replay. The private checkpoint
envelope (version 2) already covers all persisted XP, but that is a separate
checksum, not the canonical game-state hash.

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

- Private checkpoint envelope version 2 already stores and digests all persisted
  XP; it is independent of the canonical schema version.
- After approval, canonical v2 must be serialized in checkpoints with an explicit
  schema-version tag; restoring a v1 checkpoint under v2 must fail closed rather
  than migrate silently.
- Replay comparison switches to v2 hashes; existing v1 replay traces remain
  historical artifacts.

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

## 10. Owner decision

```text
Decision status: PENDING OWNER APPROVAL

Option A: Approve canonical schema v2 with complete session_progress.
Option B: Reject/defer; provide the required alternative that restores full XP hash coverage.

No option is selected.
```

No schema version constant, fixture or hash was changed by this record.
