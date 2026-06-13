package config_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/isAdamBailey/massa/backend/internal/config"
)

// setRequiredEnv sets the environment variables needed for a valid SMTP
// configuration, returning a fresh set the test can mutate.
func setRequiredEnv(t *testing.T) {
	t.Helper()
	t.Setenv("DATABASE_URL", "postgres://localhost/massa")
	t.Setenv("COOKIE_SIGNING_SECRET", "test-secret")
	t.Setenv("ALLOWED_EMAILS", "a@example.com, b@example.com")
	t.Setenv("EMAIL_PROVIDER", "smtp")
	t.Setenv("MAGIC_LINK_FROM_EMAIL", "login@example.com")
	t.Setenv("SMTP_HOST", "localhost")
	t.Setenv("SMTP_PORT", "1025")
}

func TestLoad_Valid(t *testing.T) {
	setRequiredEnv(t)

	cfg, err := config.Load()
	require.NoError(t, err)

	assert.Equal(t, "8080", cfg.Port)
	assert.Equal(t, "postgres://localhost/massa", cfg.DatabaseURL)
	assert.Equal(t, "http://localhost:3000", cfg.AppBaseURL)
	assert.Equal(t, []string{"a@example.com", "b@example.com"}, cfg.AllowedEmails)
	assert.Equal(t, "smtp", cfg.Mailer.Provider)
	assert.Equal(t, "localhost", cfg.Mailer.SMTPHost)
}

func TestLoad_MissingRequired(t *testing.T) {
	for _, name := range []string{
		"DATABASE_URL",
		"COOKIE_SIGNING_SECRET",
		"ALLOWED_EMAILS",
		"MAGIC_LINK_FROM_EMAIL",
	} {
		t.Run(name, func(t *testing.T) {
			setRequiredEnv(t)
			t.Setenv(name, "")

			_, err := config.Load()
			require.Error(t, err)
			assert.Contains(t, err.Error(), name)
		})
	}
}

func TestLoad_ResendProviderRequiresAPIKey(t *testing.T) {
	setRequiredEnv(t)
	t.Setenv("EMAIL_PROVIDER", "resend")
	t.Setenv("RESEND_API_KEY", "")

	_, err := config.Load()
	require.Error(t, err)
	assert.Contains(t, err.Error(), "RESEND_API_KEY")
}

func TestLoad_ResendProviderValid(t *testing.T) {
	setRequiredEnv(t)
	t.Setenv("EMAIL_PROVIDER", "resend")
	t.Setenv("RESEND_API_KEY", "test-key")

	cfg, err := config.Load()
	require.NoError(t, err)
	assert.Equal(t, "resend", cfg.Mailer.Provider)
	assert.Equal(t, "test-key", cfg.Mailer.ResendAPIKey)
}

func TestLoad_InvalidEmailProvider(t *testing.T) {
	setRequiredEnv(t)
	t.Setenv("EMAIL_PROVIDER", "carrier-pigeon")

	_, err := config.Load()
	require.Error(t, err)
	assert.Contains(t, err.Error(), "EMAIL_PROVIDER")
}
