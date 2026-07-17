// Package overwhelm provides access to a user's daily subjective overwhelm
// ratings, a manually logged 1-10 scale where 3 is the personal baseline,
// and the user-managed keyword vocabulary used to tag why a day felt
// overwhelming.
package overwhelm

import (
	"context"
	"errors"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgtype"

	"github.com/isAdamBailey/massa/backend/internal/db"
)

// Baseline is the neutral point on the overwhelm scale: readings above it are
// more overwhelmed than usual, below it less.
const Baseline = 3

// ErrNotFound is returned when a tag does not exist for the given user.
// Entries have no equivalent sentinel: they are addressed by day, not id,
// and every day is a valid upsert target.
var ErrNotFound = errors.New("overwhelm tag not found")

// ErrDuplicateTag is returned when renaming a tag to a name that collides
// (case-insensitively) with another of the user's tags.
var ErrDuplicateTag = errors.New("overwhelm tag name already in use")

// EntryTag is a tag attached to an overwhelm entry.
type EntryTag struct {
	ID   uuid.UUID
	Name string
}

// Entry is a single day's overwhelm rating, with the tags recorded for it.
// Tags carry names (not just ids) so an entry still renders its tags even
// after the tag is later archived from the picker.
type Entry struct {
	ID             uuid.UUID
	Day            time.Time
	OverwhelmLevel int
	Tags           []EntryTag
	CreatedAt      time.Time
	UpdatedAt      time.Time
}

// Tag is a user-defined keyword used to describe why a day was
// overwhelming.
type Tag struct {
	ID        uuid.UUID
	Name      string
	CreatedAt time.Time
	UpdatedAt time.Time
}

// Querier is the subset of db.Querier used by this package.
type Querier interface {
	ListOverwhelmEntries(ctx context.Context, arg db.ListOverwhelmEntriesParams) ([]db.ListOverwhelmEntriesRow, error)
	UpsertOverwhelmByDay(ctx context.Context, arg db.UpsertOverwhelmByDayParams) (db.UpsertOverwhelmByDayRow, error)
	ListOverwhelmTags(ctx context.Context, userID pgtype.UUID) ([]db.OverwhelmTag, error)
	CreateOrUnarchiveOverwhelmTag(ctx context.Context, arg db.CreateOrUnarchiveOverwhelmTagParams) (db.OverwhelmTag, error)
	RenameOverwhelmTag(ctx context.Context, arg db.RenameOverwhelmTagParams) (db.OverwhelmTag, error)
	ArchiveOverwhelmTag(ctx context.Context, arg db.ArchiveOverwhelmTagParams) (int64, error)
}

// Service reads and records overwhelm entries and manages the tag
// vocabulary used to describe them.
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
		entry, err := entryFromListRow(row)
		if err != nil {
			return nil, err
		}
		entries[i] = entry
	}
	return entries, nil
}

// Upsert records userID's overwhelm level and tags for day, replacing any
// existing entry for that day: overwhelm is logged once daily, so
// re-logging is a correction rather than a second reading. tagIDs that
// don't belong to userID are silently dropped rather than linked.
func (s *Service) Upsert(ctx context.Context, userID uuid.UUID, day time.Time, level int, tagIDs []uuid.UUID) (Entry, error) {
	ids := make([]pgtype.UUID, len(tagIDs))
	for i, id := range tagIDs {
		ids[i] = db.ToUUID(id)
	}

	row, err := s.q.UpsertOverwhelmByDay(ctx, db.UpsertOverwhelmByDayParams{
		UserID:         db.ToUUID(userID),
		Day:            db.ToDate(day),
		OverwhelmLevel: int16(level),
		TagIds:         ids,
	})
	if err != nil {
		return Entry{}, err
	}
	return entryFromUpsertRow(row)
}

// ListTags returns userID's active (non-archived) tags, name-ordered.
func (s *Service) ListTags(ctx context.Context, userID uuid.UUID) ([]Tag, error) {
	rows, err := s.q.ListOverwhelmTags(ctx, db.ToUUID(userID))
	if err != nil {
		return nil, err
	}

	tags := make([]Tag, len(rows))
	for i, row := range rows {
		tags[i] = tagFromRow(row)
	}
	return tags, nil
}

