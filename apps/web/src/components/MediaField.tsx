"use client";

import { useRef, useState } from "react";
import Image from "next/image";
import { api, MediaKind } from "@/services/api";

const mediaConfig: Record<MediaKind, { accept: string; maxMB: number; hint: string }> = {
  image: { accept: "image/jpeg,image/png,image/webp,image/gif", maxMB: 8, hint: "JPG, PNG, WebP, atau GIF · maksimal 8 MB" },
  audio: { accept: "audio/mpeg,audio/mp4,audio/aac,audio/ogg,audio/wav,audio/webm,audio/flac,audio/x-m4a", maxMB: 25, hint: "MP3, M4A, AAC, OGG, WAV, atau WebM · maksimal 25 MB" },
  video: { accept: "video/mp4,video/webm,video/ogg,video/quicktime,video/x-matroska", maxMB: 100, hint: "MP4, WebM, OGG, MOV, atau MKV · maksimal 100 MB" },
};

function mediaID(value: string) {
  const prefix = "/api/v1/media/";
  return value.startsWith(prefix) ? value.slice(prefix.length) : "";
}

export default function MediaField({
  label,
  kind,
  value,
  onChange,
  allowUrl = kind !== "image",
  compact = false,
}: {
  label: string;
  kind: MediaKind;
  value: string;
  onChange: (value: string) => void;
  allowUrl?: boolean;
  compact?: boolean;
}) {
  const config = mediaConfig[kind];
  const inputRef = useRef<HTMLInputElement>(null);
  const [uploading, setUploading] = useState(false);
  const [progress, setProgress] = useState(0);
  const [error, setError] = useState("");
  const [owned, setOwned] = useState<string[]>([]);

  async function choose(file?: File) {
    if (!file) return;
    setError("");
    if (file.type && !file.type.startsWith(`${kind}/`)) {
      setError(kind === "image" ? "Pilih berkas gambar." : kind === "audio" ? "Pilih berkas audio." : "Pilih berkas video.");
      return;
    }
    if (file.size > config.maxMB * 1024 * 1024) {
      setError(`Ukuran media maksimal ${config.maxMB} MB.`);
      return;
    }
    setUploading(true);
    setProgress(0);
    try {
      const asset = await api.uploadMedia(file, kind, setProgress);
      const previous = mediaID(value);
      if (previous && owned.includes(previous)) {
        void api.deleteMedia(previous).catch(() => undefined);
        setOwned((current) => current.filter((item) => item !== previous));
      }
      setOwned((current) => [...current, asset.id]);
      onChange(asset.url);
    } catch (reason) {
      setError(reason instanceof Error ? reason.message : "Media gagal diunggah.");
    } finally {
      setUploading(false);
      setProgress(0);
      if (inputRef.current) inputRef.current.value = "";
    }
  }

  function remove() {
    const id = mediaID(value);
    if (id && owned.includes(id)) {
      void api.deleteMedia(id).catch(() => undefined);
      setOwned((current) => current.filter((item) => item !== id));
    }
    setError("");
    onChange("");
  }

  return (
    <div className={`image-field${compact ? " compact" : ""}`}>
      <label>{label}</label>
      <div className="image-field-controls">
        <input
          ref={inputRef}
          type="file"
          accept={config.accept}
          disabled={uploading}
          onChange={(event) => void choose(event.target.files?.[0])}
        />
        {value && <button type="button" className="danger-action" disabled={uploading} onClick={remove}>Hapus</button>}
      </div>
      <small className="muted">{uploading ? `Mengunggah media... ${progress}%` : config.hint}</small>
      {allowUrl && (
        <input
          type="url"
          value={value}
          onChange={(event) => onChange(event.target.value)}
          placeholder="Atau tempel tautan https://..."
          aria-label={`${label} tautan`}
        />
      )}
      {error && <small className="error">{error}</small>}
      {value && kind === "image" && <Image className="question-image-preview" src={value} alt={`Pratinjau ${label}`} width={640} height={360} unoptimized />}
      {value && kind === "audio" && <audio className="question-media" controls preload="metadata" src={value} />}
      {value && kind === "video" && <video className="question-media" controls preload="metadata" src={value} />}
    </div>
  );
}
