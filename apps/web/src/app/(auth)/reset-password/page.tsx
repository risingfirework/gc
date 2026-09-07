"use client";

import Link from "next/link";
import { useSearchParams } from "next/navigation";
import { FormEvent, Suspense, useState } from "react";
import { api } from "@/services/api";

export default function ResetPasswordPage(){return <Suspense fallback={<main className="auth"><div className="card reset-card">Memuat...</div></main>}><ResetPasswordForm/></Suspense>}

function ResetPasswordForm() {
  const token = useSearchParams().get("token") ?? "";
  const [loading,setLoading] = useState(false); const [error,setError] = useState(""); const [success,setSuccess] = useState("");
  async function submit(event:FormEvent<HTMLFormElement>) {
    event.preventDefault(); const form = new FormData(event.currentTarget); const password=String(form.get("password"));
    if(password!==String(form.get("confirm_password"))){setError("Konfirmasi kata sandi tidak cocok.");return}
    setLoading(true);setError("");
    try{const result=await api.resetPassword(token,password);setSuccess(result.message)}catch(reason){setError(reason instanceof Error?reason.message:"Reset kata sandi gagal.")}finally{setLoading(false)}
  }
  return <main className="auth reset-auth"><form className="card reset-card" onSubmit={submit}>
    <Link className="reset-back" href="/login">← Kembali ke login</Link><div className="reset-icon lock" aria-hidden="true">✓</div><p className="eyebrow">Keamanan akun</p><h1>Buat kata sandi baru</h1><p className="lead">Gunakan kata sandi unik minimal 8 karakter yang belum pernah dibagikan kepada orang lain.</p>
    {!token&&<p className="error">Tautan reset tidak lengkap. Silakan ajukan permintaan baru.</p>}{error&&<p className="error">{error}</p>}{success?<div className="reset-success"><strong>Kata sandi diperbarui</strong><p>{success}</p><Link className="button full" href="/login">Masuk sekarang</Link></div>:token&&<><label htmlFor="reset-password">Kata sandi baru</label><input id="reset-password" name="password" type="password" minLength={8} maxLength={72} autoComplete="new-password" autoFocus required/><label htmlFor="reset-confirm">Konfirmasi kata sandi baru</label><input id="reset-confirm" name="confirm_password" type="password" minLength={8} maxLength={72} autoComplete="new-password" required/><button className="button full" disabled={loading}>{loading?"Memperbarui...":"Simpan kata sandi baru"}</button></>}
    <p className="reset-help">Setelah berhasil, seluruh sesi lama akan dikeluarkan untuk melindungi akun Anda.</p>
  </form></main>;
}
