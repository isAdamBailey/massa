-- name: GetUserByEmail :one
SELECT * FROM users WHERE email = $1;

-- name: GetUserByID :one
SELECT * FROM users WHERE id = $1;

-- name: CreateUser :one
INSERT INTO users (email)
VALUES ($1)
RETURNING *;

-- name: UpdateLastLoginAt :exec
UPDATE users SET last_login_at = now() WHERE id = $1;

-- name: UpdateUserSettings :one
UPDATE users
SET manual_height_cm = $2, units_preference = $3
WHERE id = $1
RETURNING *;
