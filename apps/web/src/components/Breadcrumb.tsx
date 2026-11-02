import Link from "next/link";

export type BreadcrumbItem = { label: string; href?: string };

export default function Breadcrumb({ items }: { items: BreadcrumbItem[] }) {
  return <nav className="breadcrumb" aria-label="Breadcrumb">
    <ol>
      {items.map((item, index) => {
        const current = index === items.length - 1;
        return <li key={`${item.label}-${index}`}>
          {current || !item.href
            ? <span aria-current={current ? "page" : undefined}>{item.label}</span>
            : <Link href={item.href}>{item.label}</Link>}
        </li>;
      })}
    </ol>
  </nav>;
}
