package overwhelm_test

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/isAdamBailey/massa/backend/internal/overwhelm"
)

func TestService_List(t *testing.T) {
	userID := uuid.New()
	otherUserID := uuid.New()

	day1 := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)
	day2 := time.Date(2024, 1, 2, 0, 0, 0, 0, time.UTC)

	q := newFakeQuerier()
	svc := overwhelm.NewService(q)

	_, err := svc.Upsert(context.Background(), userID, day1, 7)
	require.NoError(t, err)
	_, err = svc.Upsert(context.Background(), userID, day2, 4)
	require.NoError(t, err)
	_, err = svc.Upsert(context.Background(), otherUserID, day1, 9)
	require.NoError(t, err)

	entries, err := svc.List(context.Background(), userID, nil, nil)
	require.NoError(t, err)
	require.Len(t, entries, 2)
	assert.Equal(t, 7, entries[0].OverwhelmLevel)
	assert.Equal(t, day1, entries[0].Day)
	assert.Equal(t, 4, entries[1].OverwhelmLevel)
	assert.Equal(t, day2, entries[1].Day)
}

func TestService_List_DateRangeFilter(t *testing.T) {
	userID := uuid.New()

	day1 := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)
	day2 := time.Date(2024, 1, 2, 0, 0, 0, 0, time.UTC)

	q := newFakeQuerier()
	svc := overwhelm.NewService(q)

	_, err := svc.Upsert(context.Background(), userID, day1, 5)
	require.NoError(t, err)
	_, err = svc.Upsert(context.Background(), userID, day2, 6)
	require.NoError(t, err)

	entries, err := svc.List(context.Background(), userID, &day2, nil)
	require.NoError(t, err)
	require.Len(t, entries, 1)
	assert.Equal(t, day2, entries[0].Day)
}

func TestService_Upsert(t *testing.T) {
	userID := uuid.New()
	day := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)

	q := newFakeQuerier()
	svc := overwhelm.NewService(q)

	entry, err := svc.Upsert(context.Background(), userID, day, 5)
	require.NoError(t, err)
	assert.Equal(t, 5, entry.OverwhelmLevel)
	assert.Equal(t, day, entry.Day)
}

func TestService_Upsert_ReplacesExistingDay(t *testing.T) {
	userID := uuid.New()
	day := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)

	q := newFakeQuerier()
	svc := overwhelm.NewService(q)

	_, err := svc.Upsert(context.Background(), userID, day, 5)
	require.NoError(t, err)
	_, err = svc.Upsert(context.Background(), userID, day, 8)
	require.NoError(t, err)

	entries, err := svc.List(context.Background(), userID, nil, nil)
	require.NoError(t, err)
	require.Len(t, entries, 1)
	assert.Equal(t, 8, entries[0].OverwhelmLevel)
}
