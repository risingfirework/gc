BEGIN;
-- Opsional: 20.000 akun unik untuk K6. Password: TkaLoad123!
WITH password AS MATERIALIZED (SELECT crypt('TkaLoad123!', gen_salt('bf', 8)) AS hash)
INSERT INTO users (email, password_hash, role, school_level)
SELECT format('loadtest+%s@tka.local', sequence), password.hash, 'student', 'SMA'
FROM generate_series(1, 20000) AS sequence CROSS JOIN password
ON CONFLICT (LOWER(email)) DO UPDATE SET password_hash=EXCLUDED.password_hash, updated_at=NOW();

INSERT INTO user_packages (user_id, package_id, expired_at, status)
SELECT users.id,'20000000-0000-0000-0000-000000000002',NOW()+INTERVAL '7 days','active'
FROM users WHERE users.email LIKE 'loadtest+%@tka.local'
AND NOT EXISTS (SELECT 1 FROM user_packages WHERE user_packages.user_id=users.id
AND user_packages.package_id='20000000-0000-0000-0000-000000000002'
AND user_packages.status='active' AND user_packages.expired_at>NOW());
COMMIT;
