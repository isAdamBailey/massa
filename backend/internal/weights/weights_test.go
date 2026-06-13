package weights_test

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/isAdamBailey/massa/backend/internal/heights"
	"github.com/isAdamBailey/massa/backend/internal/weights"
)

func TestService_Create_ComputesBMI(t *testing.T) {
	q := newFakeQuerier()
	svc := weights.NewService(q, &fakeHeightResolver{heightCm: 175})

	userID := uuid.New()
	entry, err := svc.Create(context.Background(), userID, 70, time.Date(2024, 1, 1, 8, 0, 0, 0, time.UTC))
	require.NoError(t, err)

	require.NotNil(t, entry.BMI)
	assert.InDelta(t, 22.857142857142858, *entry.BMI, 1e-9)
	require.NotNil(t, entry.HeightUsedCm)
	assert.InDelta(t, 175.0, *entry.HeightUsedCm, 1e-9)
	assert.Equal(t, "manual", entry.Source)
}

func TestService_Create_NoHeightAvailable(t *testing.T) {
	q := newFakeQuerier()
	svc := weights.NewService(q, &fakeHeightResolver{err: heights.ErrNoHeight})

	entry, err := svc.Create(context.Background(), uuid.New(), 70, time.Now())
	require.NoError(t, err)

	assert.Nil(t, entry.BMI)
	assert.Nil(t, entry.HeightUsedCm)
}

func TestService_List_FiltersByDateRange(t *testing.T) {
	q := newFakeQuerier()
	svc := weights.NewService(q, &fakeHeightResolver{heightCm: 175})
	userID := uuid.New()

	_, err := svc.Create(context.Background(), userID, 70, time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC))
	require.NoError(t, err)
	_, err = svc.Create(context.Background(), userID, 71, time.Date(2024, 6, 1, 0, 0, 0, 0, time.UTC))
	require.NoError(t, err)
	_, err = svc.Create(context.Background(), userID, 72, time.Date(2024, 12, 1, 0, 0, 0, 0, time.UTC))
	require.NoError(t, err)

	from := time.Date(2024, 3, 1, 0, 0, 0, 0, time.UTC)
	to := time.Date(2024, 9, 1, 0, 0, 0, 0, time.UTC)
	entries, err := svc.List(context.Background(), userID, &from, &to)
	require.NoError(t, err)

	require.Len(t, entries, 1)
	assert.InDelta(t, 71.0, entries[0].WeightKg, 1e-9)
}

func TestService_List_NoRangeReturnsAll(t *testing.T) {
	q := newFakeQuerier()
	svc := weights.NewService(q, &fakeHeightResolver{heightCm: 175})
	userID := uuid.New()

	_, err := svc.Create(context.Background(), userID, 70, time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC))
	require.NoError(t, err)
	_, err = svc.Create(context.Background(), userID, 71, time.Date(2024, 6, 1, 0, 0, 0, 0, time.UTC))
	require.NoError(t, err)

	entries, err := svc.List(context.Background(), userID, nil, nil)
	require.NoError(t, err)
	assert.Len(t, entries, 2)
}

func TestService_Update(t *testing.T) {
	q := newFakeQuerier()
	svc := weights.NewService(q, &fakeHeightResolver{heightCm: 175})
	userID := uuid.New()

	entry, err := svc.Create(context.Background(), userID, 70, time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC))
	require.NoError(t, err)

	updated, err := svc.Update(context.Background(), userID, entry.ID, 72, time.Date(2024, 1, 2, 0, 0, 0, 0, time.UTC))
	require.NoError(t, err)

	assert.InDelta(t, 72.0, updated.WeightKg, 1e-9)
	require.NotNil(t, updated.BMI)
	assert.InDelta(t, 23.510204081632654, *updated.BMI, 1e-9)
}

func TestService_Update_NotFound(t *testing.T) {
	q := newFakeQuerier()
	svc := weights.NewService(q, &fakeHeightResolver{heightCm: 175})

	_, err := svc.Update(context.Background(), uuid.New(), uuid.New(), 70, time.Now())
	assert.ErrorIs(t, err, weights.ErrNotFound)
}

func TestService_Delete(t *testing.T) {
	q := newFakeQuerier()
	svc := weights.NewService(q, &fakeHeightResolver{heightCm: 175})
	userID := uuid.New()

	entry, err := svc.Create(context.Background(), userID, 70, time.Now())
	require.NoError(t, err)

	require.NoError(t, svc.Delete(context.Background(), userID, entry.ID))

	_, err = svc.Get(context.Background(), userID, entry.ID)
	assert.ErrorIs(t, err, weights.ErrNotFound)
}

func TestService_Delete_NotFound(t *testing.T) {
	q := newFakeQuerier()
	svc := weights.NewService(q, &fakeHeightResolver{heightCm: 175})

	err := svc.Delete(context.Background(), uuid.New(), uuid.New())
	assert.ErrorIs(t, err, weights.ErrNotFound)
}

func TestService_Latest(t *testing.T) {
	q := newFakeQuerier()
	svc := weights.NewService(q, &fakeHeightResolver{heightCm: 175})
	userID := uuid.New()

	_, err := svc.Create(context.Background(), userID, 70, time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC))
	require.NoError(t, err)
	_, err = svc.Create(context.Background(), userID, 75, time.Date(2024, 6, 1, 0, 0, 0, 0, time.UTC))
	require.NoError(t, err)

	latest, err := svc.Latest(context.Background(), userID)
	require.NoError(t, err)
	assert.InDelta(t, 75.0, latest.WeightKg, 1e-9)
}

func TestService_Latest_NotFound(t *testing.T) {
	q := newFakeQuerier()
	svc := weights.NewService(q, &fakeHeightResolver{heightCm: 175})

	_, err := svc.Latest(context.Background(), uuid.New())
	assert.ErrorIs(t, err, weights.ErrNotFound)
}
