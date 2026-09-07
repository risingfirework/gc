BEGIN;

CREATE EXTENSION IF NOT EXISTS pgcrypto;

CREATE FUNCTION set_updated_at() RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = NOW();
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

CREATE TABLE users (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    email VARCHAR(320) NOT NULL,
    password_hash TEXT NOT NULL,
    role VARCHAR(16) NOT NULL DEFAULT 'student'
        CONSTRAINT users_role_check CHECK (role IN ('student', 'admin')),
    school_level VARCHAR(8) NOT NULL
        CONSTRAINT users_school_level_check CHECK (school_level IN ('SD', 'SMP', 'SMA')),
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE UNIQUE INDEX users_email_lower_uidx ON users (LOWER(email));

CREATE TRIGGER users_set_updated_at
    BEFORE UPDATE ON users
    FOR EACH ROW EXECUTE FUNCTION set_updated_at();

CREATE TABLE packages (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    title VARCHAR(200) NOT NULL,
    description TEXT NOT NULL DEFAULT '',
    price NUMERIC(14, 2) NOT NULL CONSTRAINT packages_price_check CHECK (price >= 0),
    validity_days INTEGER NOT NULL CONSTRAINT packages_validity_days_check CHECK (validity_days > 0),
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE user_packages (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID NOT NULL REFERENCES users(id) ON UPDATE CASCADE ON DELETE CASCADE,
    package_id UUID NOT NULL REFERENCES packages(id) ON UPDATE CASCADE ON DELETE RESTRICT,
    expired_at TIMESTAMPTZ NOT NULL,
    status VARCHAR(16) NOT NULL DEFAULT 'active'
        CONSTRAINT user_packages_status_check CHECK (status IN ('active', 'expired'))
);

CREATE INDEX user_packages_user_status_idx ON user_packages (user_id, status);
CREATE INDEX user_packages_package_id_idx ON user_packages (package_id);
CREATE INDEX user_packages_expired_at_idx ON user_packages (expired_at) WHERE status = 'active';

CREATE TABLE exams (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    package_id UUID NOT NULL REFERENCES packages(id) ON UPDATE CASCADE ON DELETE RESTRICT,
    title VARCHAR(200) NOT NULL,
    duration_minutes INTEGER NOT NULL
        CONSTRAINT exams_duration_check CHECK (duration_minutes > 0),
    total_questions INTEGER NOT NULL
        CONSTRAINT exams_total_questions_check CHECK (total_questions > 0),
    passing_score NUMERIC(7, 2) NOT NULL
        CONSTRAINT exams_passing_score_check CHECK (passing_score >= 0),
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX exams_package_id_idx ON exams (package_id);

CREATE TABLE questions (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    exam_id UUID NOT NULL REFERENCES exams(id) ON UPDATE CASCADE ON DELETE CASCADE,
    subject_name VARCHAR(120) NOT NULL,
    content_text TEXT NOT NULL,
    options_json JSONB NOT NULL
        CONSTRAINT questions_options_array_check CHECK (jsonb_typeof(options_json) = 'array'),
    correct_answer TEXT NOT NULL,
    score_weight NUMERIC(8, 3) NOT NULL DEFAULT 1
        CONSTRAINT questions_score_weight_check CHECK (score_weight > 0)
);

CREATE INDEX questions_exam_id_idx ON questions (exam_id);
CREATE INDEX questions_exam_subject_idx ON questions (exam_id, subject_name);

CREATE TABLE user_exams (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID NOT NULL REFERENCES users(id) ON UPDATE CASCADE ON DELETE CASCADE,
    exam_id UUID NOT NULL REFERENCES exams(id) ON UPDATE CASCADE ON DELETE RESTRICT,
    status VARCHAR(16) NOT NULL DEFAULT 'ongoing'
        CONSTRAINT user_exams_status_check CHECK (status IN ('ongoing', 'submitted')),
    started_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    finished_at TIMESTAMPTZ,
    total_score NUMERIC(10, 3),
    CONSTRAINT user_exams_finish_state_check CHECK (
        (status = 'ongoing' AND finished_at IS NULL AND total_score IS NULL)
        OR
        (status = 'submitted' AND finished_at IS NOT NULL AND total_score IS NOT NULL)
    ),
    CONSTRAINT user_exams_finish_time_check CHECK (finished_at IS NULL OR finished_at >= started_at)
);

CREATE INDEX user_exams_user_status_idx ON user_exams (user_id, status);
CREATE INDEX user_exams_exam_id_idx ON user_exams (exam_id);
CREATE INDEX user_exams_user_exam_idx ON user_exams (user_id, exam_id);
CREATE UNIQUE INDEX user_exams_one_ongoing_uidx
    ON user_exams (user_id, exam_id)
    WHERE status = 'ongoing';

CREATE TABLE user_answers (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_exam_id UUID NOT NULL REFERENCES user_exams(id) ON UPDATE CASCADE ON DELETE CASCADE,
    question_id UUID NOT NULL REFERENCES questions(id) ON UPDATE CASCADE ON DELETE RESTRICT,
    selected_option TEXT NOT NULL,
    is_correct BOOLEAN NOT NULL,
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT user_answers_attempt_question_unique UNIQUE (user_exam_id, question_id)
);

CREATE INDEX user_answers_user_exam_id_idx ON user_answers (user_exam_id);
CREATE INDEX user_answers_question_id_idx ON user_answers (question_id);

CREATE TRIGGER user_answers_set_updated_at
    BEFORE UPDATE ON user_answers
    FOR EACH ROW EXECUTE FUNCTION set_updated_at();

CREATE TABLE transactions (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID NOT NULL REFERENCES users(id) ON UPDATE CASCADE ON DELETE RESTRICT,
    package_id UUID NOT NULL REFERENCES packages(id) ON UPDATE CASCADE ON DELETE RESTRICT,
    invoice_number VARCHAR(100) NOT NULL,
    amount NUMERIC(14, 2) NOT NULL CONSTRAINT transactions_amount_check CHECK (amount >= 0),
    payment_status VARCHAR(16) NOT NULL DEFAULT 'pending'
        CONSTRAINT transactions_payment_status_check
        CHECK (payment_status IN ('pending', 'paid', 'failed', 'expired', 'refunded')),
    payment_method VARCHAR(32)
        CONSTRAINT transactions_payment_method_check
        CHECK (payment_method IS NULL OR payment_method IN ('qris', 'virtual_account', 'e_wallet')),
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT transactions_invoice_number_unique UNIQUE (invoice_number)
);

CREATE INDEX transactions_user_created_idx ON transactions (user_id, created_at DESC);
CREATE INDEX transactions_package_id_idx ON transactions (package_id);
CREATE INDEX transactions_payment_status_idx ON transactions (payment_status);

COMMIT;
