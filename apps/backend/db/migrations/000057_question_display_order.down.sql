BEGIN;

DROP INDEX IF EXISTS questions_exam_display_order_idx;
ALTER TABLE questions DROP COLUMN IF EXISTS display_order;
DROP SEQUENCE IF EXISTS questions_display_order_seq;

COMMIT;
