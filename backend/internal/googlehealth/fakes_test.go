package googlehealth_test

import (
	"context"
	"errors"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"

	"github.com/isAdamBailey/massa/backend/internal/db"
)

// fakeQuerier is an in-memory implementation of googlehealth.Querier.
type fakeQuerier struct {
	credentials map[uuid.UUID]db.GoogleOauthCredential
	syncMeta    map[uuid.UUID]db.SyncMetadatum

	// weightEntries and heightEntries are keyed by user ID, then by either
	// the Google data point ID (if present) or the recorded_at timestamp
	// formatted as RFC3339Nano, mirroring the partial unique indexes.
	weightEntries map[uuid.UUID]map[string]db.WeightEntry
	heightEntries map[uuid.UUID]map[string]db.HeightEntry
}

func newFakeQuerier() *fakeQuerier {
	return &fakeQuerier{
		credentials:   make(map[uuid.UUID]db.GoogleOauthCredential),
		syncMeta:      make(map[uuid.UUID]db.SyncMetadatum),
		weightEntries: make(map[uuid.UUID]map[string]db.WeightEntry),
		heightEntries: make(map[uuid.UUID]map[string]db.HeightEntry),
	}
}

func (f *fakeQuerier) GetGoogleOAuthCredentialsByUserID(_ context.Context, userID pgtype.UUID) (db.GoogleOauthCredential, error) {
	row, ok := f.credentials[db.FromUUID(userID)]
	if !ok {
		return db.GoogleOauthCredential{}, pgx.ErrNoRows
	}
	return row, nil
}

func (f *fakeQuerier) UpsertGoogleOAuthCredentials(_ context.Context, arg db.UpsertGoogleOAuthCredentialsParams) (db.GoogleOauthCredential, error) {
	row := db.GoogleOauthCredential{
		ID:                    db.ToUUID(uuid.New()),
		UserID:                arg.UserID,
		GoogleHealthUserID:    arg.GoogleHealthUserID,
		RefreshTokenEncrypted: arg.RefreshTokenEncrypted,
		RefreshTokenNonce:     arg.RefreshTokenNonce,
		AccessTokenEncrypted:  arg.AccessTokenEncrypted,
		AccessTokenNonce:      arg.AccessTokenNonce,
		AccessTokenExpiresAt:  arg.AccessTokenExpiresAt,
	}
	f.credentials[db.FromUUID(arg.UserID)] = row
	return row, nil
}

func (f *fakeQuerier) DeleteGoogleOAuthCredentials(_ context.Context, userID pgtype.UUID) error {
	delete(f.credentials, db.FromUUID(userID))
	return nil
}

func (f *fakeQuerier) UpsertSyncMetadata(_ context.Context, userID pgtype.UUID) (db.SyncMetadatum, error) {
	row, ok := f.syncMeta[db.FromUUID(userID)]
	if !ok {
		row = db.SyncMetadatum{ID: db.ToUUID(uuid.New()), UserID: userID}
		f.syncMeta[db.FromUUID(userID)] = row
	}
	return row, nil
}

func (f *fakeQuerier) UpdateSyncWatermarks(_ context.Context, arg db.UpdateSyncWatermarksParams) error {
	row, ok := f.syncMeta[db.FromUUID(arg.UserID)]
	if !ok {
		return errors.New("sync metadata not found")
	}
	row.LastFullBackfillAt = arg.LastFullBackfillAt
	row.LastIncrementalSyncAt = arg.LastIncrementalSyncAt
	row.WeightSyncWatermark = arg.WeightSyncWatermark
	row.HeightSyncWatermark = arg.HeightSyncWatermark
	f.syncMeta[db.FromUUID(arg.UserID)] = row
	return nil
}

func (f *fakeQuerier) UpsertWeightEntryByGoogleID(_ context.Context, arg db.UpsertWeightEntryByGoogleIDParams) (db.WeightEntry, error) {
	return f.upsertWeightEntry(arg.UserID, *arg.GoogleDataPointID, arg.WeightKg, arg.RecordedAt, arg.GoogleDataPointID)
}

func (f *fakeQuerier) UpsertWeightEntryByRecordedAt(_ context.Context, arg db.UpsertWeightEntryByRecordedAtParams) (db.WeightEntry, error) {
	return f.upsertWeightEntry(arg.UserID, arg.RecordedAt.Time.String(), arg.WeightKg, arg.RecordedAt, nil)
}

func (f *fakeQuerier) upsertWeightEntry(userID pgtype.UUID, key string, weightKg pgtype.Numeric, recordedAt pgtype.Timestamptz, dataPointID *string) (db.WeightEntry, error) {
	entries, ok := f.weightEntries[db.FromUUID(userID)]
	if !ok {
		entries = make(map[string]db.WeightEntry)
		f.weightEntries[db.FromUUID(userID)] = entries
	}
	row := db.WeightEntry{
		ID:                db.ToUUID(uuid.New()),
		UserID:            userID,
		WeightKg:          weightKg,
		RecordedAt:        recordedAt,
		Source:            "google",
		GoogleDataPointID: dataPointID,
	}
	entries[key] = row
	return row, nil
}

func (f *fakeQuerier) UpsertHeightEntryByGoogleID(_ context.Context, arg db.UpsertHeightEntryByGoogleIDParams) (db.HeightEntry, error) {
	return f.upsertHeightEntry(arg.UserID, *arg.GoogleDataPointID, arg.HeightCm, arg.RecordedAt, arg.GoogleDataPointID)
}

func (f *fakeQuerier) UpsertHeightEntryByRecordedAt(_ context.Context, arg db.UpsertHeightEntryByRecordedAtParams) (db.HeightEntry, error) {
	return f.upsertHeightEntry(arg.UserID, arg.RecordedAt.Time.String(), arg.HeightCm, arg.RecordedAt, nil)
}

func (f *fakeQuerier) upsertHeightEntry(userID pgtype.UUID, key string, heightCm pgtype.Numeric, recordedAt pgtype.Timestamptz, dataPointID *string) (db.HeightEntry, error) {
	entries, ok := f.heightEntries[db.FromUUID(userID)]
	if !ok {
		entries = make(map[string]db.HeightEntry)
		f.heightEntries[db.FromUUID(userID)] = entries
	}
	row := db.HeightEntry{
		ID:                db.ToUUID(uuid.New()),
		UserID:            userID,
		HeightCm:          heightCm,
		RecordedAt:        recordedAt,
		Source:            "google",
		GoogleDataPointID: dataPointID,
	}
	entries[key] = row
	return row, nil
}
