package googlehealth

import (
	"golang.org/x/oauth2"
	"golang.org/x/oauth2/google"

	"github.com/isAdamBailey/massa/backend/internal/config"
)

// Scopes are the OAuth2 scopes requested when connecting a Google Health
// account, granting read and write access to weight and height
// measurements.
var Scopes = []string{
	"https://www.googleapis.com/auth/googlehealth.health_metrics_and_measurements.readonly",
	"https://www.googleapis.com/auth/googlehealth.health_metrics_and_measurements.writeonly",
}

// OAuthConfig builds an oauth2.Config for the Google Health API from the
// application's Google OAuth settings.
func OAuthConfig(cfg config.GoogleOAuthConfig) *oauth2.Config {
	return &oauth2.Config{
		ClientID:     cfg.ClientID,
		ClientSecret: cfg.ClientSecret,
		RedirectURL:  cfg.RedirectURL,
		Scopes:       Scopes,
		Endpoint:     google.Endpoint,
	}
}
