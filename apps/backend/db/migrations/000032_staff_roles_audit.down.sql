BEGIN;

UPDATE users SET role = 'admin' WHERE role IN ('owner', 'finance');

ALTER TABLE users DROP CONSTRAINT users_role_check;
ALTER TABLE users ADD CONSTRAINT users_role_check CHECK (role IN ('student', 'admin', 'teacher'));

DROP TABLE IF EXISTS audit_logs;

COMMIT;