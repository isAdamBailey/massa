package googlehealth_test

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/isAdamBailey/massa/backend/internal/googlehealth"
)

func TestSyncMetadataRepository_GetOrCreate(t *testing.T) {
	q := newFakeQuerier()
	repo := googlehealth.NewPostgresSyncMetadataRepository(q)
	userID := uuid.New()

	meta, err := repo.GetOrCreate(context.Background(), userID)
	require.NoError(t, err)
	assert.Nil(t, meta.LastFullBackfillAt)
	assert.Nil(t, meta.WeightSyncWatermark)

	// A second call returns the same (now-existing) row rather than erroring.
	meta2, err := repo.GetOrCreate(context.Background(), userID)
	require.NoError(t, err)
	assert.Equal(t, meta, meta2)
}

func TestSyncMetadataRepository_Update(t *testing.T) {
	q := newFakeQuerier()
	repo := googlehealth.NewPostgresSyncMetadataRepository(q)
	userID := uuid.New()

	_, err := repo.GetOrCreate(context.Background(), userID)
	require.NoError(t, err)

	now := time.Now().UTC().Truncate(time.Second)
	require.NoError(t, repo.Update(context.Background(), userID, googlehealth.SyncMetadata{
		LastFullBackfillAt:    &now,
		LastIncrementalSyncAt: &now,
		WeightSyncWatermark:   &now,
		HeightSyncWatermark:   &now,
	}))

	meta, err := repo.GetOrCreate(context.Background(), userID)
	require.NoError(t, err)
	require.NotNil(t, meta.LastFullBackfillAt)
	assert.True(t, now.Equal(*meta.LastFullBackfillAt))
	require.NotNil(t, meta.WeightSyncWatermark)
	assert.True(t, now.Equal(*meta.WeightSyncWatermark))
}
