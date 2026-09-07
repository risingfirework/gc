CREATE TABLE teacher_payout_accounts (
    teacher_id UUID PRIMARY KEY REFERENCES users(id) ON DELETE CASCADE,
    method VARCHAR(24) NOT NULL CHECK (method IN ('bank_transfer', 'e_wallet')),
    provider VARCHAR(80) NOT NULL,
    account_number VARCHAR(32) NOT NULL,
    account_holder_name VARCHAR(120) NOT NULL,
    phone VARCHAR(24) NOT NULL DEFAULT '',
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX teacher_payout_accounts_provider_idx
    ON teacher_payout_accounts(method, provider);
