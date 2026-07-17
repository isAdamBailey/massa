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

	_, err := svc.Upsert(context.Background(), userID, day1, 7, nil)
	require.NoError(t, err)
	_, err = svc.Upsert(context.Background(), userID, day2, 4, nil)
	require.NoError(t, err)
	_, err = svc.Upsert(context.Background(), otherUserID, day1, 9, nil)
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

	_, err := svc.Upsert(context.Background(), userID, day1, 5, nil)
	require.NoError(t, err)
	_, err = svc.Upsert(context.Background(), userID, day2, 6, nil)
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

	entry, err := svc.Upsert(context.Background(), userID, day, 5, nil)
	require.NoError(t, err)
	assert.Equal(t, 5, entry.OverwhelmLevel)
	assert.Equal(t, day, entry.Day)
	assert.Empty(t, entry.Tags)
}

func TestService_Upsert_ReplacesExistingDay(t *testing.T) {
	userID := uuid.New()
	day := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)

	q := newFakeQuerier()
	svc := overwhelm.NewService(q)

	_, err := svc.Upsert(context.Background(), userID, day, 5, nil)
	require.NoError(t, err)
	_, err = svc.Upsert(context.Background(), userID, day, 8, nil)
	require.NoError(t, err)

	entries, err := svc.List(context.Background(), userID, nil, nil)
	require.NoError(t, err)
	require.Len(t, entries, 1)
	assert.Equal(t, 8, entries[0].OverwhelmLevel)
}

func TestService_Upsert_AttachesTags(t *testing.T) {
	userID := uuid.New()
	day := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)

	q := newFakeQuerier()
	svc := overwhelm.NewService(q)

	work, err := svc.CreateTag(context.Background(), userID, "Work")
	require.NoError(t, err)
	sleep, err := svc.CreateTag(context.Background(), userID, "Sleep")
	require.NoError(t, err)

	entry, err := svc.Upsert(context.Background(), userID, day, 7, []uuid.UUID{work.ID, sleep.ID})
	require.NoError(t, err)
	require.Len(t, entry.Tags, 2)
	assert.Equal(t, "Sleep", entry.Tags[0].Name)
	assert.Equal(t, "Work", entry.Tags[1].Name)
}

func TestService_Upsert_DropsForeignTagIDs(t *testing.T) {
	userID := uuid.New()
	otherUserID := uuid.New()
	day := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)

	q := newFakeQuerier()
	svc := overwhelm.NewService(q)

	foreignTag, err := svc.CreateTag(context.Background(), otherUserID, "Work")
	require.NoError(t, err)

	entry, err := svc.Upsert(context.Background(), userID, day, 7, []uuid.UUID{foreignTag.ID})
	require.NoError(t, err)
	assert.Empty(t, entry.Tags)
}

func TestService_Upsert_ReplacesTagsOnSecondCall(t *testing.T) {
	userID := uuid.New()
	day := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)

	q := newFakeQuerier()
	svc := overwhelm.NewService(q)

	work, err := svc.CreateTag(context.Background(), userID, "Work")
	require.NoError(t, err)
	sleep, err := svc.CreateTag(context.Background(), userID, "Sleep")
	require.NoError(t, err)

	_, err = svc.Upsert(context.Background(), userID, day, 7, []uuid.UUID{work.ID})
	require.NoError(t, err)
	entry, err := svc.Upsert(context.Background(), userID, day, 7, []uuid.UUID{sleep.ID})
	require.NoError(t, err)

	require.Len(t, entry.Tags, 1)
	assert.Equal(t, "Sleep", entry.Tags[0].Name)
}

func TestService_CreateTag_Unarchives(t *testing.T) {
	userID := uuid.New()

	q := newFakeQuerier()
	svc := overwhelm.NewService(q)

	tag, err := svc.CreateTag(context.Background(), userID, "Work")
	require.NoError(t, err)
	require.NoError(t, svc.ArchiveTag(context.Background(), userID, tag.ID))

	tags, err := svc.ListTags(context.Background(), userID)
	require.NoError(t, err)
	assert.Empty(t, tags)

	reCreated, err := svc.CreateTag(context.Background(), userID, "work")
	require.NoError(t, err)
	assert.Equal(t, tag.ID, reCreated.ID)

	tags, err = svc.ListTags(context.Background(), userID)
	require.NoError(t, err)
	require.Len(t, tags, 1)
	assert.Equal(t, "work", tags[0].Name)
}

func TestService_RenameTag(t *testing.T) {
	userID := uuid.New()

	q := newFakeQuerier()
	svc := overwhelm.NewService(q)

	tag, err := svc.CreateTag(context.Background(), userID, "Work")
	require.NoError(t, err)

	renamed, err := svc.RenameTag(context.Background(), userID, tag.ID, "Job")
	require.NoError(t, err)
	assert.Equal(t, "Job", renamed.Name)
}

func TestService_RenameTag_DuplicateReturnsErrDuplicateTag(t *testing.T) {
	userID := uuid.New()

	q := newFakeQuerier()
	svc := overwhelm.NewService(q)

	_, err := svc.CreateTag(context.Background(), userID, "Work")
	require.NoError(t, err)
	sleep, err := svc.CreateTag(context.Background(), userID, "Sleep")
	require.NoError(t, err)

	_, err = svc.RenameTag(context.Background(), userID, sleep.ID, "Work")
	assert.ErrorIs(t, err, overwhelm.ErrDuplicateTag)
}

func TestService_RenameTag_NotFoundReturnsErrNotFound(t *testing.T) {
	userID := uuid.New()

	q := newFakeQuerier()
	svc := overwhelm.NewService(q)

	_, err := svc.RenameTag(context.Background(), userID, uuid.New(), "Job")
	assert.ErrorIs(t, err, overwhelm.ErrNotFound)
}

func TestService_ArchiveTag_NotFoundReturnsErrNotFound(t *testing.T) {
	userID := uuid.New()

	q := newFakeQuerier()
	svc := overwhelm.NewService(q)

	err := svc.ArchiveTag(context.Background(), userID, uuid.New())
	assert.ErrorIs(t, err, overwhelm.ErrNotFound)
}

func TestService_ArchiveTag_PreservesEntryHistory(t *testing.T) {
	userID := uuid.New()
	day := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)

	q := newFakeQuerier()
	svc := overwhelm.NewService(q)

	work, err := svc.CreateTag(context.Background(), userID, "Work")
	require.NoError(t, err)
	_, err = svc.Upsert(context.Background(), userID, day, 7, []uuid.UUID{work.ID})
	require.NoError(t, err)

	require.NoError(t, svc.ArchiveTag(context.Background(), userID, work.ID))

	tags, err := svc.ListTags(context.Background(), userID)
	require.NoError(t, err)
	assert.Empty(t, tags)

	entries, err := svc.List(context.Background(), userID, nil, nil)
	require.NoError(t, err)
	require.Len(t, entries, 1)
	require.Len(t, entries[0].Tags, 1)
	assert.Equal(t, "Work", entries[0].Tags[0].Name)
}
