package httpapi_test

import (
	"context"
	"sort"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"

	"github.com/isAdamBailey/massa/backend/internal/activeenergy"
	"github.com/isAdamBailey/massa/backend/internal/bmi"
	"github.com/isAdamBailey/massa/backend/internal/db"
	"github.com/isAdamBailey/massa/backend/internal/overwhelm"
	"github.com/isAdamBailey/massa/backend/internal/users"
	"github.com/isAdamBailey/massa/backend/internal/weights"
)

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
		ID:        db.ToUUID(uuid.New()),
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
	id := uuid.New()
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
	u := users.User{ID: uuid.New(), Email: email, UnitsPreference: "metric", CreatedAt: time.Now()}
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

// weightEntryWithUser pairs a weights.Entry with the user it belongs to,
// since weights.Entry itself has no user reference.
type weightEntryWithUser struct {
	weights.Entry
	userID uuid.UUID
}

// fakeActiveEnergyService is an in-memory implementation of
// httpapi.ActiveEnergyService.
type fakeActiveEnergyService struct {
	entries []activeenergy.Entry
}

func newFakeActiveEnergyService() *fakeActiveEnergyService {
	return &fakeActiveEnergyService{}
}

func (f *fakeActiveEnergyService) List(_ context.Context, _ uuid.UUID, _, _ *time.Time) ([]activeenergy.Entry, error) {
	return f.entries, nil
}

// overwhelmKey identifies an overwhelm entry the way the database does: by
// user and day.
type overwhelmKey struct {
	userID uuid.UUID
	day    string
}

// fakeOverwhelmService is an in-memory implementation of
// httpapi.OverwhelmService.
type fakeOverwhelmService struct {
	entries map[overwhelmKey]overwhelm.Entry
}

func newFakeOverwhelmService() *fakeOverwhelmService {
	return &fakeOverwhelmService{entries: make(map[overwhelmKey]overwhelm.Entry)}
}

func (f *fakeOverwhelmService) List(_ context.Context, userID uuid.UUID, from, to *time.Time) ([]overwhelm.Entry, error) {
	var entries []overwhelm.Entry
	for k, e := range f.entries {
		if k.userID != userID {
			continue
		}
		if from != nil && e.Day.Before(*from) {
			continue
		}
		if to != nil && e.Day.After(*to) {
			continue
		}
		entries = append(entries, e)
	}
	sort.Slice(entries, func(i, j int) bool { return entries[i].Day.Before(entries[j].Day) })
	return entries, nil
}

func (f *fakeOverwhelmService) Upsert(_ context.Context, userID uuid.UUID, day time.Time, level int) (overwhelm.Entry, error) {
	key := overwhelmKey{userID: userID, day: day.Format("2006-01-02")}
	now := time.Now()
	entry, ok := f.entries[key]
	if !ok {
		entry = overwhelm.Entry{ID: uuid.New(), Day: day, CreatedAt: now}
	}
	entry.OverwhelmLevel = level
	entry.UpdatedAt = now
	f.entries[key] = entry
	return entry, nil
}

// fakeWeightsService is an in-memory implementation of httpapi.WeightsService.
type fakeWeightsService struct {
	entries map[uuid.UUID]weightEntryWithUser
	// heightCm, if non-zero, is used to compute BMI for new/updated entries.
	heightCm float64
}

func newFakeWeightsService() *fakeWeightsService {
	return &fakeWeightsService{entries: make(map[uuid.UUID]weightEntryWithUser), heightCm: 175}
}

func (f *fakeWeightsService) bmiAndHeight(weightKg float64) (*float64, *float64) {
	if f.heightCm <= 0 {
		return nil, nil
	}
	b := bmi.Calculate(weightKg, f.heightCm)
	h := f.heightCm
	return &b, &h
}

func (f *fakeWeightsService) List(_ context.Context, userID uuid.UUID, from, to *time.Time) ([]weights.Entry, error) {
	var entries []weights.Entry
	for _, e := range f.entries {
		if e.userID != userID {
			continue
		}
		if from != nil && e.RecordedAt.Before(*from) {
			continue
		}
		if to != nil && e.RecordedAt.After(*to) {
			continue
		}
		entries = append(entries, e.Entry)
	}
	return entries, nil
}

func (f *fakeWeightsService) ListUnsynced(_ context.Context, userID uuid.UUID) ([]weights.Entry, error) {
	var entries []weights.Entry
	for _, e := range f.entries {
		if e.userID != userID || e.Source != "manual" {
			continue
		}
		if e.GoogleSyncStatus != nil && *e.GoogleSyncStatus == "synced" {
			continue
		}
		entries = append(entries, e.Entry)
	}
	return entries, nil
}

func (f *fakeWeightsService) Create(_ context.Context, userID uuid.UUID, weightKg float64, recordedAt time.Time) (weights.Entry, error) {
	now := time.Now()
	bmiValue, heightUsedCm := f.bmiAndHeight(weightKg)
	entry := weights.Entry{
		ID:           uuid.New(),
		WeightKg:     weightKg,
		RecordedAt:   recordedAt,
		BMI:          bmiValue,
		HeightUsedCm: heightUsedCm,
		Source:       "manual",
		CreatedAt:    now,
		UpdatedAt:    now,
	}
	f.entries[entry.ID] = weightEntryWithUser{Entry: entry, userID: userID}
	return entry, nil
}

func (f *fakeWeightsService) Update(_ context.Context, userID, id uuid.UUID, weightKg float64, recordedAt time.Time) (weights.Entry, error) {
	existing, ok := f.entries[id]
	if !ok || existing.userID != userID {
		return weights.Entry{}, weights.ErrNotFound
	}
	bmiValue, heightUsedCm := f.bmiAndHeight(weightKg)
	existing.WeightKg = weightKg
	existing.RecordedAt = recordedAt
	existing.BMI = bmiValue
	existing.HeightUsedCm = heightUsedCm
	existing.UpdatedAt = time.Now()
	f.entries[id] = existing
	return existing.Entry, nil
}

func (f *fakeWeightsService) UpdateGoogleSync(_ context.Context, userID, id uuid.UUID, dataPointID *string, status string) (weights.Entry, error) {
	existing, ok := f.entries[id]
	if !ok || existing.userID != userID {
		return weights.Entry{}, weights.ErrNotFound
	}
	existing.GoogleDataPointID = dataPointID
	syncStatus := status
	existing.GoogleSyncStatus = &syncStatus
	f.entries[id] = existing
	return existing.Entry, nil
}

func (f *fakeWeightsService) Delete(_ context.Context, userID, id uuid.UUID) error {
	existing, ok := f.entries[id]
	if !ok || existing.userID != userID {
		return weights.ErrNotFound
	}
	delete(f.entries, id)
	return nil
}

func (f *fakeWeightsService) Get(_ context.Context, userID, id uuid.UUID) (weights.Entry, error) {
	existing, ok := f.entries[id]
	if !ok || existing.userID != userID {
		return weights.Entry{}, weights.ErrNotFound
	}
	return existing.Entry, nil
}

func (f *fakeWeightsService) Latest(_ context.Context, userID uuid.UUID) (weights.Entry, error) {
	var latest weights.Entry
	found := false
	for _, e := range f.entries {
		if e.userID != userID {
			continue
		}
		if !found || e.RecordedAt.After(latest.RecordedAt) {
			latest = e.Entry
			found = true
		}
	}
	if !found {
		return weights.Entry{}, weights.ErrNotFound
	}
	return latest, nil
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
