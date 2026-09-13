ALTER TABLE teacher_commissions ADD COLUMN IF NOT EXISTS split_group UUID;
DROP INDEX IF EXISTS teacher_commissions_upload_once_idx;
DROP INDEX IF EXISTS teacher_commissions_sale_once_idx;
CREATE UNIQUE INDEX teacher_commissions_upload_once_idx ON teacher_commissions(package_id, kind) WHERE kind='upload_fee' AND split_group IS NULL;
CREATE UNIQUE INDEX teacher_commissions_sale_once_idx ON teacher_commissions(transaction_id, kind) WHERE kind IN ('sales_bonus','refund_reversal') AND split_group IS NULL;