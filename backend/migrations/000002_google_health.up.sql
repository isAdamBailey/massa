CREATE TABLE google_oauth_credentials (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID NOT NULL UNIQUE REFERENCES users(id) ON DELETE CASCADE,
    google_health_user_id TEXT NOT NULL,
    refresh_token_encrypted BYTEA NOT NULL,
    refresh_token_nonce BYTEA NOT NULL,
    access_token_encrypted BYTEA,
    access_token_nonce BYTEA,
    access_token_expires_at TIMESTAMPTZ,
    connected_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE height_entries (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    height_cm NUMERIC(5, 2) NOT NULL,
    recorded_at TIMESTAMPTZ NOT NULL,
    source TEXT NOT NULL CHECK (source IN ('manual', 'google')),
    google_data_point_id TEXT,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX idx_height_entries_user_id_recorded_at ON height_entries(user_id, recorded_at);

-- Dedup by Google data point ID when one is provided.
CREATE UNIQUE INDEX idx_height_entries_google_data_point_id
    ON height_entries(user_id, google_data_point_id)
    WHERE google_data_point_id IS NOT NULL;

-- Fallback dedup for Google data points without an ID (observed to be the
-- common case for weight/height in the Google Health API).
CREATE UNIQUE INDEX idx_height_entries_google_recorded_at
    ON height_entries(user_id, recorded_at)
    WHERE source = 'google' AND google_data_point_id IS NULL;

CREATE TABLE weight_entries (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    weight_kg NUMERIC(6, 2) NOT NULL,
    recorded_at TIMESTAMPTZ NOT NULL,
    bmi NUMERIC(5, 2),
    height_used_cm NUMERIC(5, 2),
    source TEXT NOT NULL CHECK (source IN ('manual', 'google')),
    google_data_point_id TEXT,
    google_sync_status TEXT CHECK (google_sync_status IN ('pending', 'synced', 'failed')),
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX idx_weight_entries_user_id_recorded_at ON weight_entries(user_id, recorded_at);

-- Dedup by Google data point ID when one is provided.
CREATE UNIQUE INDEX idx_weight_entries_google_data_point_id
    ON weight_entries(user_id, google_data_point_id)
    WHERE google_data_point_id IS NOT NULL;

-- Fallback dedup for Google data points without an ID (observed to be the
-- common case for weight/height in the Google Health API).
CREATE UNIQUE INDEX idx_weight_entries_google_recorded_at
    ON weight_entries(user_id, recorded_at)
    WHERE source = 'google' AND google_data_point_id IS NULL;

CREATE TABLE sync_metadata (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID NOT NULL UNIQUE REFERENCES users(id) ON DELETE CASCADE,
    last_full_backfill_at TIMESTAMPTZ,
    last_incremental_sync_at TIMESTAMPTZ,
    weight_sync_watermark TIMESTAMPTZ,
    height_sync_watermark TIMESTAMPTZ,
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);
