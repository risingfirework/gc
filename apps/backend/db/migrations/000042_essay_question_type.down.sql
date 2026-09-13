ALTER TABLE questions
    DROP CONSTRAINT questions_question_type_check;

ALTER TABLE questions
    ADD CONSTRAINT questions_question_type_check
        CHECK (question_type IN ('single_choice', 'multiple_choice', 'category'));