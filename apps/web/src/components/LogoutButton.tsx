"use client";

import { useEffect, useRef, useState } from "react";
import { createPortal } from "react-dom";

export default function LogoutButton({ className, loggingOut, onLogout }: { className: string; loggingOut: boolean; onLogout: () => void }) {
  const [open, setOpen] = useState(false);
  const confirmRef = useRef<HTMLButtonElement>(null);

  useEffect(() => {
    if (!open) return;
    const previousOverflow = document.body.style.overflow;
    document.body.style.overflow = "hidden";
    confirmRef.current?.focus();
    const closeOnEscape = (event: KeyboardEvent) => {
      if (event.key === "Escape" && !loggingOut) setOpen(false);
    };
    window.addEventListener("keydown", closeOnEscape);
    return () => { document.body.style.overflow = previousOverflow; window.removeEventListener("keydown", closeOnEscape); };
  }, [loggingOut, open]);

  return <>
    <button type="button" className={className} disabled={loggingOut} onClick={() => setOpen(true)}>{loggingOut ? "Keluar..." : "Logout"}</button>
    {open && createPortal(<div className="logout-modal-backdrop" role="presentation" onMouseDown={(event) => { if (event.target === event.currentTarget && !loggingOut) setOpen(false); }}>
      <section className="logout-modal" role="dialog" aria-modal="true" aria-labelledby="logout-title" aria-describedby="logout-description">
        <button type="button" className="logout-modal-close" aria-label="Tutup konfirmasi" disabled={loggingOut} onClick={() => setOpen(false)}>×</button>
        <div className="logout-modal-icon" aria-hidden="true"><svg viewBox="0 0 24 24"><path d="M10 17l5-5-5-5m5 5H3m10-8h5a2 2 0 0 1 2 2v12a2 2 0 0 1-2 2h-5"/></svg></div>
        <p className="eyebrow">Konfirmasi keluar</p>
        <h2 id="logout-title">Keluar dari akun?</h2>
        <p id="logout-description">Sesi Anda di perangkat ini akan berakhir. Anda perlu masuk kembali untuk mengakses dashboard.</p>
        <div className="logout-modal-note"><span aria-hidden="true">i</span><p>Pastikan semua pekerjaan atau perubahan sudah tersimpan sebelum keluar.</p></div>
        <div className="logout-modal-actions"><button type="button" className="logout-cancel" disabled={loggingOut} onClick={() => setOpen(false)}>Tetap di sini</button><button ref={confirmRef} type="button" className="logout-confirm" disabled={loggingOut} onClick={onLogout}>{loggingOut ? <><span className="logout-spinner"/>Sedang keluar...</> : "Ya, keluar"}</button></div>
      </section>
    </div>, document.body)}
  </>;
}
