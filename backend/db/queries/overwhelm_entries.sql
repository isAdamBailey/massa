-- name: ListOverwhelmEntries :many
SELECT
    e.id, e.user_id, e.day, e.overwhelm_level, e.created_at, e.updated_at,
    COALESCE((SELECT array_agg(t.name ORDER BY t.name) FROM overwhelm_entry_tags et
              JOIN overwhelm_tags t ON t.id = et.tag_id WHERE et.entry_id = e.id), '{}')::text[] AS tag_names,
    COALESCE((SELECT array_agg(t.id::text ORDER BY t.name) FROM overwhelm_entry_tags et
              JOIN overwhelm_tags t ON t.id = et.tag_id WHERE et.entry_id = e.id), '{}')::text[] AS tag_ids
FROM overwhelm_entries e
WHERE e.user_id = $1
  AND (sqlc.narg('from')::date IS NULL OR e.day >= sqlc.narg('from'))
  AND (sqlc.narg('to')::date IS NULL OR e.day <= sqlc.narg('to'))
ORDER BY e.day ASC;

-- name: UpsertOverwhelmByDay :one
WITH entry AS (
    INSERT INTO overwhelm_entries (user_id, day, overwhelm_level)
    VALUES ($1, $2, $3)
    ON CONFLICT (user_id, day)
    DO UPDATE SET overwhelm_level = excluded.overwhelm_level, updated_at = now()
    RETURNING *
), cleared AS (
    DELETE FROM overwhelm_entry_tags
    WHERE entry_id = (SELECT id FROM entry)
      AND tag_id <> ALL(sqlc.arg(tag_ids)::uuid[])
), inserted AS (
    INSERT INTO overwhelm_entry_tags (entry_id, tag_id)
    SELECT (SELECT id FROM entry), t.id
    FROM overwhelm_tags t
    WHERE t.user_id = $1 AND t.id = ANY(sqlc.arg(tag_ids)::uuid[])
    ON CONFLICT DO NOTHING
)
SELECT
    entry.id, entry.user_id, entry.day, entry.overwhelm_level, entry.created_at, entry.updated_at,
    COALESCE((SELECT array_agg(t.name ORDER BY t.name) FROM overwhelm_entry_tags et
              JOIN overwhelm_tags t ON t.id = et.tag_id WHERE et.entry_id = entry.id), '{}')::text[] AS tag_names,
    COALESCE((SELECT array_agg(t.id::text ORDER BY t.name) FROM overwhelm_entry_tags et
              JOIN overwhelm_tags t ON t.id = et.tag_id WHERE et.entry_id = entry.id), '{}')::text[] AS tag_ids
FROM entry;
