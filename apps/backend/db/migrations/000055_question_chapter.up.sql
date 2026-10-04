BEGIN;

ALTER TABLE questions ADD COLUMN chapter_name VARCHAR(160) NOT NULL DEFAULT '';

CREATE INDEX questions_exam_chapter_idx ON questions (exam_id, chapter_name);

COMMIT;
