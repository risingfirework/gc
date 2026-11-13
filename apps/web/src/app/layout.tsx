import "katex/dist/katex.min.css";
import "./globals.css";
import TitleSync from "@/components/TitleSync";
import { connection } from "next/server";
import type { Metadata, Viewport } from "next";

const siteURL = process.env.NEXT_PUBLIC_SITE_URL ?? "https://siap-siap.my.id";

export const metadata: Metadata = {
  metadataBase: new URL(siteURL),
  title: {
    default: "SIAP TKA — Platform Tryout SD, SMP, SMA/SMK",
    template: "%s | SIAP TKA",
  },
  description: "Platform tryout online untuk SD, SMP, dan SMA/SMK dengan soal terstandar, CBT, ranking global, dan analisis hasil per bab.",
  keywords: ["tryout online", "tryout TKA", "latihan soal SD", "latihan soal SMP", "latihan soal SMA", "CBT online", "ranking tryout", "bank soal"],
  applicationName: "SIAP TKA",
  authors: [{ name: "SIAP TKA" }],
  creator: "SIAP TKA",
  publisher: "SIAP TKA",
  category: "education",
  openGraph: {
    type: "website",
    locale: "id_ID",
    siteName: "SIAP TKA",
    title: "SIAP TKA — Platform Tryout SD, SMP, SMA/SMK",
    description: "Latihan tryout yang terukur: soal terstandar, CBT, ranking, dan analisis per bab dalam satu platform belajar.",
  },
  twitter: {
    card: "summary_large_image",
    title: "SIAP TKA — Platform Tryout SD, SMP, SMA/SMK",
    description: "Latihan tryout yang terukur: soal terstandar, CBT, ranking, dan analisis per bab dalam satu platform belajar.",
  },
  robots: {
    index: true,
    follow: true,
    googleBot: { index: true, follow: true, "max-image-preview": "large", "max-snippet": -1, "max-video-preview": -1 },
  },
};

export const viewport: Viewport = {
  width: "device-width",
  initialScale: 1,
  themeColor: "#07101e",
};

export default async function RootLayout({ children }: Readonly<{ children: React.ReactNode }>) { await connection(); return <html lang="id"><body><TitleSync/>{children}</body></html>; }
