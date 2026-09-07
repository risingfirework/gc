BEGIN;

ALTER TABLE transactions
    ADD COLUMN idempotency_key VARCHAR(128),
    ADD COLUMN payment_url TEXT,
    ADD COLUMN expires_at TIMESTAMPTZ,
    ADD COLUMN paid_at TIMESTAMPTZ,
    ADD COLUMN updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW();

CREATE UNIQUE INDEX transactions_user_idempotency_uidx
    ON transactions (user_id, idempotency_key)
    WHERE idempotency_key IS NOT NULL;

CREATE INDEX transactions_invoice_status_idx
    ON transactions (invoice_number, payment_status);

CREATE TRIGGER transactions_set_updated_at
    BEFORE UPDATE ON transactions
    FOR EACH ROW EXECUTE FUNCTION set_updated_at();

CREATE TABLE payment_webhook_events (
    event_id VARCHAR(200) PRIMARY KEY,
    transaction_id UUID REFERENCES transactions(id) ON DELETE SET NULL,
    payload_sha256 CHAR(64) NOT NULL,
    received_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

ALTER TABLE exams
    ADD COLUMN scoring_method VARCHAR(16) NOT NULL DEFAULT 'standard'
        CONSTRAINT exams_scoring_method_check CHECK (scoring_method IN ('standard', 'irt_2pl'));

ALTER TABLE questions
    ADD COLUMN explanation_text TEXT NOT NULL DEFAULT '',
    ADD COLUMN explanation_video_url TEXT,
    ADD COLUMN difficulty DOUBLE PRECISION NOT NULL DEFAULT 0,
    ADD COLUMN discrimination DOUBLE PRECISION NOT NULL DEFAULT 1,
    ADD CONSTRAINT questions_difficulty_check CHECK (difficulty BETWEEN -4 AND 4),
    ADD CONSTRAINT questions_discrimination_check CHECK (discrimination BETWEEN 0.1 AND 3),
    ADD CONSTRAINT questions_explanation_video_url_check CHECK (explanation_video_url IS NULL OR explanation_video_url ~ '^https://');

COMMIT;
