import "katex/dist/katex.min.css";
import "./globals.css";

export const metadata = { title: "TKA Juara", description: "Platform tryout TKA untuk SD, SMP, dan SMA/SMK" };
export default function RootLayout({ children }: Readonly<{ children: React.ReactNode }>) { return <html lang="id"><body>{children}</body></html>; }
