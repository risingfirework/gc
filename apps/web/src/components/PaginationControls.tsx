"use client";

import { useState } from "react";

export function usePageSlice<T>(items: T[], perPage = 50) {
  const [page, setPage] = useState(1);
  const total = items.length;
  const pages = Math.max(1, Math.ceil(total / perPage));
  const current = Math.min(Math.max(1, page), pages);
  const start = (current - 1) * perPage;
  const sliced = items.slice(start, start + perPage);
  return {
    page: current,
    perPage,
    total,
    sliced,
    setPage,
    prev: () => setPage((value) => Math.max(1, value - 1)),
    next: () => setPage((value) => Math.min(pages, value + 1)),
  };
}

export default function PaginationControls({ page, perPage, total, onPrev, onNext }: { page: number; perPage: number; total: number; onPrev: () => void; onNext: () => void }) {
  return <div className="pagination-row"><button className="table-action" disabled={page <= 1} onClick={onPrev}>‹ Sebelumnya</button><span className="muted">Hal {page} · {total} entri</span><button className="table-action" disabled={page * perPage >= total} onClick={onNext}>Berikutnya ›</button></div>;
}