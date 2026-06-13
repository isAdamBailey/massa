# Google Health Integration Setup

Massa can connect to the [Google Health API](https://developers.google.com/health/api)
(v4) to import your historical weight and height data. This is optional — if
`GOOGLE_OAUTH_CLIENT_ID` / `GOOGLE_OAUTH_CLIENT_SECRET` are not set, the
`/api/google/*` routes are disabled and the Settings page will show "Not
connected" with no way to connect.

## 1. Google Cloud project setup

1. Go to the [Google Cloud Console](https://console.cloud.google.com/) and
   create a new project (or select an existing one).
2. Enable the **Google Health API** for the project (search for "Google
   Health API" in "APIs & Services" → "Library").
3. Configure the **OAuth consent screen** ("APIs & Services" → "OAuth consent
   screen"):
   - User type: **External**.
   - Publishing status: **Testing** is fine for a personal deployment.
   - Add the email addresses in your `ALLOWED_EMAILS` list as **test users**
     (only test users can complete the OAuth flow while the app is in
     Testing).
   - Add the following scopes:
     - `https://www.googleapis.com/auth/googlehealth.health_metrics_and_measurements.readonly`
     - `https://www.googleapis.com/auth/googlehealth.health_metrics_and_measurements.writeonly`
4. Create an **OAuth Client ID** ("APIs & Services" → "Credentials" → "Create
   Credentials" → "OAuth client ID"):
   - Application type: **Web application**.
   - Authorized redirect URI: must exactly match `GOOGLE_OAUTH_REDIRECT_URL`,
     e.g. `http://localhost:8080/api/google/callback` for local dev.
5. Copy the generated **Client ID** and **Client secret** into your `.env` as
   `GOOGLE_OAUTH_CLIENT_ID` and `GOOGLE_OAUTH_CLIENT_SECRET`.

## 2. Token encryption key

Refresh and access tokens are stored encrypted at rest (AES-256-GCM). Generate
a 32-byte key and base64-encode it:

```sh
openssl rand -base64 32
```

Put the result in `OAUTH_TOKEN_ENCRYPTION_KEY`.

## 3. Environment variables

| Variable | Description |
| --- | --- |
| `GOOGLE_OAUTH_CLIENT_ID` | OAuth client ID from step 1.4 |
| `GOOGLE_OAUTH_CLIENT_SECRET` | OAuth client secret from step 1.4 |
| `GOOGLE_OAUTH_REDIRECT_URL` | Must match the redirect URI registered in step 1.4 (defaults to `http://localhost:8080/api/google/callback`) |
| `OAUTH_TOKEN_ENCRYPTION_KEY` | Base64-encoded 32-byte AES key from step 2 |

All four must be set for the integration to be enabled. If
`GOOGLE_OAUTH_CLIENT_ID`/`GOOGLE_OAUTH_CLIENT_SECRET` are both unset, the
integration is simply disabled; if only some of the four are set, the server
fails to start with a clear error.

## 4. Connecting an account

1. Log in to Massa and go to **Settings**.
2. Click **Connect Google Health**. You'll be redirected to Google's consent
   screen requesting read/write access to weight and height data.
3. After granting access, Google redirects back to
   `/api/google/callback`, which stores your credentials and runs an initial
   backfill of your full weight and height history before redirecting you
   back to Settings.
4. Use **Sync now** on the Settings page to re-run the backfill manually at
   any time, and **Disconnect** to remove the stored credentials.

## 5. Things to verify with a real account

The backfill implementation was built against Google's published API
discovery document, but the following should be confirmed once you connect a
real account:

- **Historical depth**: confirm whether the connected account's weight/height
  history goes back years (e.g. imported from old Google Fit / Fitbit data)
  or only contains recent data. This affects how long the first backfill
  takes and whether pagination (`nextPageToken`) is exercised in practice.
- **Height data availability**: confirm whether the account has any height
  data points at all. If not, the manual height override (added in a later
  milestone) becomes the primary source for BMI calculations rather than a
  fallback.
- **`DataPoint.name`**: per the discovery document, weight/height data points
  often have an empty `name` field (no per-point Google ID). The backfill
  handles this by deduplicating on `(user_id, recorded_at)` for points
  without an ID, and on `(user_id, google_data_point_id)` for points that do
  have one — verify both paths produce sensible results against real data.
- **Testing vs. Production consent screen**: while the OAuth consent screen
  is in "Testing" status, refresh tokens expire after 7 days, requiring you
  to reconnect periodically. Check
  [developers.google.com/health/app-verification](https://developers.google.com/health/app-verification)
  to see whether "In production" status is reachable for these Restricted
  scopes with only 1–3 test users — that would avoid the 7-day expiry. The
  connect flow is idempotent ("Connect" doubles as "Reconnect"), so
  reconnecting after expiry is a one-click operation either way.
