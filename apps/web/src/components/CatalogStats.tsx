export default function CatalogStats({ sales, views }: { sales: number; views: number }) {
  return <div className="catalog-stats" aria-label={`${sales} terjual dan ${views} dilihat`}>
    <span title="Jumlah terjual"><svg viewBox="0 0 24 24" aria-hidden="true"><path d="M3 4h2l2.1 10.1a2 2 0 0 0 2 1.6h7.8a2 2 0 0 0 2-1.6L20.3 8H6.1M10 20a1 1 0 1 1-2 0 1 1 0 0 1 2 0Zm9 0a1 1 0 1 1-2 0 1 1 0 0 1 2 0Z"/></svg><b>{sales.toLocaleString("id-ID")}</b><small>terjual</small></span>
    <span title="Jumlah pengunjung"><svg viewBox="0 0 24 24" aria-hidden="true"><path d="M2.5 12s3.5-6 9.5-6 9.5 6 9.5 6-3.5 6-9.5 6-9.5-6-9.5-6Z"/><circle cx="12" cy="12" r="2.5"/></svg><b>{views.toLocaleString("id-ID")}</b><small>dilihat</small></span>
  </div>;
}
