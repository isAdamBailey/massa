package httpapi_test

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"golang.org/x/oauth2"

	"github.com/isAdamBailey/massa/backend/internal/auth"
	"github.com/isAdamBailey/massa/backend/internal/db"
	"github.com/isAdamBailey/massa/backend/internal/googlehealth"
	"github.com/isAdamBailey/massa/backend/internal/httpapi"
)

const testHealthUserID = "health-user-123"

// fakeCredentialsRepository is an in-memory implementation of
// googlehealth.CredentialsRepository.
type fakeCredentialsRepository struct {
	creds map[uuid.UUID]googlehealth.Credentials
}

func newFakeCredentialsRepository() *fakeCredentialsRepository {
	return &fakeCredentialsRepository{creds: make(map[uuid.UUID]googlehealth.Credentials)}
}

func (f *fakeCredentialsRepository) Get(_ context.Context, userID uuid.UUID) (googlehealth.Credentials, error) {
	c, ok := f.creds[userID]
	if !ok {
		return googlehealth.Credentials{}, googlehealth.ErrNotConnected
	}
	return c, nil
}

func (f *fakeCredentialsRepository) Save(_ context.Context, userID uuid.UUID, creds googlehealth.Credentials) error {
	f.creds[userID] = creds
	return nil
}

func (f *fakeCredentialsRepository) Delete(_ context.Context, userID uuid.UUID) error {
	delete(f.creds, userID)
	return nil
}

// fakeSyncMetadataRepository is an in-memory implementation of
// googlehealth.SyncMetadataRepository.
type fakeSyncMetadataRepository struct {
	meta map[uuid.UUID]googlehealth.SyncMetadata
}

func newFakeSyncMetadataRepository() *fakeSyncMetadataRepository {
	return &fakeSyncMetadataRepository{meta: make(map[uuid.UUID]googlehealth.SyncMetadata)}
}

func (f *fakeSyncMetadataRepository) GetOrCreate(_ context.Context, userID uuid.UUID) (googlehealth.SyncMetadata, error) {
	return f.meta[userID], nil
}

func (f *fakeSyncMetadataRepository) Update(_ context.Context, userID uuid.UUID, meta googlehealth.SyncMetadata) error {
	f.meta[userID] = meta
	return nil
}

// fakeGoogleQuerier is a minimal in-memory implementation of
// googlehealth.Querier, used only to satisfy BackfillService's dependency
// on weight/height upserts. The credential and sync-metadata methods are
// unused because the BackfillService is given separate fake repositories.
type fakeGoogleQuerier struct{}

func (fakeGoogleQuerier) GetGoogleOAuthCredentialsByUserID(context.Context, pgtype.UUID) (db.GoogleOauthCredential, error) {
	return db.GoogleOauthCredential{}, errNotImplemented
}

func (fakeGoogleQuerier) UpsertGoogleOAuthCredentials(context.Context, db.UpsertGoogleOAuthCredentialsParams) (db.GoogleOauthCredential, error) {
	return db.GoogleOauthCredential{}, errNotImplemented
}

func (fakeGoogleQuerier) DeleteGoogleOAuthCredentials(context.Context, pgtype.UUID) error {
	return errNotImplemented
}

func (fakeGoogleQuerier) UpsertSyncMetadata(context.Context, pgtype.UUID) (db.SyncMetadatum, error) {
	return db.SyncMetadatum{}, errNotImplemented
}

func (fakeGoogleQuerier) UpdateSyncWatermarks(context.Context, db.UpdateSyncWatermarksParams) error {
	return errNotImplemented
}

func (fakeGoogleQuerier) UpsertWeightEntryByGoogleID(_ context.Context, arg db.UpsertWeightEntryByGoogleIDParams) (db.WeightEntry, error) {
	return db.WeightEntry{UserID: arg.UserID, WeightKg: arg.WeightKg, RecordedAt: arg.RecordedAt, GoogleDataPointID: arg.GoogleDataPointID}, nil
}

func (fakeGoogleQuerier) UpsertWeightEntryByRecordedAt(_ context.Context, arg db.UpsertWeightEntryByRecordedAtParams) (db.WeightEntry, error) {
	return db.WeightEntry{UserID: arg.UserID, WeightKg: arg.WeightKg, RecordedAt: arg.RecordedAt}, nil
}

