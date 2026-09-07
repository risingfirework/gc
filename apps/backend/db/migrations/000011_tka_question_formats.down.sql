DROP INDEX IF EXISTS idx_questions_exam_group;
ALTER TABLE questions
    DROP CONSTRAINT IF EXISTS questions_category_context_check,
    DROP CONSTRAINT IF EXISTS questions_group_context_check,
    DROP CONSTRAINT IF EXISTS questions_category_labels_json_check,
    DROP CONSTRAINT IF EXISTS questions_presentation_type_check,
    DROP CONSTRAINT IF EXISTS questions_question_type_check,
    DROP COLUMN IF EXISTS category_labels_json,
    DROP COLUMN IF EXISTS stimulus_text,
    DROP COLUMN IF EXISTS group_code,
    DROP COLUMN IF EXISTS presentation_type,
    DROP COLUMN IF EXISTS question_type;
