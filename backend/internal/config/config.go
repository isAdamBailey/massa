// Package config loads and validates application configuration from
// environment variables.
package config

import (
	"encoding/base64"
	"fmt"
	"os"
	"strings"

	"github.com/isAdamBailey/massa/backend/internal/mailer"
)

// Config holds all runtime configuration for the server.
type Config struct {
	Port                string
	DatabaseURL         string
	AppBaseURL          string
	CookieSigningSecret []byte
	CookieSecure        bool
	AllowedEmails       []string
	Mailer              mailer.Config
	GoogleOAuth         GoogleOAuthConfig
}

// GoogleOAuthConfig holds the settings needed to connect to the Google
// Health API. It is considered Enabled only once a client ID, secret, and
// token encryption key have all been provided.
type GoogleOAuthConfig struct {
	Enabled            bool
	ClientID           string
	ClientSecret       string
	RedirectURL        string
	TokenEncryptionKey []byte
}

// Load reads configuration from environment variables, applying defaults
// where appropriate and returning an error if any required variable is
// missing.
func Load() (Config, error) {
	var missing []string
	require := func(name string) string {
		v := os.Getenv(name)
		if v == "" {
			missing = append(missing, name)
		}
		return v
	}

	cfg := Config{
		Port:                envOrDefault("PORT", "8080"),
		DatabaseURL:         require("DATABASE_URL"),
		AppBaseURL:          envOrDefault("APP_BASE_URL", "http://localhost:3000"),
		CookieSigningSecret: []byte(require("COOKIE_SIGNING_SECRET")),
		CookieSecure:        os.Getenv("COOKIE_SECURE") == "true",
		AllowedEmails:       splitAndTrim(require("ALLOWED_EMAILS")),
		Mailer: mailer.Config{
			Provider:  os.Getenv("EMAIL_PROVIDER"),
			FromEmail: require("MAGIC_LINK_FROM_EMAIL"),
		},
	}

	switch cfg.Mailer.Provider {
	case "resend":
		cfg.Mailer.ResendAPIKey = require("RESEND_API_KEY")
	case "smtp":
		cfg.Mailer.SMTPHost = require("SMTP_HOST")
		cfg.Mailer.SMTPPort = require("SMTP_PORT")
		cfg.Mailer.SMTPUsername = os.Getenv("SMTP_USERNAME")
		cfg.Mailer.SMTPPassword = os.Getenv("SMTP_PASSWORD")
	default:
		return Config{}, fmt.Errorf(`EMAIL_PROVIDER must be "resend" or "smtp", got %q`, cfg.Mailer.Provider)
	}

	if len(missing) > 0 {
		return Config{}, fmt.Errorf("missing required environment variables: %s", strings.Join(missing, ", "))
	}

	googleOAuth, err := loadGoogleOAuth()
	if err != nil {
		return Config{}, err
	}
	cfg.GoogleOAuth = googleOAuth

	return cfg, nil
}

// loadGoogleOAuth reads the Google Health OAuth configuration. It is
// optional: if GOOGLE_OAUTH_CLIENT_ID and GOOGLE_OAUTH_CLIENT_SECRET are both
// unset, GoogleOAuthConfig.Enabled is false and Google Health features are
// disabled. If either is set, both must be set along with a valid
// OAUTH_TOKEN_ENCRYPTION_KEY.
func loadGoogleOAuth() (GoogleOAuthConfig, error) {
	cfg := GoogleOAuthConfig{
		ClientID:     os.Getenv("GOOGLE_OAUTH_CLIENT_ID"),
		ClientSecret: os.Getenv("GOOGLE_OAUTH_CLIENT_SECRET"),
		RedirectURL:  envOrDefault("GOOGLE_OAUTH_REDIRECT_URL", "http://localhost:8080/api/google/callback"),
	}

	if cfg.ClientID == "" && cfg.ClientSecret == "" {
		return cfg, nil
	}

	var missing []string
	if cfg.ClientID == "" {
		missing = append(missing, "GOOGLE_OAUTH_CLIENT_ID")
	}
	if cfg.ClientSecret == "" {
		missing = append(missing, "GOOGLE_OAUTH_CLIENT_SECRET")
	}

	keyB64 := os.Getenv("OAUTH_TOKEN_ENCRYPTION_KEY")
	if keyB64 == "" {
		missing = append(missing, "OAUTH_TOKEN_ENCRYPTION_KEY")
	}
	if len(missing) > 0 {
		return GoogleOAuthConfig{}, fmt.Errorf("missing required environment variables: %s", strings.Join(missing, ", "))
	}

	key, err := base64.StdEncoding.DecodeString(keyB64)
	if err != nil || len(key) != 32 {
		return GoogleOAuthConfig{}, fmt.Errorf("OAUTH_TOKEN_ENCRYPTION_KEY must be a base64-encoded 32-byte key")
	}

	cfg.TokenEncryptionKey = key
	cfg.Enabled = true

	return cfg, nil
}

func envOrDefault(name, def string) string {
	if v := os.Getenv(name); v != "" {
		return v
	}
	return def
}

func splitAndTrim(s string) []string {
	if s == "" {
		return nil
	}
	parts := strings.Split(s, ",")
	out := make([]string, 0, len(parts))
	for _, p := range parts {
		if p = strings.TrimSpace(p); p != "" {
			out = append(out, p)
		}
	}
	return out
}
