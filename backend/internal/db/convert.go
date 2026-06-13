package db

import (
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"
)

// ToUUID converts a uuid.UUID to the pgtype representation used by
// sqlc-generated code.
func ToUUID(id uuid.UUID) pgtype.UUID {
	return pgtype.UUID{Bytes: id, Valid: true}
}

// FromUUID converts a pgtype.UUID to uuid.UUID.
func FromUUID(id pgtype.UUID) uuid.UUID {
	return uuid.UUID(id.Bytes)
}

// ToTimestamptz converts a time.Time to the pgtype representation used by
// sqlc-generated code.
func ToTimestamptz(t time.Time) pgtype.Timestamptz {
	return pgtype.Timestamptz{Time: t, Valid: true}
}

// FromTimestamptz converts a pgtype.Timestamptz to a *time.Time, returning
// nil if the value is not set.
func FromTimestamptz(t pgtype.Timestamptz) *time.Time {
	if !t.Valid {
		return nil
	}
	return &t.Time
}
