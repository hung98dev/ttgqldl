-- +goose Up
-- Identity continuity across HMAC key rotation, and the transactional export
-- state for the durable privacy-deletion ledger (docs/plan/04_DATABASE.md,
-- "Privacy restore ledger and HMAC rotation").
--
-- The alias table stores only scoped pseudonymous material: actor_key is the
-- HMAC-SHA256 hex digest (64 lowercase hex characters), never a raw provider
-- id; key_version names the secret generation, never the secret itself. The
-- CHECK constraint rejects anything that is not a 64-character hex digest, so a
-- raw provider id cannot be stored even by mistake.

-- players must be addressable by (village_id, id) so that an alias can be
-- constrained to a player in the SAME village by a composite foreign key
-- instead of relying on application validation.
ALTER TABLE ttg.players
    ADD CONSTRAINT players_village_id_id_key UNIQUE (village_id, id);

CREATE TABLE ttg.actor_identity_aliases (
    village_id UUID NOT NULL,
    actor_key CHAR(64) NOT NULL,
    key_version TEXT NOT NULL,
    player_id UUID NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    CONSTRAINT actor_identity_aliases_pkey PRIMARY KEY (village_id, actor_key, key_version),
    CONSTRAINT actor_identity_aliases_actor_key_is_hmac_hex
        CHECK (actor_key ~ '^[0-9a-f]{64}$'),
    CONSTRAINT actor_identity_aliases_key_version_shape
        CHECK (key_version ~ '^[A-Za-z0-9._:-]{1,32}$'),
    CONSTRAINT actor_identity_aliases_village_fk
        FOREIGN KEY (village_id) REFERENCES ttg.villages (id) ON DELETE RESTRICT,
    -- Same-village invariant, enforced relationally. Cascading here is
    -- deliberate: erasing a player must not leave a live pseudonymous mapping
    -- pointing at a deleted subject. Authoritative gameplay history is not
    -- affected, because wave_members and ballots reference actor_key, not the
    -- player row.
    CONSTRAINT actor_identity_aliases_same_village_player_fk
        FOREIGN KEY (village_id, player_id) REFERENCES ttg.players (village_id, id) ON DELETE CASCADE
);

-- Identity resolution walks key_version -> actor_key for a village; the primary
-- key already covers (village_id, actor_key, key_version). A player may hold an
-- alias per retained key version, and erasure/audit removes them by player.
CREATE INDEX actor_identity_aliases_player
    ON ttg.actor_identity_aliases (player_id);

-- Transactional outbox for the independently stored encrypted deletion ledger.
-- A deletion transaction inserts the tombstone and the outbox row atomically;
-- the exporter may only mark a row exported after the external ledger has
-- durably acknowledged it. No raw identifier belongs in `payload`; it carries
-- the protected marker and scope only (M04 enforces that in code).
CREATE TABLE ttg.privacy_ledger_outbox (
    id UUID NOT NULL,
    seq BIGINT GENERATED ALWAYS AS IDENTITY,
    tombstone_id UUID NOT NULL,
    payload JSONB NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    exported_at TIMESTAMPTZ,
    ledger_ack_ref TEXT,
    attempts INTEGER NOT NULL DEFAULT 0,
    last_error TEXT,
    CONSTRAINT privacy_ledger_outbox_pkey PRIMARY KEY (id),
    CONSTRAINT privacy_ledger_outbox_seq_key UNIQUE (seq),
    CONSTRAINT privacy_ledger_outbox_tombstone_fk
        FOREIGN KEY (tombstone_id) REFERENCES ttg.privacy_tombstones (id) ON DELETE RESTRICT,
    -- An ack reference exists if and only if the export was acknowledged.
    CONSTRAINT privacy_ledger_outbox_ack_consistency
        CHECK ((exported_at IS NULL) = (ledger_ack_ref IS NULL)),
    CONSTRAINT privacy_ledger_outbox_payload_is_object
        CHECK (jsonb_typeof(payload) = 'object'),
    CONSTRAINT privacy_ledger_outbox_attempts_non_negative
        CHECK (attempts >= 0),
    CONSTRAINT privacy_ledger_outbox_ack_ref_length
        CHECK (ledger_ack_ref IS NULL OR length(ledger_ack_ref) BETWEEN 1 AND 128)
);

-- Exporter scan: oldest unexported event first.
CREATE INDEX privacy_ledger_outbox_pending
    ON ttg.privacy_ledger_outbox (seq)
    WHERE exported_at IS NULL;

CREATE INDEX privacy_ledger_outbox_tombstone
    ON ttg.privacy_ledger_outbox (tombstone_id);

-- +goose Down
-- Rollback safety: identity mappings and unacknowledged deletions are NOT
-- disposable state. This migration refuses to remove either automatically; an
-- operator must first export or resolve them deliberately. The guard makes a
-- production rollback fail closed instead of silently destroying continuity or
-- erasure evidence. On an unused development database both counts are zero and
-- the rollback proceeds.
-- +goose StatementBegin
DO $$
DECLARE
    alias_count BIGINT;
    pending_count BIGINT;
BEGIN
    SELECT count(*) INTO alias_count FROM ttg.actor_identity_aliases;
    IF alias_count > 0 THEN
        RAISE EXCEPTION
            'refusing to drop ttg.actor_identity_aliases: % identity mapping(s) present; export or erase them deliberately before rolling back',
            alias_count;
    END IF;
    SELECT count(*) INTO pending_count
        FROM ttg.privacy_ledger_outbox WHERE exported_at IS NULL;
    IF pending_count > 0 THEN
        RAISE EXCEPTION
            'refusing to drop ttg.privacy_ledger_outbox: % unexported deletion event(s) present; export them to the encrypted ledger first',
            pending_count;
    END IF;
END
$$;
-- +goose StatementEnd

DROP TABLE ttg.privacy_ledger_outbox;
DROP TABLE ttg.actor_identity_aliases;

-- Restore the pre-00004 shape of players. The redundant unique index is removed
-- last so no foreign key depends on it.
ALTER TABLE ttg.players DROP CONSTRAINT players_village_id_id_key;
