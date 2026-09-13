"use client";

import Link from "next/link";
import BrandLogo from "@/components/BrandLogo";
import { useRouter } from "next/navigation";
import { FormEvent, useEffect, useRef, useState } from "react";
import { APIError, api, SiteSettings, User } from "@/services/api";

type AuthMode = "login" | "register";

export default function LoginPage() {
  const router = useRouter();
  const [mode,setMode] = useState<AuthMode>("login");
  const [loading,setLoading] = useState(false);
  const [error,setError] = useState("");
  const [settings,setSettings] = useState<SiteSettings>();
  const [pendingGoogleToken,setPendingGoogleToken] = useState("");
  const [googleLevel,setGoogleLevel] = useState<User["school_level"]>("SMA");
  const [regRole,setRegRole] = useState<"student"|"teacher">("student");
  const [twoFAMode,setTwoFAMode] = useState(false);
  const [mfaToken,setMFAToken] = useState("");
  const [twoFACode,setTwoFACode] = useState("");
  const googleButtonRef = useRef<HTMLDivElement>(null);
  const googleClientID = process.env.NEXT_PUBLIC_GOOGLE_CLIENT_ID;

  useEffect(()=>{api.siteSettings().then(setSettings).catch(()=>undefined)},[]);

  async function loginWithGoogle(idToken:string, schoolLevel?:User["school_level"]) {
    setLoading(true); setError("");
    try { await api.loginWithGoogle(idToken,schoolLevel); router.replace("/dashboard"); }
    catch(reason) {
      if(reason instanceof APIError && reason.status===422){setPendingGoogleToken(idToken)}
      else setError(reason instanceof Error?reason.message:"Login Google gagal diproses.");
    } finally { setLoading(false); }
  }

  useEffect(()=>{
    if(!googleClientID) return;
    const clientID=googleClientID; let cancelled=false;
    function renderGoogleButton(){
      if(cancelled||!window.google||!googleButtonRef.current)return;
      window.google.accounts.id.initialize({client_id:clientID,callback:(response)=>void loginWithGoogle(response.credential)});
      googleButtonRef.current.innerHTML="";
      window.google.accounts.id.renderButton(googleButtonRef.current,{theme:"outline",size:"large",width:360,text:mode==="login"?"signin_with":"signup_with"});
    }
    if(window.google){renderGoogleButton();return}
    const existing=document.querySelector<HTMLScriptElement>('script[src="https://accounts.google.com/gsi/client"]');
    if(existing){existing.addEventListener("load",renderGoogleButton,{once:true});return()=>{cancelled=true}}
    const script=document.createElement("script");script.src="https://accounts.google.com/gsi/client";script.async=true;script.onload=renderGoogleButton;document.body.appendChild(script);
    return()=>{cancelled=true};
    // eslint-disable-next-line react-hooks/exhaustive-deps
  },[googleClientID,mode]);

  function switchMode(next:AuthMode){setMode(next);setError("");setPendingGoogleToken("");setTwoFAMode(false);setMFAToken("");setTwoFACode("")}

  async function submit(event:FormEvent<HTMLFormElement>){
    event.preventDefault();setLoading(true);setError("");const form=new FormData(event.currentTarget);const email=String(form.get("email"));const password=String(form.get("password"));
    try{
      if(mode==="register"){
        if(password!==String(form.get("confirm_password"))){setError("Konfirmasi kata sandi tidak cocok.");return}
        const role=form.get("role") as "student"|"teacher";
        const refParam=typeof window!=="undefined"?new URLSearchParams(window.location.search).get("ref"):"";
        const referralCode=String(form.get("referral_code")||"").trim()||refParam||undefined;
        const teacherKtp=String(form.get("teacher_ktp")||"").trim();
        await api.register(email,password,form.get("school_level") as User["school_level"],role,referralCode,teacherKtp);
      }
      const result=await api.login(email,password);
      if(result.requires_2fa){setMFAToken(result.mfa_token??"");setTwoFAMode(true);return}
      router.replace("/dashboard");
    }catch(reason){setError(reason instanceof Error?reason.message:mode==="login"?"Gagal masuk.":"Pendaftaran gagal.")}
    finally{setLoading(false)}
  }

  async function submitTwoFA(event:FormEvent<HTMLFormElement>){
    event.preventDefault();if(twoFACode.length!==6){setError("Masukkan kode 6 digit dari aplikasi autentikator.");return}
    setLoading(true);setError("");
    try{await api.verify2FA(mfaToken,twoFACode);router.replace("/dashboard")}
    catch(reason){setError(reason instanceof Error?reason.message:"Verifikasi autentikator gagal.")}
    finally{setLoading(false)}
  }

  return <main className="full-auth-page">
    <section className="auth-showcase">
      <Link className="brand auth-brand" href="/"><BrandLogo logoDataURL={settings?.logo_data_url}/><span>{settings?.platform_name??""}</span></Link>
      <div className="auth-showcase-copy">{settings?.platform_tagline&&<span className="auth-showcase-badge">{settings.platform_tagline}</span>}<h1>Belajar terarah.<br/>Hasil lebih maksimal.</h1><p>Kelola latihan, bank soal, dan perkembangan belajar dalam satu platform yang aman dan mudah digunakan.</p><div className="auth-benefits"><span><b>✓</b> Materi sesuai jenjang</span><span><b>✓</b> Analisis hasil lengkap</span><span><b>✓</b> Akses aman dan terintegrasi</span></div></div>
      <p className="auth-showcase-foot">© {new Date().getFullYear()}{settings?.platform_name?` ${settings.platform_name}`:""}</p>
    </section>
    <section className="auth-form-side">
      <Link className="auth-cancel" href="/" aria-label="Batalkan login dan kembali ke beranda"><span aria-hidden="true">←</span> Batal dan kembali ke beranda</Link>
      <div className="full-auth-card">
        <Link className="auth-mobile-brand brand" href="/"><BrandLogo logoDataURL={settings?.logo_data_url}/>{settings?.platform_name??""}</Link>
        {twoFAMode?<><p className="eyebrow">Verifikasi dua langkah</p><h2>Masukkan kode autentikator</h2><p className="auth-intro">Buka aplikasi autentikator Anda dan masukkan kode 6 digit yang ditampilkan untuk akun ini.</p>
          {error&&<p className="error">{error}</p>}
          <form className="full-auth-form" onSubmit={submitTwoFA}>
            <label htmlFor="login-2fa">Kode autentikator</label><input id="login-2fa" name="code" type="text" inputMode="numeric" autoComplete="one-time-code" placeholder="000000" minLength={6} maxLength={6} value={twoFACode} onChange={(event)=>setTwoFACode(event.target.value.replace(/\D/g,""))} autoFocus required/>
            <button className="button full auth-submit" disabled={loading||twoFACode.length!==6}>{loading?"Memverifikasi...":"Masuk ke dashboard"}</button>
          </form>
          <p className="auth-hint"><button type="button" className="reset-back link" onClick={()=>{setTwoFAMode(false);setMFAToken("");setTwoFACode("");setError("")}}>← Kembali ke form masuk</button></p>
        </>:<>
        <p className="eyebrow">{mode==="login"?"Selamat datang kembali":"Mulai perjalanan belajar"}</p><h2>{mode==="login"?"Masuk ke akun Anda":"Buat akun baru"}</h2><p className="auth-intro">{mode==="login"?"Gunakan akun yang telah terdaftar untuk melanjutkan.":"Daftar sebagai siswa atau guru dalam beberapa langkah."}</p>
        <div className="full-auth-tabs"><button type="button" className={mode==="login"?"active":""} onClick={()=>switchMode("login")}>Masuk</button><button type="button" className={mode==="register"?"active":""} onClick={()=>switchMode("register")}>Daftar</button></div>
        {error&&<p className="error">{error}</p>}
        <div className="google-auth-area">
          {googleClientID?<div className="google-button-slot full" ref={googleButtonRef}/>:<button type="button" className="google-placeholder" onClick={()=>setError("Login Google belum aktif. Administrator perlu mengatur GOOGLE_CLIENT_ID.")}><GoogleIcon/><span>{mode==="login"?"Masuk":"Daftar"} dengan Google</span></button>}
          {pendingGoogleToken&&<div className="google-level-confirm"><p>Akun Google baru terdeteksi. Pilih jenjang sekolah:</p><select value={googleLevel} onChange={(event)=>setGoogleLevel(event.target.value as User["school_level"])}><option value="SD">SD</option><option value="SMP">SMP</option><option value="SMA">SMA / SMK</option></select><button type="button" className="button full" disabled={loading} onClick={()=>void loginWithGoogle(pendingGoogleToken,googleLevel)}>Lanjutkan dengan Google</button></div>}
        </div>
        <div className="auth-divider"><span>atau gunakan email</span></div>
        <form className="full-auth-form" onSubmit={submit}>
          <label htmlFor="login-email">Alamat email</label><input id="login-email" name="email" type="email" placeholder="nama@email.com" autoComplete="email" required/>
          <div className="auth-label-row"><label htmlFor="login-password">Kata sandi</label>{mode==="login"&&<Link href="/forgot-password">Lupa kata sandi?</Link>}</div><input id="login-password" name="password" type="password" placeholder="Minimal 8 karakter" minLength={8} maxLength={72} autoComplete={mode==="login"?"current-password":"new-password"} required/>
          {mode==="register"&&<><label htmlFor="login-confirm">Konfirmasi kata sandi</label><input id="login-confirm" name="confirm_password" type="password" minLength={8} maxLength={72} autoComplete="new-password" required/><div className="auth-register-grid"><label>Daftar sebagai<select name="role" value={regRole} onChange={(event)=>setRegRole(event.target.value as "student"|"teacher")}><option value="student">Siswa</option><option value="teacher">Guru</option></select></label><label>Jenjang sekolah<select name="school_level" defaultValue="SMA"><option value="SD">SD</option><option value="SMP">SMP</option><option value="SMA">SMA / SMK</option></select></label></div>{regRole==="teacher"&&<><label htmlFor="login-ktp">No. KTP (NIK)</label><input id="login-ktp" name="teacher_ktp" type="text" inputMode="numeric" placeholder="16 digit nomor NIK pada KTP" minLength={16} maxLength={16} required/><p className="auth-hint">Data Anda akan diverifikasi oleh operator bahwa Anda terdaftar dalam data pendidikan nasional.</p></>}{regRole==="student"&&<><label htmlFor="login-referral">Kode rujukan <span className="auth-optional">(opsional)</span></label><input id="login-referral" name="referral_code" type="text" placeholder="Contoh: MITRA-A1B2C3D4" autoComplete="off"/><p className="auth-hint">Pakai kode dari affiliate untuk mendukung temanmu yang memperkenalkan platform ini.</p></>}</>}
          <button className="button full auth-submit" disabled={loading}>{loading?"Memproses...":mode==="login"?"Masuk ke dashboard":"Buat akun"}</button>
        </form></>}
        <p className="auth-legal">Dengan melanjutkan, Anda menyetujui penggunaan akun sesuai kebijakan layanan{settings?.platform_name?` ${settings.platform_name}`:""}.</p>
      </div>
    </section>
  </main>;
}

function GoogleIcon(){return <svg viewBox="0 0 24 24" aria-hidden="true"><path fill="#4285F4" d="M22.6 12.2c0-.7-.1-1.5-.2-2.2H12v4.3h6a5.1 5.1 0 0 1-2.2 3.3v2.8h3.6c2.1-2 3.2-4.8 3.2-8.2Z"/><path fill="#34A853" d="M12 23c3 0 5.5-1 7.4-2.6l-3.6-2.8c-1 .7-2.3 1.1-3.8 1.1-2.9 0-5.4-2-6.3-4.6H2v2.9A11.2 11.2 0 0 0 12 23Z"/><path fill="#FBBC05" d="M5.7 14.1a6.7 6.7 0 0 1 0-4.2V7H2a11 11 0 0 0 0 10l3.7-2.9Z"/><path fill="#EA4335" d="M12 5.3c1.7 0 3.1.6 4.3 1.7l3.2-3.1A10.8 10.8 0 0 0 2 7l3.7 2.9C6.6 7.2 9.1 5.3 12 5.3Z"/></svg>}
