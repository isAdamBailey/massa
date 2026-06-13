// Package httpapi wires up HTTP routes and handlers for the API.
package httpapi

import (
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"

	"github.com/isAdamBailey/massa/backend/internal/auth"
	"github.com/isAdamBailey/massa/backend/internal/users"
)

// magicLinkRateLimit caps how many magic-link requests a single IP address
// may make per minute.
const magicLinkRateLimit = 5

// Handler holds the dependencies needed to serve the API.
type Handler struct {
	auth  *auth.Service
	users users.Repository
}

// NewHandler constructs a Handler.
func NewHandler(authSvc *auth.Service, userRepo users.Repository) *Handler {
	return &Handler{auth: authSvc, users: userRepo}
}

// Register attaches all API routes to the given router.
func (h *Handler) Register(r chi.Router) {
	r.Get("/healthz", healthz)

	magicLinkLimiter := newIPRateLimiter(magicLinkRateLimit, time.Minute)

	r.Route("/api", func(r chi.Router) {
		r.With(magicLinkLimiter.middleware).Post("/auth/magic-link", h.requestMagicLink)
		r.Post("/auth/verify", h.verifyMagicLink)

		r.Group(func(r chi.Router) {
			r.Use(h.requireAuth)
			r.Get("/me", h.me)
			r.With(h.requireCSRF).Post("/auth/logout", h.logout)
		})
	})
}

func healthz(w http.ResponseWriter, _ *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write([]byte(`{"status":"ok"}`))
}
