BEGIN;

ALTER TABLE packages DROP COLUMN publisher_id;

ALTER TABLE users DROP CONSTRAINT users_role_check;
ALTER TABLE users ADD CONSTRAINT users_role_check CHECK (role IN ('student', 'admin'));

COMMIT;