func (fakeGoogleQuerier) UpsertHeightEntryByGoogleID(_ context.Context, arg db.UpsertHeightEntryByGoogleIDParams) (db.HeightEntry, error) {
	return db.HeightEntry{UserID: arg.UserID, HeightCm: arg.HeightCm, RecordedAt: arg.RecordedAt, GoogleDataPointID: arg.GoogleDataPointID}, nil
}

func (fakeGoogleQuerier) UpsertHeightEntryByRecordedAt(_ context.Context, arg db.UpsertHeightEntryByRecordedAtParams) (db.HeightEntry, error) {
	return db.HeightEntry{UserID: arg.UserID, HeightCm: arg.HeightCm, RecordedAt: arg.RecordedAt}, nil
}

var errNotImplemented = errors.New("not implemented")

// newGoogleAPIServer starts a test server that stands in for both Google's
// OAuth token endpoint and the Google Health API. Requests to
// "https://health.googleapis.com/..." are rewritten (via rewriteTransport)
// to this server's "/v4/..." paths, while the BackfillService is pointed
// directly at the server's root for its dataPoints requests.
func newGoogleAPIServer(t *testing.T) *httptest.Server {
	t.Helper()

	mux := http.NewServeMux()
	mux.HandleFunc("/token", func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"access_token":"test-access-token","refresh_token":"test-refresh-token","token_type":"Bearer","expires_in":3600}`))
	})
	mux.HandleFunc("/v4/users/me/identity", func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"name":"users/` + testHealthUserID + `","healthUserId":"` + testHealthUserID + `"}`))
	})
	mux.HandleFunc("/users/"+testHealthUserID+"/dataTypes/weight/dataPoints", func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"dataPoints":[]}`))
	})
	mux.HandleFunc("/users/"+testHealthUserID+"/dataTypes/height/dataPoints", func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"dataPoints":[]}`))
	})

	srv := httptest.NewServer(mux)
	t.Cleanup(srv.Close)
	return srv
}

// rewriteTransport redirects every request to base's scheme and host,
// preserving the original path. It is used so that requests the production
// code sends to the real Google Health API ("https://health.googleapis.com")
// land on a local test server instead.
type rewriteTransport struct {
	base *url.URL
}

func (t *rewriteTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	out := req.Clone(req.Context())
	out.URL.Scheme = t.base.Scheme
	out.URL.Host = t.base.Host
	return http.DefaultTransport.RoundTrip(out)
}

// googleTestEnv bundles everything needed to exercise the /api/google/*
// routes against a fake Google backend.
type googleTestEnv struct {
	router      chi.Router
	mailer      *fakeMailer
	users       *fakeUsers
	credentials *fakeCredentialsRepository
	syncMeta    *fakeSyncMetadataRepository
	apiServer   *httptest.Server
}

func newGoogleTestEnv(t *testing.T) *googleTestEnv {
	t.Helper()

	q := newFakeQuerier()
	u := newFakeUsers(allowedEmail)
	m := &fakeMailer{}
	svc := auth.NewService(q, u, m, []byte("test-secret"), false, "http://localhost:3000")

	apiServer := newGoogleAPIServer(t)

	oauthConfig := &oauth2.Config{
		ClientID:     "test-client",
		ClientSecret: "test-secret",
		RedirectURL:  "http://localhost:3000/api/google/callback",
		Scopes:       googlehealth.Scopes,
		Endpoint:     oauth2.Endpoint{TokenURL: apiServer.URL + "/token"},
	}

	credentials := newFakeCredentialsRepository()
	syncMeta := newFakeSyncMetadataRepository()
	backfill := googlehealth.NewBackfillServiceForTest(fakeGoogleQuerier{}, credentials, syncMeta, oauthConfig, apiServer.URL)

	google := &httpapi.GoogleHealthDeps{
		OAuthConfig: oauthConfig,
		Credentials: credentials,
		SyncMeta:    syncMeta,
		Backfill:    backfill,
	}

	r := chi.NewRouter()
	httpapi.NewHandler(svc, u, newFakeWeightsService(), false, "http://localhost:3000", google).Register(r)

	return &googleTestEnv{router: r, mailer: m, users: u, credentials: credentials, syncMeta: syncMeta, apiServer: apiServer}
}

// rewriteClient returns an *http.Client whose requests are redirected to the
// env's fake Google API server, for use as the oauth2.HTTPClient in a
// request context.
func (e *googleTestEnv) rewriteClient(t *testing.T) *http.Client {
	t.Helper()

	target, err := url.Parse(e.apiServer.URL)
	require.NoError(t, err)

	return &http.Client{Transport: &rewriteTransport{base: target}}
}

