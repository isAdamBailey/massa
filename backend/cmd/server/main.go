// Command server runs the Massa HTTP API.
package main

import (
	"context"
	"log"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/cors"

	"github.com/isAdamBailey/massa/backend/internal/auth"
	"github.com/isAdamBailey/massa/backend/internal/config"
	"github.com/isAdamBailey/massa/backend/internal/db"
	"github.com/isAdamBailey/massa/backend/internal/googlehealth"
	"github.com/isAdamBailey/massa/backend/internal/heights"
	"github.com/isAdamBailey/massa/backend/internal/httpapi"
	"github.com/isAdamBailey/massa/backend/internal/mailer"
	"github.com/isAdamBailey/massa/backend/internal/users"
	"github.com/isAdamBailey/massa/backend/internal/weights"
)

func main() {
	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("config: %v", err)
	}

	if err := db.Migrate(cfg.DatabaseURL); err != nil {
		log.Fatalf("migrate: %v", err)
	}

	ctx := context.Background()

	pool, err := db.NewPool(ctx, cfg.DatabaseURL)
	if err != nil {
		log.Fatalf("connect to database: %v", err)
	}
	defer pool.Close()

	queries := db.New(pool)
	userRepo := users.NewPostgresRepository(queries)

	if err := userRepo.SyncAllowlist(ctx, cfg.AllowedEmails); err != nil {
		log.Fatalf("sync allowlist: %v", err)
	}

	mailSvc, err := mailer.New(cfg.Mailer)
	if err != nil {
		log.Fatalf("mailer: %v", err)
	}

	authSvc := auth.NewService(queries, userRepo, mailSvc, cfg.CookieSigningSecret, cfg.CookieSecure, cfg.AppBaseURL)

	heightResolver := heights.NewResolver(queries)
	weightsSvc := weights.NewService(queries, heightResolver)

	var googleDeps *httpapi.GoogleHealthDeps
	if cfg.GoogleOAuth.Enabled {
		oauthConfig := googlehealth.OAuthConfig(cfg.GoogleOAuth)
		credentials := googlehealth.NewPostgresCredentialsRepository(queries, cfg.GoogleOAuth.TokenEncryptionKey)
		syncMeta := googlehealth.NewPostgresSyncMetadataRepository(queries)
		googleDeps = &httpapi.GoogleHealthDeps{
			OAuthConfig: oauthConfig,
			Credentials: credentials,
			SyncMeta:    syncMeta,
			Backfill:    googlehealth.NewBackfillService(queries, credentials, syncMeta, oauthConfig),
		}
	}

	r := chi.NewRouter()
	r.Use(middleware.Logger)
	r.Use(middleware.Recoverer)
	r.Use(cors.Handler(cors.Options{
		AllowedOrigins:   []string{cfg.AppBaseURL},
		AllowedMethods:   []string{http.MethodGet, http.MethodPost, http.MethodPatch, http.MethodPut, http.MethodDelete, http.MethodOptions},
		AllowedHeaders:   []string{"Content-Type", "X-CSRF-Token"},
		AllowCredentials: true,
		MaxAge:           300,
	}))

	httpapi.NewHandler(authSvc, userRepo, weightsSvc, cfg.CookieSecure, cfg.AppBaseURL, googleDeps).Register(r)

	log.Printf("listening on :%s", cfg.Port)
	if err := http.ListenAndServe(":"+cfg.Port, r); err != nil {
		log.Fatalf("server: %v", err)
	}
}