// CreateTag creates userID's tag named name, or unarchives and renames a
// previously archived tag with the same name (case-insensitive) if one
// exists - reconnecting it to its history rather than colliding on the
// unique name index.
func (s *Service) CreateTag(ctx context.Context, userID uuid.UUID, name string) (Tag, error) {
	row, err := s.q.CreateOrUnarchiveOverwhelmTag(ctx, db.CreateOrUnarchiveOverwhelmTagParams{
		UserID: db.ToUUID(userID),
		Name:   name,
	})
	if err != nil {
		return Tag{}, err
	}
	return tagFromRow(row), nil
}

// RenameTag renames userID's tag id to name. Returns ErrNotFound if id does
// not exist (or is archived) for userID, or ErrDuplicateTag if name
// collides with another of userID's tags.
func (s *Service) RenameTag(ctx context.Context, userID, id uuid.UUID, name string) (Tag, error) {
	row, err := s.q.RenameOverwhelmTag(ctx, db.RenameOverwhelmTagParams{
		ID:     db.ToUUID(id),
		UserID: db.ToUUID(userID),
		Name:   name,
	})
	if err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == "23505" {
			return Tag{}, ErrDuplicateTag
		}
		if errors.Is(err, pgx.ErrNoRows) {
			return Tag{}, ErrNotFound
		}
		return Tag{}, err
	}
	return tagFromRow(row), nil
}

// ArchiveTag archives userID's tag id, removing it from the picker while
// leaving it attached to any entry it was already logged on.
func (s *Service) ArchiveTag(ctx context.Context, userID, id uuid.UUID) error {
	n, err := s.q.ArchiveOverwhelmTag(ctx, db.ArchiveOverwhelmTagParams{
		ID:     db.ToUUID(id),
		UserID: db.ToUUID(userID),
	})
	if err != nil {
		return err
	}
	if n == 0 {
		return ErrNotFound
	}
	return nil
}

// zipTags pairs the parallel tag id/name arrays a query returns (both
// ordered by name) into EntryTags. The ids are genuine UUID columns cast to
// text by the query, so a parse failure here would indicate corrupt data
// rather than bad input.
func zipTags(tagIDs, tagNames []string) ([]EntryTag, error) {
	tags := make([]EntryTag, len(tagIDs))
	for i, idStr := range tagIDs {
		id, err := uuid.Parse(idStr)
		if err != nil {
			return nil, err
		}
		tags[i] = EntryTag{ID: id, Name: tagNames[i]}
	}
	return tags, nil
}

func entryFromListRow(row db.ListOverwhelmEntriesRow) (Entry, error) {
	tags, err := zipTags(row.TagIds, row.TagNames)
	if err != nil {
		return Entry{}, err
	}
	return Entry{
		ID:             db.FromUUID(row.ID),
		Day:            db.FromDate(row.Day),
		OverwhelmLevel: int(row.OverwhelmLevel),
		Tags:           tags,
		CreatedAt:      row.CreatedAt.Time,
		UpdatedAt:      row.UpdatedAt.Time,
	}, nil
}

func entryFromUpsertRow(row db.UpsertOverwhelmByDayRow) (Entry, error) {
	tags, err := zipTags(row.TagIds, row.TagNames)
	if err != nil {
		return Entry{}, err
	}
	return Entry{
		ID:             db.FromUUID(row.ID),
		Day:            db.FromDate(row.Day),
		OverwhelmLevel: int(row.OverwhelmLevel),
		Tags:           tags,
		CreatedAt:      row.CreatedAt.Time,
		UpdatedAt:      row.UpdatedAt.Time,
	}, nil
}

func tagFromRow(row db.OverwhelmTag) Tag {
	return Tag{
		ID:        db.FromUUID(row.ID),
		Name:      row.Name,
		CreatedAt: row.CreatedAt.Time,
		UpdatedAt: row.UpdatedAt.Time,
	}
}
