# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project overview

Massa is a personal weight/BMI tracker that syncs with the Google Health API.
It stores weight history in Postgres, computes BMI locally, charts trends,
and (where connected) syncs weight/height entries to and from Google Health.
Access is restricted to a small allowlist of emails via passwordless
magic-link login.

- **Backend**: Go, chi router, pgx v5/pgxpool, sqlc-generated queries,
  golang-migrate with embedded migrations
- **Frontend**: Nuxt 4 SPA (`ssr: false`), Tailwind v4, Pinia, Chart.js
- **Database**: PostgreSQL 16
- **Local dev**: Docker Compose (Postgres + Mailpit + backend + frontend)
- **Production**: Laravel Forge on a VPS — see `docs/DEPLOY.md`

## Commands

### Local stack

```sh
docker compose up --build
```

- Backend: http://localhost:8080 (`/healthz`)
- Frontend: http://localhost:3000
- Mailpit (catches magic-link emails in dev): http://localhost:8025
- Postgres: localhost:5432

### Backend (`backend/`)

```sh
go run ./cmd/server                 # run the API server (applies migrations on startup)
go run ./cmd/migrate                 # apply migrations only (uses DATABASE_URL)
go test ./...                        # all tests
go test ./internal/weights/...       # single package
go test ./internal/weights/ -run TestService_Create_ComputesBMI -v  # single test
go run github.com/golangci/golangci-lint/v2/cmd/golangci-lint@latest run ./...
```

Regenerating sqlc code (**must be run from `backend/`**, where `sqlc.yaml` lives):

```sh
go run github.com/sqlc-dev/sqlc/cmd/sqlc@latest generate
```

Queries live in `db/queries/*.sql`; generated code lands in `internal/db/`.
After adding/changing a query, regenerate and check the generated
`*Params`/`Querier` signatures before wiring up callers.

### Frontend (`frontend/`)

```sh
npm run dev
npm run lint
npm run test          # vitest
npm run build
npx vue-tsc --noEmit -p .nuxt/tsconfig.json   # type-check (run `npx nuxt prepare` first if .nuxt is stale)
```

### Production deploy

Forge runs `scripts/forge-deploy.sh` on push. See `docs/DEPLOY.md`.

## Architecture

### Backend package structure

- `cmd/server` — wires everything together and starts the HTTP server.
  `cmd/migrate` — standalone migration runner.
- `internal/config` — loads/validates env vars into `Config`. Google OAuth
  is optional: enabled only if client ID, secret, and a 32-byte base64
  `OAUTH_TOKEN_ENCRYPTION_KEY` are all set.
- `internal/db` — sqlc-generated code (`db.Querier`, `db.New(pool)`,
  models, per-query `*Params` structs) plus hand-written helpers in
  `convert.go` for converting between Go types and pgx types:
  - `ToUUID`/`FromUUID`, `ToTimestamptz`/`FromTimestamptz`/`ToTimestamptzPtr`
  - `ToNumeric`/`FromNumeric`, `ToNumericPtr`/`FromNumericPtr` (nullable
    `*float64` <-> `pgtype.Numeric`, where `pgtype.Numeric{}` zero value is
    SQL NULL)
- `internal/users` — user/allowlist repository (`Repository` interface +
  `PostgresRepository`). `User` includes `ManualHeightCm *float64` and
  `UnitsPreference string`.
- `internal/auth` — passwordless magic-link auth, session cookies, CSRF
  tokens.
- `internal/mailer` — email delivery via SMTP (Mailpit locally, AWS SES in
  production).
- `internal/bmi` — pure `Calculate(weightKg, heightCm) float64`.
- `internal/heights` — `Resolver.Resolve(ctx, userID)` returns the height
  (cm) to use for BMI: the most recent `height_entries` row, else the
  user's `manual_height_cm`, else `ErrNoHeight`.
- `internal/weights` — `Service` for weight-entry CRUD. BMI and the height
  used are computed once at write time (create/update) via
  `heights.Resolver` and denormalized onto the row — never recomputed
  retroactively. Entries with no resolvable height are stored with NULL
  bmi/height_used_cm.
