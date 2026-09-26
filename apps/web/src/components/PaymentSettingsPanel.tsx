"use client";

import { FormEvent, useCallback, useEffect, useState } from "react";
import { api, PaymentSettings, PaymentSettingsInput } from "@/services/api";

export default function PaymentSettingsPanel() {
  const [settings, setSettings] = useState<PaymentSettings>();
  const [saving, setSaving] = useState(false);
  const [message, setMessage] = useState("");
  const [error, setError] = useState("");
  const load = useCallback(() => api.paymentSettings().then(setSettings).catch((reason) => setError(reason instanceof Error ? reason.message : "Pengaturan pembayaran gagal dimuat.")), []);
  useEffect(() => { void load(); }, [load]);

  async function save(event: FormEvent<HTMLFormElement>) {
    event.preventDefault(); setSaving(true); setError(""); setMessage("");
    const form = new FormData(event.currentTarget);
    const input: PaymentSettingsInput = {};
    const baseUrl = String(form.get("base_url") ?? "").trim();
    const success = String(form.get("success_redirect_url") ?? "").trim();
    const failure = String(form.get("failure_redirect_url") ?? "").trim();
    const secret = String(form.get("secret_key") ?? "").trim();
    const token = String(form.get("webhook_token") ?? "").trim();
    if (baseUrl) input.base_url = baseUrl;
    // Kedua field selalu dikirim agar string kosong berarti menghapus redirect,
    // bukan mempertahankan nilai yang sebelumnya tersimpan.
    input.success_redirect_url = success;
    input.failure_redirect_url = failure;
    if (secret) input.secret_key = secret;
    if (token) input.webhook_token = token;
    try {
      const saved = await api.updatePaymentSettings(input);
      setSettings(saved);
      event.currentTarget.reset();
      setMessage("Konfigurasi tersimpan dan langsung berlaku tanpa restart server.");
    } catch (reason) { setError(reason instanceof Error ? reason.message : "Pengaturan gagal disimpan."); }
    finally { setSaving(false); }
  }

  if (!settings && !error) return <section className="admin-view"><div className="card">Memuat pengaturan pembayaran...</div></section>;
  if (!settings) return <section className="admin-view"><p className="error">{error}</p></section>;

  return <form className="settings-view" onSubmit={save}>
    <div className="settings-heading"><div><p className="eyebrow">Integrasi pembayaran</p><h2>Penyedia pembayaran (Xendit)</h2><p className="muted">Isi kredensial dari dashboard Xendit. Nilai rahasia disimpan terenkripsi dan hanya ditampilkan tersamar.</p></div><button className="button" disabled={saving}>{saving ? "Menyimpan..." : "Simpan konfigurasi"}</button></div>
    {message && <p className="finance-success">{message}</p>}{error && <p className="error">{error}</p>}
    <div className="payment-status-row">
      <span className={`status-pill ${settings.secret_key_configured ? "active" : "inactive"}`}>Secret key{settings.secret_key_configured ? `: ${settings.secret_key_masked}` : ": belum diatur"}</span>
      <span className={`status-pill ${settings.webhook_token_configured ? "active" : "inactive"}`}>Webhook token{settings.webhook_token_configured ? ": " + settings.webhook_token_masked : ": belum diatur"}</span>
      <span className={`status-pill ${settings.environment === "production" ? "active" : "inactive"}`}>Lingkungan: {settings.environment}</span>
    </div>
    <section className="card settings-card"><div className="settings-card-title"><div><h3>Kredensial</h3><p>Kosongkan bila mempertahankan nilai yang sudah tersimpan.</p></div><span>Rahasia</span></div><div className="settings-grid two">
      <label>Secret key Xendit<input type="password" name="secret_key" autoComplete="new-password" placeholder={settings.secret_key_configured ? settings.secret_key_masked : "xnd_development_..."} minLength={20}/></label>
      <label>Webhook callback token<input type="password" name="webhook_token" autoComplete="new-password" minLength={32} placeholder={settings.webhook_token_configured ? settings.webhook_token_masked : "Minimal 32 karakter"} /></label>
    </div><small className="form-helper">Callback token dipakai untuk memverifikasi webhook pembayaran via header X-Callback-Token.</small></section>
    <section className="card settings-card"><div className="settings-card-title"><div><h3>Endpoint</h3><p>URL di luar localhost wajib memakai https.</p></div><span>Networking</span></div><div className="settings-grid">
      <label>Base URL Xendit<input type="url" name="base_url" defaultValue={settings.base_url} placeholder="https://api.xendit.co"/></label>
      <label>Redirect sukses<input type="url" name="success_redirect_url" defaultValue={settings.success_redirect_url} placeholder="https://app.example.com/payment/success"/></label>
      <label>Redirect gagal<input type="url" name="failure_redirect_url" defaultValue={settings.failure_redirect_url} placeholder="https://app.example.com/payment/failure"/></label>
    </div><small className="form-helper">Redirect opsional; untuk pengembangan lokal diizinkan memakai http://localhost.</small></section>
    <section className="card settings-card"><div className="settings-card-title"><div><h3>Panduan</h3><p>Langkah penerapan di dashboard Xendit.</p></div><span>Alur</span></div><ul className="settings-tips">
      <li>Salin Secret Key dari menu Develop → API Keys (awalan <code>xnd_development_</code> untuk sandbox).</li>
      <li>Di menu Settings → Webhooks, daftarkan endpoint <code>&quot;https://&lt;domain&gt;/api/v1/webhooks/payment&quot;</code> dengan callback token acak minimal 32 karakter.</li>
      <li>Harga, masa aktif, dan metode pembayaran diatur paket per paket lewat katalog.</li>
      <li>Perubahan di panel ini langsung dipakai tanpa me-restart server.</li>
    </ul></section>
  </form>;
}
