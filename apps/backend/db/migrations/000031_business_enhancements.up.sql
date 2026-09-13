BEGIN;

ALTER TABLE transactions
    ADD COLUMN platform_commission NUMERIC(14,2) NOT NULL DEFAULT 0
        CONSTRAINT transactions_platform_commission_check CHECK (platform_commission >= 0);

ALTER TABLE exams
    ADD COLUMN max_attempts INTEGER
        CONSTRAINT exams_max_attempts_check CHECK (max_attempts IS NULL OR max_attempts > 0);

CREATE TABLE notifications (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    title VARCHAR(200) NOT NULL,
    body TEXT NOT NULL DEFAULT '',
    link VARCHAR(500) NOT NULL DEFAULT '',
    read_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX notifications_user_created_idx ON notifications (user_id, created_at DESC);

COMMIT;