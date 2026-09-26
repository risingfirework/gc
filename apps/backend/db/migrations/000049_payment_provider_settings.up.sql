CREATE TABLE payment_provider_settings (
    singleton BOOLEAN PRIMARY KEY DEFAULT TRUE CHECK (singleton),
    provider VARCHAR(20) NOT NULL DEFAULT 'xendit',
    secret_key_encrypted BYTEA NOT NULL DEFAULT ''::bytea,
    webhook_token_encrypted BYTEA NOT NULL DEFAULT ''::bytea,
    base_url VARCHAR(300) NOT NULL DEFAULT '',
    success_redirect_url VARCHAR(600) NOT NULL DEFAULT '',
    failure_redirect_url VARCHAR(600) NOT NULL DEFAULT '',
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

INSERT INTO payment_provider_settings(singleton) VALUES(TRUE);