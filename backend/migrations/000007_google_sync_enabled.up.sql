ALTER TABLE google_oauth_credentials
    ADD COLUMN sync_enabled BOOLEAN NOT NULL DEFAULT true;
