BEGIN;

UPDATE transactions SET payment_status='paid' WHERE payment_status='refund_pending';
DROP INDEX IF EXISTS transactions_provider_refund_uidx;
DROP INDEX IF EXISTS transactions_provider_payment_request_uidx;
DROP INDEX IF EXISTS transactions_provider_session_uidx;
ALTER TABLE transactions
    DROP COLUMN IF EXISTS provider_refund_id,
    DROP COLUMN IF EXISTS provider_payment_request_id,
    DROP COLUMN IF EXISTS provider_session_id,
    DROP CONSTRAINT transactions_payment_status_check,
    ADD CONSTRAINT transactions_payment_status_check
        CHECK (payment_status IN ('pending','paid','failed','expired','refunded'));

COMMIT;
