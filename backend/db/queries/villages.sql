-- Bootstrap village queries. Consumers: M05 (admin village create/get/list) and
-- M07 (operator views). Column order follows docs/plan/04_DATABASE.md.

-- name: CreateVillage :one
INSERT INTO ttg.villages (id, broadcaster_key)
VALUES ($1, $2)
RETURNING *;

-- name: GetVillage :one
SELECT * FROM ttg.villages WHERE id = $1;

-- name: GetVillageByBroadcasterKey :one
SELECT * FROM ttg.villages WHERE broadcaster_key = $1;

-- name: ListVillages :many
SELECT * FROM ttg.villages
ORDER BY created_at DESC, id
LIMIT $1 OFFSET $2;

-- name: TouchVillage :exec
UPDATE ttg.villages SET updated_at = now() WHERE id = $1;
