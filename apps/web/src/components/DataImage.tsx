import type { CSSProperties } from "react";

import Image from "next/image";

// Gambar data-URL/Base64 (disimpan di kolom TEXT database) melewati optimizer
// Next.js; gunakan unoptimized agar tetap tampil tanpa biaya optimizer.
export default function DataImage({ src, alt, width, height, className, style }: { src: string; alt: string; width?: number; height?: number; className?: string; style?: CSSProperties }) {
  return <Image src={src} alt={alt} width={width} height={height} unoptimized draggable={false} className={className} style={style} />;
}