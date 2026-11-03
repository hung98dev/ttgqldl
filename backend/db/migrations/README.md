# backend/db/migrations

Forward-only goose migrations belong here.

**Status: intentionally empty in M01 Phase A.** W0103 owns the complete schema
from `docs/plan/04_DATABASE.md` (villages, sessions, players, wave_members,
ballots, input_events, checkpoints, outbox, operator tables and privacy tables),
including every CHECK, FK, partial index and least-privilege grant.

Nothing in this directory may be invented piecemeal: the first migration must
match the owned schema and be forward-tested against PostgreSQL 18.6.
