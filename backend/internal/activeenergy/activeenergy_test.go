package activeenergy_test

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/isAdamBailey/massa/backend/internal/activeenergy"
	"github.com/isAdamBailey/massa/backend/internal/db"
)

func TestService_List(t *testing.T) {
	userID := uuid.New()
	otherUserID := uuid.New()

	day1, err := time.Parse("2006-01-02", "2024-01-01")
	require.NoError(t, err)
	day2, err := time.Parse("2006-01-02", "2024-01-02")
	require.NoError(t, err)

	kcal1, err := db.ToNumeric(320.5)
	require.NoError(t, err)
	kcal2, err := db.ToNumeric(410.25)
	require.NoError(t, err)
	otherKcal, err := db.ToNumeric(999)
	require.NoError(t, err)

	q := &fakeQuerier{entries: []db.ActiveEnergyEntry{
		{ID: db.ToUUID(uuid.New()), UserID: db.ToUUID(userID), Day: db.ToDate(day1), ActiveEnergyKcal: kcal1, Source: "google"},
		{ID: db.ToUUID(uuid.New()), UserID: db.ToUUID(userID), Day: db.ToDate(day2), ActiveEnergyKcal: kcal2, Source: "google"},
		{ID: db.ToUUID(uuid.New()), UserID: db.ToUUID(otherUserID), Day: db.ToDate(day1), ActiveEnergyKcal: otherKcal, Source: "google"},
	}}

	svc := activeenergy.NewService(q)

	entries, err := svc.List(context.Background(), userID, nil, nil)
	require.NoError(t, err)
	require.Len(t, entries, 2)
	assert.InDelta(t, 320.5, entries[0].ActiveEnergyKcal, 0.001)
	assert.InDelta(t, 410.25, entries[1].ActiveEnergyKcal, 0.001)
}

func TestService_List_DateRangeFilter(t *testing.T) {
	userID := uuid.New()

	day1, err := time.Parse("2006-01-02", "2024-01-01")
	require.NoError(t, err)
	day2, err := time.Parse("2006-01-02", "2024-01-02")
	require.NoError(t, err)

	kcal, err := db.ToNumeric(300)
	require.NoError(t, err)

	q := &fakeQuerier{entries: []db.ActiveEnergyEntry{
		{ID: db.ToUUID(uuid.New()), UserID: db.ToUUID(userID), Day: db.ToDate(day1), ActiveEnergyKcal: kcal, Source: "google"},
		{ID: db.ToUUID(uuid.New()), UserID: db.ToUUID(userID), Day: db.ToDate(day2), ActiveEnergyKcal: kcal, Source: "google"},
	}}

	svc := activeenergy.NewService(q)

	entries, err := svc.List(context.Background(), userID, &day2, nil)
	require.NoError(t, err)
	require.Len(t, entries, 1)
	assert.Equal(t, day2, entries[0].Day)
}
