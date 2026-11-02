import type { CSSProperties } from "react";

export function Skeleton({ className = "", style }: { className?: string; style?: CSSProperties }) {
  return <span className={`skeleton ${className}`.trim()} style={style} aria-hidden="true" />;
}

export function SkeletonText({ lines = 3, className = "" }: { lines?: number; className?: string }) {
  return <span className={`skeleton-text ${className}`.trim()} aria-hidden="true">
    {Array.from({ length: lines }).map((_, index) => <Skeleton className="skeleton-line" key={index} style={{ width: index === lines - 1 ? "66%" : "100%" }} />)}
  </span>;
}

export function SkeletonPackageGrid({ count = 3 }: { count?: number }) {
  return <div className="admin-package-grid student-catalog-grid" aria-hidden="true">
    {Array.from({ length: count }).map((_, index) => <div className="card skeleton-package-card" key={index}>
      <Skeleton className="skeleton-eyebrow" />
      <Skeleton className="skeleton-title" />
      <SkeletonText lines={3} />
      <Skeleton className="skeleton-price" />
      <Skeleton className="skeleton-button" />
    </div>)}
  </div>;
}

export function SkeletonOwnedGrid({ count = 2 }: { count?: number }) {
  return <div className="grid owned-grid" aria-hidden="true">
    {Array.from({ length: count }).map((_, index) => <div className="card skeleton-owned-card" key={index}>
      <Skeleton className="skeleton-eyebrow" />
      <Skeleton className="skeleton-title" />
      <SkeletonText lines={2} />
      <Skeleton className="skeleton-row" />
      <Skeleton className="skeleton-row" />
    </div>)}
  </div>;
}

export function SkeletonExamRows({ count = 2 }: { count?: number }) {
  return <div className="package-exams skeleton-exam-rows" aria-hidden="true">
    {Array.from({ length: count }).map((_, index) => <Skeleton className="skeleton-row" key={index} />)}
  </div>;
}
