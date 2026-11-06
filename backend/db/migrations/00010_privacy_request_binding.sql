-- +goose Up
-- W0406-FIX: explicit request->tombstone binding and durable restore-run audit.
ALTER TABLE ttg.privacy_erase_requests
    ADD COLUMN tombstone_id UUID REFERENCES ttg.privacy_tombstones (id) ON DELETE RESTRICT;

CREATE UNIQUE INDEX privacy_erase_requests_tombstone_unique
    ON ttg.privacy_erase_requests (tombstone_id)
    WHERE tombstone_id IS NOT NULL;

CREATE TABLE ttg.privacy_restore_runs (
    id UUID PRIMARY KEY,
    ledger_id TEXT NOT NULL,
    ledger_last_seq BIGINT NOT NULL,
    ledger_head_hash CHAR(64) NOT NULL,
    target_identity TEXT NOT NULL,
    applied_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    result_hash CHAR(64) NOT NULL,
    UNIQUE (ledger_id, ledger_last_seq, target_identity)
);

CREATE TABLE ttg.privacy_restore_erasures (
    run_id UUID NOT NULL REFERENCES ttg.privacy_restore_runs (id) ON DELETE RESTRICT,
    tombstone_id UUID NOT NULL REFERENCES ttg.privacy_tombstones (id) ON DELETE RESTRICT,
    session_id UUID NOT NULL,
    actor_ref_protected TEXT NOT NULL,
    applied_revision BIGINT NOT NULL,
    applied_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    PRIMARY KEY (run_id, tombstone_id, session_id)
);

-- +goose Down
DROP TABLE IF EXISTS ttg.privacy_restore_erasures;
DROP TABLE IF EXISTS ttg.privacy_restore_runs;
DROP INDEX IF EXISTS ttg.privacy_erase_requests_tombstone_unique;
ALTER TABLE ttg.privacy_erase_requests DROP COLUMN tombstone_id;
