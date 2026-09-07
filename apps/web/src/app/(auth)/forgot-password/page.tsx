"use client";

import Link from "next/link";
import { FormEvent, useState } from "react";
import { api } from "@/services/api";

export default function ForgotPasswordPage() {
  const [loading,setLoading] = useState(false);
  const [error,setError] = useState("");
  const [message,setMessage] = useState("");
  const [resetURL,setResetURL] = useState("");
  async function submit(event:FormEvent<HTMLFormElement>) {
    event.preventDefault(); setLoading(true); setError(""); setMessage(""); setResetURL("");
    const email = String(new FormData(event.currentTarget).get("email"));
    try { const result = await api.forgotPassword(email); setMessage(result.message); setResetURL(result.reset_url ?? ""); }
    catch (reason) { setError(reason instanceof Error ? reason.message : "Permintaan reset gagal diproses."); }
    finally { setLoading(false); }
  }
  return <main className="auth reset-auth"><form className="card reset-card" onSubmit={submit}>
    <Link className="reset-back" href="/login">← Kembali ke login</Link>
    <div className="reset-icon" aria-hidden="true">?</div><p className="eyebrow">Pemulihan akun</p><h1>Lupa kata sandi?</h1><p className="lead">Masukkan email akun Anda. Kami akan mengirimkan tautan untuk membuat kata sandi baru.</p>
    {error&&<p className="error">{error}</p>}{message&&<div className="reset-success"><strong>Periksa email Anda</strong><p>{message}</p>{resetURL&&<a className="button full" href={resetURL}>Buka tautan reset lokal</a>}</div>}
    {!message&&<><label htmlFor="forgot-email">Alamat email</label><input id="forgot-email" name="email" type="email" placeholder="nama@email.com" autoComplete="email" autoFocus required/><button className="button full" disabled={loading}>{loading?"Mengirim...":"Kirim tautan reset"}</button></>}
    <p className="reset-help">Tautan hanya berlaku selama 15 menit dan hanya dapat digunakan satu kali.</p>
  </form></main>;
}
