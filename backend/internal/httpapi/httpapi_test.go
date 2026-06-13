package httpapi_test

import (
	"bytes"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"

	"github.com/go-chi/chi/v5"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/isAdamBailey/massa/backend/internal/auth"
	"github.com/isAdamBailey/massa/backend/internal/httpapi"
)

const allowedEmail = "allowed@example.com"

func newTestRouter(allowedEmails ...string) (chi.Router, *fakeQuerier, *fakeUsers, *fakeMailer) {
	q := newFakeQuerier()
	u := newFakeUsers(allowedEmails...)
	m := &fakeMailer{}

	svc := auth.NewService(q, u, m, []byte("test-secret"), false, "http://localhost:3000")

	r := chi.NewRouter()
	httpapi.NewHandler(svc, u).Register(r)

	return r, q, u, m
}

func doRequest(t *testing.T, r chi.Router, method, path, body string, cookies []*http.Cookie, headers map[string]string) *httptest.ResponseRecorder {
	t.Helper()

	req := httptest.NewRequestWithContext(t.Context(), method, path, bytes.NewBufferString(body))
	for _, c := range cookies {
		req.AddCookie(c)
	}
	for k, v := range headers {
		req.Header.Set(k, v)
	}

	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)
	return rec
}

func TestHealthz(t *testing.T) {
	r, _, _, _ := newTestRouter()

	rec := doRequest(t, r, http.MethodGet, "/healthz", "", nil, nil)

	assert.Equal(t, http.StatusOK, rec.Code)
	assert.JSONEq(t, `{"status":"ok"}`, rec.Body.String())
}

func TestRequestMagicLink_Allowed(t *testing.T) {
	r, _, _, m := newTestRouter(allowedEmail)

	rec := doRequest(t, r, http.MethodPost, "/api/auth/magic-link", `{"email":"`+allowedEmail+`"}`, nil, nil)

	assert.Equal(t, http.StatusOK, rec.Code)
	require.Len(t, m.sent, 1)
	assert.Equal(t, allowedEmail, m.sent[0].to)
}

func TestRequestMagicLink_NotAllowed(t *testing.T) {
	r, _, _, m := newTestRouter(allowedEmail)

	rec := doRequest(t, r, http.MethodPost, "/api/auth/magic-link", `{"email":"someone-else@example.com"}`, nil, nil)

	assert.Equal(t, http.StatusOK, rec.Code)
	assert.Empty(t, m.sent)
}

func TestRequestMagicLink_RateLimited(t *testing.T) {
	r, _, _, _ := newTestRouter(allowedEmail)

	for i := 0; i < 5; i++ {
		rec := doRequest(t, r, http.MethodPost, "/api/auth/magic-link", `{"email":"`+allowedEmail+`"}`, nil, nil)
		require.Equal(t, http.StatusOK, rec.Code)
	}

	rec := doRequest(t, r, http.MethodPost, "/api/auth/magic-link", `{"email":"`+allowedEmail+`"}`, nil, nil)
	assert.Equal(t, http.StatusTooManyRequests, rec.Code)
}

func tokenFromLink(t *testing.T, link string) string {
	t.Helper()

	u, err := url.Parse(link)
	require.NoError(t, err)

	token := u.Query().Get("token")
	require.NotEmpty(t, token)
	return token
}

func TestVerifyMagicLink(t *testing.T) {
	r, _, _, m := newTestRouter(allowedEmail)

	rec := doRequest(t, r, http.MethodPost, "/api/auth/magic-link", `{"email":"`+allowedEmail+`"}`, nil, nil)
	require.Equal(t, http.StatusOK, rec.Code)
	require.Len(t, m.sent, 1)

	token := tokenFromLink(t, m.sent[0].link)

	rec = doRequest(t, r, http.MethodPost, "/api/auth/verify", `{"token":"`+token+`"}`, nil, nil)
	require.Equal(t, http.StatusOK, rec.Code)

	cookies := rec.Result().Cookies()
	var sessionCookie, csrfCookie *http.Cookie
	for _, c := range cookies {
		switch c.Name {
		case auth.SessionCookieName:
			sessionCookie = c
		case auth.CSRFCookieName:
			csrfCookie = c
		}
	}
	require.NotNil(t, sessionCookie)
	require.NotNil(t, csrfCookie)
}

