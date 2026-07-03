// Package activeenergy provides read access to a user's daily active
// energy burned totals, synced in from Google Health.
package activeenergy

import (
	"context"
	"time"

	"github.com/google/uuid"

	"github.com/isAdamBailey/massa/backend/internal/db"
)

// Entry is a single day's total active energy burned.
type Entry struct {
	ID               uuid.UUID
	Day              time.Time
	ActiveEnergyKcal float64
	Source           string
	CreatedAt        time.Time
	UpdatedAt        time.Time
}

// Querier is the subset of db.Querier used by this package.
type Querier interface {
	ListActiveEnergyEntries(ctx context.Context, arg db.ListActiveEnergyEntriesParams) ([]db.ActiveEnergyEntry, error)
}

// Service reads active energy entries.
type Service struct {
	q Querier
}

// NewService returns a Service backed by q.
func NewService(q Querier) *Service {
	return &Service{q: q}
}

// List returns userID's active energy entries with day in [from, to],
// ordered oldest first. A nil from or to leaves that bound open.
func (s *Service) List(ctx context.Context, userID uuid.UUID, from, to *time.Time) ([]Entry, error) {
	rows, err := s.q.ListActiveEnergyEntries(ctx, db.ListActiveEnergyEntriesParams{
		UserID: db.ToUUID(userID),
		From:   db.ToDatePtr(from),
		To:     db.ToDatePtr(to),
	})
	if err != nil {
		return nil, err
	}

	entries := make([]Entry, len(rows))
	for i, row := range rows {
		entry, err := fromRow(row)
		if err != nil {
			return nil, err
		}
		entries[i] = entry
	}
	return entries, nil
}

func fromRow(row db.ActiveEnergyEntry) (Entry, error) {
	kcal, err := db.FromNumeric(row.ActiveEnergyKcal)
	if err != nil {
		return Entry{}, err
	}

	return Entry{
		ID:               db.FromUUID(row.ID),
		Day:              db.FromDate(row.Day),
		ActiveEnergyKcal: kcal,
		Source:           row.Source,
		CreatedAt:        row.CreatedAt.Time,
		UpdatedAt:        row.UpdatedAt.Time,
	}, nil
}
