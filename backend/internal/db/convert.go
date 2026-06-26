package db

import (
	"strconv"
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

// ToTimestamptzPtr converts a *time.Time to the pgtype representation used
// by sqlc-generated code, returning an invalid (NULL) value for nil.
func ToTimestamptzPtr(t *time.Time) pgtype.Timestamptz {
	if t == nil {
		return pgtype.Timestamptz{}
	}
	return pgtype.Timestamptz{Time: *t, Valid: true}
}

// ToDate converts a time.Time to the pgtype representation used by
// sqlc-generated code.
func ToDate(t time.Time) pgtype.Date {
	return pgtype.Date{Time: t, Valid: true}
}

// ToNumeric converts a float64 to the pgtype representation used by
// sqlc-generated code.
func ToNumeric(f float64) (pgtype.Numeric, error) {
	var n pgtype.Numeric
	if err := n.Scan(strconv.FormatFloat(f, 'f', -1, 64)); err != nil {
		return pgtype.Numeric{}, err
	}
	return n, nil
}

// FromNumeric converts a pgtype.Numeric to a float64.
func FromNumeric(n pgtype.Numeric) (float64, error) {
	f, err := n.Float64Value()
	if err != nil {
		return 0, err
	}
	return f.Float64, nil
}

// ToNumericPtr converts a *float64 to the pgtype representation used by
// sqlc-generated code, returning an invalid (NULL) value for nil.
func ToNumericPtr(f *float64) (pgtype.Numeric, error) {
	if f == nil {
		return pgtype.Numeric{}, nil
	}
	return ToNumeric(*f)
}

// FromNumericPtr converts a pgtype.Numeric to a *float64, returning nil if
// the value is not set.
func FromNumericPtr(n pgtype.Numeric) (*float64, error) {
	if !n.Valid {
		return nil, nil
	}
	f, err := FromNumeric(n)
	if err != nil {
		return nil, err
	}
	return &f, nil
}
