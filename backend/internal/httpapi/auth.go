package httpapi

import (
	"encoding/json"
	"errors"
	"log"
	"net/http"
	"strings"

	"github.com/isAdamBailey/massa/backend/internal/auth"
)

// magicLinkRequest is the request body for POST /api/auth/magic-link.
type magicLinkRequest struct {
	Email string `json:"email"`
}

// requestMagicLink emails a sign-in link to the given address if it is
// allowed to sign in. It always responds 200 to avoid leaking allowlist
// membership.
func (h *Handler) requestMagicLink(w http.ResponseWriter, r *http.Request) {
	var req magicLinkRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	email := strings.TrimSpace(strings.ToLower(req.Email))
	if email == "" {
		writeError(w, http.StatusBadRequest, "email is required")
		return
	}

	if err := h.auth.RequestMagicLink(r.Context(), email); err != nil {
		log.Printf("httpapi: request magic link: %v", err)
	}

	w.WriteHeader(http.StatusOK)
}

// verifyRequest is the request body for POST /api/auth/verify.
type verifyRequest struct {
	Token string `json:"token"`
}

// verifyMagicLink exchanges a magic-link token for a session, setting the
// session and CSRF cookies on success.
func (h *Handler) verifyMagicLink(w http.ResponseWriter, r *http.Request) {
	var req verifyRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil || req.Token == "" {
		writeError(w, http.StatusBadRequest, "token is required")
		return
	}

	sess, err := h.auth.VerifyMagicLink(r.Context(), req.Token)
	if errors.Is(err, auth.ErrInvalidToken) {
		writeError(w, http.StatusUnauthorized, "invalid or expired token")
		return
	}
	if err != nil {
		log.Printf("httpapi: verify magic link: %v", err)
		writeError(w, http.StatusInternalServerError, "internal error")
		return
	}

	h.auth.SetSessionCookie(w, sess.ID, sess.ExpiresAt)
	if _, err := h.auth.IssueCSRFToken(w); err != nil {
		log.Printf("httpapi: issue csrf token: %v", err)
		writeError(w, http.StatusInternalServerError, "internal error")
		return
	}

	w.WriteHeader(http.StatusOK)
}

// logout deletes the caller's session and clears the session cookie. It
// requires an authenticated session and a valid CSRF token.
func (h *Handler) logout(w http.ResponseWriter, r *http.Request) {
	sessionID, err := h.auth.SessionIDFromRequest(r)
	if err == nil {
		if err := h.auth.Logout(r.Context(), sessionID); err != nil {
			log.Printf("httpapi: logout: %v", err)
		}
	}

	h.auth.ClearSessionCookie(w)
	w.WriteHeader(http.StatusOK)
}

// meResponse is the JSON body returned by GET /api/me.
type meResponse struct {
	ID    string `json:"id"`
	Email string `json:"email"`
}

// me returns the currently authenticated user and refreshes the CSRF cookie.
func (h *Handler) me(w http.ResponseWriter, r *http.Request) {
	user, ok := userFromContext(r.Context())
	if !ok {
		writeError(w, http.StatusUnauthorized, "not authenticated")
		return
	}

	if _, err := h.auth.IssueCSRFToken(w); err != nil {
		log.Printf("httpapi: issue csrf token: %v", err)
		writeError(w, http.StatusInternalServerError, "internal error")
		return
	}

	writeJSON(w, http.StatusOK, meResponse{ID: user.ID.String(), Email: user.Email})
}
