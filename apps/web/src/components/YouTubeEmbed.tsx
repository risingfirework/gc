"use client";

import { useEffect, useState } from "react";

export function youtubeVideoID(input: string): string {
  if (!input) return "";
  const trimmed = input.trim();
  const match = trimmed.match(/(?:youtube(?:-nocookie)?\.com\/(?:watch\?(?:.*&)?v=|embed\/|shorts\/|live\/)|youtu\.be\/)([\w-]{6,15})/);
  if (match) return match[1];
  return /^[\w-]{6,15}$/.test(trimmed) ? trimmed : "";
}

export default function YouTubeEmbed({ videoUrl, videoID, className }: { videoUrl?: string; videoID?: string; className?: string }) {
  const id = videoID ?? youtubeVideoID(videoUrl ?? "");
  const [open, setOpen] = useState(false);
  useEffect(() => {
    if (!open) return;
    const onKey = (event: KeyboardEvent) => { if (event.key === "Escape") setOpen(false); };
    window.addEventListener("keydown", onKey);
    return () => window.removeEventListener("keydown", onKey);
  }, [open]);
  if (!id) return null;
  return (
    <>
      <div
        className={`youtube-embed${className ? ` ${className}` : ""}`}
        role="button"
        tabIndex={0}
        aria-label="Putar video"
        onClick={() => setOpen(true)}
        onKeyDown={(event) => { if (event.key === "Enter" || event.key === " ") { event.preventDefault(); setOpen(true); } }}
        style={{ backgroundImage: `url(https://i.ytimg.com/vi/${id}/hqdefault.jpg)` }}
      >
        <span className="video-play" aria-hidden="true" />
      </div>
      {open && (
        <div className="video-lightbox" role="presentation" onClick={() => setOpen(false)}>
          <div className="video-lightbox-frame" role="dialog" aria-modal="true" aria-label="Video promosi" onClick={(event) => event.stopPropagation()}>
            <button className="video-lightbox-close" aria-label="Tutup" onClick={() => setOpen(false)}>×</button>
            <iframe
              src={`https://www.youtube-nocookie.com/embed/${id}?autoplay=1&rel=0`}
              title="Video promosi"
              allow="accelerometer; autoplay; clipboard-write; encrypted-media; gyroscope; picture-in-picture; web-share"
              allowFullScreen
            />
          </div>
        </div>
      )}
    </>
  );
}

export type YouTubeVideoItem = { id: string; title: string; url: string };

export function VideoSection({ videos = [], eyebrow = "Video", title, lead, id }: { videos?: YouTubeVideoItem[]; eyebrow?: string; title: string; lead?: string; id?: string }) {
  const items = videos.filter((video) => youtubeVideoID(video.url));
  if (items.length === 0) return null;
  return (
    <section className="section video-section" id={id}>
      <div className="section-heading"><div><p className="eyebrow">{eyebrow}</p><h2>{title}</h2>{lead && <p className="muted" style={{ marginTop: 10 }}>{lead}</p>}</div></div>
      <div className="video-grid">
        {items.map((video) => (
          <figure className="video-item" key={video.id}>
            <YouTubeEmbed videoUrl={video.url} />
            {video.title && <figcaption>{video.title}</figcaption>}
          </figure>
        ))}
      </div>
    </section>
  );
}