BEGIN;

DROP INDEX IF EXISTS notifications_unread_idx;
DROP INDEX IF EXISTS packages_exam_type_status_idx;
DROP INDEX IF EXISTS users_school_level_idx;
DROP INDEX IF EXISTS user_exams_submitted_rank_idx;

CREATE INDEX teacher_payout_accounts_provider_idx
    ON teacher_payout_accounts (method, provider);
CREATE INDEX transactions_invoice_status_idx
    ON transactions (invoice_number, payment_status);
CREATE INDEX package_views_package_idx ON package_views (package_id);
CREATE INDEX user_answers_user_exam_id_idx ON user_answers (user_exam_id);

COMMIT;