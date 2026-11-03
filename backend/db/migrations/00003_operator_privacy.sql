-- +goose Up
-- Operator surface, writer fencing, idempotency and privacy controls
-- (docs/plan/04_DATABASE.md). Secrets are stored only as hashes; the audit table
-- may hold an optional IP hash but never a raw address.

CREATE TABLE ttg.operator_accounts (
    id UUID PRIMARY KEY,
    username TEXT NOT NULL,
    password_hash TEXT NOT NULL,
    role TEXT NOT NULL CHECK (role IN ('admin', 'operator', 'viewer')),
    disabled_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

-- Case-insensitive uniqueness without requiring the citext extension.
CREATE UNIQUE INDEX operator_accounts_username_lower
    ON ttg.operator_accounts (lower(username));

CREATE TABLE ttg.operator_sessions (
    id UUID PRIMARY KEY,
    account_id UUID NOT NULL REFERENCES ttg.operator_accounts (id) ON DELETE RESTRICT,
    secret_hash CHAR(64) NOT NULL,
    expires_at TIMESTAMPTZ NOT NULL,
    revoked_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX operator_sessions_account ON ttg.operator_sessions (account_id);

CREATE TABLE ttg.operator_audit (
    id BIGINT GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    actor_id UUID,
    action TEXT NOT NULL,
    target TEXT,
    reason TEXT,
    ip_hash TEXT,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX operator_audit_created_at ON ttg.operator_audit (created_at);

CREATE TABLE ttg.writer_leases (
    village_id UUID PRIMARY KEY REFERENCES ttg.villages (id) ON DELETE RESTRICT,
    writer_epoch BIGINT NOT NULL DEFAULT 0,
    owner_instance TEXT,
    leased_until TIMESTAMPTZ
);

CREATE TABLE ttg.admin_idempotency_keys (
    account_id UUID NOT NULL,
    route TEXT NOT NULL,
    key TEXT NOT NULL,
    request_hash TEXT NOT NULL,
    response_status INT NOT NULL,
    response_json JSONB NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    expires_at TIMESTAMPTZ NOT NULL,
    PRIMARY KEY (account_id, route, key)
);

CREATE INDEX admin_idempotency_keys_expires_at ON ttg.admin_idempotency_keys (expires_at);

CREATE TABLE ttg.privacy_tombstones (
    id UUID PRIMARY KEY,
    actor_ref_protected TEXT NOT NULL,
    scope TEXT NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    restore_applied_at TIMESTAMPTZ
);

CREATE INDEX privacy_tombstones_actor ON ttg.privacy_tombstones (actor_ref_protected);

CREATE TABLE ttg.privacy_erase_requests (
    id UUID PRIMARY KEY,
    session_id UUID NOT NULL REFERENCES ttg.sessions (id) ON DELETE RESTRICT,
    actor_ref_protected TEXT NOT NULL,
    reason TEXT NOT NULL,
    status TEXT NOT NULL CHECK (status IN ('queued', 'applied', 'blocked', 'failed')),
    requested_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    applied_at TIMESTAMPTZ,
    privacy_epoch BIGINT NOT NULL,
    requested_by UUID
);

CREATE INDEX privacy_erase_requests_session ON ttg.privacy_erase_requests (session_id);

CREATE TABLE ttg.broadcaster_source_bindings (
    village_id UUID NOT NULL REFERENCES ttg.villages (id) ON DELETE RESTRICT,
    provider TEXT NOT NULL,
    broadcaster_key CHAR(64) NOT NULL,
    approval_ref TEXT,
    permission_expires_at TIMESTAMPTZ,
    PRIMARY KEY (village_id, provider)
);

-- +goose Down
DROP TABLE IF EXISTS ttg.broadcaster_source_bindings;
DROP TABLE IF EXISTS ttg.privacy_erase_requests;
DROP TABLE IF EXISTS ttg.privacy_tombstones;
DROP TABLE IF EXISTS ttg.admin_idempotency_keys;
DROP TABLE IF EXISTS ttg.writer_leases;
DROP TABLE IF EXISTS ttg.operator_audit;
DROP TABLE IF EXISTS ttg.operator_sessions;
DROP TABLE IF EXISTS ttg.operator_accounts;
