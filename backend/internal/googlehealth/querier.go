package googlehealth

import (
	"context"

	"github.com/jackc/pgx/v5/pgtype"

	"github.com/isAdamBailey/massa/backend/internal/db"
)

// Querier is the subset of db.Querier used by this package.
type Querier interface {
	GetGoogleOAuthCredentialsByUserID(ctx context.Context, userID pgtype.UUID) (db.GoogleOauthCredential, error)
	UpsertGoogleOAuthCredentials(ctx context.Context, arg db.UpsertGoogleOAuthCredentialsParams) (db.GoogleOauthCredential, error)
	DeleteGoogleOAuthCredentials(ctx context.Context, userID pgtype.UUID) error

	UpsertSyncMetadata(ctx context.Context, userID pgtype.UUID) (db.SyncMetadatum, error)
	UpdateSyncWatermarks(ctx context.Context, arg db.UpdateSyncWatermarksParams) error

	ExistsWeightEntryForDate(ctx context.Context, arg db.ExistsWeightEntryForDateParams) (bool, error)
	UpsertWeightEntryByGoogleID(ctx context.Context, arg db.UpsertWeightEntryByGoogleIDParams) (db.WeightEntry, error)
	UpsertWeightEntryByRecordedAt(ctx context.Context, arg db.UpsertWeightEntryByRecordedAtParams) (db.WeightEntry, error)

	ExistsHeightEntryForDate(ctx context.Context, arg db.ExistsHeightEntryForDateParams) (bool, error)
	UpsertHeightEntryByGoogleID(ctx context.Context, arg db.UpsertHeightEntryByGoogleIDParams) (db.HeightEntry, error)
	UpsertHeightEntryByRecordedAt(ctx context.Context, arg db.UpsertHeightEntryByRecordedAtParams) (db.HeightEntry, error)
}
