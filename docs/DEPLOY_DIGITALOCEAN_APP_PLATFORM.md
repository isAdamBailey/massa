# Deploying Massa on DigitalOcean App Platform

Deploy from GitHub with a managed PostgreSQL database, two container services
(Go API + Nuxt frontend), and App Platform's built-in HTTPS — no droplet,
Docker Compose, or Caddy required.

**Repo:** https://github.com/isAdamBailey/massa  
**App spec:** `.do/app.yaml`

## Architecture

```
https://your-app.ondigitalocean.app  (or custom domain)
         │
         ▼
   App Platform ingress (TLS terminated)
         ├─ /api/*, /healthz  →  api service (Go, :8080)
         └─ /*                →  web service (Nuxt, :3000)
         
   db (Managed PostgreSQL 16)  →  api only via ${db.DATABASE_URL}
```

Cookie auth requires one public origin. The app spec routes `/api` to the backend
and `/` to the frontend on the **same domain**.

---

## Prerequisites

- DigitalOcean account
- GitHub repo access (`isAdamBailey/massa`)
- [AWS SES](./AWS_SES_SETUP.md) configured for magic-link email
- Optional: Google OAuth — [GOOGLE_HEALTH_SETUP.md](./GOOGLE_HEALTH_SETUP.md)

---

## 1. Create the app

### Option A — Control panel (recommended)

