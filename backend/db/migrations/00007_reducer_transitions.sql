-- +goose Up
-- W0403 durable transition journal: one immutable row per committed reducer revision.
-- Checkpoints and replay use this to prove operation order and revision continuity;
-- it is not a second gameplay engine and stores no private input payload.
CREATE TABLE ttg.reducer_transitions (
    session_id UUID NOT NULL REFERENCES ttg.sessions (id) ON DELETE RESTRICT,
    revision BIGINT NOT NULL,
    operation TEXT NOT NULL,
    wave_no BIGINT NOT NULL,
    wave_tick INT NOT NULL,
    state_hash CHAR(64) NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    CONSTRAINT reducer_transitions_pkey PRIMARY KEY (session_id, revision),
    CONSTRAINT reducer_transitions_revision_check CHECK (revision > 0),
    CONSTRAINT reducer_transitions_operation_check CHECK (operation IN ('start', 'advance', 'rollover'))
);

-- +goose Down
DROP TABLE IF EXISTS ttg.reducer_transitions;
