// Package auth implements passwordless magic-link authentication and
// server-side session management.
package auth

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"

	"github.com/isAdamBailey/massa/backend/internal/db"
	"github.com/isAdamBailey/massa/backend/internal/mailer"
	"github.com/isAdamBailey/massa/backend/internal/users"
)

// Querier is the subset of db.Querier used by Service.
type Querier interface {
	CountRecentMagicLinkTokensForEmail(ctx context.Context, userEmail string) (int64, error)
	CreateMagicLinkToken(ctx context.Context, arg db.CreateMagicLinkTokenParams) (db.MagicLinkToken, error)
	GetValidMagicLinkToken(ctx context.Context, tokenHash string) (db.MagicLinkToken, error)
	MarkMagicLinkTokenUsed(ctx context.Context, id pgtype.UUID) error
	CreateSession(ctx context.Context, arg db.CreateSessionParams) (db.Session, error)
	GetSession(ctx context.Context, id pgtype.UUID) (db.Session, error)
	DeleteSession(ctx context.Context, id pgtype.UUID) error
}

const (
	magicLinkTTL         = 15 * time.Minute
	sessionTTL           = 30 * 24 * time.Hour
	maxMagicLinksPerHour = 5
)

// ErrInvalidToken is returned when a magic link token is missing, expired,
// already used, or otherwise invalid.
var ErrInvalidToken = errors.New("invalid or expired token")

// ErrSessionNotFound is returned when a session ID does not correspond to a
// valid, unexpired session.
var ErrSessionNotFound = errors.New("session not found")

// Session is an authenticated user session.
type Session struct {
	ID        uuid.UUID
	UserID    uuid.UUID
	ExpiresAt time.Time
}

// Service implements magic-link authentication and session management.
type Service struct {
	queries      Querier
	users        users.Repository
	mailer       mailer.Mailer
	cookieSecret []byte
	cookieSecure bool
	appBaseURL   string
}

// NewService constructs a Service.
func NewService(queries Querier, userRepo users.Repository, m mailer.Mailer, cookieSecret []byte, cookieSecure bool, appBaseURL string) *Service {
	return &Service{
		queries:      queries,
		users:        userRepo,
		mailer:       m,
		cookieSecret: cookieSecret,
		cookieSecure: cookieSecure,
		appBaseURL:   appBaseURL,
	}
}

// RequestMagicLink emails a sign-in link to email if it is on the allowlist
// and has not exceeded the rate limit. It returns nil in both of those
// non-error cases so callers can respond uniformly without leaking allowlist
// membership.
func (s *Service) RequestMagicLink(ctx context.Context, email string) error {
	allowed, err := s.users.IsEmailAllowed(ctx, email)
	if err != nil {
		return fmt.Errorf("check allowlist: %w", err)
	}
	if !allowed {
		return nil
	}

	count, err := s.queries.CountRecentMagicLinkTokensForEmail(ctx, email)
	if err != nil {
		return fmt.Errorf("check rate limit: %w", err)
	}
	if count >= maxMagicLinksPerHour {
		return nil
	}

	raw, hash, err := generateToken()
	if err != nil {
		return err
	}

	if _, err := s.queries.CreateMagicLinkToken(ctx, db.CreateMagicLinkTokenParams{
		UserEmail: email,
		TokenHash: hash,
		ExpiresAt: db.ToTimestamptz(time.Now().Add(magicLinkTTL)),
	}); err != nil {
		return fmt.Errorf("create magic link token: %w", err)
	}

	link := s.appBaseURL + "/auth/callback?token=" + raw
	if err := s.mailer.SendMagicLink(ctx, email, link); err != nil {
		return fmt.Errorf("send magic link email: %w", err)
	}

	return nil
}

// VerifyMagicLink validates rawToken, marks it used, finds or creates the
// corresponding user, and creates a new session.
func (s *Service) VerifyMagicLink(ctx context.Context, rawToken string) (Session, error) {
	tok, err := s.queries.GetValidMagicLinkToken(ctx, hashToken(rawToken))
	if errors.Is(err, pgx.ErrNoRows) {
		return Session{}, ErrInvalidToken
	}
	if err != nil {
		return Session{}, fmt.Errorf("lookup magic link token: %w", err)
	}

	if err := s.queries.MarkMagicLinkTokenUsed(ctx, tok.ID); err != nil {
		return Session{}, fmt.Errorf("mark token used: %w", err)
	}

	user, err := s.users.GetByEmail(ctx, tok.UserEmail)
	if errors.Is(err, users.ErrNotFound) {
		user, err = s.users.Create(ctx, tok.UserEmail)
	}
	if err != nil {
		return Session{}, fmt.Errorf("resolve user: %w", err)
	}

	if err := s.users.UpdateLastLoginAt(ctx, user.ID); err != nil {
		return Session{}, fmt.Errorf("update last login: %w", err)
	}

	row, err := s.queries.CreateSession(ctx, db.CreateSessionParams{
		UserID:    db.ToUUID(user.ID),
		ExpiresAt: db.ToTimestamptz(time.Now().Add(sessionTTL)),
	})
	if err != nil {
		return Session{}, fmt.Errorf("create session: %w", err)
	}

	return sessionFromRow(row), nil
}

// GetSession returns the session with the given ID, or ErrSessionNotFound.
func (s *Service) GetSession(ctx context.Context, id uuid.UUID) (Session, error) {
	row, err := s.queries.GetSession(ctx, db.ToUUID(id))
	if errors.Is(err, pgx.ErrNoRows) {
		return Session{}, ErrSessionNotFound
	}
	if err != nil {
		return Session{}, fmt.Errorf("get session: %w", err)
	}
	return sessionFromRow(row), nil
}

// Logout deletes the session with the given ID.
func (s *Service) Logout(ctx context.Context, id uuid.UUID) error {
	return s.queries.DeleteSession(ctx, db.ToUUID(id))
}

func sessionFromRow(row db.Session) Session {
	return Session{
		ID:        db.FromUUID(row.ID),
		UserID:    db.FromUUID(row.UserID),
		ExpiresAt: row.ExpiresAt.Time,
	}
}