1. Go to [DigitalOcean App Platform](https://cloud.digitalocean.com/apps) → **Create App**.
2. Choose **GitHub** → authorize → select `isAdamBailey/massa`, branch `main`.
3. App Platform should detect `.do/app.yaml`. Confirm the spec shows:
   - **api** service (`backend/`, Dockerfile, port 8080)
   - **web** service (`frontend/`, Dockerfile, port 3000)
   - **db** database (PostgreSQL 16)
   - Ingress rules for `/api`, `/healthz`, `/`
4. Choose a region (match the spec or edit `region:` in `.do/app.yaml`).
5. **Do not deploy yet** — set secrets first (step 2).

### Option B — CLI

```sh
doctl apps create --spec .do/app.yaml
```

---

## 2. Set secret environment variables

In the App Platform dashboard → your app → **api** component → **Environment Variables**, set these **SECRET** values:

| Variable | How to get it |
|----------|---------------|
| `COOKIE_SIGNING_SECRET` | `openssl rand -base64 32` |
| `OAUTH_TOKEN_ENCRYPTION_KEY` | `openssl rand -base64 32` (required if using Google OAuth) |
| `ALLOWED_EMAILS` | Comma-separated login emails, e.g. `you@example.com` |
| `SES_REGION` | AWS region, e.g. `us-east-1` |
| `SMTP_USERNAME` | AWS SES → SMTP settings |
| `SMTP_PASSWORD` | AWS SES → SMTP settings |
| `MAGIC_LINK_FROM_EMAIL` | Verified SES sender, e.g. `login@yourdomain.com` |

**Optional — Google Health sync** (add to **api** component if needed):

| Variable | Value |
|----------|-------|
| `OAUTH_TOKEN_ENCRYPTION_KEY` | `openssl rand -base64 32` |
| `GOOGLE_OAUTH_CLIENT_ID` | From Google Cloud Console |
| `GOOGLE_OAUTH_CLIENT_SECRET` | From Google Cloud Console |
| `GOOGLE_OAUTH_REDIRECT_URL` | `https://YOUR-APP-URL/api/google/callback` |

**Important:** After the first deploy, copy your app URL from the dashboard
(e.g. `https://massa-xxxxx.ondigitalocean.app`) and set
`GOOGLE_OAUTH_REDIRECT_URL` if using Google OAuth.

Non-secret vars (`APP_BASE_URL`, `DATABASE_URL`, `COOKIE_SECURE`, `EMAIL_PROVIDER`)
are set in `.do/app.yaml` using `${APP_URL}` and `${db.DATABASE_URL}`.

The **web** service needs only `NUXT_PUBLIC_API_BASE=${APP_URL}` (already in spec).

---

## 3. Deploy

Click **Create Resources** / **Deploy**. First build takes several minutes
(Go compile + Nuxt build).

Watch build logs for **api** and **web** components. The **api** service runs
database migrations on startup automatically.

---

## 4. Verify

```sh
curl -s https://YOUR-APP-URL.ondigitalocean.app/healthz
# → {"status":"ok"}
```

Open the app URL in a browser → request a magic link → confirm email via SES.

---

## 5. Custom domain (optional)

1. App Platform → **Settings** → **Domains** → **Add Domain**.
2. Add your domain (e.g. `massa.example.com`).
3. Create the CNAME record DigitalOcean shows at your DNS provider.
4. After the domain is active, `${APP_URL}` updates automatically for new deploys.
5. Update `GOOGLE_OAUTH_REDIRECT_URL` to `https://massa.example.com/api/google/callback`.

---

## 6. Deploy updates

Push to `main` with `deploy_on_push: true` (already in spec), or manually
**Deploy** from the dashboard.

---

## Cost estimate

Typical monthly cost (varies by region):

| Resource | Size |
|----------|------|
| api service | `apps-s-1vcpu-0.5gb` × 1 |
| web service | `apps-s-1vcpu-0.5gb` × 1 |
| Managed Postgres | production tier |

Use `production: false` on the database in `.do/app.yaml` for a cheaper dev
database (not recommended for real data long-term).

---

## Troubleshooting

| Symptom | Fix |
|---------|-----|
| 404 on `/api/...` | Check ingress rules; `/api` rule must have `preserve_path_prefix: true` |
| Login / cookies broken | Confirm `APP_BASE_URL` and `NUXT_PUBLIC_API_BASE` both resolve to `${APP_URL}` |
| API won't start | Check **api** logs; usually missing secret env or bad `DATABASE_URL` |
| Build fails (web) | Check Node build logs; ensure `source_dir: frontend` |
| Build fails (api) | Check Go build logs; ensure `source_dir: backend` |
| No magic link | SES credentials, sandbox mode, or unverified sender — see [AWS_SES_SETUP.md](./AWS_SES_SETUP.md) |
| Google OAuth redirect error | `GOOGLE_OAUTH_REDIRECT_URL` must exactly match Google Console |

Useful dashboard locations: **Activity** (deploy logs), **Console** (runtime logs per component).

---

## Droplet vs App Platform

| | App Platform (this doc) | [Droplet + Docker](./DEPLOY_DIGITALOCEAN.md) |
|--|-------------------------|-----------------------------------------------|
| Ops | Low — no SSH required | You manage the server |
| HTTPS | Built-in | Caddy + Let's Encrypt |
| Database | Managed Postgres add-on | Postgres in Docker or DO Managed DB |
| Cost | Higher at small scale | Usually cheaper on one droplet |
| Best for | Push-to-deploy, minimal ops | Full control, Forge, etc. |

---

## Gemini walkthrough prompt

Paste into Gemini while setting up App Platform:

```
Help me deploy Massa on DigitalOcean App Platform from GitHub.

Repo: https://github.com/isAdamBailey/massa.git
App spec: .do/app.yaml (api + web services + managed Postgres)

Architecture:
- One public URL via App Platform ingress
- /api/* and /healthz → Go backend (backend/Dockerfile)
- /* → Nuxt frontend (frontend/Dockerfile)
- PostgreSQL via ${db.DATABASE_URL}
- HTTPS is automatic (no Caddy/certbot)
- Email via AWS SES (EMAIL_PROVIDER=ses)

My values:
- GitHub repo: isAdamBailey/massa, branch main
- Allowed login email: [YOUR EMAIL]
- SES region: [e.g. us-east-1]
- Custom domain (optional): [YOUR DOMAIN or none yet]

Walk me through ONE step at a time:
1. Creating the app from GitHub in DO control panel
2. Which secret env vars to set on the api component (and how to generate them)
3. Setting GOOGLE_OAUTH_REDIRECT_URL after first deploy
4. Verifying /healthz and magic-link login
5. Adding a custom domain

Wait for my confirmation after each step. If something fails, ask for logs/error text.
Reference docs/DEPLOY_DIGITALOCEAN_APP_PLATFORM.md in the repo.
```
