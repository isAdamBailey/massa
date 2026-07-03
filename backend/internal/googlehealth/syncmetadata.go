package googlehealth

import (
	"context"
	"time"

	"github.com/google/uuid"

	"github.com/isAdamBailey/massa/backend/internal/db"
)

// SyncMetadata tracks the progress of a user's Google Health sync.
type SyncMetadata struct {
	LastFullBackfillAt        *time.Time
	LastIncrementalSyncAt     *time.Time
	WeightSyncWatermark       *time.Time
	HeightSyncWatermark       *time.Time
	ActiveEnergySyncWatermark *time.Time
}

// SyncMetadataRepository stores per-user Google Health sync progress.
type SyncMetadataRepository interface {
	// GetOrCreate returns the sync metadata for userID, creating an empty
	// row if one does not yet exist.
	GetOrCreate(ctx context.Context, userID uuid.UUID) (SyncMetadata, error)
	// Update overwrites the stored sync metadata for userID.
	Update(ctx context.Context, userID uuid.UUID, meta SyncMetadata) error
}

// PostgresSyncMetadataRepository implements SyncMetadataRepository using
// sqlc-generated queries.
type PostgresSyncMetadataRepository struct {
	q Querier
}

// NewPostgresSyncMetadataRepository returns a SyncMetadataRepository backed
// by q.
func NewPostgresSyncMetadataRepository(q Querier) *PostgresSyncMetadataRepository {
	return &PostgresSyncMetadataRepository{q: q}
}

// GetOrCreate implements SyncMetadataRepository.
func (r *PostgresSyncMetadataRepository) GetOrCreate(ctx context.Context, userID uuid.UUID) (SyncMetadata, error) {
	row, err := r.q.UpsertSyncMetadata(ctx, db.ToUUID(userID))
	if err != nil {
		return SyncMetadata{}, err
	}
	return SyncMetadata{
		LastFullBackfillAt:        db.FromTimestamptz(row.LastFullBackfillAt),
		LastIncrementalSyncAt:     db.FromTimestamptz(row.LastIncrementalSyncAt),
		WeightSyncWatermark:       db.FromTimestamptz(row.WeightSyncWatermark),
		HeightSyncWatermark:       db.FromTimestamptz(row.HeightSyncWatermark),
		ActiveEnergySyncWatermark: db.FromTimestamptz(row.ActiveEnergySyncWatermark),
	}, nil
}

// Update implements SyncMetadataRepository.
func (r *PostgresSyncMetadataRepository) Update(ctx context.Context, userID uuid.UUID, meta SyncMetadata) error {
	return r.q.UpdateSyncWatermarks(ctx, db.UpdateSyncWatermarksParams{
		UserID:                    db.ToUUID(userID),
		LastFullBackfillAt:        db.ToTimestamptzPtr(meta.LastFullBackfillAt),
		LastIncrementalSyncAt:     db.ToTimestamptzPtr(meta.LastIncrementalSyncAt),
		WeightSyncWatermark:       db.ToTimestamptzPtr(meta.WeightSyncWatermark),
		HeightSyncWatermark:       db.ToTimestamptzPtr(meta.HeightSyncWatermark),
		ActiveEnergySyncWatermark: db.ToTimestamptzPtr(meta.ActiveEnergySyncWatermark),
	})
}
