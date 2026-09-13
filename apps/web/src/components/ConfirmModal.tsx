"use client";

import { createPortal } from "react-dom";

export default function ConfirmModal({ title, message, busy = false, confirmLabel = "Ya, hapus", onClose, onConfirm }: { title: string; message: string; busy?: boolean; confirmLabel?: string; onClose: () => void; onConfirm: () => void }) {
  return createPortal(
    <div className="verify-modal-backdrop" role="presentation" onMouseDown={(event) => { if (event.target === event.currentTarget) onClose(); }}>
      <section className="verify-modal" role="dialog" aria-modal="true" aria-labelledby="confirm-title" onMouseDown={(event) => event.stopPropagation()}>
        <button type="button" className="verify-modal-close" disabled={busy} onClick={onClose}>×</button>
        <div className="verify-modal-icon verify-modal-icon--danger" aria-hidden="true"><svg viewBox="0 0 24 24"><path d="M18 6L6 18M6 6l12 12"/></svg></div>
        <p className="eyebrow">Konfirmasi</p>
        <h2 id="confirm-title">{title}</h2>
        <p className="muted">{message}</p>
        <div className="verify-modal-actions">
          <button type="button" className="verify-cancel" disabled={busy} onClick={onClose}>Batal</button>
          <button type="button" className="verify-confirm-reject" disabled={busy} onClick={onConfirm}>{busy ? "Memproses..." : confirmLabel}</button>
        </div>
      </section>
    </div>,
    document.body,
  );
}