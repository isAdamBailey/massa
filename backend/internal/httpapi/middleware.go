package httpapi

import (
	"context"
	"net/http"

	"github.com/isAdamBailey/massa/backend/internal/users"
)

type contextKey int

const userContextKey contextKey = iota

// userFromContext returns the authenticated user stored in ctx by
// requireAuth, if any.
func userFromContext(ctx context.Context) (users.User, bool) {
	u, ok := ctx.Value(userContextKey).(users.User)
	return u, ok
}

// requireAuth resolves the session cookie on the request to an authenticated
// user, storing it in the request context. It responds 401 Unauthorized if
// the request has no valid session.
func (h *Handler) requireAuth(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		sessionID, err := h.auth.SessionIDFromRequest(r)
		if err != nil {
			writeError(w, http.StatusUnauthorized, "not authenticated")
			return
		}

		sess, err := h.auth.GetSession(r.Context(), sessionID)
		if err != nil {
			writeError(w, http.StatusUnauthorized, "not authenticated")
			return
		}

		user, err := h.users.GetByID(r.Context(), sess.UserID)
		if err != nil {
			writeError(w, http.StatusUnauthorized, "not authenticated")
			return
		}

		ctx := context.WithValue(r.Context(), userContextKey, user)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

// requireCSRF rejects requests that do not present a valid CSRF token via the
// double-submit cookie pattern. It must run after requireAuth.
func (h *Handler) requireCSRF(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !h.auth.ValidateCSRF(r) {
			writeError(w, http.StatusForbidden, "invalid csrf token")
			return
		}
		next.ServeHTTP(w, r)
	})
}
