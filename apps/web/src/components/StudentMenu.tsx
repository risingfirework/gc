"use client";

import Link from "next/link";
import LogoutButton from "./LogoutButton";

export type StudentMenuView = "belajar" | "katalog" | "ranking" | "video" | "cbt" | "testimoni" | "profil";

export default function StudentMenu({ active, loggingOut, onLogout }: { active: StudentMenuView; loggingOut: boolean; onLogout: () => void }) {
  return (
    <div className="dashboard-menu" aria-label="Menu pengguna">
      <Link className={active === "belajar" ? "active" : ""} href="/dashboard?view=belajar">Belajar Saya</Link>
      <Link className={active === "katalog" ? "active" : ""} href="/dashboard?view=katalog">Daftar Katalog</Link>
      <Link className={active === "cbt" ? "active" : ""} href="/dashboard?view=cbt">Ujian CBT</Link>
      <Link className={active === "ranking" ? "active" : ""} href="/dashboard?view=ranking">Ranking Global</Link>
      <Link className={active === "video" ? "active" : ""} href="/dashboard?view=video">Video</Link>
      <Link className={active === "testimoni" ? "active" : ""} href="/testimoni">Testimoni</Link>
      <Link className={active === "profil" ? "active" : ""} href="/dashboard?view=profil">Profil</Link>
      <LogoutButton className="logout-button" loggingOut={loggingOut} onLogout={onLogout}/>
    </div>
  );
}
