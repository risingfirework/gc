"use client";

import { FormEvent, useState } from "react";
import { api, SiteSettings, YouTubeVideo } from "@/services/api";
import { youtubeVideoID } from "./YouTubeEmbed";

export default function VideoManagerPanel({ initial, onSaved }: { initial: SiteSettings; onSaved: (settings: SiteSettings) => void }) {
  const [value, setValue] = useState<YouTubeVideo[]>(() => structuredClone(initial.youtube_videos));
  const [saving, setSaving] = useState(false); const [message, setMessage] = useState(""); const [error, setError] = useState("");
  function addVideo() {
    setValue((current) => [...current, { id: `video-${Date.now()}`, title: "Video baru", url: "" }]);
  }
  function changeVideo(id: string, patch: Partial<YouTubeVideo>) {
    setValue((current) => current.map((item) => item.id === id ? { ...item, ...patch } : item));
  }
  function removeVideo(id: string) {
    if (!window.confirm("Hapus video ini? Perubahan tersimpan setelah Anda menekan Simpan.")) return;
    setValue((current) => current.filter((item) => item.id !== id));
  }
  async function save(event: FormEvent) {
    event.preventDefault(); setSaving(true); setMessage(""); setError("");
    try {
      const saved = await api.updateSiteSettings({ ...initial, youtube_videos: value });
      setValue(structuredClone(saved.youtube_videos));
      onSaved(saved);
      setMessage("Menu video berhasil disimpan dan langsung diterapkan.");
    } catch (reason) { setError(reason instanceof Error ? reason.message : "Menu video gagal disimpan."); }
    finally { setSaving(false); }
  }
  return <form className="settings-view" onSubmit={save}>
    <div className="settings-heading"><div><p className="eyebrow">Menu video</p><h2>Video YouTube</h2><p className="muted">Dikelola penuh di sini: tambah, ubah, susun, atau hapus video yang tampil di panel siswa.</p></div><button className="button" disabled={saving}>{saving ? "Menyimpan..." : "Simpan semua perubahan"}</button></div>
    {message && <p className="finance-success">{message}</p>}{error && <p className="error">{error}</p>}
    <section className="settings-slides settings-videos"><div className="settings-card-title"><div><h3>Daftar video</h3><p>Video dibuka dalam mode layar penuh tengah saat diputar.</p></div><button type="button" className="button secondary" onClick={addVideo}>+ Tambah video</button></div>
      <p className="muted">Pakai tautan berbagi YouTube, contoh: https://www.youtube.com/watch?v=VIDEO_ID</p>
      {value.map((video) => {
        const videoId = youtubeVideoID(video.url);
        return <article className="card slide-editor" key={video.id}>
          <div className="slide-editor-head"><strong>{video.title || "Video"}</strong><button type="button" className="danger-action" onClick={() => removeVideo(video.id)}>Hapus</button></div>
          <div className="video-manager-row">
            <aside className="video-manager-preview">{videoId ? <span style={{ backgroundImage: `url(https://i.ytimg.com/vi/${videoId}/hqdefault.jpg)` }}></span> : <span>Tanpa pratinjau</span>}</aside>
            <div className="settings-grid">
              <label>Judul<input value={video.title} maxLength={180} placeholder="Judul video" onChange={(event) => changeVideo(video.id, { title: event.target.value })}/></label>
              <label>URL video<input type="url" value={video.url} maxLength={500} placeholder="https://www.youtube.com/watch?v=..." onChange={(event) => changeVideo(video.id, { url: event.target.value })}/></label>
            </div>
          </div>
        </article>;
      })}
      {value.length === 0 && <div className="card empty-state">Belum ada video. Klik &quot;+ Tambah video&quot; untuk menambahkan embed YouTube.</div>}
    </section>
  </form>;
}