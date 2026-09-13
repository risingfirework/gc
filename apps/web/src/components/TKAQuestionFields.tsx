"use client";

import { useState } from "react";
import Image from "next/image";
import { AdminQuestion, QuestionOption, QuestionType } from "@/services/api";

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

  return <>
    <div className="admin-form-section"><h3>Bentuk soal</h3><label>Bentuk soal TKA</label><select name="question_type" value={questionType} onChange={(event) => handleTypeChange(event.target.value as QuestionType)}><option value="single_choice">Pilihan Ganda Sederhana</option><option value="multiple_choice">PGK MCMA (jawaban jamak)</option><option value="category">PGK Kategori</option><option value="essay">Esai (jawaban uraian)</option></select>
    <input type="hidden" name="presentation_type" value="single"/><input type="hidden" name="group_code" value=""/><input type="hidden" name="stimulus_text" value=""/></div>
    <div className="admin-form-section"><h3>{questionType === "category" ? "Pertanyaan & petunjuk klasifikasi" : "Pertanyaan & media"}</h3><label>{questionType === "category" ? "Pertanyaan/petunjuk klasifikasi" : "Pertanyaan"}</label><textarea name="content_text" defaultValue={`${legacyStimulus}${item?.content_text ?? ""}`} required/><ImageField label="Gambar pertanyaan (opsional)" value={questionImage} onChange={setQuestionImage}/></div>
    {questionType === "category" && <div className="admin-form-section"><h3>Kategori jawaban</h3><label>Kategori jawaban</label><div className="option-rows">{categoryLabels.map((label, index) => <div className="option-row" key={index}><input value={label} onChange={(event) => setCategoryLabels((current) => current.map((value, itemIndex) => itemIndex === index ? event.target.value : value))} placeholder="Contoh: Benar" required/><button type="button" className="danger-action" disabled={categoryLabels.length <= 2} onClick={() => setCategoryLabels((current) => current.filter((_, itemIndex) => itemIndex !== index))}>Hapus</button></div>)}</div><button type="button" className="table-action" onClick={() => setCategoryLabels((current) => [...current, ""])}>+ Tambah kategori</button></div>}
    {questionType !== "essay" && <div className="admin-form-section"><h3>{questionType === "category" ? "Daftar pernyataan" : "Pilihan jawaban"}</h3><label>{questionType === "category" ? "Daftar pernyataan" : "Pilihan jawaban"}</label>
    <div className="option-rows">{options.map((option, index) => <div className="option-editor" key={index}><div className="option-row"><input className="option-key" value={option.key} onChange={(event) => updateOption(index, "key", event.target.value.toUpperCase())} placeholder="A" required/><input value={option.content} onChange={(event) => updateOption(index, "content", event.target.value)} placeholder={questionType === "category" ? "Isi pernyataan" : "Isi opsi"} required/><button type="button" className="danger-action" disabled={options.length <= 2} onClick={() => setOptions((current) => current.filter((_, itemIndex) => itemIndex !== index))}>Hapus</button></div><ImageField compact label={`Gambar ${questionType === "category" ? "pernyataan" : "pilihan"} ${option.key || index + 1} (opsional)`} value={option.image_url ?? ""} onChange={(value) => setOptions((current) => current.map((entry, itemIndex) => itemIndex === index ? { ...entry, image_url: value } : entry))}/></div>)}</div>
    <button type="button" className="table-action" onClick={addOption}>+ Tambah {questionType === "category" ? "pernyataan" : "opsi"}</button></div>}
    <div className="admin-form-section"><h3>Kunci jawaban</h3><label>Kunci jawaban</label>
    {questionType === "single_choice" && <select value={singleAnswer} onChange={(event) => setSingleAnswer(event.target.value)} required>{keys.map((key) => <option key={key} value={key}>{key}</option>)}</select>}
    {questionType === "multiple_choice" && <div className="answer-key-grid">{keys.map((key) => <label className="check-card" key={key}><input type="checkbox" checked={multipleAnswers.includes(key)} onChange={() => setMultipleAnswers((current) => current.includes(key) ? current.filter((value) => value !== key) : [...current, key])}/><span>{key}</span></label>)}<small>Pilih minimal dua jawaban benar.</small></div>}
    {questionType === "category" && <div className="category-key-grid">{options.map((option) => { const key = option.key.trim().toUpperCase(); return <div key={key || option.content}><span><b>{key}</b> {option.content || "Pernyataan"}</span><select value={categoryAnswers[key] ?? ""} onChange={(event) => setCategoryAnswers((current) => ({ ...current, [key]: event.target.value }))} required><option value="">Pilih kategori</option>{categoryLabels.filter(Boolean).map((label) => <option value={label} key={label}>{label}</option>)}</select></div>; })}</div>}
    {questionType === "essay" && <><label>Jawaban referensi (kunci uraian)</label><textarea value={essayAnswer} onChange={(event) => setEssayAnswer(event.target.value)} placeholder="Tuliskan kunci jawaban atau referensi pedoman penilaian esai" required/><p className="form-helper">Jawaban esai siswa disimpan dan akan dinilai manual oleh guru. Skor otomatis dari esai dihitung 0 sampai dinilai.</p></>}</div>
    <input type="hidden" name="correct_answer" value={correctAnswer}/><input type="hidden" name="category_labels_json" value={JSON.stringify(questionType === "category" ? categoryLabels : [])}/><input type="hidden" name="question_image_url" value={questionImage}/><input type="hidden" name="stimulus_image_url" value=""/>
  </>;
}

function ImageField({ label, value, onChange, compact = false }: { label: string; value: string; onChange: (value: string) => void; compact?: boolean }) {
  const [error, setError] = useState("");
  function choose(file?: File) {
    setError("");
    if (!file) return;
    if (!(["image/jpeg", "image/png", "image/webp", "image/gif"].includes(file.type))) { setError("Gunakan JPG, PNG, WebP, atau GIF."); return; }
    if (file.size > 1_500_000) { setError("Ukuran gambar maksimal 1,5 MB."); return; }
    const reader = new FileReader();
    reader.onload = () => onChange(String(reader.result));
    reader.onerror = () => setError("Gambar gagal dibaca.");
    reader.readAsDataURL(file);
  }
  return <div className={`image-field ${compact ? "compact" : ""}`}><label>{label}</label><div className="image-field-controls"><input type="file" accept="image/jpeg,image/png,image/webp,image/gif" onChange={(event) => choose(event.target.files?.[0])}/>{value && <button type="button" className="danger-action" onClick={() => onChange("")}>Hapus gambar</button>}</div>{error && <small className="error">{error}</small>}{value && <Image className="question-image-preview" src={value} alt="Pratinjau media soal" width={640} height={360} unoptimized/>}</div>;
}
