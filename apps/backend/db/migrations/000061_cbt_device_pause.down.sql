ALTER TABLE user_exams
  DROP COLUMN IF EXISTS paused_seconds,
  DROP COLUMN IF EXISTS paused_at,
  DROP COLUMN IF EXISTS device_id;

DROP INDEX IF EXISTS user_exams_paused_idx;