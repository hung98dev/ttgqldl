-- +goose Up
-- W0403 audit fix: accepted_tick repeats every wave. Persist the acceptance wave so
-- replay can distinguish wave-1 tick-800 from wave-2 tick-800. Historical rows that
-- cannot prove their acceptance wave remain NULL and replay fails closed.
ALTER TABLE ttg.input_events ADD COLUMN accepted_wave BIGINT;

-- Exact historical visibility boundary for replay: all input_seq values accepted at or
-- before the transition's commit. A later input appended at the same tick must not
-- influence replay to an earlier revision.
ALTER TABLE ttg.reducer_transitions ADD COLUMN input_seq_high BIGINT NOT NULL DEFAULT 0;

CREATE INDEX input_events_replay_boundary ON ttg.input_events (session_id, accepted_wave, accepted_tick, input_seq);

-- +goose Down
DROP INDEX IF EXISTS ttg.input_events_replay_boundary;
ALTER TABLE ttg.reducer_transitions DROP COLUMN input_seq_high;
ALTER TABLE ttg.input_events DROP COLUMN accepted_wave;
