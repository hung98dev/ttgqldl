# backend/db/migrations

Canonical, forward-only schema migrations, executed by `cmd/migrate` via goose.

| File | Contents |
|---|---|
| `00001_core.sql` | schema `ttg`, `villages`, `sessions`, `players`, `session_player_progress`, `wave_members`, `ballots` |
| `00002_inputs.sql` | `input_receipts`, `input_events`, `checkpoints`, `domain_events`, `stream_outbox` |
| `00003_operator_privacy.sql` | `operator_accounts`, `operator_sessions`, `operator_audit`, `writer_leases`, `admin_idempotency_keys`, `privacy_tombstones`, `privacy_erase_requests`, `broadcaster_source_bindings` |

The migrations implement the contract in `docs/plan/04_DATABASE.md`: every CHECK,
unique constraint, partial index and FK cascade policy is written there, and
authoritative history is `ON DELETE RESTRICT` so it can only be removed by an
explicit audited purge.

The files are embedded (`embed.go`), so the migrate binary cannot run a different
set of migrations than the one that was reviewed.

## Running

```bash
cd tooling
./migrate-up.ps1        # apply all pending migrations as the owner role
./migrate-down.ps1      # roll back ONE migration
./migrate-status.ps1    # show applied/pending
```

The server connects as the least-privileged `ttg_app` role and cannot run DDL;
migrations use `TTG_MIGRATION_DATABASE_URL` (the `ttg_owner` role). In
development both DSNs live in the gitignored `.env`.

## sqlc

`backend/sqlc.yaml` reads `schema.sql`, which is a **generated** snapshot of the
migrated schema (see the README in `backend/db/queries`). Regenerate it after
changing a migration:

```bash
./migrate-up.ps1
./schema-dump.ps1       # writes backend/db/schema.sql from the live database
```

`sqlc generate` must never be pointed at the goose files directly: they contain
down migrations and goose markers.
