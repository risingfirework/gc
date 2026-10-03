BEGIN;

ALTER TABLE exams
    DROP CONSTRAINT IF EXISTS exams_screen_lock_seconds_check,
    DROP COLUMN IF EXISTS screen_lock_seconds,
    DROP COLUMN IF EXISTS screen_lock_enabled;

COMMIT;
