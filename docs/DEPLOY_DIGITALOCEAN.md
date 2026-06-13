# Deploying Massa on DigitalOcean

Guide for deploying Massa from GitHub onto an Ubuntu DigitalOcean droplet using
Docker Compose and Caddy for HTTPS.

> **Using App Platform instead?** See [DEPLOY_DIGITALOCEAN_APP_PLATFORM.md](./DEPLOY_DIGITALOCEAN_APP_PLATFORM.md) — no droplet, Docker Compose, or Caddy required.

**Repo:** https://github.com/isAdamBailey/massa

## Architecture

```
Internet → Caddy (:443)
              ├─ /api/*, /healthz  →  backend (:8080, internal)
              └─ /*                →  frontend (:3000, internal)
           postgres (internal only, no public port)
```

Massa uses cookie-based auth. The frontend and API must share **one origin**
(e.g. `https://massa.example.com`). Caddy routes `/api/*` to the Go backend and
everything else to the Nuxt frontend.

## Requirements

- Ubuntu 22.04 or 24.04 droplet (2 GB RAM recommended)
- DNS A record pointing your domain at the droplet IP
- [AWS SES](https://aws.amazon.com/ses/) for magic-link email — see [AWS_SES_SETUP.md](./AWS_SES_SETUP.md)
- Optional: Google OAuth credentials — see [GOOGLE_HEALTH_SETUP.md](./GOOGLE_HEALTH_SETUP.md)

---

## Quick reference

```sh
# Production (on the server, from /opt/massa)
docker compose -f docker-compose.yml -f docker-compose.prod.yml up -d --build

# Deploy updates
git pull origin main
docker compose -f docker-compose.yml -f docker-compose.prod.yml up -d --build

# Logs
docker compose -f docker-compose.yml -f docker-compose.prod.yml logs -f --tail=100

# Health check
curl -s https://your-domain.com/healthz
# → {"status":"ok"}
```

---

## Step-by-step deployment

### 1. DNS (DigitalOcean control panel)

1. Go to **Networking → Domains** (or your registrar).
2. Add an **A record**: `@` or a subdomain (e.g. `massa`) → your droplet's public IP.
3. Verify propagation: `dig +short massa.example.com`

### 2. Server setup (SSH)

```sh
apt update && apt upgrade -y
apt install -y git curl ufw

ufw allow OpenSSH
ufw allow 80
ufw allow 443
ufw --force enable
```

### 3. Install Docker

```sh
curl -fsSL https://get.docker.com | sh
usermod -aG docker $USER
```

Log out and back in so the `docker` group applies, then verify:

```sh
docker --version
docker compose version
```

### 4. Clone the repo

```sh
sudo mkdir -p /opt/massa
sudo chown $USER:$USER /opt/massa
cd /opt/massa
git clone https://github.com/isAdamBailey/massa.git .
```

**Private repo:** add a [deploy key](https://docs.github.com/en/authentication/connecting-to-github-with-ssh/managing-deploy-keys) to GitHub, then clone with `git@github.com:isAdamBailey/massa.git`.

### 5. Generate secrets

```sh
echo "COOKIE_SIGNING_SECRET=$(openssl rand -base64 32)"
echo "OAUTH_TOKEN_ENCRYPTION_KEY=$(openssl rand -base64 32)"
echo "POSTGRES_PASSWORD=$(openssl rand -base64 24 | tr -d '/+=' | head -c 32)"
```

Save the output — you need it for `.env`.

### 6. Create production `.env`

```sh
cd /opt/massa
cp .env.example .env
nano .env
```

Set these values for production:

```env
POSTGRES_USER=massa
POSTGRES_PASSWORD=<generated-postgres-password>
POSTGRES_DB=massa

DATABASE_URL=postgres://massa:<generated-postgres-password>@postgres:5432/massa?sslmode=disable

APP_BASE_URL=https://massa.example.com
COOKIE_SIGNING_SECRET=<generated-cookie-secret>
COOKIE_SECURE=true

OAUTH_TOKEN_ENCRYPTION_KEY=<generated-oauth-key>

GOOGLE_OAUTH_CLIENT_ID=<optional>
GOOGLE_OAUTH_CLIENT_SECRET=<optional>
GOOGLE_OAUTH_REDIRECT_URL=https://massa.example.com/api/google/callback

EMAIL_PROVIDER=ses
SES_REGION=us-east-1
SMTP_USERNAME=<ses-smtp-username>
SMTP_PASSWORD=<ses-smtp-password>
MAGIC_LINK_FROM_EMAIL=login@massa.example.com

ALLOWED_EMAILS=you@example.com

NUXT_PUBLIC_API_BASE=https://massa.example.com
```

Replace `massa.example.com` with your domain everywhere.

### 7. Configure Caddy

```sh
cp caddy/Caddyfile.example caddy/Caddyfile
nano caddy/Caddyfile
```

Replace `massa.example.com` with your domain. The example routes `/api/*` and
`/healthz` to the backend and all other paths to the frontend.

### 8. Build and start

```sh
cd /opt/massa
docker compose -f docker-compose.yml -f docker-compose.prod.yml up -d --build
```

Watch startup logs:

```sh
docker compose -f docker-compose.yml -f docker-compose.prod.yml logs -f --tail=50
```

Caddy obtains a Let's Encrypt certificate automatically on first request.

### 9. Verify

```sh
curl -s https://massa.example.com/healthz
```

Expected: `{"status":"ok"}`

Open `https://massa.example.com` in a browser and request a magic link. Check
that the email arrives via AWS SES.

### 10. Enable Docker on boot

```sh
sudo systemctl enable docker
```

Containers use `restart: unless-stopped` in the production compose overlay.

---

## Google OAuth (optional)

If using Google Health sync, set `GOOGLE_OAUTH_CLIENT_ID` and
`GOOGLE_OAUTH_CLIENT_SECRET` in `.env`, then in Google Cloud Console add this
authorized redirect URI:

```
https://massa.example.com/api/google/callback
```

Full setup: [GOOGLE_HEALTH_SETUP.md](./GOOGLE_HEALTH_SETUP.md).

---

## Deploying updates

```sh
cd /opt/massa
git pull origin main
docker compose -f docker-compose.yml -f docker-compose.prod.yml up -d --build
```

The backend runs database migrations automatically on startup.

---

## Troubleshooting

| Symptom | Likely cause |
|---------|--------------|
| Login fails / cookies not set | `APP_BASE_URL` or `NUXT_PUBLIC_API_BASE` doesn't match your HTTPS domain; or `COOKIE_SECURE` is not `true` |
| Magic link never arrives | Wrong SES SMTP credentials, unverified `MAGIC_LINK_FROM_EMAIL`, or SES sandbox blocking the recipient |
| 502 from Caddy | Backend or frontend not running — check `docker compose ps` and logs |
| Google OAuth redirect error | Redirect URI in Google Console doesn't exactly match `GOOGLE_OAUTH_REDIRECT_URL` |
| Caddy can't get certificate | DNS not propagated, or ports 80/443 blocked |

Useful commands:

```sh
docker compose -f docker-compose.yml -f docker-compose.prod.yml ps
docker compose -f docker-compose.yml -f docker-compose.prod.yml logs backend
docker compose -f docker-compose.yml -f docker-compose.prod.yml logs frontend
docker compose -f docker-compose.yml -f docker-compose.prod.yml logs caddy
```

---

## Gemini assistant prompt

Copy this into Gemini (e.g. in the browser while on your DigitalOcean droplet
page) for guided, step-by-step help:

```
You are helping me deploy Massa on a fresh Ubuntu DigitalOcean droplet.

Repo: https://github.com/isAdamBailey/massa.git
Docs: docs/DEPLOY_DIGITALOCEAN.md in the repo

Stack: PostgreSQL 16 + Go API + Nuxt 4 SPA, deployed with:
  docker compose -f docker-compose.yml -f docker-compose.prod.yml up -d --build

Architecture: Caddy reverse proxy on one domain —
  /api/* and /healthz → backend:8080
  /* → frontend:3000
  Postgres is internal only.

My domain: [YOUR DOMAIN]
My allowed login email: [YOUR EMAIL]

Walk me through deployment one step at a time. Give exact Ubuntu shell commands,
tell me what output to expect, and wait for me to confirm before the next step.
If a command fails, ask for the full error output.

Key production .env values:
- APP_BASE_URL=https://[YOUR DOMAIN]
- NUXT_PUBLIC_API_BASE=https://[YOUR DOMAIN]
- COOKIE_SECURE=true
- EMAIL_PROVIDER=ses
- SES_REGION=us-east-1
- SMTP_USERNAME / SMTP_PASSWORD (AWS SES SMTP credentials)
- MAGIC_LINK_FROM_EMAIL verified in SES
- GOOGLE_OAUTH_REDIRECT_URL=https://[YOUR DOMAIN]/api/google/callback
```

---

## Files added for production

| File | Purpose |
|------|---------|
| `docker-compose.prod.yml` | Production overlay: Caddy, no public app/db ports, restart policies, skips Mailpit |
| `caddy/Caddyfile.example` | Template reverse-proxy config — copy to `caddy/Caddyfile` on the server |

Local development is unchanged: `docker compose up --build` still starts Mailpit
and exposes ports for local use.