- `internal/googlehealth` — Google Health OAuth, encrypted credential
  storage (AES-256-GCM via `crypto.go`), and the backfill/sync service that
  pulls weight/height history from the Google Health API.
- `internal/httpapi` — HTTP handlers and route registration
  (`Handler.Register`). Conventions:
  - `writeJSON(w, status, v)` / `writeError(w, status, msg)` (`response.go`)
  - `userFromContext(ctx)` returns `(users.User, bool)`, set by
    `requireAuth` middleware
  - `requireCSRF` enforces the double-submit CSRF cookie pattern on
    state-changing requests
  - Authenticated routes are registered inside
    `r.Group(func(r chi.Router) { r.Use(h.requireAuth); ... })`
  - `/api/google/*` routes are only registered if `GoogleHealthDeps` is
    non-nil (i.e. Google OAuth is configured)

Each internal package that talks to the database defines its own minimal
`Querier` interface — a subset of `db.Querier` covering only the methods it
needs (e.g. `auth.Querier`, `heights.Querier`, `weights.Querier`,
`googlehealth.Querier`). This keeps fakes in tests small.

### Database

Migrations are embedded (`migrations/embed.go`) and applied automatically by
`cmd/server` on startup (and by `cmd/migrate` standalone). Key tables:
`users`, `allowed_users`, `sessions`, `magic_link_tokens`,
`google_oauth_credentials`, `height_entries`, `weight_entries`,
`sync_metadata`. Weight/height entries have a `source` of `manual` or
`google`, with unique indexes to dedupe Google-synced data points (by
`google_data_point_id`, or by `(user_id, recorded_at)` when Google doesn't
provide one).

### Frontend structure

- `app/composables/useApi.ts` — `apiFetch`, the shared fetch wrapper that
  sends session cookies and attaches the CSRF header (from the
  `massa_csrf` cookie) on non-GET requests.
- `app/composables/useBmi.ts` — BMI categorization and metric/imperial unit
  conversions (kg<->lb, cm<->in), shared across pages/components.
- `app/stores/` — Pinia stores: `auth`, `googlehealth`, `weights`,
  `settings`. Stores are auto-imported (`useXStore`); types like
  `WeightEntry`/`Settings`/`UnitsPreference` need explicit `import type`.
- `app/middleware/auth.global.ts` — redirects unauthenticated users to
  `/login`, calling `auth.fetchMe()` once on first navigation.
- `app/components/WeightChart.vue` — Chart.js line chart (via vue-chartjs
  + chartjs-adapter-date-fns) of weight over time, unit-aware.
- `app/pages/index.vue` — dashboard: latest weight/BMI, date-range presets,
  chart, add-entry form, Google sync status banner.
- `app/pages/settings/index.vue` — Google Health connect/disconnect/sync,
  units preference, and manual height override.

## Testing conventions

- Backend tests use hand-written in-memory fakes (`fakes_test.go` in each
  package) implementing that package's `Querier`/`Repository` interfaces —
  no real database or HTTP calls in unit tests.
- `internal/httpapi` tests spin up a `chi.Router` via a local
  `newTestRouter`/`newGoogleAPIServer` helper and exercise it with
  `httptest`; a `login` helper performs the magic-link flow to get session
  + CSRF cookies.
- OAuth-dependent tests (`googlehealth`, `httpapi/google_test.go`) run
  against a local `httptest.Server` by injecting it via the
  `oauth2.HTTPClient` context key, rather than calling real Google
  endpoints.

## Design Context

`PRODUCT.md` (and `DESIGN.md`, once generated) at the repo root define the
frontend's design direction for use with the `/impeccable` skill. Register:
**product**. Personality: calm and quiet — a private journal, not a coach or
leaderboard; numbers presented without judgment, closer to Oura/Whoop's
restrained quantified-self feel than a generic SaaS dashboard or a gamified
fitness app. One accent color, used deliberately. Read `PRODUCT.md` before
making frontend design decisions.
