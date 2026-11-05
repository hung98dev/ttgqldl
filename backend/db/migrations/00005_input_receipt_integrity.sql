-- +goose Up
-- W0305 durable ingestion: bind each stable provider-event receipt to the compact
-- payload fingerprint and the immutable input event it produced. Historical rows keep
-- NULL associations explicitly; they remain valid duplicates but cannot prove which
-- input event they acknowledged.
ALTER TABLE ttg.input_receipts
    ADD COLUMN payload_fingerprint CHAR(64),
    ADD COLUMN input_seq BIGINT;

ALTER TABLE ttg.input_receipts
    ADD CONSTRAINT input_receipts_fingerprint_shape
        CHECK (payload_fingerprint IS NULL OR payload_fingerprint ~ '^[0-9a-f]{64}$'),
    ADD CONSTRAINT input_receipts_input_seq_positive
        CHECK (input_seq IS NULL OR input_seq > 0),
    ADD CONSTRAINT input_receipts_input_event_fk
        FOREIGN KEY (session_id, input_seq)
        REFERENCES ttg.input_events (session_id, input_seq)
        ON DELETE RESTRICT;

-- New receipts always carry both values; old rows are the only NULLs. A partial index
-- keeps the "unassociated historical receipt" audit cheap without indexing every row.
CREATE INDEX input_receipts_unassociated
    ON ttg.input_receipts (session_id)
    WHERE input_seq IS NULL;

-- +goose Down
-- Identity/input associations are not disposable state: refuse to drop them while any
-- durable receipt exists. An operator must export or erase deliberately first.
-- +goose StatementBegin
DO $$
DECLARE
    receipt_count BIGINT;
BEGIN
    SELECT count(*) INTO receipt_count FROM ttg.input_receipts;
    IF receipt_count > 0 THEN
        RAISE EXCEPTION
            'refusing to drop W0305 receipt integrity: % receipt(s) present; export or erase them deliberately before rolling back',
            receipt_count;
    END IF;
END
$$;
-- +goose StatementEnd

DROP INDEX ttg.input_receipts_unassociated;
ALTER TABLE ttg.input_receipts
    DROP CONSTRAINT input_receipts_input_event_fk,
    DROP CONSTRAINT input_receipts_input_seq_positive,
    DROP CONSTRAINT input_receipts_fingerprint_shape;
ALTER TABLE ttg.input_receipts
    DROP COLUMN input_seq,
    DROP COLUMN payload_fingerprint;
