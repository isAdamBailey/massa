-- name: ListOverwhelmEntries :many
SELECT * FROM overwhelm_entries
WHERE user_id = $1
  AND (sqlc.narg('from')::date IS NULL OR day >= sqlc.narg('from'))
  AND (sqlc.narg('to')::date IS NULL OR day <= sqlc.narg('to'))
ORDER BY day ASC;

-- name: UpsertOverwhelmByDay :one
INSERT INTO overwhelm_entries (user_id, day, overwhelm_level)
VALUES ($1, $2, $3)
ON CONFLICT (user_id, day)
DO UPDATE SET overwhelm_level = excluded.overwhelm_level, updated_at = now()
RETURNING *;
