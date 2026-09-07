"use client";

import { forwardRef, useImperativeHandle, useState } from "react";
import { ExamEntry, MasterItem, Status } from "@/services/api";

export const PackageExamFields = forwardRef<{ getValues: () => ExamEntry }, { defaultExam?: Partial<ExamEntry>; mapels: MasterItem[]; academicYears: MasterItem[]; showHint?: boolean; defaults?:{duration_minutes:number;total_questions:number;passing_score:number} }>(
  function PackageExamFields({ defaultExam, mapels, academicYears, showHint = true, defaults }, ref) {
    const [exam, setExam] = useState<ExamEntry>({
      title: defaultExam?.title ?? "",
      mapel_id: defaultExam?.mapel_id && mapels.some((item) => item.id === defaultExam.mapel_id) ? defaultExam.mapel_id : mapels[0]?.id ?? "",
      tahun_ajaran_id: defaultExam?.tahun_ajaran_id && academicYears.some((item) => item.id === defaultExam.tahun_ajaran_id) ? defaultExam.tahun_ajaran_id : academicYears[0]?.id ?? "",
      duration_minutes: defaultExam?.duration_minutes ?? defaults?.duration_minutes ?? 60,
      total_questions: defaultExam?.total_questions ?? defaults?.total_questions ?? 1,
      passing_score: defaultExam?.passing_score ?? defaults?.passing_score ?? 50,
      status: defaultExam?.status ?? "active",
    });
    useImperativeHandle(ref, () => ({ getValues: () => exam }), [exam]);
    function update(field: keyof ExamEntry, value: string | number) {
      setExam((current) => ({ ...current, [field]: value }));
    }
    return <div className="bundle-section">
      <h3>Detail Ujian</h3>
      <label>Mapel</label>
      <select value={exam.mapel_id} onChange={(event) => update("mapel_id", event.target.value)} required>
        {mapels.map((item) => <option key={item.id} value={item.id}>{item.nama}</option>)}
      </select>
      <label>Tahun Ajaran</label>
      <select value={exam.tahun_ajaran_id} onChange={(event) => update("tahun_ajaran_id", event.target.value)} required>
        {academicYears.map((item) => <option key={item.id} value={item.id}>{item.nama}</option>)}
      </select>
      <div className="form-grid">
        <div><label>Durasi (menit)</label><input type="number" min="1" value={exam.duration_minutes} onChange={(event) => update("duration_minutes", Number(event.target.value))} required /></div>
        <div><label>Jumlah soal</label><input type="number" min="1" value={exam.total_questions} onChange={(event) => update("total_questions", Number(event.target.value))} required /></div>
        <div><label>Nilai lulus</label><input type="number" min="0" max="100" step="0.01" value={exam.passing_score} onChange={(event) => update("passing_score", Number(event.target.value))} required /></div>
        <div><label>Status ujian</label><select value={exam.status} onChange={(event) => update("status", event.target.value as Status)}><option value="active">Aktif</option><option value="inactive">Nonaktif</option></select></div>
      </div>
      {showHint && <p className="muted">Isi soal nanti di dalam paket, setelah paket dibuat.</p>}
    </div>;
  },
);
