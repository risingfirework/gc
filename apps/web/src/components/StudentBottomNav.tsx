"use client";

import Link from "next/link";
import type { ReactNode } from "react";
import type { StudentMenuView } from "./StudentMenu";

type BottomNavItem = { view: StudentMenuView; href: string; label: string; icon: ReactNode };

const items: BottomNavItem[] = [
  {
    view: "belajar",
    href: "/dashboard?view=belajar",
    label: "Belajar",
    icon: <><path d="M4 5.5A2.5 2.5 0 0 1 6.5 3H11v17H6.5A2.5 2.5 0 0 0 4 22V5.5Z" /><path d="M20 5.5A2.5 2.5 0 0 0 17.5 3H13v17h4.5A2.5 2.5 0 0 1 20 22V5.5Z" /></>,
  },
  {
    view: "katalog",
    href: "/dashboard?view=katalog",
    label: "Katalog",
    icon: <><rect x="3" y="3" width="7" height="7" rx="2" /><rect x="14" y="3" width="7" height="7" rx="2" /><rect x="3" y="14" width="7" height="7" rx="2" /><rect x="14" y="14" width="7" height="7" rx="2" /></>,
  },
  {
    view: "cbt",
    href: "/dashboard?view=cbt",
    label: "CBT",
    icon: <><rect x="5" y="4" width="14" height="17" rx="2.5" /><path d="M9 4.5V3h6v1.5M9 9h6M9 13h6M9 17h4" /></>,
  },
  {
    view: "ranking",
    href: "/dashboard?view=ranking",
    label: "Ranking",
    icon: <><path d="M8 21h8M12 17v4M7 4h10v5a5 5 0 0 1-10 0V4Z" /><path d="M7 6H4v2a3 3 0 0 0 3 3M17 6h3v2a3 3 0 0 1-3 3" /></>,
  },
  {
    view: "profil",
    href: "/dashboard?view=profil",
    label: "Profil",
    icon: <><circle cx="12" cy="8" r="4" /><path d="M4.5 21a7.5 7.5 0 0 1 15 0" /></>,
  },
];

export default function StudentBottomNav({ active }: { active: StudentMenuView }) {
  return <nav className="student-bottom-nav" aria-label="Navigasi utama siswa">
    <div className="student-bottom-nav-inner">
      {items.map((item) => <Link key={item.view} className={active === item.view ? "active" : ""} href={item.href} aria-current={active === item.view ? "page" : undefined}>
        <svg viewBox="0 0 24 24" aria-hidden="true">{item.icon}</svg>
        <span>{item.label}</span>
      </Link>)}
    </div>
  </nav>;
}
