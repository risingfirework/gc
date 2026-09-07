-- Sinkronisasi paket guru yang sudah publish sebelum mekanisme komisi Versi B aktif.
-- Indeks unik pada (package_id, kind) menjamin honorarium tetap hanya satu kali.
INSERT INTO teacher_commissions (
    teacher_id,
    package_id,
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
    'upload_fee',
    0,
    0,
    fs.teacher_upload_fee,
    'available',
    NOW()
FROM packages p
CROSS JOIN finance_settings fs
WHERE p.publisher_id IS NOT NULL
  AND p.status = 'active'
  AND fs.singleton = TRUE
ON CONFLICT (package_id, kind) WHERE kind = 'upload_fee' DO NOTHING;
