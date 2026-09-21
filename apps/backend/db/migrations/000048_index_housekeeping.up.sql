BEGIN;

-- Buang index redundan yang merupakan prefix dari index unik/PK yang sudah ada.
DROP INDEX IF EXISTS user_answers_user_exam_id_idx;
DROP INDEX IF EXISTS package_views_package_idx;
DROP INDEX IF EXISTS transactions_invoice_status_idx;
DROP INDEX IF EXISTS teacher_payout_accounts_provider_idx;

-- Hot path ranking/leaderboard (ListGlobalRanking): window ROW_NUMBER/RANK/DISTINCT ON
-- atas attempt submitted diurut (total_score, durasi, finished_at).
CREATE INDEX user_exams_submitted_rank_idx
    ON user_exams (
        total_score DESC,
        (EXTRACT(EPOCH FROM (finished_at - started_at))),
        finished_at
    )
    WHERE status = 'submitted' AND total_score IS NOT NULL AND finished_at IS NOT NULL;

-- Ranking global & admin user list: filter siswa berdasarkan jenjang.
CREATE INDEX users_school_level_idx ON users (school_level) WHERE role = 'student';

-- Katalog publik CBT: paket aktif bertipe cbt.
CREATE INDEX packages_exam_type_status_idx ON packages (exam_type, status);

-- Badge notifikasi belum dibaca (UnreadCount) + list notifikasi unread.
CREATE INDEX notifications_unread_idx ON notifications (user_id, created_at DESC) WHERE read_at IS NULL;

COMMIT;