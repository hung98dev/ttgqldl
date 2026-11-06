-- +goose Up
-- W0406-FIX2 durable village-scope parent request and child inventory.
CREATE TABLE ttg.privacy_scope_requests (
    id UUID PRIMARY KEY,
    village_id UUID NOT NULL REFERENCES ttg.villages (id) ON DELETE RESTRICT,
    actor_ref_protected TEXT NOT NULL,
    scope TEXT NOT NULL CHECK (scope IN ('village')),
    status TEXT NOT NULL CHECK (status IN ('queued', 'applied', 'blocked', 'failed')),
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    completed_at TIMESTAMPTZ,
    UNIQUE (village_id, actor_ref_protected, scope)
);

ALTER TABLE ttg.privacy_erase_requests
    ADD COLUMN parent_id UUID REFERENCES ttg.privacy_scope_requests (id) ON DELETE RESTRICT;

CREATE INDEX privacy_erase_requests_parent ON ttg.privacy_erase_requests (parent_id);

-- +goose Down
DROP INDEX IF EXISTS ttg.privacy_erase_requests_parent;
ALTER TABLE ttg.privacy_erase_requests DROP COLUMN parent_id;
DROP TABLE IF EXISTS ttg.privacy_scope_requests;
