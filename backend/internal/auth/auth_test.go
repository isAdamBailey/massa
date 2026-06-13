package auth_test

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/isAdamBailey/massa/backend/internal/auth"
)

func newService(allowedEmails ...string) (*auth.Service, *fakeQuerier, *fakeUsers, *fakeMailer) {
	q := newFakeQuerier()
	u := newFakeUsers(allowedEmails...)
	m := &fakeMailer{}
	svc := auth.NewService(q, u, m, []byte("test-secret"), false, "https://app.example.com")
	return svc, q, u, m
}

func TestRequestMagicLink_NotAllowed(t *testing.T) {
	svc, _, _, m := newService("allowed@example.com")

	err := svc.RequestMagicLink(t.Context(), "stranger@example.com")
	require.NoError(t, err)
	assert.Empty(t, m.sent)
}

func TestRequestMagicLink_Allowed(t *testing.T) {
	svc, _, _, m := newService("allowed@example.com")

	err := svc.RequestMagicLink(t.Context(), "allowed@example.com")
	require.NoError(t, err)
	require.Len(t, m.sent, 1)
	assert.Equal(t, "allowed@example.com", m.sent[0].to)
	assert.Contains(t, m.sent[0].link, "https://app.example.com/auth/callback?token=")
}

func TestRequestMagicLink_RateLimited(t *testing.T) {
	svc, q, _, m := newService("allowed@example.com")
	q.recentForEmail = 5

	err := svc.RequestMagicLink(t.Context(), "allowed@example.com")
	require.NoError(t, err)
	assert.Empty(t, m.sent)
}

func TestVerifyMagicLink(t *testing.T) {
	svc, _, u, m := newService("allowed@example.com")

	require.NoError(t, svc.RequestMagicLink(t.Context(), "allowed@example.com"))
	require.Len(t, m.sent, 1)

	token := tokenFromLink(t, m.sent[0].link)

	sess, err := svc.VerifyMagicLink(t.Context(), token)
	require.NoError(t, err)
	assert.NotEqual(t, sess.ID.String(), "00000000-0000-0000-0000-000000000000")

	created, err := u.GetByEmail(t.Context(), "allowed@example.com")
	require.NoError(t, err)
	assert.Equal(t, sess.UserID, created.ID)
	require.NotNil(t, created.LastLoginAt)

	// The token can only be used once.
	_, err = svc.VerifyMagicLink(t.Context(), token)
	assert.ErrorIs(t, err, auth.ErrInvalidToken)
}

func TestVerifyMagicLink_InvalidToken(t *testing.T) {
	svc, _, _, _ := newService("allowed@example.com")

	_, err := svc.VerifyMagicLink(t.Context(), "not-a-real-token")
	assert.ErrorIs(t, err, auth.ErrInvalidToken)
}

func TestSessionCookieRoundTrip(t *testing.T) {
	svc, _, _, m := newService("allowed@example.com")

	require.NoError(t, svc.RequestMagicLink(t.Context(), "allowed@example.com"))
	sess, err := svc.VerifyMagicLink(t.Context(), tokenFromLink(t, m.sent[0].link))
	require.NoError(t, err)

	rec := httptest.NewRecorder()
	svc.SetSessionCookie(rec, sess.ID, sess.ExpiresAt)

	req := httptest.NewRequestWithContext(t.Context(), http.MethodGet, "/", nil)
	for _, c := range rec.Result().Cookies() {
		req.AddCookie(c)
	}

	gotID, err := svc.SessionIDFromRequest(req)
	require.NoError(t, err)
	assert.Equal(t, sess.ID, gotID)
}

func TestSessionIDFromRequest_TamperedCookie(t *testing.T) {
	svc, _, _, _ := newService("allowed@example.com")

	req := httptest.NewRequestWithContext(t.Context(), http.MethodGet, "/", nil)
	req.AddCookie(&http.Cookie{Name: auth.SessionCookieName, Value: "not-a-valid-signed-value"})

	_, err := svc.SessionIDFromRequest(req)
	assert.ErrorIs(t, err, auth.ErrInvalidCookie)
}

func TestCSRFValidation(t *testing.T) {
	svc, _, _, _ := newService("allowed@example.com")

	rec := httptest.NewRecorder()
	token, err := svc.IssueCSRFToken(rec)
	require.NoError(t, err)

	req := httptest.NewRequestWithContext(t.Context(), http.MethodPost, "/", nil)
	for _, c := range rec.Result().Cookies() {
		req.AddCookie(c)
	}

	assert.False(t, svc.ValidateCSRF(req), "request without header should fail")

	req.Header.Set("X-CSRF-Token", token)
	assert.True(t, svc.ValidateCSRF(req), "request with matching header should pass")

	req.Header.Set("X-CSRF-Token", "wrong-token")
	assert.False(t, svc.ValidateCSRF(req), "request with mismatched header should fail")
}

// tokenFromLink extracts the raw token query parameter from a magic link URL.
func tokenFromLink(t *testing.T, link string) string {
	t.Helper()
	const prefix = "https://app.example.com/auth/callback?token="
	require.Contains(t, link, prefix)
	return link[len(prefix):]
}