func TestVerifyMagicLink_InvalidToken(t *testing.T) {
	r, _, _, _ := newTestRouter(allowedEmail)

	rec := doRequest(t, r, http.MethodPost, "/api/auth/verify", `{"token":"not-a-real-token"}`, nil, nil)

	assert.Equal(t, http.StatusUnauthorized, rec.Code)
}

func TestVerifyMagicLink_ReuseFails(t *testing.T) {
	r, _, _, m := newTestRouter(allowedEmail)

	rec := doRequest(t, r, http.MethodPost, "/api/auth/magic-link", `{"email":"`+allowedEmail+`"}`, nil, nil)
	require.Equal(t, http.StatusOK, rec.Code)

	token := tokenFromLink(t, m.sent[0].link)

	rec = doRequest(t, r, http.MethodPost, "/api/auth/verify", `{"token":"`+token+`"}`, nil, nil)
	require.Equal(t, http.StatusOK, rec.Code)

	rec = doRequest(t, r, http.MethodPost, "/api/auth/verify", `{"token":"`+token+`"}`, nil, nil)
	assert.Equal(t, http.StatusUnauthorized, rec.Code)
}

// login performs a full magic-link login and returns the resulting session
// and CSRF cookies.
func login(t *testing.T, r chi.Router, m *fakeMailer, email string) (*http.Cookie, *http.Cookie) {
	t.Helper()

	rec := doRequest(t, r, http.MethodPost, "/api/auth/magic-link", `{"email":"`+email+`"}`, nil, nil)
	require.Equal(t, http.StatusOK, rec.Code)
	require.NotEmpty(t, m.sent)

	token := tokenFromLink(t, m.sent[len(m.sent)-1].link)

	rec = doRequest(t, r, http.MethodPost, "/api/auth/verify", `{"token":"`+token+`"}`, nil, nil)
	require.Equal(t, http.StatusOK, rec.Code)

	var sessionCookie, csrfCookie *http.Cookie
	for _, c := range rec.Result().Cookies() {
		switch c.Name {
		case auth.SessionCookieName:
			sessionCookie = c
		case auth.CSRFCookieName:
			csrfCookie = c
		}
	}
	require.NotNil(t, sessionCookie)
	require.NotNil(t, csrfCookie)

	return sessionCookie, csrfCookie
}

func TestMe_RequiresAuth(t *testing.T) {
	r, _, _, _ := newTestRouter(allowedEmail)

	rec := doRequest(t, r, http.MethodGet, "/api/me", "", nil, nil)

	assert.Equal(t, http.StatusUnauthorized, rec.Code)
}

func TestMe_Authenticated(t *testing.T) {
	r, _, _, m := newTestRouter(allowedEmail)

	sessionCookie, _ := login(t, r, m, allowedEmail)

	rec := doRequest(t, r, http.MethodGet, "/api/me", "", []*http.Cookie{sessionCookie}, nil)

	require.Equal(t, http.StatusOK, rec.Code)
	assert.Contains(t, rec.Body.String(), allowedEmail)
}

func TestLogout_RequiresCSRF(t *testing.T) {
	r, _, _, m := newTestRouter(allowedEmail)

	sessionCookie, _ := login(t, r, m, allowedEmail)

	rec := doRequest(t, r, http.MethodPost, "/api/auth/logout", "", []*http.Cookie{sessionCookie}, nil)

	assert.Equal(t, http.StatusForbidden, rec.Code)
}

func TestLogout(t *testing.T) {
	r, _, _, m := newTestRouter(allowedEmail)

	sessionCookie, csrfCookie := login(t, r, m, allowedEmail)

	rec := doRequest(t, r, http.MethodPost, "/api/auth/logout", "", []*http.Cookie{sessionCookie, csrfCookie}, map[string]string{
		"X-CSRF-Token": csrfCookie.Value,
	})
	require.Equal(t, http.StatusOK, rec.Code)

	rec = doRequest(t, r, http.MethodGet, "/api/me", "", []*http.Cookie{sessionCookie}, nil)
	assert.Equal(t, http.StatusUnauthorized, rec.Code)
}
