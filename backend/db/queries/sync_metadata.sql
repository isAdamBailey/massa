-- name: UpsertSyncMetadata :one
INSERT INTO sync_metadata (user_id)
VALUES ($1)
ON CONFLICT (user_id) DO UPDATE SET user_id = excluded.user_id
RETURNING *;

-- name: GetSyncMetadataByUserID :one
SELECT * FROM sync_metadata WHERE user_id = $1;

-- name: UpdateSyncWatermarks :exec
UPDATE sync_metadata SET
    last_full_backfill_at = $2,
    last_incremental_sync_at = $3,
    weight_sync_watermark = $4,
    height_sync_watermark = $5,
    updated_at = now()
WHERE user_id = $1;
