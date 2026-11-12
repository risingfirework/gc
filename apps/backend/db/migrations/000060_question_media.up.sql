BEGIN;

ALTER TABLE questions
    ADD COLUMN question_audio_url TEXT NOT NULL DEFAULT '',
    ADD COLUMN question_video_url TEXT NOT NULL DEFAULT '';

-- Media yang diunggah ke server disajikan melalui path relatif
-- /api/v1/media/{id}; tautan https eksternal tetap diterima.
ALTER TABLE questions DROP CONSTRAINT IF EXISTS questions_explanation_video_url_check;
ALTER TABLE questions
    ADD CONSTRAINT questions_explanation_video_url_check CHECK (
        explanation_video_url IS NULL
        OR explanation_video_url = ''
        OR explanation_video_url ~ '^https://'
        OR explanation_video_url ~ '^/api/v1/media/'
    );

ALTER TABLE questions
    ADD CONSTRAINT questions_question_audio_url_check CHECK (
        question_audio_url = ''
        OR question_audio_url ~ '^https://'
        OR question_audio_url ~ '^/api/v1/media/'
    ),
    ADD CONSTRAINT questions_question_video_url_check CHECK (
        question_video_url = ''
        OR question_video_url ~ '^https://'
        OR question_video_url ~ '^/api/v1/media/'
    );

COMMIT;
