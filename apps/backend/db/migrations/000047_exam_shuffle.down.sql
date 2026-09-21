ALTER TABLE user_exams DROP COLUMN IF EXISTS option_order_json;
ALTER TABLE user_exams DROP COLUMN IF EXISTS question_order_json;
ALTER TABLE exams DROP COLUMN IF EXISTS shuffle_options;
ALTER TABLE exams DROP COLUMN IF EXISTS shuffle_questions;