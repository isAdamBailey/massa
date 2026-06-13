// Package weights manages a user's weight entries, computing and
// denormalizing BMI at write time using the height resolved at that time.
package weights

import (
	"context"
	"errors"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"

	"github.com/isAdamBailey/massa/backend/internal/bmi"
	"github.com/isAdamBailey/massa/backend/internal/db"
	"github.com/isAdamBailey/massa/backend/internal/heights"
)

// ErrNotFound is returned when a weight entry does not exist for the given
// user.
var ErrNotFound = errors.New("weight entry not found")

// Entry is a single weight measurement, with BMI computed from the height
// resolved at the time it was recorded.
type Entry struct {
	ID                uuid.UUID
	WeightKg          float64
	RecordedAt        time.Time
	BMI               *float64
	HeightUsedCm      *float64
	Source            string
	GoogleDataPointID *string
	GoogleSyncStatus  *string
	CreatedAt         time.Time
	UpdatedAt         time.Time
}

// Querier is the subset of db.Querier used by this package.
type Querier interface {
	CreateWeightEntry(ctx context.Context, arg db.CreateWeightEntryParams) (db.WeightEntry, error)
	ListWeightEntries(ctx context.Context, arg db.ListWeightEntriesParams) ([]db.WeightEntry, error)
	GetWeightEntryByID(ctx context.Context, arg db.GetWeightEntryByIDParams) (db.WeightEntry, error)
	GetLatestWeightEntry(ctx context.Context, userID pgtype.UUID) (db.WeightEntry, error)
	UpdateWeightEntry(ctx context.Context, arg db.UpdateWeightEntryParams) (db.WeightEntry, error)
	UpdateWeightEntryGoogleSync(ctx context.Context, arg db.UpdateWeightEntryGoogleSyncParams) (db.WeightEntry, error)
	DeleteWeightEntry(ctx context.Context, arg db.DeleteWeightEntryParams) (int64, error)
}

// HeightResolver resolves the height to use for a user's BMI calculations.
type HeightResolver interface {
	Resolve(ctx context.Context, userID uuid.UUID) (float64, error)
}

// Service manages weight entries.
type Service struct {
	q       Querier
	heights HeightResolver
}

// NewService returns a Service backed by q, resolving heights via heightResolver.
func NewService(q Querier, heightResolver HeightResolver) *Service {
	return &Service{q: q, heights: heightResolver}
}

