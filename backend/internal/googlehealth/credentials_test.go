package googlehealth_test

import (
	"context"
	"crypto/rand"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/isAdamBailey/massa/backend/internal/googlehealth"
)

func testKey(t *testing.T) []byte {
	t.Helper()
	key := make([]byte, 32)
	_, err := rand.Read(key)
	require.NoError(t, err)
	return key
}

func TestCredentialsRepository_SaveAndGet(t *testing.T) {
	q := newFakeQuerier()
	repo := googlehealth.NewPostgresCredentialsRepository(q, testKey(t))
	userID := uuid.New()
	expiry := time.Now().Add(time.Hour).UTC().Truncate(time.Second)

	want := googlehealth.Credentials{
		HealthUserID:         "health-user-123",
		RefreshToken:         "refresh-token-value",
		AccessToken:          "access-token-value",
		AccessTokenExpiresAt: &expiry,
	}

	require.NoError(t, repo.Save(context.Background(), userID, want))

	got, err := repo.Get(context.Background(), userID)
	require.NoError(t, err)
	assert.Equal(t, want.HealthUserID, got.HealthUserID)
	assert.Equal(t, want.RefreshToken, got.RefreshToken)
	assert.Equal(t, want.AccessToken, got.AccessToken)
	require.NotNil(t, got.AccessTokenExpiresAt)
	assert.True(t, expiry.Equal(*got.AccessTokenExpiresAt))
}

func TestCredentialsRepository_GetNotConnected(t *testing.T) {
	q := newFakeQuerier()
	repo := googlehealth.NewPostgresCredentialsRepository(q, testKey(t))

	_, err := repo.Get(context.Background(), uuid.New())
	require.ErrorIs(t, err, googlehealth.ErrNotConnected)
}

func TestCredentialsRepository_Delete(t *testing.T) {
	q := newFakeQuerier()
	repo := googlehealth.NewPostgresCredentialsRepository(q, testKey(t))
	userID := uuid.New()

	require.NoError(t, repo.Save(context.Background(), userID, googlehealth.Credentials{
		HealthUserID: "health-user-123",
		RefreshToken: "refresh-token-value",
	}))

	require.NoError(t, repo.Delete(context.Background(), userID))

	_, err := repo.Get(context.Background(), userID)
	require.ErrorIs(t, err, googlehealth.ErrNotConnected)
}
