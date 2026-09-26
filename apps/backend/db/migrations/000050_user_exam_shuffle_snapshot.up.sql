ALTER TABLE user_exams ADD COLUMN shuffle_questions BOOLEAN;
ALTER TABLE user_exams ADD COLUMN shuffle_options BOOLEAN;

-- Existing attempts retain whichever shuffle dimensions have already been
-- materialized. New attempts copy both flags from the exam at creation time.
UPDATE user_exams
SET shuffle_questions = question_order_json IS NOT NULL,
    shuffle_options = option_order_json IS NOT NULL;

ALTER TABLE user_exams ALTER COLUMN shuffle_questions SET NOT NULL;
ALTER TABLE user_exams ALTER COLUMN shuffle_options SET NOT NULL;
ALTER TABLE user_exams ALTER COLUMN shuffle_questions SET DEFAULT FALSE;
ALTER TABLE user_exams ALTER COLUMN shuffle_options SET DEFAULT FALSE;
