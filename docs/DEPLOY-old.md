# Deploying Massa

Production setup: a **VPS managed by [Laravel Forge](https://forge.laravel.com)**,
PostgreSQL on the same server, Nuxt via PM2, and Go API as a Forge daemon.

**Repo:** https://github.com/isAdamBailey/massa

## Architecture

```
https://massa.example.com
         │
         ▼
      Nginx (Forge-managed)
         ├─ /api/*, /healthz  → 127.0.0.1:8080  (Go daemon)
         └─ /*                → 127.0.0.1:3001  (Nuxt via PM2; port varies)

Postgres → 127.0.0.1 on the VPS
Email    → your existing SMTP provider (e.g. SES)
```

Cookie auth requires **one domain**. Nginx proxies `/api` to Go; everything else
goes to Nuxt. Production builds use same-origin API requests (`/api/...`), so
`NUXT_PUBLIC_API_BASE` is optional when frontend and API share a domain.

---

## 1. Server prerequisites

Use an existing Forge server or provision a new one (e.g. DigitalOcean droplet).

### PostgreSQL

Install Postgres on the VPS if Forge does not provide it (some servers only have
MySQL in the Forge UI):

```sh
sudo apt update
sudo apt install -y postgresql postgresql-contrib
sudo -u postgres createuser --pwprompt massa
sudo -u postgres createdb -O massa massa
sudo -u postgres psql -d massa -c "CREATE EXTENSION IF NOT EXISTS pgcrypto;"
```

Note `DATABASE_URL` for step 3, e.g.
`postgres://massa:PASSWORD@127.0.0.1:5432/massa?sslmode=disable`.

### Go

Forge may not include Go. Install once per server:

```sh
uname -m   # x86_64 → amd64, aarch64 → arm64
cd /tmp
curl -LO https://go.dev/dl/go1.26.4.linux-amd64.tar.gz   # or linux-arm64
sudo rm -rf /usr/local/go
sudo tar -C /usr/local -xzf go1.26.4.linux-amd64.tar.gz
/usr/local/go/bin/go version
```

The deploy script adds `/usr/local/go/bin` to `PATH`.

---

## 2. Create the Forge site

1. **Sites → New Site** → your domain (e.g. `massa.example.com`).
2. Project type: **Nuxt**.
3. **Web directory:** leave blank (repo root). The deploy script builds from the
   monorepo; do not set `frontend`.
4. **Server port:** use an free port (e.g. `3001` if another app uses `3000`).
   Set `NUXT_PORT` in environment to match (see step 3).
5. Connect **GitHub** → `isAdamBailey/massa`, branch `main`.
6. Enable **Push to deploy**.
7. **Deploy script:**

   ```bash
   bash $FORGE_SITE_PATH/scripts/forge-deploy.sh
   ```

8. **SSL** → obtain a Let's Encrypt certificate.

Forge clones the repo under `/home/forge/massa.example.com/` with `current/`
pointing at the active release.

---

## 3. Environment variables

Forge → site → **Environment**. Forge writes these to
`/home/forge/massa.example.com/.env` (site root, not inside each release).

Copy values from `.env.example` for local dev, then swap the following for
production:

| Variable | Local (`.env.example`) | Production (Forge) |
| --- | --- | --- |
| `DATABASE_URL` | `postgres://…@postgres:5432/…` | `postgres://…@127.0.0.1:5432/massa?sslmode=disable` |
| `APP_BASE_URL` | `http://localhost:3000` | `https://massa.example.com` |
| `COOKIE_SIGNING_SECRET` | dev placeholder | `openssl rand -base64 32` |
| `COOKIE_SECURE` | `false` | `true` |
| `GOOGLE_OAUTH_REDIRECT_URL` | `http://localhost:8080/api/google/callback` | `https://massa.example.com/api/google/callback` |
| `EMAIL_PROVIDER` | `smtp` | `ses` |
| `SMTP_HOST` | `mailpit` | omit (auto from `SES_REGION` when using `ses`) |
| `SMTP_PORT` | `1025` | omit (defaults to `587`) |
| `SMTP_USERNAME` | empty | your SMTP username |
| `SMTP_PASSWORD` | empty | your SMTP password |
| `SES_REGION` | empty | your SES region, e.g. `us-west-2` |
| `MAGIC_LINK_FROM_EMAIL` | `login@massa.local` | your verified sender address |
| `ALLOWED_EMAILS` | your email | same — must match what you type at login |
| `NUXT_PUBLIC_API_BASE` | `http://localhost:8080` | omit (same-origin `/api` in production) |

Production-only (not in `.env.example`):

| Variable | Value |
| --- | --- |
| `PORT` | `8080` (Go API — nginx proxies `/api` here) |
| `NUXT_PORT` | Forge site port, e.g. `3001` if `3000` is taken |
| `FORGE_API_DAEMON` | Supervisor name from Forge → Daemons, e.g. `daemon-1234567` |

Do **not** set `SMTP_HOST` to a template like `email-smtp.{SES_REGION}.amazonaws.com`
— either omit it or use the real hostname.

Changing env vars only requires restarting the API daemon — no full redeploy:

```sh
sudo supervisorctl restart daemon-1234567
```

---

## 4. Go API daemon

1. **Server → Daemons → New Daemon**
2. **Command:** `/home/forge/massa.example.com/current/scripts/run-api.sh`
3. **Directory:** `/home/forge/massa.example.com/current`
4. **User:** `forge`

Copy the supervisor name from the daemon page (e.g. `daemon-1234567`, including
the `daemon-` prefix) into `FORGE_API_DAEMON`.

The deploy script runs `git pull`, builds the Go binary, rebuilds Nuxt, reloads
PM2, and restarts this daemon.

---

## 5. Nginx — proxy `/api` to Go

Forge → site → **Nginx** — add **before** the existing `location /` block:

```nginx
location /api/ {
    proxy_pass http://127.0.0.1:8080;
    proxy_http_version 1.1;
    proxy_set_header Host $host;
    proxy_set_header X-Real-IP $remote_addr;
    proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for;
    proxy_set_header X-Forwarded-Proto $scheme;
}

location = /healthz {
    proxy_pass http://127.0.0.1:8080;
    proxy_http_version 1.1;
    proxy_set_header Host $host;
    proxy_set_header X-Real-IP $remote_addr;
    proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for;
    proxy_set_header X-Forwarded-Proto $scheme;
}
```

Ensure `location /` proxies to the Nuxt port (`127.0.0.1:3001` if that is your
`NUXT_PORT`). Save — Forge reloads Nginx.

---

## 6. First deploy

1. **Deploy Now** in Forge (or push to `main` with push-to-deploy enabled).
2. Confirm the log shows Go build, `npm run build`, PM2 start, and daemon restart.
3. Verify:

   ```sh
   curl -s https://massa.example.com/healthz
   curl -I http://127.0.0.1:3001
   pm2 list
   ```

4. Request a magic link and complete login.

---

## 7. Google OAuth (optional)

Add the production redirect URI in Google Cloud Console:

```
https://massa.example.com/api/google/callback
```

See [GOOGLE_HEALTH_SETUP.md](./GOOGLE_HEALTH_SETUP.md).

---

## Troubleshooting

| Symptom | Fix |
|---------|-----|
| 502 on `/` | `pm2 list` — `massa-web` must listen on `NUXT_PORT`. See Nuxt 502 below. |
| 502 on `/api` | Check **Server → Daemons**; run `scripts/run-api.sh` manually for errors |
| Nuxt binds to wrong port | Set `NUXT_PORT=3001` in Forge env (not `PORT` — that is Go on 8080). Restart PM2 or redeploy. |
| Nuxt 502 after deploy | `cd current/frontend && export NUXT_PORT=3001 PORT=3001 && pm2 delete massa-web; pm2 start ecosystem.config.cjs --update-env && pm2 save` |
| Magic link 200, no email | Email not on `ALLOWED_EMAILS`, or SMTP misconfigured — check daemon logs |
| Bad SMTP host | Remove `SMTP_HOST` from env or set the real hostname |
| Login redirect broken | `APP_BASE_URL` must match your HTTPS domain |
| Deploy fails on `go build` | Install Go (see step 1) |
| Daemon restart fails in deploy | Set `FORGE_API_DAEMON` to the full supervisor name (`daemon-1234567`) |

**Diagnose magic links:**

```sh
sudo -u postgres psql -d massa -c \
  "SELECT user_email, created_at FROM magic_link_tokens ORDER BY created_at DESC LIMIT 5;"
sudo -u postgres psql -d massa -c "SELECT email FROM allowed_users;"
sudo supervisorctl tail -100 daemon-1234567
```

No token rows → address not allowed or rate-limited (5/hour). Token rows but no
mail → check SMTP credentials and sender address in Forge env.

---

## Cost

Typical monthly cost for personal use: VPS (~$6–12) and an existing Forge plan.
