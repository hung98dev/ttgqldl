-- +goose Up
-- W0406 privacy erasure idempotency and anonymized-checkpoint metadata.
ALTER TABLE ttg.privacy_erase_requests
    ADD COLUMN scope TEXT NOT NULL DEFAULT 'session';

CREATE UNIQUE INDEX privacy_erase_requests_idempotency
    ON ttg.privacy_erase_requests (session_id, actor_ref_protected, scope);

ALTER TABLE ttg.checkpoints
    ADD COLUMN privacy_epoch BIGINT NOT NULL DEFAULT 0;

ALTER TABLE ttg.reducer_transitions
    DROP CONSTRAINT reducer_transitions_operation_check;

ALTER TABLE ttg.reducer_transitions
    ADD CONSTRAINT reducer_transitions_operation_check
    CHECK (operation IN ('start', 'advance', 'rollover', 'privacy'));

-- +goose Down
UPDATE ttg.reducer_transitions SET operation = 'advance' WHERE operation = 'privacy';
ALTER TABLE ttg.reducer_transitions
    DROP CONSTRAINT reducer_transitions_operation_check;
ALTER TABLE ttg.reducer_transitions
    ADD CONSTRAINT reducer_transitions_operation_check
    CHECK (operation IN ('start', 'advance', 'rollover'));
ALTER TABLE ttg.checkpoints DROP COLUMN privacy_epoch;
DROP INDEX IF EXISTS ttg.privacy_erase_requests_idempotency;
ALTER TABLE ttg.privacy_erase_requests DROP COLUMN scope;

