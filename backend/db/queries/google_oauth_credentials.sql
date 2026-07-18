-- name: UpsertGoogleOAuthCredentials :one
INSERT INTO google_oauth_credentials (
    user_id, google_health_user_id,
    refresh_token_encrypted, refresh_token_nonce,
    access_token_encrypted, access_token_nonce, access_token_expires_at,
    sync_enabled
)
VALUES ($1, $2, $3, $4, $5, $6, $7, true)
ON CONFLICT (user_id) DO UPDATE SET
    google_health_user_id = excluded.google_health_user_id,
    refresh_token_encrypted = excluded.refresh_token_encrypted,
    refresh_token_nonce = excluded.refresh_token_nonce,
    access_token_encrypted = excluded.access_token_encrypted,
    access_token_nonce = excluded.access_token_nonce,
    access_token_expires_at = excluded.access_token_expires_at,
    sync_enabled = true,
    updated_at = now()
RETURNING *;

-- name: GetGoogleOAuthCredentialsByUserID :one
SELECT * FROM google_oauth_credentials WHERE user_id = $1;

-- name: DeleteGoogleOAuthCredentials :exec
DELETE FROM google_oauth_credentials WHERE user_id = $1;

-- name: UpdateGoogleSyncEnabled :exec
UPDATE google_oauth_credentials
SET sync_enabled = $2, updated_at = now()
WHERE user_id = $1;

-- name: UpdateGoogleOAuthTokens :exec
UPDATE google_oauth_credentials
SET
    refresh_token_encrypted = $2,
    refresh_token_nonce = $3,
    access_token_encrypted = $4,
    access_token_nonce = $5,
    access_token_expires_at = $6,
    updated_at = now()
WHERE user_id = $1;
