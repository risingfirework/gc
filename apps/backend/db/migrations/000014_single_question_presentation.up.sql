UPDATE questions
SET content_text = CASE
        WHEN BTRIM(stimulus_text) <> '' THEN 'Stimulus:' || E'\n' || stimulus_text || E'\n\n' || content_text
        ELSE content_text
    END,
    question_image_url = CASE
        WHEN question_image_url = '' AND stimulus_image_url <> '' THEN stimulus_image_url
        ELSE question_image_url
    END,
    presentation_type = 'single',
    group_code = NULL,
    stimulus_text = '',
    stimulus_image_url = ''
WHERE presentation_type = 'group';

ALTER TABLE questions DROP CONSTRAINT questions_presentation_type_check;
ALTER TABLE questions ADD CONSTRAINT questions_presentation_type_check CHECK (presentation_type = 'single');
