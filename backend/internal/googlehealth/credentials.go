package googlehealth

import (
	"context"
	"errors"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"github.com/isAdamBailey/massa/backend/internal/db"
)

// ErrNotConnected is returned when a user has not connected a Google Health
// account.
var ErrNotConnected = errors.New("google health account not connected")

// ErrReauthRequired is returned when a user's stored Google credentials are
// no longer valid — the OAuth refresh token has expired or been revoked
// (Google responds with "invalid_grant"). The user must reconnect their
// Google Health account to mint a fresh refresh token.
var ErrReauthRequired = errors.New("google health reauthorization required")

// Credentials holds a user's decrypted Google OAuth tokens and the Google
// Health user ID used to address API requests.
type Credentials struct {
	HealthUserID         string
	RefreshToken         string
	AccessToken          string
	AccessTokenExpiresAt *time.Time
	SyncEnabled          bool
}

// CredentialsRepository stores and retrieves a user's Google OAuth
// credentials, encrypting tokens at rest.
type CredentialsRepository interface {
	// Get returns the stored credentials for userID, or ErrNotConnected if
	// none exist.
	Get(ctx context.Context, userID uuid.UUID) (Credentials, error)
	// Save creates or replaces the stored credentials for userID. Used only
	// for an explicit connect/reconnect, since it always sets sync_enabled
	// to true — use UpdateTokens to persist a refreshed token without
	// disturbing a user's pause setting.
	Save(ctx context.Context, userID uuid.UUID, creds Credentials) error
	// UpdateTokens updates the stored OAuth tokens for userID without
	// touching sync_enabled. Only called for a userID whose credentials
	// were just read via Get, so a missing row is not expected.
	UpdateTokens(ctx context.Context, userID uuid.UUID, creds Credentials) error
	// Delete removes any stored credentials for userID.
	Delete(ctx context.Context, userID uuid.UUID) error
	// SetSyncEnabled pauses or resumes syncing for userID without discarding
	// the stored credentials. Returns ErrNotConnected if none exist.
	SetSyncEnabled(ctx context.Context, userID uuid.UUID, enabled bool) error
}

// PostgresCredentialsRepository implements CredentialsRepository using
// sqlc-generated queries, encrypting tokens with AES-256-GCM.
type PostgresCredentialsRepository struct {
	q   Querier
	key []byte
}

// NewPostgresCredentialsRepository returns a CredentialsRepository backed
// by q, encrypting tokens with key (which must be 32 bytes).
func NewPostgresCredentialsRepository(q Querier, key []byte) *PostgresCredentialsRepository {
	return &PostgresCredentialsRepository{q: q, key: key}
}

// Get implements CredentialsRepository.
func (r *PostgresCredentialsRepository) Get(ctx context.Context, userID uuid.UUID) (Credentials, error) {
	row, err := r.q.GetGoogleOAuthCredentialsByUserID(ctx, db.ToUUID(userID))
	if errors.Is(err, pgx.ErrNoRows) {
		return Credentials{}, ErrNotConnected
	}
	if err != nil {
		return Credentials{}, err
	}

	refreshToken, err := Decrypt(r.key, row.RefreshTokenNonce, row.RefreshTokenEncrypted)
	if err != nil {
		return Credentials{}, err
	}

	var accessToken string
	if row.AccessTokenEncrypted != nil {
		plaintext, err := Decrypt(r.key, row.AccessTokenNonce, row.AccessTokenEncrypted)
		if err != nil {
			return Credentials{}, err
		}
		accessToken = string(plaintext)
	}

	return Credentials{
		HealthUserID:         row.GoogleHealthUserID,
		RefreshToken:         string(refreshToken),
		AccessToken:          accessToken,
		AccessTokenExpiresAt: db.FromTimestamptz(row.AccessTokenExpiresAt),
		SyncEnabled:          row.SyncEnabled,
	}, nil
}

// Save implements CredentialsRepository.
func (r *PostgresCredentialsRepository) Save(ctx context.Context, userID uuid.UUID, creds Credentials) error {
	refreshCiphertext, refreshNonce, accessCiphertext, accessNonce, err := r.encryptTokens(creds)
	if err != nil {
		return err
	}

	_, err = r.q.UpsertGoogleOAuthCredentials(ctx, db.UpsertGoogleOAuthCredentialsParams{
		UserID:                db.ToUUID(userID),
		GoogleHealthUserID:    creds.HealthUserID,
		RefreshTokenEncrypted: refreshCiphertext,
		RefreshTokenNonce:     refreshNonce,
		AccessTokenEncrypted:  accessCiphertext,
		AccessTokenNonce:      accessNonce,
		AccessTokenExpiresAt:  db.ToTimestamptzPtr(creds.AccessTokenExpiresAt),
	})
	return err
}

// UpdateTokens implements CredentialsRepository.
func (r *PostgresCredentialsRepository) UpdateTokens(ctx context.Context, userID uuid.UUID, creds Credentials) error {
	refreshCiphertext, refreshNonce, accessCiphertext, accessNonce, err := r.encryptTokens(creds)
	if err != nil {
		return err
	}

	return r.q.UpdateGoogleOAuthTokens(ctx, db.UpdateGoogleOAuthTokensParams{
		UserID:                db.ToUUID(userID),
		RefreshTokenEncrypted: refreshCiphertext,
		RefreshTokenNonce:     refreshNonce,
		AccessTokenEncrypted:  accessCiphertext,
		AccessTokenNonce:      accessNonce,
		AccessTokenExpiresAt:  db.ToTimestamptzPtr(creds.AccessTokenExpiresAt),
	})
}

// encryptTokens encrypts creds' refresh and (if present) access tokens.
func (r *PostgresCredentialsRepository) encryptTokens(creds Credentials) (refreshCiphertext, refreshNonce, accessCiphertext, accessNonce []byte, err error) {
	refreshCiphertext, refreshNonce, err = Encrypt(r.key, []byte(creds.RefreshToken))
	if err != nil {
		return nil, nil, nil, nil, err
	}

	if creds.AccessToken != "" {
		accessCiphertext, accessNonce, err = Encrypt(r.key, []byte(creds.AccessToken))
		if err != nil {
			return nil, nil, nil, nil, err
		}
	}

	return refreshCiphertext, refreshNonce, accessCiphertext, accessNonce, nil
}

// Delete implements CredentialsRepository.
func (r *PostgresCredentialsRepository) Delete(ctx context.Context, userID uuid.UUID) error {
	return r.q.DeleteGoogleOAuthCredentials(ctx, db.ToUUID(userID))
}

// SetSyncEnabled implements CredentialsRepository.
func (r *PostgresCredentialsRepository) SetSyncEnabled(ctx context.Context, userID uuid.UUID, enabled bool) error {
	// Confirm the row exists first so we can return the same ErrNotConnected
	// sentinel Get uses, rather than silently no-op'ing an UPDATE that
	// matched zero rows.
	if _, err := r.Get(ctx, userID); err != nil {
		return err
	}
	return r.q.UpdateGoogleSyncEnabled(ctx, db.UpdateGoogleSyncEnabledParams{
		UserID:      db.ToUUID(userID),
		SyncEnabled: enabled,
	})
}
