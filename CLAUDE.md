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

The Compose `backend` service runs the compiled server only (no Go toolchain).
Run Go commands via the same `golang:1.26-alpine` image the Dockerfile builds
with — do **not** assume a host `go` binary:

```sh
# helper: from repo root
alias massa-go='docker run --rm -v "$PWD/backend:/src" -w /src golang:1.26-alpine'

massa-go go test ./...               # all tests
massa-go go test ./internal/weights/...
massa-go go test ./internal/weights/ -run TestService_Create_ComputesBMI -v
massa-go go run github.com/golangci/golangci-lint/v2/cmd/golangci-lint@latest run ./...
```

The API server itself is the Compose service (`docker compose up`); migrations
apply on startup. Standalone migrate / local `go run ./cmd/server` are only
needed outside Compose.

Regenerating sqlc code (**must be run against `backend/`**, where `sqlc.yaml` lives):

```sh
massa-go go run github.com/sqlc-dev/sqlc/cmd/sqlc@latest generate
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
- `internal/activeenergy` — `Service` for read access to daily active energy
  totals synced in from Google Health (day-keyed, `source` always `google`).
- `internal/overwhelm` — `Service` for a manually logged daily 1-10
  subjective overwhelm rating (`Baseline = 3`). Day-keyed with
  `UNIQUE(user_id, day)`; entry writes go through a single `Upsert` (no
  create/update/delete), so re-logging the same day is a correction, not a
  second reading. Also manages the user-editable tag vocabulary
  (`overwhelm_tags`, `overwhelm_entry_tags`) used to describe *why* a day
  was overwhelming: `ListTags`/`CreateTag`/`RenameTag`/`ArchiveTag`.
  `CreateTag` unarchives-and-renames a matching archived tag instead of
  erroring, so recreating a removed tag reconnects its history. Archiving
  a tag removes it from the picker but leaves it attached to entries
  already tagged with it — tags are never hard-deleted, since that would
  silently rewrite past days. `Upsert`'s `UpsertOverwhelmByDay` query
  attaches/detaches an entry's tags atomically via data-modifying CTEs
  rather than a transaction (no package in this codebase uses `pgx.Tx`).
  `ErrNotFound`/`ErrDuplicateTag` apply to tags, which are addressable by
  id; entries have no such sentinel, since every day is a valid upsert
  target.
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
`active_energy_entries`, `overwhelm_entries`, `overwhelm_tags`,
`overwhelm_entry_tags`, `sync_metadata`. Weight/height entries have a
`source` of `manual` or `google`, with unique indexes to dedupe
Google-synced data points (by `google_data_point_id`, or by
`(user_id, recorded_at)` when Google doesn't provide one). Active energy and
overwhelm entries are day-keyed (`day DATE`) with a `UNIQUE(user_id, day)`
index instead — active energy is Google-only, overwhelm is manual-only.
`overwhelm_tags` has a case-insensitive unique index on `(user_id,
lower(name))` and an `archived_at` column (soft-delete only);
`overwhelm_entry_tags` is the join table, `(entry_id, tag_id)` as its
primary key.

### Frontend structure

- `app/composables/useApi.ts` — `apiFetch`, the shared fetch wrapper that
  sends session cookies and attaches the CSRF header (from the
  `massa_csrf` cookie) on non-GET requests.
- `app/composables/useBmi.ts` — BMI categorization and metric/imperial unit
  conversions (kg<->lb, cm<->in), shared across pages/components.
- `app/stores/` — Pinia stores: `auth`, `googlehealth`, `weights`,
  `activeEnergy`, `overwhelm`, `overwhelmTags`, `settings`. Stores are
  auto-imported (`useXStore`); types like
  `WeightEntry`/`Settings`/`UnitsPreference` need explicit `import type`.
  `overwhelm`'s `saveEntry` replaces (not appends) the local entry for a
  day, matching the API's upsert-by-day semantics — copying `weights.ts`'s
  append-and-sort pattern here would draw a duplicate point until the next
  fetch.
- `app/middleware/auth.global.ts` — redirects unauthenticated users to
  `/login`, calling `auth.fetchMe()` once on first navigation.
- `app/components/SegmentedControl.vue` — shared toggle group with
  `emphasis: 'primary' | 'quiet'`. Primary uses a Graphite track and the
  metric accent for the active option; quiet is trackless Fog/Mist chips
  (no accent) for secondary controls. `stretch` lays primary options out as
  a 2-column grid on small screens and a flex row from `sm` up. `scrollable`
  keeps long quiet option sets (time spans) in a horizontal touch scroller
  so every option stays a ≥44px tap target. All options use `min-h-11`.
- `app/components/MetricChart.vue` — Chart.js chart (via vue-chartjs +
  chartjs-adapter-date-fns). Control hierarchy: primary metric switcher
  (weight/BMI/energy/overwhelm, accent-aware) on its own row; quiet
  daily/weekly aggregation beside a `#range` slot for the parent’s time-span
  control. Unit-aware. In overwhelm mode the tooltip footer shows that day's
  tags (daily) or the week's top 3 tags by frequency (weekly), computed
  client-side from already-fetched entries.
- `app/composables/useOverwhelm.ts` — `OVERWHELM_BASELINE` (3) and
  `useOverwhelmSummary()`, which averages the current Monday-starting week
  and, when that average is **over** `OVERWHELM_ELEVATED_THRESHOLD` (4),
  returns the top 2 tags by frequency for the dashboard sentence.
- `app/components/LogCard.vue` — tabbed daily-entry card (Weight /
  Overwhelm); defaults to whichever metric isn't logged yet today, since
  the two are typically logged at different times of day. The overwhelm
  tab's tag chips are optional — a bare 1-10 tap-and-save with no tags is
  the fast path.
- `app/pages/index.vue` — dashboard: latest weight + this-week verdict;
  overwhelm block only when current-week avg is **over 4** (cobalt avg +
  top 2 tags). Log card, Trend chart, Google sync status banner. Trend
  time spans are 7d / 30d / 90d / **6m (182 days)** / 1y / all, passed
  into MetricChart via the `#range` slot as a quiet scrollable
  `SegmentedControl`.
- `app/pages/settings/index.vue` — Google Health connect/disconnect/sync,
  units preference, manual height override, and the overwhelm tag
  vocabulary editor (create/rename/archive).

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
