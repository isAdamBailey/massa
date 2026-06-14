# Massa

A personal weight/BMI tracker that syncs with the Google Health API.

Massa stores your weight history in its own Postgres database, computes BMI
locally, charts trends over time, and (where supported) syncs weight entries
to and from your Google account. Access is restricted to a small, pre-set
allowlist of email addresses via passwordless magic-link login.

## Stack

- **Backend**: Go, [chi](https://github.com/go-chi/chi), pgx, sqlc, golang-migrate
- **Frontend**: Nuxt 4, Tailwind CSS v4, Pinia, Chart.js
- **Database**: PostgreSQL 16
- **Local dev**: Docker Compose

## Project layout

```
backend/    Go API server (cmd/server, cmd/migrate, internal/...)
frontend/   Nuxt 4 PWA
```

## Deployment

See [docs/DEPLOY_FORGE.md](docs/DEPLOY_FORGE.md) for deploying on a VPS with
[Laravel Forge](https://forge.laravel.com) (push-to-deploy from GitHub).

Magic-link email in production uses [AWS SES](docs/AWS_SES_SETUP.md).

## Local development

1. Copy `.env.example` to `.env` and fill in the values (Google OAuth
   credentials, email provider, allowed emails, etc.).
2. Start everything with Docker Compose:

   ```sh
   docker compose up --build
   ```

   - Backend API: http://localhost:8080 (`/healthz` for a liveness check)
   - Frontend: http://localhost:3000
   - Postgres: localhost:5432

### Backend only

```sh
cd backend
go run ./cmd/server
go test ./...
go run github.com/golangci/golangci-lint/v2/cmd/golangci-lint@latest run ./...
```

### Frontend only

```sh
cd frontend
npm install
npm run dev
npm run lint
npm run test
npm run build
```
