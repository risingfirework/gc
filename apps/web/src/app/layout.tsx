import "katex/dist/katex.min.css";
import "./globals.css";
import TitleSync from "@/components/TitleSync";

export const metadata = { title: "Platform Tryout", description: "Platform tryout untuk SD, SMP, dan SMA/SMK" };
export default function RootLayout({ children }: Readonly<{ children: React.ReactNode }>) { return <html lang="id"><body><TitleSync/>{children}</body></html>; }
