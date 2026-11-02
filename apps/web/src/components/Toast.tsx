"use client";

import { useEffect, useRef } from "react";

export type ToastTone = "success" | "error" | "info";
export type ToastState = { message: string; tone: ToastTone } | null;

const icons: Record<ToastTone, string> = { success: "✓", error: "!", info: "i" };

export default function Toast({ toast, onClose }: { toast: ToastState; onClose: () => void }) {
  const onCloseRef = useRef(onClose);
  useEffect(() => { onCloseRef.current = onClose; }, [onClose]);
  useEffect(() => {
    if (!toast) return;
    const timer = window.setTimeout(() => onCloseRef.current(), 4800);
    return () => window.clearTimeout(timer);
  }, [toast]);
  if (!toast) return null;
  return <div className={`toast toast-${toast.tone}`} role={toast.tone === "error" ? "alert" : "status"} aria-live={toast.tone === "error" ? "assertive" : "polite"}>
    <span className="toast-icon" aria-hidden="true">{icons[toast.tone]}</span>
    <p>{toast.message}</p>
    <button type="button" aria-label="Tutup notifikasi" onClick={onClose}>×</button>
  </div>;
}
