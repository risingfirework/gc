BEGIN;

ALTER TABLE users
    DROP COLUMN totp_secret_base32,
    DROP COLUMN totp_enabled;

COMMIT;