func TestGoogleAuthURL_RequiresAuth(t *testing.T) {
	env := newGoogleTestEnv(t)

	rec := doRequest(t, env.router, http.MethodGet, "/api/google/auth-url", "", nil, nil)

	assert.Equal(t, http.StatusUnauthorized, rec.Code)
}

func TestGoogleAuthURL(t *testing.T) {
	env := newGoogleTestEnv(t)
	sessionCookie, _ := login(t, env.router, env.mailer, allowedEmail)

	rec := doRequest(t, env.router, http.MethodGet, "/api/google/auth-url", "", []*http.Cookie{sessionCookie}, nil)
	require.Equal(t, http.StatusOK, rec.Code)

	var body struct {
		URL string `json:"url"`
	}
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &body))

	authURL, err := url.Parse(body.URL)
	require.NoError(t, err)
	q := authURL.Query()
	assert.Equal(t, "test-client", q.Get("client_id"))
	assert.Equal(t, "offline", q.Get("access_type"))
	assert.Equal(t, "consent", q.Get("prompt"))
	require.NotEmpty(t, q.Get("state"))

	var stateCookie *http.Cookie
	for _, c := range rec.Result().Cookies() {
		if c.Name == "massa_google_oauth_state" {
			stateCookie = c
		}
	}
	require.NotNil(t, stateCookie)
	assert.Equal(t, q.Get("state"), stateCookie.Value)
}

func TestGoogleCallback_MissingStateCookie(t *testing.T) {
	env := newGoogleTestEnv(t)
	sessionCookie, _ := login(t, env.router, env.mailer, allowedEmail)

	rec := doRequest(t, env.router, http.MethodGet, "/api/google/callback?code=test-code&state=abc", "", []*http.Cookie{sessionCookie}, nil)

	assert.Equal(t, http.StatusBadRequest, rec.Code)
}

func TestGoogleCallback_StateMismatch(t *testing.T) {
	env := newGoogleTestEnv(t)
	sessionCookie, _ := login(t, env.router, env.mailer, allowedEmail)

	stateCookie := &http.Cookie{Name: "massa_google_oauth_state", Value: "expected-state"}
	rec := doRequest(t, env.router, http.MethodGet, "/api/google/callback?code=test-code&state=wrong-state", "", []*http.Cookie{sessionCookie, stateCookie}, nil)

	assert.Equal(t, http.StatusBadRequest, rec.Code)
}

func TestGoogleCallback_Success(t *testing.T) {
	env := newGoogleTestEnv(t)
	sessionCookie, _ := login(t, env.router, env.mailer, allowedEmail)

	state := "test-state"
	stateCookie := &http.Cookie{Name: "massa_google_oauth_state", Value: state}

	ctx := context.WithValue(t.Context(), oauth2.HTTPClient, env.rewriteClient(t))
	req := httptest.NewRequestWithContext(ctx, http.MethodGet, "/api/google/callback?code=test-code&state="+state, nil)
	req.AddCookie(sessionCookie)
	req.AddCookie(stateCookie)

	rec := httptest.NewRecorder()
	env.router.ServeHTTP(rec, req)

	require.Equal(t, http.StatusFound, rec.Code)
	assert.Equal(t, "http://localhost:3000/settings?google=connected", rec.Header().Get("Location"))

	for _, c := range rec.Result().Cookies() {
		if c.Name == "massa_google_oauth_state" {
			assert.Negative(t, c.MaxAge)
		}
	}

	user, err := env.users.GetByEmail(t.Context(), allowedEmail)
	require.NoError(t, err)

	creds, err := env.credentials.Get(t.Context(), user.ID)
	require.NoError(t, err)
	assert.Equal(t, testHealthUserID, creds.HealthUserID)
	assert.Equal(t, "test-refresh-token", creds.RefreshToken)
	assert.Equal(t, "test-access-token", creds.AccessToken)

	meta, err := env.syncMeta.GetOrCreate(t.Context(), user.ID)
	require.NoError(t, err)
	require.NotNil(t, meta.LastFullBackfillAt)
}

