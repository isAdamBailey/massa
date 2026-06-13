-- name: CreateWeightEntry :one
INSERT INTO weight_entries (user_id, weight_kg, recorded_at, bmi, height_used_cm, source)
VALUES ($1, $2, $3, $4, $5, 'manual')
RETURNING *;

-- name: ListWeightEntries :many
SELECT * FROM weight_entries
WHERE user_id = $1
  AND (sqlc.narg('from')::timestamptz IS NULL OR recorded_at >= sqlc.narg('from'))
  AND (sqlc.narg('to')::timestamptz IS NULL OR recorded_at <= sqlc.narg('to'))
ORDER BY recorded_at ASC;

-- name: GetWeightEntryByID :one
SELECT * FROM weight_entries WHERE id = $1 AND user_id = $2;

-- name: GetLatestWeightEntry :one
SELECT * FROM weight_entries
WHERE user_id = $1
ORDER BY recorded_at DESC
LIMIT 1;

-- name: UpdateWeightEntry :one
UPDATE weight_entries
SET weight_kg = $3, recorded_at = $4, bmi = $5, height_used_cm = $6, updated_at = now()
WHERE id = $1 AND user_id = $2
RETURNING *;

-- name: DeleteWeightEntry :execrows
DELETE FROM weight_entries WHERE id = $1 AND user_id = $2;

-- name: UpsertWeightEntryByGoogleID :one
INSERT INTO weight_entries (user_id, weight_kg, recorded_at, source, google_data_point_id)
VALUES ($1, $2, $3, 'google', $4)
ON CONFLICT (user_id, google_data_point_id) WHERE google_data_point_id IS NOT NULL
DO UPDATE SET weight_kg = excluded.weight_kg, recorded_at = excluded.recorded_at, updated_at = now()
RETURNING *;

-- name: UpsertWeightEntryByRecordedAt :one
INSERT INTO weight_entries (user_id, weight_kg, recorded_at, source, google_data_point_id)
VALUES ($1, $2, $3, 'google', NULL)
ON CONFLICT (user_id, recorded_at) WHERE source = 'google' AND google_data_point_id IS NULL
DO UPDATE SET weight_kg = excluded.weight_kg, updated_at = now()
RETURNING *;
