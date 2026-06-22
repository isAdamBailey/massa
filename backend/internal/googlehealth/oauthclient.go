package googlehealth

import (
	"context"
	"errors"

	"github.com/google/uuid"
	"golang.org/x/oauth2"
)

// isReauthError reports whether err is a Google OAuth token-endpoint failure
// indicating the stored refresh token is no longer usable (expired or
// revoked), so the user must reconnect. These surface from the lazy token
// refresh performed during an API request, wrapped in *url.Error and our own
// fmt.Errorf context, so we unwrap via errors.As.
func isReauthError(err error) bool {
	var retrieveErr *oauth2.RetrieveError
	if !errors.As(err, &retrieveErr) {
		return false
	}
	switch retrieveErr.ErrorCode {
	case "invalid_grant", "invalid_client", "unauthorized_client":
		return true
	default:
		return false
	}
}

// newAuthorizedClient returns a Client authenticated as userID against
// apiBaseURL, along with a function that persists any refreshed OAuth
// token. Returns ErrNotConnected if userID has not connected a Google
// Health account.
func newAuthorizedClient(ctx context.Context, credentials CredentialsRepository, oauthConfig *oauth2.Config, userID uuid.UUID, apiBaseURL string) (*Client, func(context.Context) error, error) {
	creds, err := credentials.Get(ctx, userID)
	if err != nil {
		return nil, nil, err
	}

	token := &oauth2.Token{
		RefreshToken: creds.RefreshToken,
		AccessToken:  creds.AccessToken,
	}
	if creds.AccessTokenExpiresAt != nil {
		token.Expiry = *creds.AccessTokenExpiresAt
	}

	tokenSource := oauthConfig.TokenSource(ctx, token)
	client := newClient(oauth2.NewClient(ctx, tokenSource), apiBaseURL)

	persist := func(ctx context.Context) error {
		return persistRefreshedToken(ctx, credentials, userID, creds, tokenSource)
	}

	return client, persist, nil
}

// persistRefreshedToken saves tokenSource's current token if it differs from
// creds, the credentials originally loaded for userID.
func persistRefreshedToken(ctx context.Context, credentials CredentialsRepository, userID uuid.UUID, creds Credentials, tokenSource oauth2.TokenSource) error {
	newToken, err := tokenSource.Token()
	if err != nil {
		return err
	}

	changed := newToken.AccessToken != creds.AccessToken
	if newToken.RefreshToken != "" && newToken.RefreshToken != creds.RefreshToken {
		creds.RefreshToken = newToken.RefreshToken
		changed = true
	}
	if !changed {
		return nil
	}

	creds.AccessToken = newToken.AccessToken
	if !newToken.Expiry.IsZero() {
		expiry := newToken.Expiry
		creds.AccessTokenExpiresAt = &expiry
	}

	return credentials.Save(ctx, userID, creds)
}
