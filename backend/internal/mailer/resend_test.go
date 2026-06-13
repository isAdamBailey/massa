package mailer_test

import (
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/isAdamBailey/massa/backend/internal/mailer"
)

func TestResendMailer_SendMagicLink(t *testing.T) {
	var gotAuth, gotBody string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotAuth = r.Header.Get("Authorization")
		body, _ := io.ReadAll(r.Body)
		gotBody = string(body)
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	m := mailer.NewResendMailer("test-key", "login@example.com")
	m.SetBaseURL(server.URL)

	err := m.SendMagicLink(t.Context(), "user@example.com", "https://app.example.com/auth/callback?token=abc")
	require.NoError(t, err)

	assert.Equal(t, "Bearer test-key", gotAuth)
	assert.Contains(t, gotBody, "user@example.com")
	assert.Contains(t, gotBody, "https://app.example.com/auth/callback?token=abc")
}

func TestResendMailer_SendMagicLink_ErrorStatus(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
	}))
	defer server.Close()

	m := mailer.NewResendMailer("bad-key", "login@example.com")
	m.SetBaseURL(server.URL)

	err := m.SendMagicLink(t.Context(), "user@example.com", "https://app.example.com/auth/callback?token=abc")
	require.Error(t, err)
}
