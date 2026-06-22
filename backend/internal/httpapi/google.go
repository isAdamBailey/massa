package httpapi

import (
	"crypto/rand"
	"encoding/base64"
	"errors"
	"log"
	"net/http"
	"time"

	"golang.org/x/oauth2"

	"github.com/isAdamBailey/massa/backend/internal/googlehealth"
)

// googleOAuthStateCookie holds a random value used to verify that an OAuth
// callback corresponds to a connect attempt initiated by this session.
const googleOAuthStateCookie = "massa_google_oauth_state"

// GoogleHealthDeps bundles the dependencies needed to serve the Google
// Health connect flow. If nil, the /api/google/* routes are not registered.
type GoogleHealthDeps struct {
	OAuthConfig *oauth2.Config
	Credentials googlehealth.CredentialsRepository
	SyncMeta    googlehealth.SyncMetadataRepository
	Backfill    *googlehealth.BackfillService
	Push        *googlehealth.PushService
}

// authURLResponse is the JSON body returned by GET /api/google/auth-url.
type authURLResponse struct {
	URL string `json:"url"`
}

// googleAuthURL returns a Google OAuth consent URL and sets a state cookie
// used to verify the subsequent callback.
func (h *Handler) googleAuthURL(w http.ResponseWriter, _ *http.Request) {
	state, err := randomToken()
	if err != nil {
		log.Printf("httpapi: generate oauth state: %v", err)
		writeError(w, http.StatusInternalServerError, "internal error")
		return
	}

	http.SetCookie(w, &http.Cookie{
		Name:     googleOAuthStateCookie,
		Value:    state,
		Path:     "/api/google",
		MaxAge:   600,
		HttpOnly: true,
		Secure:   h.cookieSecure,
		SameSite: http.SameSiteLaxMode,
	})

	url := h.google.OAuthConfig.AuthCodeURL(state, oauth2.AccessTypeOffline, oauth2.SetAuthURLParam("prompt", "consent"))
	writeJSON(w, http.StatusOK, authURLResponse{URL: url})
}

// googleCallback exchanges the OAuth code for tokens, fetches the user's
// Google Health identity, stores the credentials, runs an initial backfill,
// and redirects back to the frontend.
func (h *Handler) googleCallback(w http.ResponseWriter, r *http.Request) {
	user, ok := userFromContext(r.Context())
	if !ok {
		writeError(w, http.StatusUnauthorized, "not authenticated")
		return
	}

	stateCookie, err := r.Cookie(googleOAuthStateCookie)
	if err != nil || stateCookie.Value == "" || r.URL.Query().Get("state") != stateCookie.Value {
		writeError(w, http.StatusBadRequest, "invalid oauth state")
		return
	}
	clearGoogleOAuthStateCookie(w, h.cookieSecure)

	code := r.URL.Query().Get("code")
	if code == "" {
		writeError(w, http.StatusBadRequest, "missing code")
		return
	}

	token, err := h.google.OAuthConfig.Exchange(r.Context(), code, oauth2.AccessTypeOffline)
	if err != nil {
		log.Printf("httpapi: exchange google oauth code: %v", err)
		writeError(w, http.StatusBadGateway, "failed to connect google account")
		return
	}

	refreshToken := token.RefreshToken
	if refreshToken == "" {
		if existing, err := h.google.Credentials.Get(r.Context(), user.ID); err == nil {
			refreshToken = existing.RefreshToken
		}
	}
	if refreshToken == "" {
		writeError(w, http.StatusBadGateway, "google did not return a refresh token")
		return
	}

	client := googlehealth.NewClient(h.google.OAuthConfig.Client(r.Context(), token))
	identity, err := client.GetIdentity(r.Context())
	if err != nil {
		log.Printf("httpapi: get google health identity: %v", err)
		writeError(w, http.StatusBadGateway, "failed to connect google account")
		return
	}

	creds := googlehealth.Credentials{
		HealthUserID: identity.HealthUserID,
		RefreshToken: refreshToken,
		AccessToken:  token.AccessToken,
	}
	if !token.Expiry.IsZero() {
		expiry := token.Expiry
		creds.AccessTokenExpiresAt = &expiry
	}

	if err := h.google.Credentials.Save(r.Context(), user.ID, creds); err != nil {
		log.Printf("httpapi: save google credentials: %v", err)
		writeError(w, http.StatusInternalServerError, "internal error")
		return
	}

	if err := h.google.Backfill.Run(r.Context(), user.ID); err != nil {
		log.Printf("httpapi: initial google health backfill: %v", err)
	}

	http.Redirect(w, r, h.appBaseURL+"/settings?google=connected", http.StatusFound)
}

