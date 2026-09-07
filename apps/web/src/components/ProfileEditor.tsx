"use client";

import { FormEvent, useState } from "react";
import { api, APIError, User } from "@/services/api";

const JENJANGS = ["SD", "SMP", "SMA"] as const;
const roleLabel: Record<User["role"], string> = { student: "Siswa", teacher: "Guru", admin: "Administrator" };
type ProfileTab = "personal" | "security";
type Props = { user: User; onUpdated?: (user: User) => void };

export default function ProfileEditor({ user, onUpdated }: Props) {
  const [tab, setTab] = useState<ProfileTab>("personal");
  const [name, setName] = useState(user.name ?? "");
  const [birthDate, setBirthDate] = useState(user.birth_date ?? "");
  const [phone, setPhone] = useState(user.phone ?? "");
  const [schoolLevel, setSchoolLevel] = useState(user.school_level);
  const [currentPassword, setCurrentPassword] = useState("");
  const [newPassword, setNewPassword] = useState("");
  const [confirmPassword, setConfirmPassword] = useState("");
  const [saving, setSaving] = useState(false);
  const [message, setMessage] = useState("");
  const [error, setError] = useState("");

  function switchTab(next: ProfileTab) {
    setTab(next); setMessage(""); setError("");
  }

  async function savePersonal(event: FormEvent<HTMLFormElement>) {
    event.preventDefault();
    setSaving(true); setError(""); setMessage("");
    try {
      const updated = await api.updateProfile({ name, birth_date: birthDate || undefined, phone, school_level: schoolLevel });
      setMessage("Data pribadi berhasil diperbarui.");
      onUpdated?.(updated);
    } catch (reason) { setError(reason instanceof APIError ? reason.message : reason instanceof Error ? reason.message : "Data pribadi gagal disimpan."); }
    finally { setSaving(false); }
  }

  async function savePassword(event: FormEvent<HTMLFormElement>) {
    event.preventDefault();
    if (newPassword.length < 8) { setError("Kata sandi baru minimal 8 karakter."); return; }
    if (newPassword !== confirmPassword) { setError("Konfirmasi kata sandi tidak cocok."); return; }
    setSaving(true); setError(""); setMessage("");
    try {
      const updated = await api.updateProfile({ name, birth_date: birthDate || undefined, phone, school_level: schoolLevel, current_password: currentPassword, new_password: newPassword });
      setMessage("Kata sandi berhasil diperbarui.");
      onUpdated?.(updated);
      setCurrentPassword(""); setNewPassword(""); setConfirmPassword("");
    } catch (reason) { setError(reason instanceof APIError ? reason.message : reason instanceof Error ? reason.message : "Kata sandi gagal diperbarui."); }
    finally { setSaving(false); }
  }

  return <div className="card profile-editor">
    <div className="profile-editor-head">
      <div className="profile-avatar large">{name ? name[0].toUpperCase() : user.email[0].toUpperCase()}</div>
      <div className="profile-editor-identity"><h3>{name || "Nama belum dilengkapi"}</h3><p>{user.email}</p><span className={`profile-role ${user.role}`}>{roleLabel[user.role]}</span></div>
    </div>

    <div className="profile-tabs" role="tablist" aria-label="Pengaturan profil">
      <button type="button" role="tab" aria-selected={tab === "personal"} className={tab === "personal" ? "active" : ""} onClick={() => switchTab("personal")}><span aria-hidden="true">◎</span><div>Data Pribadi<small>Identitas dan informasi kontak</small></div></button>
      <button type="button" role="tab" aria-selected={tab === "security"} className={tab === "security" ? "active" : ""} onClick={() => switchTab("security")}><span aria-hidden="true">⌑</span><div>Keamanan<small>Ubah kata sandi akun</small></div></button>
    </div>

    {tab === "personal" && <form className="profile-tab-panel" onSubmit={savePersonal}>
      <div className="profile-form-grid">
        <div className="form-field"><label htmlFor="pe-name">Nama lengkap</label><input id="pe-name" value={name} onChange={(event) => setName(event.target.value)} placeholder="Masukkan nama lengkap" maxLength={120}/></div>
        <div className="form-field"><label htmlFor="pe-email">Alamat email</label><input id="pe-email" value={user.email} disabled readOnly/><small>Email digunakan sebagai identitas masuk.</small></div>
        <div className="form-field"><label htmlFor="pe-birth">Tanggal lahir</label><input id="pe-birth" type="date" value={birthDate} onChange={(event) => setBirthDate(event.target.value)}/></div>
        <div className="form-field"><label htmlFor="pe-phone">Nomor telepon</label><input id="pe-phone" value={phone} onChange={(event) => setPhone(event.target.value)} placeholder="Contoh: 081234567890" maxLength={20} inputMode="tel" autoComplete="tel"/></div>
        <div className="form-field"><label htmlFor="pe-level">Jenjang sekolah</label><select id="pe-level" value={schoolLevel} onChange={(event) => setSchoolLevel(event.target.value as User["school_level"])}>{JENJANGS.map((level) => <option key={level} value={level}>{level === "SMA" ? "SMA / SMK" : level}</option>)}</select></div>
      </div>
      {(message || error) && <p className={error ? "form-error" : "form-success"}>{error || message}</p>}
      <div className="profile-form-actions"><span>Terakhir diperbarui {new Date(user.updated_at).toLocaleDateString("id-ID", { day:"2-digit", month:"short", year:"numeric" })}</span><button className="button" disabled={saving}>{saving ? "Menyimpan..." : "Simpan data pribadi"}</button></div>
    </form>}

    {tab === "security" && <form className="profile-tab-panel security-panel" onSubmit={savePassword}>
      <div className="security-note"><span aria-hidden="true">✓</span><div><strong>Jaga keamanan akun Anda</strong><p>Gunakan kata sandi unik minimal 8 karakter dan jangan membagikannya kepada siapa pun.</p></div></div>
      <div className="profile-form-grid password-grid">
        <div className="form-field current-password"><label htmlFor="pe-cur">Kata sandi saat ini</label><input id="pe-cur" type="password" value={currentPassword} onChange={(event) => setCurrentPassword(event.target.value)} autoComplete="current-password" required/></div>
        <div className="form-field"><label htmlFor="pe-new">Kata sandi baru</label><input id="pe-new" type="password" value={newPassword} onChange={(event) => setNewPassword(event.target.value)} autoComplete="new-password" minLength={8} maxLength={72} required/><small>Minimal 8 karakter.</small></div>
        <div className="form-field"><label htmlFor="pe-confirm">Konfirmasi kata sandi baru</label><input id="pe-confirm" type="password" value={confirmPassword} onChange={(event) => setConfirmPassword(event.target.value)} autoComplete="new-password" minLength={8} maxLength={72} required/></div>
      </div>
      {(message || error) && <p className={error ? "form-error" : "form-success"}>{error || message}</p>}
      <div className="profile-form-actions"><span>Anda akan tetap masuk setelah kata sandi diperbarui.</span><button className="button" disabled={saving}>{saving ? "Memperbarui..." : "Perbarui kata sandi"}</button></div>
    </form>}
  </div>;
}
