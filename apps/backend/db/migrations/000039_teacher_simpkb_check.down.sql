ALTER TABLE users
    DROP COLUMN IF EXISTS teacher_simpkb_checked_at,
    DROP COLUMN IF EXISTS teacher_simpkb_image,
    DROP COLUMN IF EXISTS teacher_simpkb_status;