-- name: CreateMagicLinkToken :one
INSERT INTO magic_link_tokens (user_email, token_hash, expires_at)
VALUES ($1, $2, $3)
RETURNING *;

-- name: GetValidMagicLinkToken :one
SELECT * FROM magic_link_tokens
WHERE token_hash = $1
  AND used_at IS NULL
  AND expires_at > now();

-- name: MarkMagicLinkTokenUsed :exec
UPDATE magic_link_tokens SET used_at = now() WHERE id = $1;

-- name: CountRecentMagicLinkTokensForEmail :one
SELECT count(*) FROM magic_link_tokens
WHERE user_email = $1
  AND created_at > now() - interval '1 hour';
