-- One-time cleanup for entries duplicated by a Google Health sync bug that
-- inserted a Google-sourced entry even when an entry already existed for
-- that day. For each user/day with more than one weight_entries row, keep
-- the manual entry if one exists, otherwise keep the row with the latest
-- recorded_at, and delete the rest.
WITH ranked AS (
    SELECT id,
           ROW_NUMBER() OVER (
               PARTITION BY user_id, recorded_at::date
               ORDER BY (source = 'manual') DESC, recorded_at DESC
           ) AS rn
    FROM weight_entries
)
DELETE FROM weight_entries
WHERE id IN (SELECT id FROM ranked WHERE rn > 1);

-- height_entries has no manual source in practice, so this just keeps the
-- latest recorded_at per user/day.
WITH ranked AS (
    SELECT id,
           ROW_NUMBER() OVER (
               PARTITION BY user_id, recorded_at::date
               ORDER BY recorded_at DESC
           ) AS rn
    FROM height_entries
)
DELETE FROM height_entries
WHERE id IN (SELECT id FROM ranked WHERE rn > 1);
