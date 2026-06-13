-- name: GetLatestHeightEntry :one
SELECT * FROM height_entries
WHERE user_id = $1
ORDER BY recorded_at DESC
LIMIT 1;

-- name: UpsertHeightEntryByGoogleID :one
INSERT INTO height_entries (user_id, height_cm, recorded_at, source, google_data_point_id)
VALUES ($1, $2, $3, 'google', $4)
ON CONFLICT (user_id, google_data_point_id) WHERE google_data_point_id IS NOT NULL
DO UPDATE SET height_cm = excluded.height_cm, recorded_at = excluded.recorded_at
RETURNING *;

-- name: UpsertHeightEntryByRecordedAt :one
INSERT INTO height_entries (user_id, height_cm, recorded_at, source, google_data_point_id)
VALUES ($1, $2, $3, 'google', NULL)
ON CONFLICT (user_id, recorded_at) WHERE source = 'google' AND google_data_point_id IS NULL
DO UPDATE SET height_cm = excluded.height_cm
RETURNING *;
