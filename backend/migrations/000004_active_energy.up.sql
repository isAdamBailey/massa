CREATE TABLE active_energy_entries (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    day DATE NOT NULL,
    active_energy_kcal NUMERIC(8, 2) NOT NULL,
    source TEXT NOT NULL CHECK (source IN ('google')),
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE UNIQUE INDEX idx_active_energy_entries_user_id_day ON active_energy_entries(user_id, day);

ALTER TABLE sync_metadata ADD COLUMN active_energy_sync_watermark TIMESTAMPTZ;
