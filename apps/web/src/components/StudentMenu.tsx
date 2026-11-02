"use client";

import Link from "next/link";
import LogoutButton from "./LogoutButton";

export type StudentMenuView = "belajar" | "katalog" | "ranking" | "video" | "cbt" | "testimoni" | "profil";

function itemClass(active: StudentMenuView, view: StudentMenuView, primary = false) {
  return `${active === view ? "active" : ""}${primary ? " menu-primary" : ""}`.trim();
}

export default function StudentMenu({ active, loggingOut, onLogout }: { active: StudentMenuView; loggingOut: boolean; onLogout: () => void }) {
  return (
    <div className="dashboard-menu" aria-label="Menu pengguna">
      <Link className={itemClass(active, "belajar", true)} href="/dashboard?view=belajar">Belajar Saya</Link>
      <Link className={itemClass(active, "katalog", true)} href="/dashboard?view=katalog">Daftar Katalog</Link>
      <Link className={itemClass(active, "cbt", true)} href="/dashboard?view=cbt">Ujian CBT</Link>
      <Link className={itemClass(active, "ranking", true)} href="/dashboard?view=ranking">Ranking Global</Link>
      <Link className={itemClass(active, "video")} href="/dashboard?view=video">Video</Link>
      <Link className={itemClass(active, "testimoni")} href="/testimoni">Testimoni</Link>
      <Link className={itemClass(active, "profil", true)} href="/dashboard?view=profil">Profil</Link>
      <LogoutButton className="logout-button" loggingOut={loggingOut} onLogout={onLogout}/>
    </div>
  );
}
