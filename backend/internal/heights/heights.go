// Package heights resolves the height to use for a user's BMI calculations.
package heights

import (
	"context"
	"errors"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"

	"github.com/isAdamBailey/massa/backend/internal/db"
)

// ErrNoHeight is returned when a user has neither a height_entries row nor a
// manual height override set.
var ErrNoHeight = errors.New("no height available")

// Querier is the subset of db.Querier used by this package.
type Querier interface {
	GetLatestHeightEntry(ctx context.Context, userID pgtype.UUID) (db.HeightEntry, error)
	GetUserByID(ctx context.Context, id pgtype.UUID) (db.User, error)
}

// Resolver resolves the height to use for a user's BMI calculations.
type Resolver struct {
	q Querier
}

// NewResolver returns a Resolver backed by q.
func NewResolver(q Querier) *Resolver {
	return &Resolver{q: q}
}

// Resolve returns the height in centimeters to use for userID: the user's
// manual height override if set, otherwise the most recently recorded
// height_entries row. It returns ErrNoHeight if neither is set.
func (r *Resolver) Resolve(ctx context.Context, userID uuid.UUID) (float64, error) {
	user, err := r.q.GetUserByID(ctx, db.ToUUID(userID))
	if err != nil {
		return 0, err
	}
	if user.ManualHeightCm.Valid {
		return db.FromNumeric(user.ManualHeightCm)
	}

	entry, err := r.q.GetLatestHeightEntry(ctx, db.ToUUID(userID))
	if err == nil {
		return db.FromNumeric(entry.HeightCm)
	}
	if errors.Is(err, pgx.ErrNoRows) {
		return 0, ErrNoHeight
	}
	return 0, err
}
