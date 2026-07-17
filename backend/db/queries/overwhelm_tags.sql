-- name: ListOverwhelmTags :many
SELECT * FROM overwhelm_tags
WHERE user_id = $1 AND archived_at IS NULL
ORDER BY name ASC;

-- name: CreateOrUnarchiveOverwhelmTag :one
INSERT INTO overwhelm_tags (user_id, name)
VALUES ($1, $2)
ON CONFLICT (user_id, lower(name))
DO UPDATE SET name = excluded.name, archived_at = NULL, updated_at = now()
RETURNING *;

-- name: RenameOverwhelmTag :one
UPDATE overwhelm_tags
SET name = $3, updated_at = now()
WHERE id = $1 AND user_id = $2 AND archived_at IS NULL
RETURNING *;

-- name: ArchiveOverwhelmTag :execrows
UPDATE overwhelm_tags
SET archived_at = now(), updated_at = now()
WHERE id = $1 AND user_id = $2 AND archived_at IS NULL;
