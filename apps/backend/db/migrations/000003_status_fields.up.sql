BEGIN;

ALTER TABLE packages
    ADD COLUMN status VARCHAR(16) NOT NULL DEFAULT 'active'
        CONSTRAINT packages_status_check CHECK (status IN ('active', 'inactive'));

ALTER TABLE exams
    ADD COLUMN status VARCHAR(16) NOT NULL DEFAULT 'active'
        CONSTRAINT exams_status_check CHECK (status IN ('active', 'inactive'));

ALTER TABLE questions
    ADD COLUMN status VARCHAR(16) NOT NULL DEFAULT 'active'
        CONSTRAINT questions_status_check CHECK (status IN ('active', 'inactive'));

COMMIT;