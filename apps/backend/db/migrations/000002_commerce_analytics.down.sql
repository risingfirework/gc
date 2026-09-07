BEGIN;

ALTER TABLE questions
    DROP CONSTRAINT IF EXISTS questions_explanation_video_url_check,
    DROP CONSTRAINT IF EXISTS questions_discrimination_check,
    DROP CONSTRAINT IF EXISTS questions_difficulty_check,
    DROP COLUMN IF EXISTS discrimination,
    DROP COLUMN IF EXISTS difficulty,
    DROP COLUMN IF EXISTS explanation_video_url,
    DROP COLUMN IF EXISTS explanation_text;

ALTER TABLE exams
    DROP CONSTRAINT IF EXISTS exams_scoring_method_check,
    DROP COLUMN IF EXISTS scoring_method;

DROP TABLE IF EXISTS payment_webhook_events;
DROP TRIGGER IF EXISTS transactions_set_updated_at ON transactions;
DROP INDEX IF EXISTS transactions_invoice_status_idx;
DROP INDEX IF EXISTS transactions_user_idempotency_uidx;

ALTER TABLE transactions
    DROP COLUMN IF EXISTS updated_at,
    DROP COLUMN IF EXISTS paid_at,
    DROP COLUMN IF EXISTS expires_at,
    DROP COLUMN IF EXISTS payment_url,
    DROP COLUMN IF EXISTS idempotency_key;

COMMIT;
