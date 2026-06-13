package config_test

import (
	"encoding/base64"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/isAdamBailey/massa/backend/internal/config"
)

var validEncryptionKey = base64.StdEncoding.EncodeToString(make([]byte, 32))

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

func TestLoad_SESProviderValid(t *testing.T) {
	setRequiredEnv(t)
	t.Setenv("EMAIL_PROVIDER", "ses")
	t.Setenv("SES_REGION", "us-east-1")
	t.Setenv("SMTP_USERNAME", "ses-user")
	t.Setenv("SMTP_PASSWORD", "ses-pass")
	t.Setenv("SMTP_HOST", "")
	t.Setenv("SMTP_PORT", "")

	cfg, err := config.Load()
	require.NoError(t, err)
	assert.Equal(t, "ses", cfg.Mailer.Provider)
	assert.Equal(t, "email-smtp.us-east-1.amazonaws.com", cfg.Mailer.SMTPHost)
	assert.Equal(t, "587", cfg.Mailer.SMTPPort)
	assert.Equal(t, "ses-user", cfg.Mailer.SMTPUsername)
	assert.Equal(t, "ses-pass", cfg.Mailer.SMTPPassword)
}

func TestLoad_SESProviderRequiresCredentials(t *testing.T) {
	setRequiredEnv(t)
	t.Setenv("EMAIL_PROVIDER", "ses")
	t.Setenv("SES_REGION", "us-east-1")
	t.Setenv("SMTP_USERNAME", "")
	t.Setenv("SMTP_PASSWORD", "")

	_, err := config.Load()
	require.Error(t, err)
	assert.Contains(t, err.Error(), "SMTP_USERNAME")
	assert.Contains(t, err.Error(), "SMTP_PASSWORD")
}

func TestLoad_InvalidEmailProvider(t *testing.T) {
	setRequiredEnv(t)
	t.Setenv("EMAIL_PROVIDER", "carrier-pigeon")

	_, err := config.Load()
	require.Error(t, err)
	assert.Contains(t, err.Error(), "EMAIL_PROVIDER")
}

func TestLoad_GoogleOAuthNotConfigured(t *testing.T) {
	setRequiredEnv(t)

	cfg, err := config.Load()
	require.NoError(t, err)
	assert.False(t, cfg.GoogleOAuth.Enabled)
}

func TestLoad_GoogleOAuthValid(t *testing.T) {
	setRequiredEnv(t)
	t.Setenv("GOOGLE_OAUTH_CLIENT_ID", "client-id")
	t.Setenv("GOOGLE_OAUTH_CLIENT_SECRET", "client-secret")
	t.Setenv("GOOGLE_OAUTH_REDIRECT_URL", "http://localhost:8080/api/google/callback")
	t.Setenv("OAUTH_TOKEN_ENCRYPTION_KEY", validEncryptionKey)

	cfg, err := config.Load()
	require.NoError(t, err)
	assert.True(t, cfg.GoogleOAuth.Enabled)
	assert.Equal(t, "client-id", cfg.GoogleOAuth.ClientID)
	assert.Equal(t, "client-secret", cfg.GoogleOAuth.ClientSecret)
	assert.Equal(t, "http://localhost:8080/api/google/callback", cfg.GoogleOAuth.RedirectURL)
	assert.Len(t, cfg.GoogleOAuth.TokenEncryptionKey, 32)
}

func TestLoad_GoogleOAuthPartialConfig(t *testing.T) {
	setRequiredEnv(t)
	t.Setenv("GOOGLE_OAUTH_CLIENT_ID", "client-id")

	_, err := config.Load()
	require.Error(t, err)
	assert.Contains(t, err.Error(), "GOOGLE_OAUTH_CLIENT_SECRET")
	assert.Contains(t, err.Error(), "OAUTH_TOKEN_ENCRYPTION_KEY")
}

func TestLoad_GoogleOAuthInvalidEncryptionKey(t *testing.T) {
	setRequiredEnv(t)
	t.Setenv("GOOGLE_OAUTH_CLIENT_ID", "client-id")
	t.Setenv("GOOGLE_OAUTH_CLIENT_SECRET", "client-secret")
	t.Setenv("OAUTH_TOKEN_ENCRYPTION_KEY", "not-base64-and-wrong-length")

	_, err := config.Load()
	require.Error(t, err)
	assert.Contains(t, err.Error(), "OAUTH_TOKEN_ENCRYPTION_KEY")
}
