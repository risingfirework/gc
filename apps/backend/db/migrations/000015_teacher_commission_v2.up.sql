ALTER TABLE finance_settings
    ADD COLUMN teacher_upload_fee NUMERIC(14,2) NOT NULL DEFAULT 30000 CHECK (teacher_upload_fee >= 0),
    ADD COLUMN teacher_sales_bonus_percent NUMERIC(5,2) NOT NULL DEFAULT 10 CHECK (teacher_sales_bonus_percent BETWEEN 0 AND 100),
    ADD COLUMN commission_hold_days INTEGER NOT NULL DEFAULT 7 CHECK (commission_hold_days BETWEEN 0 AND 90);

CREATE TABLE teacher_commissions (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    teacher_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    package_id UUID NOT NULL REFERENCES packages(id) ON DELETE CASCADE,
    transaction_id UUID REFERENCES transactions(id) ON DELETE SET NULL,
    kind VARCHAR(24) NOT NULL CHECK (kind IN ('upload_fee','sales_bonus','refund_reversal')),
    base_amount NUMERIC(14,2) NOT NULL DEFAULT 0,
    rate_percent NUMERIC(5,2) NOT NULL DEFAULT 0,
    amount NUMERIC(14,2) NOT NULL,
    status VARCHAR(16) NOT NULL DEFAULT 'pending' CHECK (status IN ('pending','available','paid','cancelled')),
    available_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    paid_at TIMESTAMPTZ,
    payout_reference VARCHAR(120) NOT NULL DEFAULT '',
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE UNIQUE INDEX teacher_commissions_upload_once_idx
    ON teacher_commissions(package_id, kind) WHERE kind='upload_fee';
CREATE UNIQUE INDEX teacher_commissions_sale_once_idx
    ON teacher_commissions(transaction_id, kind) WHERE kind IN ('sales_bonus','refund_reversal');
CREATE INDEX teacher_commissions_teacher_status_idx
    ON teacher_commissions(teacher_id, status, available_at DESC);

CREATE TABLE teacher_payouts (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    teacher_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    amount NUMERIC(14,2) NOT NULL CHECK (amount > 0),
    reference VARCHAR(120) NOT NULL,
    paid_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX teacher_payouts_teacher_idx ON teacher_payouts(teacher_id, paid_at DESC);
