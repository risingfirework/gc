"use client";

import Image from "next/image";

type Props = {
  image?: string;
  audio?: string;
  video?: string;
  alt: string;
};

// QuestionMedia menampilkan media soal yang tersimpan di server maupun tautan
// eksternal. Gambar memakai optimasi Next.js, sedangkan audio/video diputar
// langsung oleh browser.
export default function QuestionMedia({ image, audio, video, alt }: Props) {
  return (
    <>
      {image ? <Image className="question-media" src={image} alt={alt} width={800} height={500} unoptimized/> : null}
      {audio ? <audio className="question-media" controls preload="metadata" src={audio} aria-label={`Audio ${alt}`}/> : null}
      {video ? <video className="question-media" controls preload="metadata" src={video} aria-label={`Video ${alt}`}/> : null}
    </>
  );
}
