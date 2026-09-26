"use client";

import { useState } from "react";
import type { AdminExam } from "@/services/api";

export default function QuestionShuffleSettings({
  exam,
  saving,
  onSave,
}: {
  exam: AdminExam;
  saving: boolean;
  onSave: (shuffleQuestions: boolean, shuffleOptions: boolean) => Promise<void> | void;
}) {
  const [shuffleQuestions, setShuffleQuestions] = useState(exam.shuffle_questions);
  const [shuffleOptions, setShuffleOptions] = useState(exam.shuffle_options);
  const changed = shuffleQuestions !== exam.shuffle_questions || shuffleOptions !== exam.shuffle_options;

  return <section className="card question-shuffle-settings">
    <div className="question-shuffle-heading">
      <div>
        <p className="eyebrow">Pengaturan bank soal</p>
        <h3>Pengacakan peserta</h3>
        <p className="muted">Perubahan berlaku untuk attempt baru; sesi yang sedang berjalan tetap memakai urutannya saat halaman dimuat ulang.</p>
      </div>
      <button className="button small-btn" type="button" disabled={saving || !changed} onClick={() => void onSave(shuffleQuestions, shuffleOptions)}>
        {saving ? "Menyimpan..." : changed ? "Simpan pengaturan" : "Tersimpan"}
      </button>
    </div>
    <div className="question-shuffle-options">
      <label className="question-shuffle-option">
        <input type="checkbox" checked={shuffleQuestions} disabled={saving} onChange={(event) => setShuffleQuestions(event.target.checked)} />
        <span><strong>Acak urutan soal</strong><small>Setiap attempt memperoleh susunan soal acak yang dipersistenkan.</small></span>
      </label>
      <label className="question-shuffle-option">
        <input type="checkbox" checked={shuffleOptions} disabled={saving} onChange={(event) => setShuffleOptions(event.target.checked)} />
        <span><strong>Acak urutan opsi jawaban</strong><small>Opsi diacak dan jawaban tetap diterjemahkan ke kunci asli saat dinilai.</small></span>
      </label>
    </div>
  </section>;
}
