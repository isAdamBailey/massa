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
	auth         *auth.Service
	users        users.Repository
	weights      WeightsService
	cookieSecure bool
	appBaseURL   string
	google       *GoogleHealthDeps
}

// NewHandler constructs a Handler. google may be nil, in which case the
// /api/google/* routes are not registered.
func NewHandler(authSvc *auth.Service, userRepo users.Repository, weightsSvc WeightsService, cookieSecure bool, appBaseURL string, google *GoogleHealthDeps) *Handler {
	return &Handler{
		auth:         authSvc,
		users:        userRepo,
		weights:      weightsSvc,
		cookieSecure: cookieSecure,
		appBaseURL:   appBaseURL,
		google:       google,
	}
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

			r.Get("/weights", h.listWeights)
			r.With(h.requireCSRF).Post("/weights", h.createWeight)
			r.Get("/weights/{id}", h.getWeight)
			r.With(h.requireCSRF).Patch("/weights/{id}", h.updateWeight)
			r.With(h.requireCSRF).Delete("/weights/{id}", h.deleteWeight)

			r.Get("/settings", h.getSettings)
			r.With(h.requireCSRF).Put("/settings", h.updateSettings)

			r.Get("/bmi/latest", h.bmiLatest)

			if h.google != nil {
				r.Get("/google/auth-url", h.googleAuthURL)
				r.Get("/google/callback", h.googleCallback)
				r.Get("/google/status", h.googleStatus)
				r.With(h.requireCSRF).Post("/google/disconnect", h.googleDisconnect)
				r.With(h.requireCSRF).Post("/google/sync", h.googleSync)
			}
		})
	})
}

func healthz(w http.ResponseWriter, _ *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write([]byte(`{"status":"ok"}`))
}
