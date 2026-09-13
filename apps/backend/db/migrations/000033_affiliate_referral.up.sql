BEGIN;

-- Role affiliate baru: penjual rujukan, terpisah dari teacher.
ALTER TABLE users DROP CONSTRAINT users_role_check;
ALTER TABLE users ADD CONSTRAINT users_role_check CHECK (role IN ('student', 'teacher', 'admin', 'owner', 'finance', 'affiliate'));

-- Tingkat komisi affiliate sebagai persentase dari jatah platform (dari gross).
ALTER TABLE finance_settings
    ADD COLUMN affiliate_rate_percent NUMERIC(5,2) NOT NULL DEFAULT 10
        CONSTRAINT finance_settings_affiliate_rate_percent_check CHECK (affiliate_rate_percent >= 0 AND affiliate_rate_percent <= 100);

-- Profil affiliate dengan kode rujukan unik.
CREATE TABLE affiliates (
    id            UUID PRIMARY KEY REFERENCES users(id) ON DELETE CASCADE,
    referral_code VARCHAR(24) NOT NULL,
    created_at    TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE UNIQUE INDEX affiliates_referral_code_uq ON affiliates (referral_code);

-- Atribusi first-touch: satu pengguna hanya dapat dirujuk sekali.
CREATE TABLE referrals (
    id               UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    affiliate_id     UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    referred_user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    referred_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    first_purchase_at TIMESTAMPTZ,
    CONSTRAINT referrals_referred_user_uq UNIQUE (referred_user_id)
);

CREATE INDEX referrals_affiliate_idx ON referrals (affiliate_id, referred_at DESC);

-- Perluas jenis komisi untuk bonus rujukan affiliate (+ reversal refund).
ALTER TABLE teacher_commissions DROP CONSTRAINT teacher_commissions_kind_check;
ALTER TABLE teacher_commissions ADD CONSTRAINT teacher_commissions_kind_check CHECK (kind IN ('upload_fee','sales_bonus','refund_reversal','referral_bonus','referral_reversal'));

-- Satu bonus rujukan per transaksi terkonfirmasi (anti duplikat pembayaran/webhook).
CREATE UNIQUE INDEX teacher_commissions_referral_once_idx
    ON teacher_commissions (transaction_id, kind)
    WHERE kind IN ('referral_bonus','referral_reversal') AND split_group IS NULL;

COMMIT;