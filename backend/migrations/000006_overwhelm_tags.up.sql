CREATE TABLE overwhelm_tags (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    name TEXT NOT NULL CHECK (length(trim(name)) BETWEEN 1 AND 30),
    archived_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE UNIQUE INDEX idx_overwhelm_tags_user_id_name ON overwhelm_tags(user_id, lower(name));

CREATE TABLE overwhelm_entry_tags (
    entry_id UUID NOT NULL REFERENCES overwhelm_entries(id) ON DELETE CASCADE,
    tag_id UUID NOT NULL REFERENCES overwhelm_tags(id) ON DELETE CASCADE,
    PRIMARY KEY (entry_id, tag_id)
);

CREATE INDEX idx_overwhelm_entry_tags_tag_id ON overwhelm_entry_tags(tag_id);
