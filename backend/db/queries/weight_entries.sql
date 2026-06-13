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
