// Package users manages the allowlist and user records used by
// authentication.
package users

import (
	"context"
	"errors"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"github.com/isAdamBailey/massa/backend/internal/db"
)

// ErrNotFound is returned when a user does not exist.
var ErrNotFound = errors.New("user not found")

// User is a registered application user.
type User struct {
	ID              uuid.UUID
	Email           string
	ManualHeightCm  *float64
	UnitsPreference string
	CreatedAt       time.Time
	LastLoginAt     *time.Time
}

// Repository manages the allowlist and user records.
type Repository interface {
	// IsEmailAllowed reports whether email is on the allowlist.
	IsEmailAllowed(ctx context.Context, email string) (bool, error)
	// GetByEmail returns the user with the given email, or ErrNotFound.
	GetByEmail(ctx context.Context, email string) (User, error)
	// GetByID returns the user with the given ID, or ErrNotFound.
	GetByID(ctx context.Context, id uuid.UUID) (User, error)
	// Create inserts a new user with the given email.
	Create(ctx context.Context, email string) (User, error)
	// UpdateLastLoginAt sets the user's last_login_at to now.
	UpdateLastLoginAt(ctx context.Context, id uuid.UUID) error
	// UpdateSettings updates the user's manual height override and units
	// preference.
	UpdateSettings(ctx context.Context, id uuid.UUID, manualHeightCm *float64, unitsPreference string) (User, error)
	// SyncAllowlist makes the allowed_users table match emails exactly,
	// adding and removing entries as needed.
	SyncAllowlist(ctx context.Context, emails []string) error
}

// PostgresRepository implements Repository using sqlc-generated queries.
type PostgresRepository struct {
	q db.Querier
}

// NewPostgresRepository returns a Repository backed by q.
func NewPostgresRepository(q db.Querier) *PostgresRepository {
	return &PostgresRepository{q: q}
}

// IsEmailAllowed implements Repository.
func (r *PostgresRepository) IsEmailAllowed(ctx context.Context, email string) (bool, error) {
	return r.q.IsEmailAllowed(ctx, email)
}

// GetByEmail implements Repository.
func (r *PostgresRepository) GetByEmail(ctx context.Context, email string) (User, error) {
	row, err := r.q.GetUserByEmail(ctx, email)
	if errors.Is(err, pgx.ErrNoRows) {
		return User{}, ErrNotFound
	}
	if err != nil {
		return User{}, err
	}
	return fromRow(row)
}

// GetByID implements Repository.
func (r *PostgresRepository) GetByID(ctx context.Context, id uuid.UUID) (User, error) {
	row, err := r.q.GetUserByID(ctx, db.ToUUID(id))
	if errors.Is(err, pgx.ErrNoRows) {
		return User{}, ErrNotFound
	}
	if err != nil {
		return User{}, err
	}
	return fromRow(row)
}

// Create implements Repository.
func (r *PostgresRepository) Create(ctx context.Context, email string) (User, error) {
	row, err := r.q.CreateUser(ctx, email)
	if err != nil {
		return User{}, err
	}
	return fromRow(row)
}

// UpdateLastLoginAt implements Repository.
func (r *PostgresRepository) UpdateLastLoginAt(ctx context.Context, id uuid.UUID) error {
	return r.q.UpdateLastLoginAt(ctx, db.ToUUID(id))
}

// UpdateSettings implements Repository.
func (r *PostgresRepository) UpdateSettings(ctx context.Context, id uuid.UUID, manualHeightCm *float64, unitsPreference string) (User, error) {
	manualHeightCmNumeric, err := db.ToNumericPtr(manualHeightCm)
	if err != nil {
		return User{}, err
	}

	row, err := r.q.UpdateUserSettings(ctx, db.UpdateUserSettingsParams{
		ID:              db.ToUUID(id),
		ManualHeightCm:  manualHeightCmNumeric,
		UnitsPreference: unitsPreference,
	})
	if errors.Is(err, pgx.ErrNoRows) {
		return User{}, ErrNotFound
	}
	if err != nil {
		return User{}, err
	}
	return fromRow(row)
}

// SyncAllowlist implements Repository.
func (r *PostgresRepository) SyncAllowlist(ctx context.Context, emails []string) error {
	existing, err := r.q.ListAllowedEmails(ctx)
	if err != nil {
		return err
	}

	desired := make(map[string]bool, len(emails))
	for _, e := range emails {
		desired[e] = true
	}

	current := make(map[string]bool, len(existing))
	for _, e := range existing {
		current[e] = true
	}

	for email := range desired {
		if !current[email] {
			if err := r.q.UpsertAllowedUser(ctx, email); err != nil {
				return err
			}
		}
	}

	for email := range current {
		if !desired[email] {
			if err := r.q.DeleteAllowedUserByEmail(ctx, email); err != nil {
				return err
			}
		}
	}

	return nil
}

func fromRow(row db.User) (User, error) {
	manualHeightCm, err := db.FromNumericPtr(row.ManualHeightCm)
	if err != nil {
		return User{}, err
	}

	return User{
		ID:              db.FromUUID(row.ID),
		Email:           row.Email,
		ManualHeightCm:  manualHeightCm,
		UnitsPreference: row.UnitsPreference,
		CreatedAt:       row.CreatedAt.Time,
		LastLoginAt:     db.FromTimestamptz(row.LastLoginAt),
	}, nil
}
