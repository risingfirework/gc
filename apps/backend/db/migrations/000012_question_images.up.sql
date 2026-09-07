ALTER TABLE questions
    ADD COLUMN question_image_url TEXT NOT NULL DEFAULT '',
    ADD COLUMN stimulus_image_url TEXT NOT NULL DEFAULT '';
