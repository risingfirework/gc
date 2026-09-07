CREATE TABLE teacher_payout_requests (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    teacher_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    amount NUMERIC(14,2) NOT NULL CHECK (amount > 0),
    status VARCHAR(16) NOT NULL DEFAULT 'submitted' CHECK (status IN ('submitted','approved','rejected','cancelled','paid')),
    payout_method VARCHAR(24) NOT NULL,
    provider VARCHAR(80) NOT NULL,
    account_number VARCHAR(32) NOT NULL,
    account_holder_name VARCHAR(120) NOT NULL,
    phone VARCHAR(24) NOT NULL DEFAULT '',
    admin_note VARCHAR(500) NOT NULL DEFAULT '',
    transfer_reference VARCHAR(120) NOT NULL DEFAULT '',
    submitted_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    reviewed_at TIMESTAMPTZ,
    paid_at TIMESTAMPTZ,
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE UNIQUE INDEX teacher_payout_requests_one_active_idx
    ON teacher_payout_requests(teacher_id) WHERE status IN ('submitted','approved');
CREATE INDEX teacher_payout_requests_status_idx
    ON teacher_payout_requests(status, submitted_at DESC);

ALTER TABLE teacher_commissions
    ADD COLUMN payout_request_id UUID REFERENCES teacher_payout_requests(id) ON DELETE SET NULL;
CREATE INDEX teacher_commissions_payout_request_idx
    ON teacher_commissions(payout_request_id) WHERE payout_request_id IS NOT NULL;

ALTER TABLE teacher_payouts
    ADD COLUMN payout_request_id UUID UNIQUE REFERENCES teacher_payout_requests(id) ON DELETE SET NULL;
