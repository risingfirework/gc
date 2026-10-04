BEGIN;

DROP INDEX IF EXISTS questions_exam_chapter_idx;
ALTER TABLE questions DROP COLUMN IF EXISTS chapter_name;

COMMIT;
