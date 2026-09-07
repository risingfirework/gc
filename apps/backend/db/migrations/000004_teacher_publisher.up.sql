BEGIN;

ALTER TABLE users DROP CONSTRAINT users_role_check;
ALTER TABLE users ADD CONSTRAINT users_role_check CHECK (role IN ('student', 'admin', 'teacher'));

ALTER TABLE packages ADD COLUMN publisher_id UUID REFERENCES users(id) ON DELETE SET NULL;

CREATE INDEX packages_publisher_id_idx ON packages (publisher_id);

COMMIT;