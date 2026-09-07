DROP TABLE IF EXISTS teacher_payouts;
DROP TABLE IF EXISTS teacher_commissions;
ALTER TABLE finance_settings
    DROP COLUMN IF EXISTS commission_hold_days,
    DROP COLUMN IF EXISTS teacher_sales_bonus_percent,
    DROP COLUMN IF EXISTS teacher_upload_fee;
