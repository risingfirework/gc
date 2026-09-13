"use client";

import { Suspense, useEffect, useRef, useState } from "react";
import { useRouter } from "next/navigation";
import { api, AppNotification } from "@/services/api";

function formatWhen(value: string) {
  const seconds = Math.max(0, Math.floor((Date.now() - new Date(value).getTime()) / 1000));
  if (seconds < 60) return "Baru saja";
  const minutes = Math.floor(seconds / 60);
  if (minutes < 60) return `${minutes} menit lalu`;
  const hours = Math.floor(minutes / 60);
  if (hours < 24) return `${hours} jam lalu`;
  const days = Math.floor(hours / 24);
  if (days < 7) return `${days} hari lalu`;
  return new Date(value).toLocaleDateString("id-ID");
}

function NotificationBellInner() {
  const router = useRouter();
  const [open, setOpen] = useState(false);
  const [items, setItems] = useState<AppNotification[]>([]);
  const [unread, setUnread] = useState(0);
  const rootRef = useRef<HTMLDivElement>(null);

  async function refresh() {
    try {
      const [list, count] = await Promise.all([api.notifications(), api.notificationUnreadCount()]);
      setItems(list);
      setUnread(count);
    } catch {
      /* bell degrades silently when offline */
    }
  }

  useEffect(() => {
    const initial = window.setTimeout(() => void refresh(), 0);
    const timer = window.setInterval(() => void refresh(), 60_000);
    return () => { window.clearTimeout(initial); window.clearInterval(timer); };
  }, []);

  useEffect(() => {
    if (!open) return;
    function onPointerDown(event: MouseEvent) {
      if (rootRef.current && !rootRef.current.contains(event.target as Node)) setOpen(false);
    }
    function onKeyDown(event: KeyboardEvent) {
      if (event.key === "Escape") setOpen(false);
    }
    document.addEventListener("mousedown", onPointerDown);
    document.addEventListener("keydown", onKeyDown);
    return () => {
      document.removeEventListener("mousedown", onPointerDown);
      document.removeEventListener("keydown", onKeyDown);
    };
  }, [open]);

  async function openItem(item: AppNotification) {
    if (!item.read_at) {
      setItems((current) => current.map((entry) => entry.id === item.id ? { ...entry, read_at: entry.read_at ?? new Date().toISOString() } : entry));
      setUnread((count) => Math.max(0, count - 1));
      void api.markNotificationRead(item.id).catch(() => undefined);
    }
    setOpen(false);
    if (item.link) router.push(item.link);
  }

  async function markAllRead() {
    setItems((current) => current.map((entry) => entry.read_at ? entry : { ...entry, read_at: new Date().toISOString() }));
    setUnread(0);
    setOpen(false);
    void api.markAllNotificationsRead().catch(() => undefined);
  }

  return <div className="notification-bell" ref={rootRef}>
    <button type="button" className="notification-bell-button" aria-label={unread > 0 ? `${unread} notifikasi belum dibaca` : "Notifikasi"} aria-expanded={open} onClick={() => { setOpen((current) => !current); if (!open) void refresh(); }}>
      <svg viewBox="0 0 24 24" width="20" height="20" fill="none" stroke="currentColor" strokeWidth="2" strokeLinecap="round" strokeLinejoin="round" aria-hidden="true"><path d="M18 8a6 6 0 0 0-12 0c0 7-3 9-3 9h18s-3-2-3-9"/><path d="M13.73 21a2 2 0 0 1-3.46 0"/></svg>
      {unread > 0 && <span className="notification-bell-badge">{unread > 9 ? "9+" : unread}</span>}
    </button>
    {open && <div className="notification-popover">
      <div className="notification-popover-head"><strong>Notifikasi</strong>{unread > 0 && <button type="button" onClick={() => void markAllRead()}>Tandai semua dibaca</button>}</div>
      {items.length === 0
        ? <p className="notification-empty">Tidak ada notifikasi.</p>
        : <ul className="notification-list">{items.map((item) => <li key={item.id}><button type="button" className={item.read_at ? "read" : "unread"} onClick={() => void openItem(item)}><span className="notification-dot"/><span className="notification-text"><strong>{item.title}</strong><span>{item.body}</span><small>{formatWhen(item.created_at)}</small></span></button></li>)}</ul>}
    </div>}
  </div>;
}

export default function NotificationBell() {
  return <Suspense fallback={<span className="notification-bell" />}><NotificationBellInner /></Suspense>;
}