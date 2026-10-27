# Massa

[![Laravel Forge Site Deployment Status](https://img.shields.io/endpoint?url=https%3A%2F%2Fforge.laravel.com%2Fsite-badges%2F714b1a56-7744-47a1-9872-c24736452e3f&style=plastic)](https://forge.laravel.com/adam-f6w/stunning-fern-s7f/3463871)

A personal weight/BMI tracker that syncs with the Google Health API.

Massa stores your weight history in its own Postgres database, computes BMI
locally, charts trends over time, and (where supported) syncs weight entries
to and from your Google account. Access is restricted to a small, pre-set
allowlist of email addresses via passwordless magic-link login.

## Stack

- **Backend**: Go, [chi](https://github.com/go-chi/chi), pgx, sqlc, golang-migrate
- **Frontend**: Nuxt 4, Tailwind CSS v4, Pinia, Chart.js
- **Database**: PostgreSQL 18
- **Local dev**: Docker Compose

## Project layout

```
backend/    Go API server (cmd/server, cmd/migrate, internal/...)
frontend/   Nuxt 4 SPA
scripts/    Forge deploy script and API daemon wrapper
docs/       Deployment, SES, and Google Health setup
```

## Deployment

Production is [massa.adambailey.io](https://massa.adambailey.io) on the shared Forge server. A push to `main` deploys it. See [docs/DEPLOY.md](docs/DEPLOY.md), and [docs/GOOGLE_HEALTH_SETUP.md](docs/GOOGLE_HEALTH_SETUP.md) for Google OAuth.

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
