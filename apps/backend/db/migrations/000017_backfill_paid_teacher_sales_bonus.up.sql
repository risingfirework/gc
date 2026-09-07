-- Sinkronisasi transaksi berhasil milik paket guru yang terjadi sebelum
-- ledger komisi Versi B tersedia. Bonus memakai pembayaran aktual siswa.
INSERT INTO teacher_commissions (
    teacher_id,
    package_id,
    transaction_id,
    kind,
    base_amount,
    rate_percent,
    amount,
    status,
    available_at
)
SELECT
    p.publisher_id,
    p.id,
    t.id,
    'sales_bonus',
    t.amount,
    fs.teacher_sales_bonus_percent,
    ROUND((t.amount * fs.teacher_sales_bonus_percent / 100)::numeric, 2),
    'available',
    NOW()
FROM transactions t
JOIN packages p ON p.id = t.package_id
CROSS JOIN finance_settings fs
WHERE t.payment_status = 'paid'
  AND p.publisher_id IS NOT NULL
  AND fs.singleton = TRUE
ON CONFLICT (transaction_id, kind)
WHERE kind IN ('sales_bonus', 'refund_reversal')
DO NOTHING;
