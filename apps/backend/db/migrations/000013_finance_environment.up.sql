CREATE TABLE finance_settings (
    singleton BOOLEAN PRIMARY KEY DEFAULT TRUE CHECK (singleton),
    platform_commission_percent NUMERIC(5,2) NOT NULL DEFAULT 20 CHECK (platform_commission_percent BETWEEN 0 AND 100),
    default_discount_percent NUMERIC(5,2) NOT NULL DEFAULT 0 CHECK (default_discount_percent BETWEEN 0 AND 100),
    tax_percent NUMERIC(5,2) NOT NULL DEFAULT 0 CHECK (tax_percent BETWEEN 0 AND 100),
    minimum_payout NUMERIC(14,2) NOT NULL DEFAULT 100000 CHECK (minimum_payout >= 0),
    payout_cycle VARCHAR(16) NOT NULL DEFAULT 'monthly' CHECK (payout_cycle IN ('weekly','monthly','manual')),
    auto_payout BOOLEAN NOT NULL DEFAULT FALSE,
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

INSERT INTO finance_settings(singleton) VALUES(TRUE);

CREATE TABLE user_finance_profiles (
    user_id UUID PRIMARY KEY REFERENCES users(id) ON DELETE CASCADE,
    commission_percent NUMERIC(5,2) CHECK (commission_percent IS NULL OR commission_percent BETWEEN 0 AND 100),
    discount_percent NUMERIC(5,2) CHECK (discount_percent IS NULL OR discount_percent BETWEEN 0 AND 100),
    account_status VARCHAR(16) NOT NULL DEFAULT 'active' CHECK (account_status IN ('active','hold')),
    notes VARCHAR(500) NOT NULL DEFAULT '',
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX user_finance_profiles_status_idx ON user_finance_profiles(account_status);
