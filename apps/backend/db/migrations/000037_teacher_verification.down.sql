ALTER TABLE users DROP CONSTRAINT IF EXISTS users_teacher_verification_status_check;
ALTER TABLE users DROP INDEX IF EXISTS idx_users_teacher_verification;
ALTER TABLE users DROP COLUMN IF EXISTS teacher_verified_at;
ALTER TABLE users DROP COLUMN IF EXISTS teacher_appeal_image;
ALTER TABLE users DROP COLUMN IF EXISTS teacher_rejection_reason;
ALTER TABLE users DROP COLUMN IF EXISTS teacher_verification_status;
ALTER TABLE users DROP COLUMN IF EXISTS teacher_ktp;