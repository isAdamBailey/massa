package auth

import (
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/base64"
	"errors"
	"net/http"
	"strings"
	"time"

	"github.com/google/uuid"
)

// SessionCookieName is the name of the cookie holding the signed session ID.
const SessionCookieName = "massa_session"

// CSRFCookieName is the name of the cookie holding the CSRF token, readable
// by frontend JavaScript and echoed back in the X-CSRF-Token header on
// state-changing requests (double-submit pattern).
const CSRFCookieName = "massa_csrf"

// ErrInvalidCookie is returned when a signed cookie value fails verification.
var ErrInvalidCookie = errors.New("invalid cookie")

// SetSessionCookie writes a signed, HttpOnly cookie identifying the session.
func (s *Service) SetSessionCookie(w http.ResponseWriter, sessionID uuid.UUID, expiresAt time.Time) {
	http.SetCookie(w, &http.Cookie{
		Name:     SessionCookieName,
		Value:    s.sign(sessionID.String()),
		Path:     "/",
		Expires:  expiresAt,
		HttpOnly: true,
		Secure:   s.cookieSecure,
		SameSite: http.SameSiteLaxMode,
	})
}

// ClearSessionCookie removes the session cookie.
func (s *Service) ClearSessionCookie(w http.ResponseWriter) {
	http.SetCookie(w, &http.Cookie{
		Name:     SessionCookieName,
		Value:    "",
		Path:     "/",
		MaxAge:   -1,
		HttpOnly: true,
		Secure:   s.cookieSecure,
		SameSite: http.SameSiteLaxMode,
	})
}

// SessionIDFromRequest extracts and verifies the session ID from the request
// cookies.
func (s *Service) SessionIDFromRequest(r *http.Request) (uuid.UUID, error) {
	c, err := r.Cookie(SessionCookieName)
	if err != nil {
		return uuid.Nil, ErrInvalidCookie
	}

	raw, err := s.verify(c.Value)
	if err != nil {
		return uuid.Nil, err
	}

	id, err := uuid.Parse(raw)
	if err != nil {
		return uuid.Nil, ErrInvalidCookie
	}

	return id, nil
}

// IssueCSRFToken generates a new CSRF token and writes it as a readable
// cookie. The same value must be echoed back in the X-CSRF-Token header on
// state-changing requests.
func (s *Service) IssueCSRFToken(w http.ResponseWriter) (string, error) {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	token := base64.RawURLEncoding.EncodeToString(b)

	http.SetCookie(w, &http.Cookie{
		Name:     CSRFCookieName,
		Value:    token,
		Path:     "/",
		HttpOnly: false,
		Secure:   s.cookieSecure,
		SameSite: http.SameSiteLaxMode,
	})

	return token, nil
}

// ValidateCSRF reports whether the X-CSRF-Token header matches the CSRF
// cookie.
func (s *Service) ValidateCSRF(r *http.Request) bool {
	cookie, err := r.Cookie(CSRFCookieName)
	if err != nil || cookie.Value == "" {
		return false
	}

	header := r.Header.Get("X-CSRF-Token")
	if header == "" {
		return false
	}

	return subtle.ConstantTimeCompare([]byte(cookie.Value), []byte(header)) == 1
}

// sign returns value with an HMAC-SHA256 signature appended.
func (s *Service) sign(value string) string {
	mac := hmac.New(sha256.New, s.cookieSecret)
	mac.Write([]byte(value))
	sig := base64.RawURLEncoding.EncodeToString(mac.Sum(nil))
	return value + "." + sig
}

// verify checks the signature produced by sign and returns the original
// value.
func (s *Service) verify(signed string) (string, error) {
	idx := strings.LastIndex(signed, ".")
	if idx < 0 {
		return "", ErrInvalidCookie
	}

	value, sigPart := signed[:idx], signed[idx+1:]

	sig, err := base64.RawURLEncoding.DecodeString(sigPart)
	if err != nil {
		return "", ErrInvalidCookie
	}

	mac := hmac.New(sha256.New, s.cookieSecret)
	mac.Write([]byte(value))
	expected := mac.Sum(nil)

	if !hmac.Equal(sig, expected) {
		return "", ErrInvalidCookie
	}

	return value, nil
}
