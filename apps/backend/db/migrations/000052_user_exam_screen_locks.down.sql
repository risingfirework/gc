BEGIN;

DROP INDEX IF EXISTS user_exam_screen_locks_active_idx;
DROP TABLE IF EXISTS user_exam_screen_locks;

COMMIT;