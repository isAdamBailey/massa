package overwhelm_test

import (
	"context"
	"sort"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgtype"

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
	entries   map[overwhelmKey]db.OverwhelmEntry
	tags      map[uuid.UUID]db.OverwhelmTag
	entryTags map[uuid.UUID]map[uuid.UUID]bool // entry id -> set of tag ids
}

func newFakeQuerier() *fakeQuerier {
	return &fakeQuerier{
		entries:   make(map[overwhelmKey]db.OverwhelmEntry),
		tags:      make(map[uuid.UUID]db.OverwhelmTag),
		entryTags: make(map[uuid.UUID]map[uuid.UUID]bool),
	}
}

// tagsForEntry returns the id/name arrays a real query would produce for
// entryID, both name-ordered.
func (f *fakeQuerier) tagsForEntry(entryID uuid.UUID) (ids []string, names []string) {
	type pair struct{ id, name string }
	var pairs []pair
	for tagID := range f.entryTags[entryID] {
		tag := f.tags[tagID]
		pairs = append(pairs, pair{id: tagID.String(), name: tag.Name})
	}
	sort.Slice(pairs, func(i, j int) bool { return pairs[i].name < pairs[j].name })
	for _, p := range pairs {
		ids = append(ids, p.id)
		names = append(names, p.name)
	}
	if ids == nil {
		ids = []string{}
	}
	if names == nil {
		names = []string{}
	}
	return ids, names
}

func (f *fakeQuerier) ListOverwhelmEntries(_ context.Context, arg db.ListOverwhelmEntriesParams) ([]db.ListOverwhelmEntriesRow, error) {
	var rows []db.ListOverwhelmEntriesRow
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
		ids, names := f.tagsForEntry(db.FromUUID(row.ID))
		rows = append(rows, db.ListOverwhelmEntriesRow{
			ID: row.ID, UserID: row.UserID, Day: row.Day, OverwhelmLevel: row.OverwhelmLevel,
			CreatedAt: row.CreatedAt, UpdatedAt: row.UpdatedAt, TagIds: ids, TagNames: names,
		})
	}
	sort.Slice(rows, func(i, j int) bool { return rows[i].Day.Time.Before(rows[j].Day.Time) })
	return rows, nil
}

func (f *fakeQuerier) UpsertOverwhelmByDay(_ context.Context, arg db.UpsertOverwhelmByDayParams) (db.UpsertOverwhelmByDayRow, error) {
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

	entryID := db.FromUUID(row.ID)
	wanted := make(map[uuid.UUID]bool, len(arg.TagIds))
	for _, id := range arg.TagIds {
		tagID := db.FromUUID(id)
		if tag, ok := f.tags[tagID]; ok && db.FromUUID(tag.UserID) == db.FromUUID(arg.UserID) {
			wanted[tagID] = true
		}
	}
	f.entryTags[entryID] = wanted

	ids, names := f.tagsForEntry(entryID)
	return db.UpsertOverwhelmByDayRow{
		ID: row.ID, UserID: row.UserID, Day: row.Day, OverwhelmLevel: row.OverwhelmLevel,
		CreatedAt: row.CreatedAt, UpdatedAt: row.UpdatedAt, TagIds: ids, TagNames: names,
	}, nil
}

func (f *fakeQuerier) ListOverwhelmTags(_ context.Context, userID pgtype.UUID) ([]db.OverwhelmTag, error) {
	var tags []db.OverwhelmTag
	for _, tag := range f.tags {
		if db.FromUUID(tag.UserID) != db.FromUUID(userID) || tag.ArchivedAt.Valid {
			continue
		}
		tags = append(tags, tag)
	}
	sort.Slice(tags, func(i, j int) bool { return tags[i].Name < tags[j].Name })
	return tags, nil
}

func (f *fakeQuerier) CreateOrUnarchiveOverwhelmTag(_ context.Context, arg db.CreateOrUnarchiveOverwhelmTagParams) (db.OverwhelmTag, error) {
	now := db.ToTimestamptz(time.Now())
	for id, tag := range f.tags {
		if db.FromUUID(tag.UserID) == db.FromUUID(arg.UserID) && strings.EqualFold(tag.Name, arg.Name) {
			tag.Name = arg.Name
			tag.ArchivedAt = pgtype.Timestamptz{}
			tag.UpdatedAt = now
			f.tags[id] = tag
			return tag, nil
		}
	}
	tag := db.OverwhelmTag{
		ID: db.ToUUID(uuid.New()), UserID: arg.UserID, Name: arg.Name,
		CreatedAt: now, UpdatedAt: now,
	}
	f.tags[db.FromUUID(tag.ID)] = tag
	return tag, nil
}

func (f *fakeQuerier) RenameOverwhelmTag(_ context.Context, arg db.RenameOverwhelmTagParams) (db.OverwhelmTag, error) {
	id := db.FromUUID(arg.ID)
	tag, ok := f.tags[id]
	if !ok || db.FromUUID(tag.UserID) != db.FromUUID(arg.UserID) || tag.ArchivedAt.Valid {
		return db.OverwhelmTag{}, pgx.ErrNoRows
	}
	for otherID, other := range f.tags {
		if otherID == id || other.ArchivedAt.Valid {
			continue
		}
		if db.FromUUID(other.UserID) == db.FromUUID(arg.UserID) && strings.EqualFold(other.Name, arg.Name) {
			return db.OverwhelmTag{}, &pgconn.PgError{Code: "23505"}
		}
	}
	tag.Name = arg.Name
	tag.UpdatedAt = db.ToTimestamptz(time.Now())
	f.tags[id] = tag
	return tag, nil
}

func (f *fakeQuerier) ArchiveOverwhelmTag(_ context.Context, arg db.ArchiveOverwhelmTagParams) (int64, error) {
	id := db.FromUUID(arg.ID)
	tag, ok := f.tags[id]
	if !ok || db.FromUUID(tag.UserID) != db.FromUUID(arg.UserID) || tag.ArchivedAt.Valid {
		return 0, nil
	}
	now := db.ToTimestamptz(time.Now())
	tag.ArchivedAt = now
	tag.UpdatedAt = now
	f.tags[id] = tag
	return 1, nil
}
