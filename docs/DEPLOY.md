# Deploying Massa

Production is a Forge VPS shared with the other Go apps. One Postgres cluster, one Go process per app, and nginx serving the frontend as static files. Platform rules (ports, build lock, backups) are [face-value#18](https://github.com/isAdamBailey/face-value/issues/18). The move onto this server is [massa#15](https://github.com/isAdamBailey/massa/issues/15).

The previous server's PM2 layout is [DEPLOY-old.md](./DEPLOY-old.md).

**Repo:** https://github.com/isAdamBailey/massa
**Domain:** `massa.adambailey.io`
**Go API port:** `8102`

## Architecture

```
https://massa.adambailey.io
         │
         ▼
      Nginx (Forge-managed)
         ├─ /api/*, /healthz  → 127.0.0.1:8102  (Go daemon)
         └─ /*                → frontend/.output/public  (static files)

Postgres → 127.0.0.1:5432, database massa
Email    → SES
Google   → Google Health API (OAuth tokens encrypted in Postgres)
```

Cookie auth needs this one domain. Production API calls are same-origin (`/api/...`), so `NUXT_PUBLIC_API_BASE` is unset and `nuxt generate` bakes in an empty base.

---

## 1. Postgres role and database

Postgres 18 is already installed on the server, listening on localhost. Create the app role and database once. Type the password at the prompt.

```sh
sudo -u postgres createuser --pwprompt massa
sudo -u postgres createdb -O massa massa
sudo -u postgres psql -c "REVOKE CONNECT ON DATABASE massa FROM PUBLIC; GRANT CONNECT ON DATABASE massa TO massa;"
```

`DATABASE_URL` is `postgres://massa:PASSWORD@127.0.0.1:5432/massa?sslmode=disable`.

The owning role creates `pgcrypto` from migration `000001`. No superuser is required at deploy time.

Go is installed at `/usr/local/go` (1.26.x). The deploy script adds it to `PATH`.

---

## 2. Forge site

1. **Sites → New Site** → `massa.adambailey.io`.
2. Project type: Nuxt, set to Static. This gives a zero-downtime site (`releases/` and `current`) with no Node process. Branch `main`. Push-to-deploy is on: a push to `main` deploys this site.
3. Repository: `isAdamBailey/massa`.
4. Replace the deploy script with:

```bash
$CREATE_RELEASE()

cd "$FORGE_RELEASE_DIRECTORY"
bash scripts/forge-deploy.sh

$ACTIVATE_RELEASE()

sudo supervisorctl restart daemon-<id>:*
```

`<id>` is the massa daemon from step 4.

`scripts/forge-deploy.sh` takes `/tmp/node-build.lock`, builds the Go API with `GOFLAGS=-p=2`, runs migrations, and runs `npm run generate`. Nginx serves `frontend/.output/public`. There is no Node process.

---

## 3. Environment

Forge → site → **Environment**. The file is `/home/forge/massa.adambailey.io/.env`.

Copy the old site's environment, then change only:

| Variable | Production |
| --- | --- |
| `DATABASE_URL` | `postgres://massa:PASSWORD@127.0.0.1:5432/massa?sslmode=disable` |
| `PORT` | `8102` |
| `APP_BASE_URL` | `https://massa.adambailey.io` |
| `COOKIE_SECURE` | `true` |
| `GOOGLE_OAUTH_REDIRECT_URL` | `https://massa.adambailey.io/api/google/callback` |
| `EMAIL_PROVIDER` | `ses` |
| `SMTP_PORT` | `2587`. This server cannot open 587 or 465; SES answers on 2587. |

Two secrets must stay the same as the old server:

- `COOKIE_SIGNING_SECRET`, or the existing user is logged out.
- `OAUTH_TOKEN_ENCRYPTION_KEY`, or the stored Google Health tokens cannot be decrypted and Google has to be reconnected.

Remove `FORGE_API_DAEMON` and `NUXT_PORT`. Omit `NUXT_PUBLIC_API_BASE` and `SMTP_HOST` (the host comes from `SES_REGION`).

Changing `.env` only needs a daemon restart: `sudo supervisorctl restart daemon-<id>:*`.

Google Cloud setup is in [GOOGLE_HEALTH_SETUP.md](./GOOGLE_HEALTH_SETUP.md).

---

## 4. Go API daemon

Forge → **Background processes** → Custom.

- Directory: `/home/forge/massa.adambailey.io/current`
- User: `forge`
- Command: `bash scripts/run-api.sh`

`PORT` comes from `.env`. Supervisor starts it after a reboot.

---

## 5. Nginx

Site settings: `root /home/forge/massa.adambailey.io/current/frontend/.output/public;`.

Before the catch-all:

```nginx
location /api/ {
    proxy_pass http://127.0.0.1:8102;
    proxy_http_version 1.1;
    proxy_set_header Host $host;
    proxy_set_header X-Real-IP $remote_addr;
    proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for;
    proxy_set_header X-Forwarded-Proto $scheme;
    proxy_read_timeout 120s;
}

location = /healthz {
    proxy_pass http://127.0.0.1:8102;
}

location /_nuxt/ {
    expires 1y;
    add_header Cache-Control "public, immutable";
    try_files $uri =404;
}

location / {
    try_files $uri $uri/ /200.html;
}
```

`sudo nginx -t`, then reload.

`proxy_read_timeout 120s` covers a long Google Health backfill.

`try_files $uri $uri/` redirects a generated page such as `/auth/callback` to `/auth/callback/`. The auth middleware strips a trailing slash before the public-route check, so the magic link still signs the user in.

---

## 6. Check a deploy

The deploy log should show the build lock, `go build`, migrate, and `npm run generate`, then the daemon restart.

```sh
curl -s 127.0.0.1:8102/healthz
curl -s -H 'Host: massa.adambailey.io' 127.0.0.1/healthz
curl -s -H 'Host: massa.adambailey.io' 127.0.0.1/ | head
```

The first is the daemon. The second is nginx proxying `/healthz`. The third is the generated HTML.

---

## Troubleshooting

| Symptom | Fix |
| --- | --- |
| 502 on `/api` | `sudo supervisorctl status`. Run `bash scripts/run-api.sh` from `current` for the real error. |
| Daemon restart fails | `sudo supervisorctl restart daemon-<id>:*` — the `:*` suffix matches the process group. |
| `bind: address already in use` | `sudo supervisorctl stop daemon-<id>:*` before killing the port. `ss -tlnp` shows what holds `8102`. |
| Login redirect broken | `APP_BASE_URL` must be `https://massa.adambailey.io`. |
| Magic link opens the login screen | Nginx added a trailing slash and the public-route check missed `/auth/callback/`. The auth middleware strips that slash. |
| Magic link never arrives | `SMTP_PORT` must be `2587`. Connections to 587 and 465 time out from this server. |
| Magic link 200, no email | Email not on `ALLOWED_EMAILS`, or SES misconfigured. Check the daemon log. |
| Google sync fails with `decrypt: authentication failed` in the daemon log | `OAUTH_TOKEN_ENCRYPTION_KEY` differs from the key that stored the tokens. Restore the old key, or disconnect and reconnect in Settings. |
| Google callback error | `GOOGLE_OAUTH_REDIRECT_URL` must match the URI in Google Cloud Console. |
| Deploy cannot find `.env` | The file belongs at the site root, `/home/forge/massa.adambailey.io/.env`. |

**Diagnose magic links:**

```sh
sudo -u postgres psql -d massa -c \
  "SELECT user_email, created_at FROM magic_link_tokens ORDER BY created_at DESC LIMIT 5;"
sudo -u postgres psql -d massa -c "SELECT email FROM allowed_users;"
sudo supervisorctl tail -100 daemon-<id>
```

No token rows → address not allowed or rate-limited (5/hour). Token rows but no mail → check SES credentials and the sender address in `.env`.
