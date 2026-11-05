-- W0305 durable ingestion queries. All sequence allocation and receipt/input append
-- happens inside one transaction with the session row locked.

-- name: LockSession :one
SELECT * FROM ttg.sessions WHERE id = $1 FOR UPDATE;

-- name: GetInputReceipt :one
SELECT * FROM ttg.input_receipts
WHERE session_id = $1 AND provider_source = $2 AND provider_event_id = $3;

-- name: InsertInputEvent :exec
INSERT INTO ttg.input_events (
    session_id, input_seq, accepted_tick, scheduled_tick, wave_no,
    actor_key, kind, canonical_payload, rules_version
) VALUES (
    $1, $2, $3, $4, $5, $6, $7, $8, $9
);

-- name: InsertInputReceipt :exec
INSERT INTO ttg.input_receipts (
    session_id, provider_source, provider_event_id, payload_fingerprint, input_seq
) VALUES ($1, $2, $3, $4, $5);

-- name: SetSessionInputSequence :exec
UPDATE ttg.sessions SET input_sequence = $2 WHERE id = $1;

-- name: GetInputEvent :one
SELECT * FROM ttg.input_events WHERE session_id = $1 AND input_seq = $2;