// List returns userID's weight entries with recorded_at in [from, to],
// ordered oldest first. A nil from or to leaves that bound open.
func (s *Service) List(ctx context.Context, userID uuid.UUID, from, to *time.Time) ([]Entry, error) {
	rows, err := s.q.ListWeightEntries(ctx, db.ListWeightEntriesParams{
		UserID: db.ToUUID(userID),
		From:   db.ToTimestamptzPtr(from),
		To:     db.ToTimestamptzPtr(to),
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

// Create records a new manual weight entry for userID, computing BMI from
// the height resolved at this moment.
func (s *Service) Create(ctx context.Context, userID uuid.UUID, weightKg float64, recordedAt time.Time) (Entry, error) {
	bmiValue, heightUsedCm, err := s.resolveBMI(ctx, userID, weightKg)
	if err != nil {
		return Entry{}, err
	}

	weightKgNumeric, err := db.ToNumeric(weightKg)
	if err != nil {
		return Entry{}, err
	}

	row, err := s.q.CreateWeightEntry(ctx, db.CreateWeightEntryParams{
		UserID:       db.ToUUID(userID),
		WeightKg:     weightKgNumeric,
		RecordedAt:   db.ToTimestamptz(recordedAt),
		Bmi:          bmiValue,
		HeightUsedCm: heightUsedCm,
	})
	if err != nil {
		return Entry{}, err
	}
	return fromRow(row)
}

// Update changes the weight and/or recorded time of an existing entry,
// recomputing BMI from the height resolved at update time.
func (s *Service) Update(ctx context.Context, userID, id uuid.UUID, weightKg float64, recordedAt time.Time) (Entry, error) {
	bmiValue, heightUsedCm, err := s.resolveBMI(ctx, userID, weightKg)
	if err != nil {
		return Entry{}, err
	}

	weightKgNumeric, err := db.ToNumeric(weightKg)
	if err != nil {
		return Entry{}, err
	}

	row, err := s.q.UpdateWeightEntry(ctx, db.UpdateWeightEntryParams{
		ID:           db.ToUUID(id),
		UserID:       db.ToUUID(userID),
		WeightKg:     weightKgNumeric,
		RecordedAt:   db.ToTimestamptz(recordedAt),
		Bmi:          bmiValue,
		HeightUsedCm: heightUsedCm,
	})
	if errors.Is(err, pgx.ErrNoRows) {
		return Entry{}, ErrNotFound
	}
	if err != nil {
		return Entry{}, err
	}
	return fromRow(row)
}

// UpdateGoogleSync records the outcome of pushing a manual weight entry to
// Google Health, returning ErrNotFound if it does not exist for userID.
func (s *Service) UpdateGoogleSync(ctx context.Context, userID, id uuid.UUID, dataPointID *string, status string) (Entry, error) {
	row, err := s.q.UpdateWeightEntryGoogleSync(ctx, db.UpdateWeightEntryGoogleSyncParams{
		ID:                db.ToUUID(id),
		UserID:            db.ToUUID(userID),
		GoogleDataPointID: dataPointID,
		GoogleSyncStatus:  &status,
	})
	if errors.Is(err, pgx.ErrNoRows) {
		return Entry{}, ErrNotFound
	}
	if err != nil {
		return Entry{}, err
	}
	return fromRow(row)
}

// Delete removes a weight entry, returning ErrNotFound if it does not exist
// for userID.
func (s *Service) Delete(ctx context.Context, userID, id uuid.UUID) error {
	n, err := s.q.DeleteWeightEntry(ctx, db.DeleteWeightEntryParams{ID: db.ToUUID(id), UserID: db.ToUUID(userID)})
	if err != nil {
		return err
	}
	if n == 0 {
		return ErrNotFound
	}
	return nil
}

// Get returns a single weight entry, or ErrNotFound if it does not exist for
// userID.
func (s *Service) Get(ctx context.Context, userID, id uuid.UUID) (Entry, error) {
	row, err := s.q.GetWeightEntryByID(ctx, db.GetWeightEntryByIDParams{ID: db.ToUUID(id), UserID: db.ToUUID(userID)})
	if errors.Is(err, pgx.ErrNoRows) {
		return Entry{}, ErrNotFound
	}
	if err != nil {
		return Entry{}, err
	}
	return fromRow(row)
}

// Latest returns userID's most recently recorded weight entry, or
// ErrNotFound if none exist.
func (s *Service) Latest(ctx context.Context, userID uuid.UUID) (Entry, error) {
	row, err := s.q.GetLatestWeightEntry(ctx, db.ToUUID(userID))
	if errors.Is(err, pgx.ErrNoRows) {
		return Entry{}, ErrNotFound
	}
	if err != nil {
		return Entry{}, err
	}
	return fromRow(row)
}

// resolveBMI computes the BMI and height to denormalize onto a weight entry.
// If the user has no resolvable height, both return values are zero-valued
// (NULL in the database) rather than an error.
func (s *Service) resolveBMI(ctx context.Context, userID uuid.UUID, weightKg float64) (pgtype.Numeric, pgtype.Numeric, error) {
	heightCm, err := s.heights.Resolve(ctx, userID)
	if errors.Is(err, heights.ErrNoHeight) {
		return pgtype.Numeric{}, pgtype.Numeric{}, nil
	}
	if err != nil {
		return pgtype.Numeric{}, pgtype.Numeric{}, err
	}

	bmiValue, err := db.ToNumeric(bmi.Calculate(weightKg, heightCm))
	if err != nil {
		return pgtype.Numeric{}, pgtype.Numeric{}, err
	}
	heightUsedCm, err := db.ToNumeric(heightCm)
	if err != nil {
		return pgtype.Numeric{}, pgtype.Numeric{}, err
	}
	return bmiValue, heightUsedCm, nil
}

func fromRow(row db.WeightEntry) (Entry, error) {
	weightKg, err := db.FromNumeric(row.WeightKg)
	if err != nil {
		return Entry{}, err
	}

	entry := Entry{
		ID:                db.FromUUID(row.ID),
		WeightKg:          weightKg,
		RecordedAt:        row.RecordedAt.Time,
		Source:            row.Source,
		GoogleDataPointID: row.GoogleDataPointID,
		GoogleSyncStatus:  row.GoogleSyncStatus,
		CreatedAt:         row.CreatedAt.Time,
		UpdatedAt:         row.UpdatedAt.Time,
	}

	if row.Bmi.Valid {
		v, err := db.FromNumeric(row.Bmi)
		if err != nil {
			return Entry{}, err
		}
		entry.BMI = &v
	}
	if row.HeightUsedCm.Valid {
		v, err := db.FromNumeric(row.HeightUsedCm)
		if err != nil {
			return Entry{}, err
		}
		entry.HeightUsedCm = &v
	}

	return entry, nil
}
