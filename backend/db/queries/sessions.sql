-- Bootstrap session queries. Consumers: M04 (writer lease / reducer bootstrap)
-- and M05 (admin session start/list). The partial unique index guarantees at most
-- one non-stopped session per village, so GetActiveSessionForVillage is singular.

-- name: CreateSession :one
INSERT INTO ttg.sessions (id, village_id, mode, lifecycle, rules_version)
VALUES ($1, $2, $3, $4, $5)
RETURNING *;

-- name: GetSession :one
SELECT * FROM ttg.sessions WHERE id = $1;

-- name: GetActiveSessionForVillage :one
SELECT * FROM ttg.sessions
WHERE village_id = $1 AND lifecycle <> 'stopped';

-- name: ListSessions :many
SELECT * FROM ttg.sessions
ORDER BY COALESCE(started_at, 'epoch'::timestamptz) DESC, id
LIMIT $1 OFFSET $2;

-- name: UpdateSessionLifecycle :exec
UPDATE ttg.sessions
SET lifecycle = $3,
    started_at = CASE WHEN $3 = 'running' AND started_at IS NULL THEN now() ELSE started_at END,
    stopped_at = CASE WHEN $3 = 'stopped' THEN now() ELSE stopped_at END
WHERE id = $1 AND village_id = $2;
