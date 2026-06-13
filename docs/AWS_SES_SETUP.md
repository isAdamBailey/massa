# AWS SES setup for magic-link email

Massa sends passwordless sign-in links by email. In production, use **AWS SES**
via its SMTP interface (`EMAIL_PROVIDER=ses`).

## 1. Verify your sender

In the [SES console](https://console.aws.amazon.com/ses/), choose your region,
then verify either:

- **Domain** (recommended): e.g. `yourdomain.com` — then use
  `login@yourdomain.com` as `MAGIC_LINK_FROM_EMAIL`
- **Email address**: verify a single address if you don't have DNS set up yet

`MAGIC_LINK_FROM_EMAIL` must match a verified identity.

## 2. Create SMTP credentials

1. SES console → **SMTP settings** → **Create SMTP credentials**
2. Save the **SMTP username** and **SMTP password** (shown once)

These are not the same as AWS access keys — they are SES-specific SMTP credentials.

## 3. Sandbox vs production

New SES accounts start in **sandbox mode**:

- You can only send **to** verified email addresses
- Add your login email(s) from `ALLOWED_EMAILS` as verified identities, **or**
- Request **production access** in SES → Account dashboard → Request production access

Until you leave sandbox, magic links only arrive for verified recipient addresses.

## 4. Environment variables

Set these in your production `.env`:

```env
EMAIL_PROVIDER=ses
SES_REGION=us-east-1
SMTP_USERNAME=AKIA....................   # from SES SMTP credentials
SMTP_PASSWORD=............................  # from SES SMTP credentials
MAGIC_LINK_FROM_EMAIL=login@yourdomain.com
```

Optional overrides:

```env
SMTP_HOST=email-smtp.us-east-1.amazonaws.com   # default: email-smtp.{SES_REGION}.amazonaws.com
SMTP_PORT=587                                   # default: 587 (STARTTLS)
```

The backend uses port **587** with STARTTLS, which is what AWS recommends.

## 5. Example production `.env` snippet

```env
EMAIL_PROVIDER=ses
SES_REGION=us-east-1
SMTP_USERNAME=your-ses-smtp-username
SMTP_PASSWORD=your-ses-smtp-password
MAGIC_LINK_FROM_EMAIL=login@massa.example.com
ALLOWED_EMAILS=you@example.com
```

## 6. Verify it works

After deploying, open the app and request a sign-in link. Check:

- SES console → **Account dashboard** → sending statistics
- If email doesn't arrive, check spam and SES suppression/bounce lists

Common errors:

| Issue | Fix |
|-------|-----|
| Message rejected | `MAGIC_LINK_FROM_EMAIL` not verified in SES |
| Auth failed | Wrong SMTP username/password (regenerate in SES) |
| No mail in sandbox | Recipient email not verified, or still in sandbox |
| Connection timeout | Droplet can't reach SES on port 587 — check security groups |

## Alternative: generic SMTP

If you prefer not to use the `ses` provider alias, set `EMAIL_PROVIDER=smtp`
with SES SMTP host/credentials directly:

```env
EMAIL_PROVIDER=smtp
SMTP_HOST=email-smtp.us-east-1.amazonaws.com
SMTP_PORT=587
SMTP_USERNAME=...
SMTP_PASSWORD=...
MAGIC_LINK_FROM_EMAIL=login@yourdomain.com
```

Both approaches use the same SMTP mailer in the backend.
