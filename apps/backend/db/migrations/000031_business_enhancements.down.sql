BEGIN;

DROP TABLE IF EXISTS notifications;
ALTER TABLE exams DROP COLUMN IF EXISTS max_attempts;
ALTER TABLE transactions DROP COLUMN IF EXISTS platform_commission;

COMMIT;