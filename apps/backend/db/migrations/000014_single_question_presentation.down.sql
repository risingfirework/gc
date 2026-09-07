ALTER TABLE questions DROP CONSTRAINT questions_presentation_type_check;
ALTER TABLE questions ADD CONSTRAINT questions_presentation_type_check CHECK (presentation_type IN ('single', 'group'));
