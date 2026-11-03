-- +goose Up
-- Durable input log, reducer checkpoints and the WebSocket outbox
-- (docs/plan/04_DATABASE.md). `input_seq` is monotonic per session but is NOT a
-- single global processing cursor: pending next-wave events may carry a lower
-- input_seq than a later current-wave event, so `applied_at_tick` is tracked per
-- event and the pending index is partial.

CREATE TABLE ttg.input_receipts (
    session_id UUID NOT NULL REFERENCES ttg.sessions (id) ON DELETE RESTRICT,
    provider_source TEXT NOT NULL,
    provider_event_id TEXT NOT NULL,
    inserted_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    CONSTRAINT input_receipts_unique_event UNIQUE (session_id, provider_source, provider_event_id)
);

-- Retention/TTL cleanup scan.
CREATE INDEX input_receipts_inserted_at ON ttg.input_receipts (inserted_at);

CREATE TABLE ttg.input_events (
    session_id UUID NOT NULL REFERENCES ttg.sessions (id) ON DELETE RESTRICT,
    input_seq BIGINT NOT NULL,
    accepted_tick INT NOT NULL,
    scheduled_tick INT NOT NULL,
    applied_at_tick INT,
    wave_no BIGINT NOT NULL CHECK (wave_no >= 1),
    actor_key CHAR(64) NOT NULL,
    kind TEXT NOT NULL CHECK (kind IN ('role', 'vote')),
    canonical_payload JSONB NOT NULL,
    rules_version TEXT NOT NULL,
    inserted_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    PRIMARY KEY (session_id, input_seq)
);

CREATE INDEX input_events_pending
    ON ttg.input_events (session_id, wave_no, scheduled_tick, input_seq)
    WHERE applied_at_tick IS NULL;

CREATE TABLE ttg.checkpoints (
    session_id UUID NOT NULL REFERENCES ttg.sessions (id) ON DELETE RESTRICT,
    revision BIGINT NOT NULL,
    wave_no BIGINT NOT NULL,
    wave_tick INT NOT NULL,
    input_seq BIGINT NOT NULL,
    state_json JSONB NOT NULL,
    state_hash CHAR(64) NOT NULL,
    rules_version TEXT NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    -- The primary key is a btree on (session_id, revision); a backward scan of
    -- that key is the "newest checkpoint per session" lookup, so no second
    -- index is created.
    PRIMARY KEY (session_id, revision)
);

CREATE TABLE ttg.domain_events (
    session_id UUID NOT NULL REFERENCES ttg.sessions (id) ON DELETE RESTRICT,
    revision BIGINT NOT NULL,
    event_index INT NOT NULL,
    event_type TEXT NOT NULL,
    payload JSONB NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    PRIMARY KEY (session_id, revision, event_index)
);

CREATE TABLE ttg.stream_outbox (
    session_id UUID NOT NULL REFERENCES ttg.sessions (id) ON DELETE RESTRICT,
    stream_seq BIGINT NOT NULL,
    revision BIGINT NOT NULL,
    type TEXT NOT NULL,
    payload JSONB NOT NULL,
    checkpoint_ref BIGINT,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    PRIMARY KEY (session_id, stream_seq)
);

-- +goose Down
DROP TABLE IF EXISTS ttg.stream_outbox;
DROP TABLE IF EXISTS ttg.domain_events;
DROP TABLE IF EXISTS ttg.checkpoints;
DROP TABLE IF EXISTS ttg.input_events;
DROP TABLE IF EXISTS ttg.input_receipts;
