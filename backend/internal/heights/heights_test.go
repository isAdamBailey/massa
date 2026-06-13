package heights_test

import (
	"context"
	"testing"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/isAdamBailey/massa/backend/internal/db"
	"github.com/isAdamBailey/massa/backend/internal/heights"
)

type fakeQuerier struct {
	heightEntry *db.HeightEntry
	user        db.User
}

func (f *fakeQuerier) GetLatestHeightEntry(_ context.Context, _ pgtype.UUID) (db.HeightEntry, error) {
	if f.heightEntry == nil {
		return db.HeightEntry{}, pgx.ErrNoRows
	}
	return *f.heightEntry, nil
}

func (f *fakeQuerier) GetUserByID(_ context.Context, _ pgtype.UUID) (db.User, error) {
	return f.user, nil
}

func numeric(t *testing.T, f float64) pgtype.Numeric {
	t.Helper()
	n, err := db.ToNumeric(f)
	require.NoError(t, err)
	return n
}

func TestResolve_PrefersManualHeightOverride(t *testing.T) {
	q := &fakeQuerier{
		heightEntry: &db.HeightEntry{HeightCm: numeric(t, 180)},
		user:        db.User{ManualHeightCm: numeric(t, 170)},
	}
	r := heights.NewResolver(q)

	got, err := r.Resolve(context.Background(), uuid.New())
	require.NoError(t, err)
	assert.InDelta(t, 170.0, got, 1e-9)
}

func TestResolve_FallsBackToLatestHeightEntry(t *testing.T) {
	q := &fakeQuerier{
		heightEntry: &db.HeightEntry{HeightCm: numeric(t, 180)},
	}
	r := heights.NewResolver(q)

	got, err := r.Resolve(context.Background(), uuid.New())
	require.NoError(t, err)
	assert.InDelta(t, 180.0, got, 1e-9)
}

func TestResolve_NoHeightAvailable(t *testing.T) {
	q := &fakeQuerier{user: db.User{}}
	r := heights.NewResolver(q)

	_, err := r.Resolve(context.Background(), uuid.New())
	assert.ErrorIs(t, err, heights.ErrNoHeight)
}
