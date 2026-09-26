import "katex/dist/katex.min.css";
import "./globals.css";
import TitleSync from "@/components/TitleSync";
import { connection } from "next/server";

export const metadata = { title: "Platform Tryout", description: "Platform tryout untuk SD, SMP, dan SMA/SMK" };
export default async function RootLayout({ children }: Readonly<{ children: React.ReactNode }>) { await connection(); return <html lang="id"><body><TitleSync/>{children}</body></html>; }
