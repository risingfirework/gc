BEGIN;

ALTER TABLE questions
    DROP CONSTRAINT IF EXISTS questions_question_video_url_check,
    DROP CONSTRAINT IF EXISTS questions_question_audio_url_check,
    DROP CONSTRAINT IF EXISTS questions_explanation_video_url_check;

UPDATE questions SET explanation_video_url = NULL WHERE explanation_video_url = '';

ALTER TABLE questions
    ADD CONSTRAINT questions_explanation_video_url_check CHECK (
        explanation_video_url IS NULL OR explanation_video_url ~ '^https://'
    );

ALTER TABLE questions
    DROP COLUMN IF EXISTS question_video_url,
    DROP COLUMN IF EXISTS question_audio_url;

COMMIT;
