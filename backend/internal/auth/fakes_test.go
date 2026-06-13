package auth_test

import (
	"context"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"

	"github.com/isAdamBailey/massa/backend/internal/db"
	"github.com/isAdamBailey/massa/backend/internal/users"
)

func mustUUID() uuid.UUID {
	return uuid.New()
}

// fakeQuerier is an in-memory implementation of auth.Querier.
type fakeQuerier struct {
	magicLinkTokens map[string]db.MagicLinkToken
	sessions        map[uuid.UUID]db.Session
	recentForEmail  int64
}

func newFakeQuerier() *fakeQuerier {
	return &fakeQuerier{
		magicLinkTokens: make(map[string]db.MagicLinkToken),
		sessions:        make(map[uuid.UUID]db.Session),
	}
}

func (f *fakeQuerier) CountRecentMagicLinkTokensForEmail(_ context.Context, _ string) (int64, error) {
	return f.recentForEmail, nil
}

func (f *fakeQuerier) CreateMagicLinkToken(_ context.Context, arg db.CreateMagicLinkTokenParams) (db.MagicLinkToken, error) {
	tok := db.MagicLinkToken{
		ID:        db.ToUUID(mustUUID()),
		UserEmail: arg.UserEmail,
		TokenHash: arg.TokenHash,
		ExpiresAt: arg.ExpiresAt,
		CreatedAt: db.ToTimestamptz(time.Now()),
	}
	f.magicLinkTokens[arg.TokenHash] = tok
	return tok, nil
}

func (f *fakeQuerier) GetValidMagicLinkToken(_ context.Context, tokenHash string) (db.MagicLinkToken, error) {
	tok, ok := f.magicLinkTokens[tokenHash]
	if !ok || tok.UsedAt.Valid || tok.ExpiresAt.Time.Before(time.Now()) {
		return db.MagicLinkToken{}, pgx.ErrNoRows
	}
	return tok, nil
}

func (f *fakeQuerier) MarkMagicLinkTokenUsed(_ context.Context, id pgtype.UUID) error {
	for hash, tok := range f.magicLinkTokens {
		if tok.ID == id {
			tok.UsedAt = db.ToTimestamptz(time.Now())
			f.magicLinkTokens[hash] = tok
		}
	}
	return nil
}

func (f *fakeQuerier) CreateSession(_ context.Context, arg db.CreateSessionParams) (db.Session, error) {
	id := mustUUID()
	sess := db.Session{
		ID:        db.ToUUID(id),
		UserID:    arg.UserID,
		ExpiresAt: arg.ExpiresAt,
		CreatedAt: db.ToTimestamptz(time.Now()),
	}
	f.sessions[id] = sess
	return sess, nil
}

func (f *fakeQuerier) GetSession(_ context.Context, id pgtype.UUID) (db.Session, error) {
	sess, ok := f.sessions[db.FromUUID(id)]
	if !ok || sess.ExpiresAt.Time.Before(time.Now()) {
		return db.Session{}, pgx.ErrNoRows
	}
	return sess, nil
}

func (f *fakeQuerier) DeleteSession(_ context.Context, id pgtype.UUID) error {
	delete(f.sessions, db.FromUUID(id))
	return nil
}

// fakeUsers is an in-memory implementation of users.Repository.
type fakeUsers struct {
	allowed map[string]bool
	byEmail map[string]users.User
}

func newFakeUsers(allowed ...string) *fakeUsers {
	m := make(map[string]bool, len(allowed))
	for _, e := range allowed {
		m[e] = true
	}
	return &fakeUsers{allowed: m, byEmail: make(map[string]users.User)}
}

func (f *fakeUsers) IsEmailAllowed(_ context.Context, email string) (bool, error) {
	return f.allowed[email], nil
}

func (f *fakeUsers) GetByEmail(_ context.Context, email string) (users.User, error) {
	u, ok := f.byEmail[email]
	if !ok {
		return users.User{}, users.ErrNotFound
	}
	return u, nil
}

func (f *fakeUsers) GetByID(_ context.Context, id uuid.UUID) (users.User, error) {
	for _, u := range f.byEmail {
		if u.ID == id {
			return u, nil
		}
	}
	return users.User{}, users.ErrNotFound
}

func (f *fakeUsers) Create(_ context.Context, email string) (users.User, error) {
	u := users.User{ID: mustUUID(), Email: email, UnitsPreference: "metric", CreatedAt: time.Now()}
	f.byEmail[email] = u
	return u, nil
}

func (f *fakeUsers) UpdateLastLoginAt(_ context.Context, id uuid.UUID) error {
	for email, u := range f.byEmail {
		if u.ID == id {
			now := time.Now()
			u.LastLoginAt = &now
			f.byEmail[email] = u
		}
	}
	return nil
}

func (f *fakeUsers) UpdateSettings(_ context.Context, id uuid.UUID, manualHeightCm *float64, unitsPreference string) (users.User, error) {
	for email, u := range f.byEmail {
		if u.ID == id {
			u.ManualHeightCm = manualHeightCm
			u.UnitsPreference = unitsPreference
			f.byEmail[email] = u
			return u, nil
		}
	}
	return users.User{}, users.ErrNotFound
}

func (f *fakeUsers) SyncAllowlist(_ context.Context, emails []string) error {
	allowed := make(map[string]bool, len(emails))
	for _, e := range emails {
		allowed[e] = true
	}
	f.allowed = allowed
	return nil
}

// fakeMailer is an in-memory implementation of mailer.Mailer.
type fakeMailer struct {
	sent []sentEmail
}

type sentEmail struct {
	to   string
	link string
}

func (f *fakeMailer) SendMagicLink(_ context.Context, toEmail, link string) error {
	f.sent = append(f.sent, sentEmail{to: toEmail, link: link})
	return nil
}
