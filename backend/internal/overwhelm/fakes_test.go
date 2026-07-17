package overwhelm_test

import (
	"context"
	"sort"
	"time"

	"github.com/google/uuid"

	"github.com/isAdamBailey/massa/backend/internal/db"
)

// overwhelmKey identifies an overwhelm entry the way the database does: by
// user and day.
type overwhelmKey struct {
	userID string
	day    string
}

// fakeQuerier is an in-memory implementation of overwhelm.Querier.
type fakeQuerier struct {
	entries map[overwhelmKey]db.OverwhelmEntry
}

func newFakeQuerier() *fakeQuerier {
	return &fakeQuerier{entries: make(map[overwhelmKey]db.OverwhelmEntry)}
}

func (f *fakeQuerier) ListOverwhelmEntries(_ context.Context, arg db.ListOverwhelmEntriesParams) ([]db.OverwhelmEntry, error) {
	var rows []db.OverwhelmEntry
	for _, row := range f.entries {
		if db.FromUUID(row.UserID) != db.FromUUID(arg.UserID) {
			continue
		}
		if arg.From.Valid && row.Day.Time.Before(arg.From.Time) {
			continue
		}
		if arg.To.Valid && row.Day.Time.After(arg.To.Time) {
			continue
		}
		rows = append(rows, row)
	}
	sort.Slice(rows, func(i, j int) bool { return rows[i].Day.Time.Before(rows[j].Day.Time) })
	return rows, nil
}

func (f *fakeQuerier) UpsertOverwhelmByDay(_ context.Context, arg db.UpsertOverwhelmByDayParams) (db.OverwhelmEntry, error) {
	key := overwhelmKey{userID: db.FromUUID(arg.UserID).String(), day: arg.Day.Time.Format("2006-01-02")}
	now := db.ToTimestamptz(time.Now())
	row, ok := f.entries[key]
	if !ok {
		row = db.OverwhelmEntry{
			ID:        db.ToUUID(uuid.New()),
			UserID:    arg.UserID,
			Day:       arg.Day,
			CreatedAt: now,
		}
	}
	row.OverwhelmLevel = arg.OverwhelmLevel
	row.UpdatedAt = now
	f.entries[key] = row
	return row, nil
}
