BEGIN;

-- Hapus duplikat lisensi: sisakan satu baris terbaik per (user, paket).
DELETE FROM user_packages up
USING (
    SELECT id,
           ROW_NUMBER() OVER (
               PARTITION BY user_id, package_id
               ORDER BY (status = 'active') DESC, expired_at DESC, id DESC
           ) AS rn
    FROM user_packages
) ranked
WHERE up.id = ranked.id AND ranked.rn > 1;

-- Maksimal satu lisensi per (user, paket) agar pembayaran ganda
-- tidak menciptakan hak ganda yang sulit dicabut saat refund.
CREATE UNIQUE INDEX user_packages_user_package_uidx ON user_packages (user_id, package_id);

-- Mempercepat job yang menandai invoice pending kedaluwarsa.
CREATE INDEX transactions_pending_expiry_idx ON transactions (payment_status, expires_at)
    WHERE payment_status = 'pending';

COMMIT;