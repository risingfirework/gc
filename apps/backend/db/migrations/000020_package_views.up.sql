CREATE TABLE package_views (
    package_id UUID NOT NULL REFERENCES packages(id) ON DELETE CASCADE,
    visitor_key UUID NOT NULL,
    first_viewed_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    last_viewed_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    PRIMARY KEY (package_id, visitor_key)
);

CREATE INDEX package_views_package_idx ON package_views(package_id);
