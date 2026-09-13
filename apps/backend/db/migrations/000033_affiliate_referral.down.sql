BEGIN;

DROP INDEX IF EXISTS teacher_commissions_referral_once_idx;

ALTER TABLE teacher_commissions DROP CONSTRAINT teacher_commissions_kind_check;
ALTER TABLE teacher_commissions ADD CONSTRAINT teacher_commissions_kind_check CHECK (kind IN ('upload_fee','sales_bonus','refund_reversal'));

DROP TABLE IF EXISTS referrals;
DROP TABLE IF EXISTS affiliates;

ALTER TABLE finance_settings DROP CONSTRAINT IF EXISTS finance_settings_affiliate_rate_percent_check;
ALTER TABLE finance_settings DROP COLUMN IF EXISTS affiliate_rate_percent;

ALTER TABLE users DROP CONSTRAINT users_role_check;
ALTER TABLE users ADD CONSTRAINT users_role_check CHECK (role IN ('student', 'teacher', 'admin', 'owner', 'finance'));

COMMIT;