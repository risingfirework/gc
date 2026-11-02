"use client";

import Link from "next/link";
import type { ReactNode } from "react";

export type EmptyStateIcon = "book" | "catalog" | "exam" | "ranking" | "video" | "cart" | "search";

const icons: Record<EmptyStateIcon, ReactNode> = {
  book: <><path d="M4 5.5A2.5 2.5 0 0 1 6.5 3H11v17H6.5A2.5 2.5 0 0 0 4 22V5.5Z" /><path d="M20 5.5A2.5 2.5 0 0 0 17.5 3H13v17h4.5A2.5 2.5 0 0 1 20 22V5.5Z" /></>,
  catalog: <><rect x="3" y="3" width="7" height="7" rx="2" /><rect x="14" y="3" width="7" height="7" rx="2" /><rect x="3" y="14" width="7" height="7" rx="2" /><rect x="14" y="14" width="7" height="7" rx="2" /></>,
  exam: <><rect x="5" y="4" width="14" height="17" rx="2.5" /><path d="M9 4.5V3h6v1.5M9 9h6M9 13h6M9 17h4" /></>,
  ranking: <><path d="M8 21h8M12 17v4M7 4h10v5a5 5 0 0 1-10 0V4Z" /><path d="M7 6H4v2a3 3 0 0 0 3 3M17 6h3v2a3 3 0 0 1-3 3" /></>,
  video: <><rect x="3" y="5" width="18" height="14" rx="3" /><path d="m10 9 5 3-5 3V9Z" /></>,
  cart: <><path d="M3 4h2l2.4 10.4A2 2 0 0 0 9.35 16H18a2 2 0 0 0 1.95-1.55L21.5 8H6" /><circle cx="9.5" cy="20" r="1.2" /><circle cx="18" cy="20" r="1.2" /></>,
  search: <><circle cx="11" cy="11" r="7" /><path d="m20 20-3.5-3.5" /></>,
};

type EmptyStateAction = { label: string; href?: string; onClick?: () => void };

export default function EmptyState({
  icon = "book",
  title,
  description,
  action,
  secondaryAction,
  variant = "card",
}: {
  icon?: EmptyStateIcon;
  title: string;
  description?: string;
  action?: EmptyStateAction;
  secondaryAction?: EmptyStateAction;
  variant?: "card" | "plain";
}) {
  return <div className={`empty-state-card ${variant === "plain" ? "plain" : ""}`.trim()}>
    <span className="empty-state-icon" aria-hidden="true">
      <svg viewBox="0 0 24 24">{icons[icon]}</svg>
    </span>
    <div>
      <strong>{title}</strong>
      {description && <p>{description}</p>}
    </div>
    {(action || secondaryAction) && <div className="empty-state-actions">
      {action && (action.href
        ? <Link className="button" href={action.href}>{action.label}</Link>
        : <button className="button" type="button" onClick={action.onClick}>{action.label}</button>)}
      {secondaryAction && (secondaryAction.href
        ? <Link className="button secondary" href={secondaryAction.href}>{secondaryAction.label}</Link>
        : <button className="button secondary" type="button" onClick={secondaryAction.onClick}>{secondaryAction.label}</button>)}
    </div>}
  </div>;
}
