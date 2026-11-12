BEGIN;

CREATE TABLE media_assets (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    owner_id UUID NOT NULL REFERENCES users(id) ON UPDATE CASCADE ON DELETE CASCADE,
    kind VARCHAR(16) NOT NULL
        CONSTRAINT media_assets_kind_check CHECK (kind IN ('image', 'audio', 'video')),
    mime_type VARCHAR(128) NOT NULL,
    size_bytes BIGINT NOT NULL
        CONSTRAINT media_assets_size_bytes_check CHECK (size_bytes > 0),
    original_name TEXT NOT NULL DEFAULT '',
    storage_key TEXT NOT NULL UNIQUE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX media_assets_owner_id_idx ON media_assets (owner_id);
CREATE INDEX media_assets_created_at_idx ON media_assets (created_at);

COMMIT;
