// Package overwhelm provides access to a user's daily subjective overwhelm
// ratings, a manually logged 1-10 scale where 3 is the personal baseline.
package overwhelm

import (
	"context"
	"time"

	"github.com/google/uuid"

	"github.com/isAdamBailey/massa/backend/internal/db"
)

// Baseline is the neutral point on the overwhelm scale: readings above it are
// more overwhelmed than usual, below it less.
const Baseline = 3

// Entry is a single day's overwhelm rating.
type Entry struct {
	ID             uuid.UUID
	Day            time.Time
	OverwhelmLevel int
	CreatedAt      time.Time
	UpdatedAt      time.Time
}

// Querier is the subset of db.Querier used by this package.
type Querier interface {
	ListOverwhelmEntries(ctx context.Context, arg db.ListOverwhelmEntriesParams) ([]db.OverwhelmEntry, error)
	UpsertOverwhelmByDay(ctx context.Context, arg db.UpsertOverwhelmByDayParams) (db.OverwhelmEntry, error)
}

// Service reads and records overwhelm entries.
type Service struct {
	q Querier
}

// NewService returns a Service backed by q.
func NewService(q Querier) *Service {
	return &Service{q: q}
}

// List returns userID's overwhelm entries with day in [from, to], ordered
// oldest first. A nil from or to leaves that bound open.
func (s *Service) List(ctx context.Context, userID uuid.UUID, from, to *time.Time) ([]Entry, error) {
	rows, err := s.q.ListOverwhelmEntries(ctx, db.ListOverwhelmEntriesParams{
		UserID: db.ToUUID(userID),
		From:   db.ToDatePtr(from),
		To:     db.ToDatePtr(to),
	})
	if err != nil {
		return nil, err
	}

	entries := make([]Entry, len(rows))
	for i, row := range rows {
		entries[i] = fromRow(row)
	}
	return entries, nil
}

// Upsert records userID's overwhelm level for day, replacing any existing
// entry for that day: overwhelm is logged once daily, so re-logging is a
// correction rather than a second reading.
func (s *Service) Upsert(ctx context.Context, userID uuid.UUID, day time.Time, level int) (Entry, error) {
	row, err := s.q.UpsertOverwhelmByDay(ctx, db.UpsertOverwhelmByDayParams{
		UserID:         db.ToUUID(userID),
		Day:            db.ToDate(day),
		OverwhelmLevel: int16(level),
	})
	if err != nil {
		return Entry{}, err
	}
	return fromRow(row), nil
}

func fromRow(row db.OverwhelmEntry) Entry {
	return Entry{
		ID:             db.FromUUID(row.ID),
		Day:            db.FromDate(row.Day),
		OverwhelmLevel: int(row.OverwhelmLevel),
		CreatedAt:      row.CreatedAt.Time,
		UpdatedAt:      row.UpdatedAt.Time,
	}
}
