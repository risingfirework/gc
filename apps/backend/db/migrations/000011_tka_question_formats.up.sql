ALTER TABLE questions
    ADD COLUMN question_type VARCHAR(24) NOT NULL DEFAULT 'single_choice',
    ADD COLUMN presentation_type VARCHAR(16) NOT NULL DEFAULT 'single',
    ADD COLUMN group_code VARCHAR(64),
    ADD COLUMN stimulus_text TEXT NOT NULL DEFAULT '',
    ADD COLUMN category_labels_json JSONB NOT NULL DEFAULT '[]'::jsonb;

ALTER TABLE questions
    ADD CONSTRAINT questions_question_type_check
        CHECK (question_type IN ('single_choice', 'multiple_choice', 'category')),
    ADD CONSTRAINT questions_presentation_type_check
        CHECK (presentation_type IN ('single', 'group')),
    ADD CONSTRAINT questions_category_labels_json_check
        CHECK (jsonb_typeof(category_labels_json) = 'array'),
    ADD CONSTRAINT questions_group_context_check
        CHECK (presentation_type = 'single' OR (NULLIF(BTRIM(group_code), '') IS NOT NULL AND NULLIF(BTRIM(stimulus_text), '') IS NOT NULL)),
    ADD CONSTRAINT questions_category_context_check
        CHECK (question_type <> 'category' OR jsonb_array_length(category_labels_json) >= 2);

CREATE INDEX idx_questions_exam_group ON questions(exam_id, group_code) WHERE presentation_type = 'group';
