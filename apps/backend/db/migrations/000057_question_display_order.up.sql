BEGIN;

CREATE SEQUENCE questions_display_order_seq;

ALTER TABLE questions
    ADD COLUMN display_order BIGINT;

WITH numbered_questions AS (
    SELECT id, row_number() OVER (ORDER BY ctid) AS display_order
    FROM questions
)
UPDATE questions AS question
SET display_order = numbered_questions.display_order
FROM numbered_questions
WHERE question.id = numbered_questions.id;

SELECT setval(
    'questions_display_order_seq',
    COALESCE((SELECT MAX(display_order) FROM questions), 1),
    (SELECT COUNT(*) > 0 FROM questions)
);

ALTER TABLE questions
    ALTER COLUMN display_order SET DEFAULT nextval('questions_display_order_seq'),
    ALTER COLUMN display_order SET NOT NULL;

ALTER SEQUENCE questions_display_order_seq OWNED BY questions.display_order;

CREATE INDEX questions_exam_display_order_idx ON questions (exam_id, display_order DESC);

COMMIT;
