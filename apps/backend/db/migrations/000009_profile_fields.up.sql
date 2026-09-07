-- Profil pengguna: nama, tanggal lahir, dan nomor HP yang bisa diubah sendiri.
ALTER TABLE users
    ADD COLUMN name VARCHAR(120) NOT NULL DEFAULT '',
    ADD COLUMN birth_date DATE NULL,
    ADD COLUMN phone VARCHAR(20) NOT NULL DEFAULT '';

UPDATE users SET name = split_part(email, '@', 1) WHERE name = '';