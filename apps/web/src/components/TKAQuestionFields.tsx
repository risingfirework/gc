"use client";

import { useRef, useState } from "react";
import { AdminQuestion, QuestionOption, QuestionType } from "@/services/api";
import MediaField from "@/components/MediaField";

type Props = {
  item?: AdminQuestion;
  options: QuestionOption[];
  setOptions: React.Dispatch<React.SetStateAction<QuestionOption[]>>;
};

function parseCategoryAnswer(value = "") {
  return Object.fromEntries(value.split(";").map((part) => part.split("=", 2)).filter((part) => part.length === 2));
}

export default function TKAQuestionFields({ item, options, setOptions }: Props) {
  const [questionType, setQuestionType] = useState<QuestionType>(item?.question_type ?? "single_choice");
  const [singleAnswer, setSingleAnswer] = useState(item?.question_type === "single_choice" ? item.correct_answer : item?.options[0]?.key ?? "A");
  const [multipleAnswers, setMultipleAnswers] = useState<string[]>(item?.question_type === "multiple_choice" ? item.correct_answer.split(",") : []);
  const [categoryLabels, setCategoryLabels] = useState<string[]>(item?.category_labels?.length ? item.category_labels : ["Benar", "Salah"]);
  const [categoryAnswers, setCategoryAnswers] = useState<Record<string, string>>(item?.question_type === "category" ? parseCategoryAnswer(item.correct_answer) : {});
  const [essayAnswer, setEssayAnswer] = useState(item?.question_type === "essay" ? item.correct_answer : "");
  const legacyStimulus = item?.presentation_type === "group" && item.stimulus_text ? `Stimulus:\n${item.stimulus_text}\n\n` : "";
  const [questionImage, setQuestionImage] = useState(item?.question_image_url || item?.stimulus_image_url || "");
  const [questionAudio, setQuestionAudio] = useState(item?.question_audio_url || "");
  const [questionVideo, setQuestionVideo] = useState(item?.question_video_url || "");
  const [explanationVideo, setExplanationVideo] = useState(item?.explanation_video_url || "");
  const contentRef = useRef<HTMLTextAreaElement>(null);
  const contentSelection = useRef<{ start: number; end: number }>({ start: 0, end: 0 });
  const optionRefs = useRef<(HTMLInputElement | null)[]>([]);
  const optionSelections = useRef<{ start: number; end: number }[]>([]);
  function insertFraction() {
    const el = contentRef.current;
    if (!el) return;
    const start = el.selectionStart ?? el.value.length;
    const end = el.selectionEnd ?? el.value.length;
    const selected = el.value.slice(start, end).trim();
    let numerator = "";
    let denominator = "";
    if (selected) {
      const slash = selected.match(/^(-?\d+)\s*\/\s*(-?\d+)$/);
      if (slash) {
        numerator = slash[1];
        denominator = slash[2];
      } else {
        numerator = selected;
      }
    }
    const template = `$\\frac{${numerator}}{${denominator}}$`;
    el.value = el.value.slice(0, start) + template + el.value.slice(end);
    el.focus();
    // Tempatkan kursor di dalam pembilang jika kosong, atau penyebut jika pembilang terisi.
    const numStart = start + "$\\frac{".length;
    if (!numerator) {
      el.setSelectionRange(numStart, numStart);
    } else if (!denominator) {
      const denStart = numStart + numerator.length + 3; // }{
      el.setSelectionRange(denStart, denStart);
    } else {
      el.setSelectionRange(start + template.length, start + template.length);
    }
  }

  const keys = options.map((option) => option.key.trim().toUpperCase()).filter(Boolean);
  const correctAnswer = (() => {
    if (questionType === "single_choice") return singleAnswer;
    if (questionType === "multiple_choice") return [...multipleAnswers].sort().join(",");
    if (questionType === "essay") return essayAnswer;
    return keys.filter((key) => categoryAnswers[key]).sort().map((key) => `${key}=${categoryAnswers[key]}`).join(";");
  })();

  function updateOption(index: number, field: "key" | "content", value: string) {
    setOptions((current) => current.map((option, optionIndex) => optionIndex === index ? { ...option, [field]: value } : option));
  }
  function addOption() {
    const nextKey = String.fromCharCode(65 + options.length);
    setOptions((current) => [...current, { key: nextKey, content: "" }]);
  }
  function handleTypeChange(value: QuestionType) {
    setQuestionType(value);
    if (value === "essay") setOptions([]);
    else if (options.length === 0) setOptions([{ key: "A", content: "" }, { key: "B", content: "" }]);
  }

  function insertScript(kind: "^" | "_") {
    const el = contentRef.current;
    if (!el) return;
    const start = contentSelection.current.start ?? el.value.length;
    const end = contentSelection.current.end ?? el.value.length;
    const selected = el.value.slice(start, end).trim();
    const before = el.value.slice(0, start);
    const baseMatch = before.match(/([A-Za-z0-9]+)\s*$/);
    let template = "";
    let replaceStart = start;
    if (baseMatch && selected) {
      // base dari teks sebelum seleksi, seleksi jadi eksponen
      const base = baseMatch[1];
      replaceStart = start - baseMatch[0].length + (baseMatch[0].length - base.length);
      template = `$${base}${kind}{${selected}}$`;
    } else if (selected) {
      // tidak ada base sebelumnya: seleksi jadi base, eksponen kosong
      template = `$${selected}${kind}{}$`;
    } else {
      template = `$x${kind}{}$`;
    }
    el.value = el.value.slice(0, replaceStart) + template + el.value.slice(end);
    el.focus();
    const cursor = replaceStart + template.length - 2;
    el.setSelectionRange(cursor, cursor);
  }

  return <>
    <div className="admin-form-section"><h3>Bentuk soal</h3><label>Bentuk soal TKA</label><select name="question_type" value={questionType} onChange={(event) => handleTypeChange(event.target.value as QuestionType)}><option value="single_choice">Pilihan Ganda Sederhana</option><option value="multiple_choice">PGK MCMA (jawaban jamak)</option><option value="category">PGK Kategori</option><option value="essay">Esai (jawaban uraian)</option></select>
    <input type="hidden" name="presentation_type" value="single"/><input type="hidden" name="group_code" value=""/><input type="hidden" name="stimulus_text" value=""/></div>
    <div className="admin-form-section"><h3>{questionType === "category" ? "Pertanyaan & petunjuk klasifikasi" : "Pertanyaan & media"}</h3><label>{questionType === "category" ? "Pertanyaan/petunjuk klasifikasi" : "Pertanyaan"}</label><div style={{display:"flex",gap:8,marginBottom:6,alignItems:"center",flexWrap:"wrap"}}><button type="button" className="button secondary" onClick={insertFraction} title="Sisipkan pecahan">Pecahan a/b</button><button type="button" className="button secondary" onClick={() => insertScript("^")} title="Pangkat atas">x²</button><button type="button" className="button secondary" onClick={() => insertScript("_")} title="Pangkat bawah">x₂</button><small className="muted">{"LaTeX: $\\frac{a}{b}$, $x^2$, $x_1$"}</small></div><textarea ref={contentRef} name="content_text" defaultValue={`${legacyStimulus}${item?.content_text ?? ""}`} required onSelect={(e) => { contentSelection.current = { start: e.currentTarget.selectionStart ?? 0, end: e.currentTarget.selectionEnd ?? 0 }; }} onChange={(e) => { contentSelection.current = { start: e.target.selectionStart ?? e.target.value.length, end: e.target.selectionEnd ?? e.target.value.length }; }}/><MediaField label="Gambar pertanyaan (opsional)" kind="image" value={questionImage} onChange={setQuestionImage}/><MediaField label="Audio pertanyaan (opsional)" kind="audio" value={questionAudio} onChange={setQuestionAudio}/><MediaField label="Video pertanyaan (opsional)" kind="video" value={questionVideo} onChange={setQuestionVideo}/><label>Video pembahasan (tautan YouTube):</label><input type="text" name="explanation_video_url" value={explanationVideo} onChange={(event) => setExplanationVideo(event.target.value)} placeholder="Contoh: https://youtu.be/..."/></div>
    {questionType === "category" && <div className="admin-form-section"><h3>Kategori jawaban</h3><label>Kategori jawaban</label><div className="option-rows">{categoryLabels.map((label, index) => <div className="option-row" key={index}><input value={label} onChange={(event) => setCategoryLabels((current) => current.map((value, itemIndex) => itemIndex === index ? event.target.value : value))} placeholder="Contoh: Benar" required/><button type="button" className="danger-action" disabled={categoryLabels.length <= 2} onClick={() => setCategoryLabels((current) => current.filter((_, itemIndex) => itemIndex !== index))}>Hapus</button></div>)}</div><button type="button" className="table-action" onClick={() => setCategoryLabels((current) => [...current, ""])}>+ Tambah kategori</button></div>}
    {questionType !== "essay" && <div className="admin-form-section"><h3>{questionType === "category" ? "Daftar pernyataan" : "Pilihan jawaban"}</h3><label>{questionType === "category" ? "Daftar pernyataan" : "Pilihan jawaban"}</label>
    <div className="option-rows">{options.map((option, index) => <div className="option-editor" key={index}><div className="option-row"><input className="option-key" value={option.key} onChange={(event) => updateOption(index, "key", event.target.value.toUpperCase())} placeholder="A" required/><input ref={(el) => { optionRefs.current[index] = el; }} value={option.content} onChange={(event) => { updateOption(index, "content", event.target.value); optionSelections.current[index] = { start: event.target.selectionStart ?? event.target.value.length, end: event.target.selectionEnd ?? event.target.value.length }; }} onSelect={(event) => { optionSelections.current[index] = { start: event.currentTarget.selectionStart ?? 0, end: event.currentTarget.selectionEnd ?? 0 }; }} onMouseUp={(event) => { optionSelections.current[index] = { start: event.currentTarget.selectionStart ?? 0, end: event.currentTarget.selectionEnd ?? 0 }; }} onKeyUp={(event) => { optionSelections.current[index] = { start: event.currentTarget.selectionStart ?? 0, end: event.currentTarget.selectionEnd ?? 0 }; }} placeholder={questionType === "category" ? "Isi pernyataan" : "Isi opsi"} required/><button type="button" className="button small-btn" onClick={() => {
    const content = option.content;
    const sel = optionSelections.current[index] || { start: content.length, end: content.length };
    const start = Math.min(sel.start, content.length);
    const end = Math.min(sel.end, content.length);
    const selected = content.slice(start, end).trim();
    let numerator = "";
    let denominator = "";
    if (selected) {
      const slash = selected.match(/^(-?\d+)\s*\/\s*(-?\d+)$/);
      if (slash) { numerator = slash[1]; denominator = slash[2]; } else { numerator = selected; }
    }
    const template = `$\\frac{${numerator}}{${denominator}}$`;
    const newContent = content.slice(0, start) + template + content.slice(end);
    setOptions((current) => current.map((entry, itemIndex) => itemIndex === index ? { ...entry, content: newContent } : entry));
    setTimeout(() => {
      const input = optionRefs.current[index];
      if (!input) return;
      input.focus();
      const cursor = numerator ? (denominator ? start + template.length : start + numerator.length + 7) : start + 7;
      input.setSelectionRange(cursor, cursor);
    }, 0);
  }} title="Blok angka dulu, lalu klik Pecahan">Pecahan</button>
<button type="button" className="button small-btn" onClick={() => {
    const content = option.content;
    const sel = optionSelections.current[index] || { start: content.length, end: content.length };
    const start = Math.min(sel.start, content.length);
    const end = Math.min(sel.end, content.length);
    const selected = content.slice(start, end).trim() || "x";
    const template = `$${selected}^{}$`;
    const newContent = content.slice(0, start) + template + content.slice(end);
    setOptions((current) => current.map((entry, itemIndex) => itemIndex === index ? { ...entry, content: newContent } : entry));
    setTimeout(() => {
      const input = optionRefs.current[index];
      if (!input) return;
      input.focus();
      const cursor = start + selected.length + 3;
      input.setSelectionRange(cursor, cursor);
    }, 0);
  }} title="Pangkat atas">x²</button>
<button type="button" className="button small-btn" onClick={() => {
    const content = option.content;
    const sel = optionSelections.current[index] || { start: content.length, end: content.length };
    const start = Math.min(sel.start, content.length);
    const end = Math.min(sel.end, content.length);
    const selected = content.slice(start, end).trim() || "x";
    const template = `$${selected}_{}$`;
    const newContent = content.slice(0, start) + template + content.slice(end);
    setOptions((current) => current.map((entry, itemIndex) => itemIndex === index ? { ...entry, content: newContent } : entry));
    setTimeout(() => {
      const input = optionRefs.current[index];
      if (!input) return;
      input.focus();
      const cursor = start + selected.length + 3;
      input.setSelectionRange(cursor, cursor);
    }, 0);
  }} title="Pangkat bawah">x₂</button><button type="button" className="danger-action" disabled={options.length <= 2} onClick={() => setOptions((current) => current.filter((_, itemIndex) => itemIndex !== index))}>Hapus</button></div><MediaField compact kind="image" label={`Gambar ${questionType === "category" ? "pernyataan" : "pilihan"} ${option.key || index + 1} (opsional)`} value={option.image_url ?? ""} onChange={(value) => setOptions((current) => current.map((entry, itemIndex) => itemIndex === index ? { ...entry, image_url: value } : entry))}/></div>)}</div>
    <button type="button" className="table-action" onClick={addOption}>+ Tambah {questionType === "category" ? "pernyataan" : "opsi"}</button></div>}
    <div className="admin-form-section"><h3>Kunci jawaban</h3><label>Kunci jawaban</label>
    {questionType === "single_choice" && <select value={singleAnswer} onChange={(event) => setSingleAnswer(event.target.value)} required>{keys.map((key) => <option key={key} value={key}>{key}</option>)}</select>}
    {questionType === "multiple_choice" && <div className="answer-key-grid">{keys.map((key) => <label className="check-card" key={key}><input type="checkbox" checked={multipleAnswers.includes(key)} onChange={() => setMultipleAnswers((current) => current.includes(key) ? current.filter((value) => value !== key) : [...current, key])}/><span>{key}</span></label>)}<small>Pilih minimal dua jawaban benar.</small></div>}
    {questionType === "category" && <div className="category-key-grid">{options.map((option) => { const key = option.key.trim().toUpperCase(); return <div key={key || option.content}><span><b>{key}</b> {option.content || "Pernyataan"}</span><select value={categoryAnswers[key] ?? ""} onChange={(event) => setCategoryAnswers((current) => ({ ...current, [key]: event.target.value }))} required><option value="">Pilih kategori</option>{categoryLabels.filter(Boolean).map((label) => <option value={label} key={label}>{label}</option>)}</select></div>; })}</div>}
    {questionType === "essay" && <><label>Jawaban referensi (kunci uraian)</label><textarea value={essayAnswer} onChange={(event) => setEssayAnswer(event.target.value)} placeholder="Tuliskan kunci jawaban atau referensi pedoman penilaian esai" required/><p className="form-helper">Jawaban esai siswa disimpan dan akan dinilai manual oleh guru. Skor otomatis dari esai dihitung 0 sampai dinilai.</p></>}</div>
    <input type="hidden" name="correct_answer" value={correctAnswer}/><input type="hidden" name="category_labels_json" value={JSON.stringify(questionType === "category" ? categoryLabels : [])}/><input type="hidden" name="question_image_url" value={questionImage}/><input type="hidden" name="stimulus_image_url" value=""/><input type="hidden" name="question_audio_url" value={questionAudio}/><input type="hidden" name="question_video_url" value={questionVideo}/><input type="hidden" name="explanation_video_url" value={explanationVideo}/>
  </>;
}
