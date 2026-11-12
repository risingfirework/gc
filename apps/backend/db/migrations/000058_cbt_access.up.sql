BEGIN;

CREATE TABLE cbt_access_settings (
    id SMALLINT PRIMARY KEY DEFAULT 1 CHECK (id = 1),
    access_mode VARCHAR(20) NOT NULL DEFAULT 'free'
        CHECK (access_mode IN ('free', 'trial_then_paid', 'paid')),
    price_amount BIGINT NOT NULL DEFAULT 0 CHECK (price_amount >= 0),
    currency CHAR(3) NOT NULL DEFAULT 'IDR',
    validity_days INTEGER NOT NULL DEFAULT 365 CHECK (validity_days BETWEEN 0 AND 3650),
    free_quota INTEGER NOT NULL DEFAULT 1 CHECK (free_quota BETWEEN 0 AND 100),
    free_quota_reset VARCHAR(10) NOT NULL DEFAULT 'never'
        CHECK (free_quota_reset IN ('never', 'yearly')),
    payment_mode VARCHAR(20) NOT NULL DEFAULT 'manual'
        CHECK (payment_mode IN ('manual', 'gateway')),
    payment_instructions TEXT NOT NULL DEFAULT '',
    contact_info TEXT NOT NULL DEFAULT '',
    lock_scope VARCHAR(30) NOT NULL DEFAULT 'create_new_cbt'
        CHECK (lock_scope IN ('create_new_cbt', 'all_cbt_features')),
    allow_manage_existing BOOLEAN NOT NULL DEFAULT TRUE,
    grace_days INTEGER NOT NULL DEFAULT 7 CHECK (grace_days BETWEEN 0 AND 90),
    require_verified_teacher BOOLEAN NOT NULL DEFAULT TRUE,
    updated_by UUID REFERENCES users(id) ON DELETE SET NULL,
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

INSERT INTO cbt_access_settings (id) VALUES (1);

CREATE TABLE teacher_cbt_entitlements (
    user_id UUID PRIMARY KEY REFERENCES users(id) ON DELETE CASCADE,
    status VARCHAR(20) NOT NULL DEFAULT 'trial_available'
        CHECK (status IN ('trial_available', 'trial_used', 'active', 'expired', 'pending', 'blocked')),
    free_quota_used INTEGER NOT NULL DEFAULT 0 CHECK (free_quota_used >= 0),
    active_from TIMESTAMPTZ,
    active_until TIMESTAMPTZ,
    source VARCHAR(20)
        CHECK (source IN ('free', 'trial', 'payment', 'admin_grant')),
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX idx_cbt_entitlements_status ON teacher_cbt_entitlements (status, active_until);

CREATE TABLE teacher_cbt_payments (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    amount BIGINT NOT NULL CHECK (amount >= 0),
    currency CHAR(3) NOT NULL DEFAULT 'IDR',
    proof_image TEXT NOT NULL DEFAULT '',
    note TEXT NOT NULL DEFAULT '',
    status VARCHAR(20) NOT NULL DEFAULT 'pending'
        CHECK (status IN ('pending', 'approved', 'rejected', 'cancelled')),
    period_start TIMESTAMPTZ,
    period_end TIMESTAMPTZ,
    reviewed_by UUID REFERENCES users(id) ON DELETE SET NULL,
    reviewed_at TIMESTAMPTZ,
    rejection_reason TEXT NOT NULL DEFAULT '',
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX idx_cbt_payments_status ON teacher_cbt_payments (status, created_at DESC);
CREATE INDEX idx_cbt_payments_user ON teacher_cbt_payments (user_id, created_at DESC);

-- Guru yang sudah memiliki paket CBT sebelum fitur ini aktif tetap dapat
-- mengakses CBT (grandfathered) dan dianggap sudah memakai jatah gratisnya.
INSERT INTO teacher_cbt_entitlements (user_id, status, free_quota_used, active_from, active_until, source)
SELECT DISTINCT p.publisher_id, 'active', 1, now(), NULL, 'admin_grant'
FROM packages p
WHERE p.exam_type = 'cbt' AND p.publisher_id IS NOT NULL
  AND EXISTS (SELECT 1 FROM users u WHERE u.id = p.publisher_id AND u.role = 'teacher')
ON CONFLICT (user_id) DO NOTHING;

COMMIT;
