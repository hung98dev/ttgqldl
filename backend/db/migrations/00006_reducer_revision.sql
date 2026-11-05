-- +goose Up
-- W0402 durable reducer revision. This is a monotonic transition counter distinct
-- from input_sequence, writer_epoch, wave_tick and stream_sequence; it survives
-- wave rollover and is the revision key used by domain_events/stream_outbox.
ALTER TABLE ttg.sessions
    ADD COLUMN reducer_revision BIGINT NOT NULL DEFAULT 0;

ALTER TABLE ttg.sessions
    ADD CONSTRAINT sessions_reducer_revision_check CHECK (reducer_revision >= 0);

-- +goose Down
ALTER TABLE ttg.sessions DROP CONSTRAINT sessions_reducer_revision_check;
ALTER TABLE ttg.sessions DROP COLUMN reducer_revision;
