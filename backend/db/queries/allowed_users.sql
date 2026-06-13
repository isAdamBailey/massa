-- name: IsEmailAllowed :one
SELECT EXISTS (
    SELECT 1 FROM allowed_users WHERE email = $1
) AS allowed;

-- name: ListAllowedEmails :many
SELECT email FROM allowed_users ORDER BY email;

-- name: UpsertAllowedUser :exec
INSERT INTO allowed_users (email) VALUES ($1)
ON CONFLICT (email) DO NOTHING;

-- name: DeleteAllowedUserByEmail :exec
DELETE FROM allowed_users WHERE email = $1;
