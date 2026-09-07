-- Honorarium upload adalah nominal tetap, bukan persentase harga jual.
-- Karena itu base_amount harus nol; harga dasar hanya dipakai bonus penjualan.
UPDATE teacher_commissions
SET base_amount = 0,
    rate_percent = 0
WHERE kind = 'upload_fee';