func TestGoogleStatus_NotConnected(t *testing.T) {
	env := newGoogleTestEnv(t)
	sessionCookie, _ := login(t, env.router, env.mailer, allowedEmail)

	rec := doRequest(t, env.router, http.MethodGet, "/api/google/status", "", []*http.Cookie{sessionCookie}, nil)
	require.Equal(t, http.StatusOK, rec.Code)
	assert.JSONEq(t, `{"connected":false}`, rec.Body.String())
}

func TestGoogleStatus_Connected(t *testing.T) {
	env := newGoogleTestEnv(t)
	sessionCookie, _ := login(t, env.router, env.mailer, allowedEmail)

	user, err := env.users.GetByEmail(t.Context(), allowedEmail)
	require.NoError(t, err)
	require.NoError(t, env.credentials.Save(t.Context(), user.ID, googlehealth.Credentials{
		HealthUserID: testHealthUserID,
		RefreshToken: "refresh-token",
	}))

	rec := doRequest(t, env.router, http.MethodGet, "/api/google/status", "", []*http.Cookie{sessionCookie}, nil)
	require.Equal(t, http.StatusOK, rec.Code)

	var body struct {
		Connected    bool   `json:"connected"`
		HealthUserID string `json:"healthUserId"`
	}
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &body))
	assert.True(t, body.Connected)
	assert.Equal(t, testHealthUserID, body.HealthUserID)
}

func TestGoogleDisconnect(t *testing.T) {
	env := newGoogleTestEnv(t)
	sessionCookie, csrfCookie := login(t, env.router, env.mailer, allowedEmail)

	user, err := env.users.GetByEmail(t.Context(), allowedEmail)
	require.NoError(t, err)
	require.NoError(t, env.credentials.Save(t.Context(), user.ID, googlehealth.Credentials{
		HealthUserID: testHealthUserID,
		RefreshToken: "refresh-token",
	}))

	rec := doRequest(t, env.router, http.MethodPost, "/api/google/disconnect", "", []*http.Cookie{sessionCookie, csrfCookie}, map[string]string{
		"X-CSRF-Token": csrfCookie.Value,
	})
	require.Equal(t, http.StatusOK, rec.Code)

	_, err = env.credentials.Get(t.Context(), user.ID)
	assert.ErrorIs(t, err, googlehealth.ErrNotConnected)
}

func TestGoogleDisconnect_RequiresCSRF(t *testing.T) {
	env := newGoogleTestEnv(t)
	sessionCookie, _ := login(t, env.router, env.mailer, allowedEmail)

	rec := doRequest(t, env.router, http.MethodPost, "/api/google/disconnect", "", []*http.Cookie{sessionCookie}, nil)

	assert.Equal(t, http.StatusForbidden, rec.Code)
}

func TestGoogleSync_NotConnected(t *testing.T) {
	env := newGoogleTestEnv(t)
	sessionCookie, csrfCookie := login(t, env.router, env.mailer, allowedEmail)

	rec := doRequest(t, env.router, http.MethodPost, "/api/google/sync", "", []*http.Cookie{sessionCookie, csrfCookie}, map[string]string{
		"X-CSRF-Token": csrfCookie.Value,
	})

	assert.Equal(t, http.StatusConflict, rec.Code)
}

func TestGoogleSync_Success(t *testing.T) {
	env := newGoogleTestEnv(t)
	sessionCookie, csrfCookie := login(t, env.router, env.mailer, allowedEmail)

	user, err := env.users.GetByEmail(t.Context(), allowedEmail)
	require.NoError(t, err)
	require.NoError(t, env.credentials.Save(t.Context(), user.ID, googlehealth.Credentials{
		HealthUserID: testHealthUserID,
		RefreshToken: "refresh-token",
	}))

	rec := doRequest(t, env.router, http.MethodPost, "/api/google/sync", "", []*http.Cookie{sessionCookie, csrfCookie}, map[string]string{
		"X-CSRF-Token": csrfCookie.Value,
	})

	require.Equal(t, http.StatusOK, rec.Code)

	meta, err := env.syncMeta.GetOrCreate(t.Context(), user.ID)
	require.NoError(t, err)
	require.NotNil(t, meta.LastFullBackfillAt)
}

func TestGoogleRoutes_NotRegisteredWhenDisabled(t *testing.T) {
	r, _, _, _ := newTestRouter(allowedEmail)

	rec := doRequest(t, r, http.MethodGet, "/api/google/status", "", nil, nil)

	assert.Equal(t, http.StatusNotFound, rec.Code)
}
