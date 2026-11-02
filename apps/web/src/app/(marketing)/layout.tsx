import type { Metadata } from "next";

export const metadata: Metadata = {
  alternates: { canonical: "/" },
};

export default function MarketingLayout({ children }: Readonly<{ children: React.ReactNode }>) {
  return children;
}
