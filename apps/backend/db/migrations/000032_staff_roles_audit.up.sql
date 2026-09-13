BEGIN;

-- Audit trail untuk keputusan finansial & manajemen akses.
CREATE TABLE audit_logs (
    id          uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    actor_id    uuid REFERENCES users(id) ON DELETE SET NULL,
    actor_email text NOT NULL,
    action      text NOT NULL,
    entity_type text NOT NULL,
    entity_id   text NOT NULL DEFAULT '',
    detail      jsonb NOT NULL DEFAULT '{}'::jsonb,
    created_at  timestamptz NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_audit_logs_created_at ON audit_logs (created_at DESC);

-- Pemisahan peran: admin lama dianggap pemilik (owner) agar akses penuh tetap,
-- sedangkan role 'admin' kini digunakan untuk operator konten.
-- Perluas cakupan role yang sah sebelum mempromosikan admin menjadi owner.
ALTER TABLE users DROP CONSTRAINT users_role_check;
ALTER TABLE users ADD CONSTRAINT users_role_check CHECK (role IN ('student', 'teacher', 'admin', 'owner', 'finance'));

UPDATE users SET role = 'owner' WHERE role = 'admin';

COMMIT;