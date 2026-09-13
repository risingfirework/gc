BEGIN;

DROP INDEX IF EXISTS transactions_pending_expiry_idx;
DROP INDEX IF EXISTS user_packages_user_package_uidx;

COMMIT;