// googleStatusResponse is the JSON body returned by GET /api/google/status.
type googleStatusResponse struct {
	Connected             bool       `json:"connected"`
	HealthUserID          string     `json:"healthUserId,omitempty"`
	LastFullBackfillAt    *time.Time `json:"lastFullBackfillAt,omitempty"`
	LastIncrementalSyncAt *time.Time `json:"lastIncrementalSyncAt,omitempty"`
}

// googleStatus reports whether the current user has connected a Google
// Health account and, if so, when it last synced.
func (h *Handler) googleStatus(w http.ResponseWriter, r *http.Request) {
	user, ok := userFromContext(r.Context())
	if !ok {
		writeError(w, http.StatusUnauthorized, "not authenticated")
		return
	}

	creds, err := h.google.Credentials.Get(r.Context(), user.ID)
	if errors.Is(err, googlehealth.ErrNotConnected) {
		writeJSON(w, http.StatusOK, googleStatusResponse{Connected: false})
		return
	}
	if err != nil {
		log.Printf("httpapi: get google credentials: %v", err)
		writeError(w, http.StatusInternalServerError, "internal error")
		return
	}

	meta, err := h.google.SyncMeta.GetOrCreate(r.Context(), user.ID)
	if err != nil {
		log.Printf("httpapi: get google sync metadata: %v", err)
		writeError(w, http.StatusInternalServerError, "internal error")
		return
	}

	writeJSON(w, http.StatusOK, googleStatusResponse{
		Connected:             true,
		HealthUserID:          creds.HealthUserID,
		LastFullBackfillAt:    meta.LastFullBackfillAt,
		LastIncrementalSyncAt: meta.LastIncrementalSyncAt,
	})
}

// googleDisconnect removes the current user's stored Google credentials.
func (h *Handler) googleDisconnect(w http.ResponseWriter, r *http.Request) {
	user, ok := userFromContext(r.Context())
	if !ok {
		writeError(w, http.StatusUnauthorized, "not authenticated")
		return
	}

	if err := h.google.Credentials.Delete(r.Context(), user.ID); err != nil {
		log.Printf("httpapi: delete google credentials: %v", err)
		writeError(w, http.StatusInternalServerError, "internal error")
		return
	}

	w.WriteHeader(http.StatusOK)
}

// googleSync re-runs the backfill for the current user, picking up any new
// weight or height history from Google.
func (h *Handler) googleSync(w http.ResponseWriter, r *http.Request) {
	user, ok := userFromContext(r.Context())
	if !ok {
		writeError(w, http.StatusUnauthorized, "not authenticated")
		return
	}

	err := h.google.Backfill.Run(r.Context(), user.ID)
	if errors.Is(err, googlehealth.ErrNotConnected) {
		writeError(w, http.StatusConflict, "google account not connected")
		return
	}
	if errors.Is(err, googlehealth.ErrReauthRequired) {
		log.Printf("httpapi: google health sync: reauthorization required: %v", err)
		writeError(w, http.StatusConflict, "reconnect_required")
		return
	}
	if err != nil {
		log.Printf("httpapi: google health sync: %v", err)
		writeError(w, http.StatusBadGateway, "sync failed")
		return
	}

	w.WriteHeader(http.StatusOK)
}

func clearGoogleOAuthStateCookie(w http.ResponseWriter, secure bool) {
	http.SetCookie(w, &http.Cookie{
		Name:     googleOAuthStateCookie,
		Value:    "",
		Path:     "/api/google",
		MaxAge:   -1,
		HttpOnly: true,
		Secure:   secure,
		SameSite: http.SameSiteLaxMode,
	})
}

func randomToken() (string, error) {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return base64.RawURLEncoding.EncodeToString(b), nil
}
