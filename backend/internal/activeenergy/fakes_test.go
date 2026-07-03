package activeenergy_test

import (
	"context"

	"github.com/isAdamBailey/massa/backend/internal/db"
)

// fakeQuerier is an in-memory implementation of activeenergy.Querier.
type fakeQuerier struct {
	entries []db.ActiveEnergyEntry
}

func (f *fakeQuerier) ListActiveEnergyEntries(_ context.Context, arg db.ListActiveEnergyEntriesParams) ([]db.ActiveEnergyEntry, error) {
	var rows []db.ActiveEnergyEntry
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
	return rows, nil
}
