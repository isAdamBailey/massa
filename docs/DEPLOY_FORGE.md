# Deploying Massa on Laravel Forge

Deploy Massa on a single VPS (e.g. a DigitalOcean droplet provisioned through
Forge) with PostgreSQL on the same server, a Nuxt frontend, and a Go API daemon.

**Repo:** https://github.com/isAdamBailey/massa

## Architecture

```
https://massa.example.com
         │
         ▼
      Nginx (Forge-managed)
         ├─ /api/*, /healthz  → 127.0.0.1:8080  (Go daemon)
         └─ /*                → 127.0.0.1:3000  (Nuxt via PM2)

Postgres → localhost (Forge database on same server)
Email    → AWS SES (see [AWS_SES_SETUP.md](./AWS_SES_SETUP.md))
```

Cookie auth requires **one domain**. Nginx proxies `/api` to the Go backend;
everything else goes to Nuxt.

---

## Auto-deploy on git push

**Yes.** Forge supports **Push to Deploy** when the site is connected to
GitHub, GitLab, or Bitbucket.

1. Connect the site to `isAdamBailey/massa` on branch `main`.
2. In the site → **Apps** tab, ensure **Push to deploy** is enabled (on by
   default for new GitHub sites).
3. Set the **Deploy script** to `bash $FORGE_SITE_PATH/scripts/forge-deploy.sh` (included in
   this repo).
4. Every push to `main` triggers: `git pull` → build Go binary → build Nuxt →
   reload PM2 → restart the API daemon.

You can also deploy manually from the Forge dashboard or via the Forge CLI.

---

## 1. Server and database

1. In Forge, create or connect a server (DigitalOcean, AWS, etc.).
2. **Server → Database → Create database**
   - Name: `massa`
   - User: `massa` (Forge generates a password)
3. Note the connection details for `DATABASE_URL`.
4. After creating the database, enable UUID support (required for migrations):

   ```sh
   sudo -u postgres psql -d massa -c "CREATE EXTENSION IF NOT EXISTS pgcrypto;"
   ```

---

## 2. Create the Nuxt site

1. **Sites → New Site** → your domain (e.g. `massa.example.com`).
2. Choose **Nuxt** as the project type.
3. **Web directory:** leave blank (repo root). Do not set `frontend` — the deploy script builds from the monorepo root.
4. **Server port:** use what Forge assigns (e.g. `3001` if `3000` is taken). Add `NUXT_PORT=3001` to the site environment to match (do **not** reuse `PORT` — that is the Go API on `8080`).
5. Connect **GitHub** → `isAdamBailey/massa`, branch `main`.
6. Enable **Push to deploy**.
7. Set **Deploy Script** to:

   ```bash
   bash $FORGE_SITE_PATH/scripts/forge-deploy.sh
   ```

8. **SSL** → obtain a Let's Encrypt certificate.

The site clones the full monorepo to `/home/forge/massa.example.com`.

---

## 3. Environment variables

In Forge → site → **Environment**, set (do not commit these):

```env
# Postgres (localhost on the Forge server)
DATABASE_URL=postgres://massa:PASSWORD@127.0.0.1:5432/massa?sslmode=disable

PORT=8080
NUXT_PORT=3001

APP_BASE_URL=https://massa.example.com
COOKIE_SIGNING_SECRET=<openssl rand -base64 32>
COOKIE_SECURE=true

EMAIL_PROVIDER=ses
SES_REGION=us-east-1
SMTP_USERNAME=<ses-smtp-username>
SMTP_PASSWORD=<ses-smtp-password>
MAGIC_LINK_FROM_EMAIL=login@yourdomain.com

ALLOWED_EMAILS=you@example.com

NUXT_PUBLIC_API_BASE=https://massa.example.com

# Optional — Google Health
OAUTH_TOKEN_ENCRYPTION_KEY=<openssl rand -base64 32>
GOOGLE_OAUTH_CLIENT_ID=
GOOGLE_OAUTH_CLIENT_SECRET=
GOOGLE_OAUTH_REDIRECT_URL=https://massa.example.com/api/google/callback
```

Forge writes this to the site `.env` file on deploy. The Go daemon reads the
same file via `scripts/run-api.sh`.

---

## 4. Go API daemon

1. **Server → Daemons → New Daemon**
2. **Command:** `/home/forge/massa.example.com/scripts/run-api.sh`
3. **Directory:** `/home/forge/massa.example.com`
4. **User:** `forge`

After creating the daemon, copy its supervisor name from the Forge daemon page
(e.g. `daemon-1234567`) and add to the site **Environment**:

```env
FORGE_API_DAEMON=daemon-1234567
```

Use the full name including the `daemon-` prefix — not the numeric id alone.

The deploy script restarts this daemon after each push.

---

## 5. Nginx — proxy `/api` to Go

In Forge → site → **Nginx**, add this block **before** the existing
`location /` that proxies to Nuxt:

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

Click **Save** — Forge reloads Nginx.

---

## 6. First deploy

1. Click **Deploy Now** in Forge (or push to `main`).
2. Watch the deploy log for Go build, `npm run build`, and PM2 reload.
3. Verify:

   ```sh
   curl -s https://massa.example.com/healthz
   # → {"status":"ok"}
   ```

4. Open the site and test magic-link login.

---

## 7. Google OAuth (optional)

Add the redirect URI in Google Cloud Console:

```
https://massa.example.com/api/google/callback
```

See [GOOGLE_HEALTH_SETUP.md](./GOOGLE_HEALTH_SETUP.md).

---

## Troubleshooting

| Symptom | Fix |
|---------|-----|
| 502 on `/api` | Daemon not running — check **Server → Daemons** logs |
| Login broken | `APP_BASE_URL` / `NUXT_PUBLIC_API_BASE` must match your HTTPS domain |
| Nuxt 502 | `pm2 list` on server; re-run deploy |
| API won't start | SSH in, run `scripts/run-api.sh` manually and read errors |
| No magic link | [AWS_SES_SETUP.md](./AWS_SES_SETUP.md) — credentials, sandbox, verified sender |
| Deploy fails on `go build` | Install Go 1.26 on the server (see below) |

### Install Go on the server (one-time)

SSH in, then:

```sh
uname -m   # x86_64 → amd64, aarch64 → arm64
```

For **amd64** (most droplets):

```sh
cd /tmp
curl -LO https://go.dev/dl/go1.26.4.linux-amd64.tar.gz
sudo rm -rf /usr/local/go
sudo tar -C /usr/local -xzf go1.26.4.linux-amd64.tar.gz
/usr/local/go/bin/go version
```

For **arm64**, use `go1.26.4.linux-arm64.tar.gz` instead.

Then redeploy from Forge.

---

## Cost

Typical monthly cost for personal use:

| Item | Cost |
|------|------|
| DigitalOcean Basic droplet (1–2 GB) | ~$6–12 |
| Forge | your existing plan |
| AWS SES | negligible at low volume |

No managed database or App Platform fees required.
