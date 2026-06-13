package weights_test

import (
	"context"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"

	"github.com/isAdamBailey/massa/backend/internal/db"
)

// fakeQuerier is an in-memory implementation of weights.Querier.
type fakeQuerier struct {
	entries map[uuid.UUID]db.WeightEntry
}

func newFakeQuerier() *fakeQuerier {
	return &fakeQuerier{entries: make(map[uuid.UUID]db.WeightEntry)}
}

func (f *fakeQuerier) CreateWeightEntry(_ context.Context, arg db.CreateWeightEntryParams) (db.WeightEntry, error) {
	id := uuid.New()
	now := db.ToTimestamptz(arg.RecordedAt.Time)
	row := db.WeightEntry{
		ID:           db.ToUUID(id),
		UserID:       arg.UserID,
		WeightKg:     arg.WeightKg,
		RecordedAt:   arg.RecordedAt,
		Bmi:          arg.Bmi,
		HeightUsedCm: arg.HeightUsedCm,
		Source:       "manual",
		CreatedAt:    now,
		UpdatedAt:    now,
	}
	f.entries[id] = row
	return row, nil
}

func (f *fakeQuerier) ListWeightEntries(_ context.Context, arg db.ListWeightEntriesParams) ([]db.WeightEntry, error) {
	var rows []db.WeightEntry
	for _, row := range f.entries {
		if db.FromUUID(row.UserID) != db.FromUUID(arg.UserID) {
			continue
		}
		if arg.From.Valid && row.RecordedAt.Time.Before(arg.From.Time) {
			continue
		}
		if arg.To.Valid && row.RecordedAt.Time.After(arg.To.Time) {
			continue
		}
		rows = append(rows, row)
	}
	return rows, nil
}

func (f *fakeQuerier) GetWeightEntryByID(_ context.Context, arg db.GetWeightEntryByIDParams) (db.WeightEntry, error) {
	row, ok := f.entries[db.FromUUID(arg.ID)]
	if !ok || db.FromUUID(row.UserID) != db.FromUUID(arg.UserID) {
		return db.WeightEntry{}, pgx.ErrNoRows
	}
	return row, nil
}

func (f *fakeQuerier) GetLatestWeightEntry(_ context.Context, userID pgtype.UUID) (db.WeightEntry, error) {
	var latest db.WeightEntry
	found := false
	for _, row := range f.entries {
		if db.FromUUID(row.UserID) != db.FromUUID(userID) {
			continue
		}
		if !found || row.RecordedAt.Time.After(latest.RecordedAt.Time) {
			latest = row
			found = true
		}
	}
	if !found {
		return db.WeightEntry{}, pgx.ErrNoRows
	}
	return latest, nil
}

func (f *fakeQuerier) UpdateWeightEntry(_ context.Context, arg db.UpdateWeightEntryParams) (db.WeightEntry, error) {
	row, ok := f.entries[db.FromUUID(arg.ID)]
	if !ok || db.FromUUID(row.UserID) != db.FromUUID(arg.UserID) {
		return db.WeightEntry{}, pgx.ErrNoRows
	}
	row.WeightKg = arg.WeightKg
	row.RecordedAt = arg.RecordedAt
	row.Bmi = arg.Bmi
	row.HeightUsedCm = arg.HeightUsedCm
	f.entries[db.FromUUID(arg.ID)] = row
	return row, nil
}

func (f *fakeQuerier) UpdateWeightEntryGoogleSync(_ context.Context, arg db.UpdateWeightEntryGoogleSyncParams) (db.WeightEntry, error) {
	row, ok := f.entries[db.FromUUID(arg.ID)]
	if !ok || db.FromUUID(row.UserID) != db.FromUUID(arg.UserID) {
		return db.WeightEntry{}, pgx.ErrNoRows
	}
	row.GoogleDataPointID = arg.GoogleDataPointID
	row.GoogleSyncStatus = arg.GoogleSyncStatus
	f.entries[db.FromUUID(arg.ID)] = row
	return row, nil
}

func (f *fakeQuerier) DeleteWeightEntry(_ context.Context, arg db.DeleteWeightEntryParams) (int64, error) {
	row, ok := f.entries[db.FromUUID(arg.ID)]
	if !ok || db.FromUUID(row.UserID) != db.FromUUID(arg.UserID) {
		return 0, nil
	}
	delete(f.entries, db.FromUUID(arg.ID))
	return 1, nil
}

// fakeHeightResolver is a stub heights.Resolver.
type fakeHeightResolver struct {
	heightCm float64
	err      error
}

func (f *fakeHeightResolver) Resolve(_ context.Context, _ uuid.UUID) (float64, error) {
	if f.err != nil {
		return 0, f.err
	}
	return f.heightCm, nil
}
