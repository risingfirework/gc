ALTER TABLE questions
    DROP COLUMN IF EXISTS stimulus_image_url,
    DROP COLUMN IF EXISTS question_image_url;
