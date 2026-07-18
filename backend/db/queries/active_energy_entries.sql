-- name: ListActiveEnergyEntries :many
SELECT * FROM active_energy_entries
WHERE user_id = $1
  AND (sqlc.narg('from')::date IS NULL OR day >= sqlc.narg('from'))
  AND (sqlc.narg('to')::date IS NULL OR day <= sqlc.narg('to'))
ORDER BY day ASC;

-- name: ExistsActiveEnergyForDate :one
SELECT EXISTS (
    SELECT 1 FROM active_energy_entries
    WHERE user_id = $1
      AND day = $2
) AS exists;

-- name: UpsertActiveEnergyByDay :one
INSERT INTO active_energy_entries (user_id, day, active_energy_kcal, source)
VALUES ($1, $2, $3, 'google')
ON CONFLICT (user_id, day)
DO UPDATE SET active_energy_kcal = excluded.active_energy_kcal, updated_at = now()
RETURNING *;
