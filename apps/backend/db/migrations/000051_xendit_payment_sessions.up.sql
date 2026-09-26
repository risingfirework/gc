BEGIN;

ALTER TABLE transactions
    DROP CONSTRAINT transactions_payment_status_check,
    ADD CONSTRAINT transactions_payment_status_check
        CHECK (payment_status IN ('pending','paid','failed','expired','refund_pending','refunded')),
    ADD COLUMN provider_session_id VARCHAR(255),
    ADD COLUMN provider_payment_request_id VARCHAR(255),
    ADD COLUMN provider_refund_id VARCHAR(255);

CREATE UNIQUE INDEX transactions_provider_session_uidx
    ON transactions(provider_session_id) WHERE provider_session_id IS NOT NULL;
CREATE UNIQUE INDEX transactions_provider_payment_request_uidx
    ON transactions(provider_payment_request_id) WHERE provider_payment_request_id IS NOT NULL;
CREATE UNIQUE INDEX transactions_provider_refund_uidx
    ON transactions(provider_refund_id) WHERE provider_refund_id IS NOT NULL;

COMMIT;
