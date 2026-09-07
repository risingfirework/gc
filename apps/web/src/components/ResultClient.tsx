"use client";

import Link from "next/link";
import BrandLogo from "./BrandLogo";
import Image from "next/image";
import { useEffect, useState } from "react";
import RichText from "@/components/RichText";
import { AnswerReview, api, ExamResult } from "@/services/api";

export default function ResultClient({ attemptID }: { attemptID: string }) {
  const [result, setResult] = useState<ExamResult>();
  const [error, setError] = useState("");
  const [reviewIndex, setReviewIndex] = useState(0);
  useEffect(() => { api.examResult(attemptID).then(setResult).catch((reason) => setError(reason instanceof Error ? reason.message : "Hasil tidak dapat dimuat.")); }, [attemptID]);
  if (error) return <main className="auth"><section className="card"><p className="error">{error}</p><Link className="button secondary" href="/dashboard">Kembali</Link></section></main>;
  if (!result) return <main className="auth"><div className="exam-loader"/><p>Menyiapkan hasil dan pembahasan...</p></main>;
  return <main><div className="container result-page"><nav className="nav"><Link className="brand" href="/dashboard"><BrandLogo/>TKA Juara</Link><Link href="/dashboard">Dashboard</Link></nav>
    <header className="card result-hero"><div><p className="eyebrow">Hasil ujian · {result.scoring_method === "irt_2pl" ? "IRT 2PL" : "Skor berbobot"}</p><h2>{result.title}</h2><div className="result-summary"><span className="correct"><b>{result.correct_answers}</b> Benar</span><span className="wrong"><b>{result.wrong_answers}</b> Salah</span><span className="empty"><b>{result.unanswered}</b> Kosong</span><span><b>{result.passing_score}</b> Nilai minimum</span></div></div><div className={`score-ring ${result.passed ? "passed" : "failed"}`}><strong>{result.total_score.toFixed(2)}</strong><span>{result.passed ? "LULUS" : "TIDAK LULUS"}</span></div></header>
    <section className="review-section"><div className="review-heading"><div><p className="eyebrow">Analisis dan pembahasan</p><h2>Review setiap soal</h2><p className="muted">Pilih nomor untuk melihat soal, jawaban, dan pembahasannya.</p></div><span>{result.review.length} soal</span></div>{result.review.length ? <div className="review-browser"><div className="review-browser-main"><QuestionReviewCard item={result.review[reviewIndex]} index={reviewIndex} total={result.review.length}/><div className="review-navigation-actions"><button className="button secondary" disabled={reviewIndex===0} onClick={()=>setReviewIndex((current)=>Math.max(0,current-1))}>← Sebelumnya</button><span>Soal <b>{reviewIndex+1}</b> dari {result.review.length}</span><button className="button" disabled={reviewIndex===result.review.length-1} onClick={()=>setReviewIndex((current)=>Math.min(result.review.length-1,current+1))}>Selanjutnya →</button></div></div><aside className="card review-number-panel"><div><p className="review-analysis-title">Navigasi soal</p><span className="muted">Pilih nomor soal</span></div><div className="review-number-grid">{result.review.map((item,index)=>{const status=!item.selected_option?"empty":item.is_correct?"correct":"wrong";return <button type="button" className={`${status} ${reviewIndex===index?"current":""}`} aria-label={`Soal ${index+1}, ${status==="correct"?"benar":status==="wrong"?"salah":"tidak dijawab"}`} aria-current={reviewIndex===index?"true":undefined} onClick={()=>setReviewIndex(index)} key={item.question_id}>{index+1}</button>})}</div><div className="review-number-legend"><span><i className="correct"/>Benar</span><span><i className="wrong"/>Salah</span><span><i className="empty"/>Kosong</span></div></aside></div> : <div className="card empty-state">Belum ada soal yang dapat direview.</div>}</section>
  </div></main>;
}

function QuestionReviewCard({ item, index, total }: { item: AnswerReview; index: number; total: number }) {
  const unanswered = !item.selected_option;
  const selectedKeys = new Set((item.selected_option ?? "").split(",").filter(Boolean));
  const correctKeys = new Set(item.correct_answer.split(",").filter(Boolean));
  const selectedCategories = parseCategoryAnswer(item.selected_option);
  const correctCategories = parseCategoryAnswer(item.correct_answer);
  const status = unanswered ? "empty" : item.is_correct ? "correct" : "wrong";
  return <article className={`card review-question-card ${status}`}>
    <header className="review-question-header"><div className="question-meta"><span>{item.subject_name}</span><span>Soal {index + 1} dari {total}</span></div><span className={`review-status ${status}`}><i aria-hidden="true">{status === "correct" ? "✓" : status === "wrong" ? "×" : "–"}</i>{status === "correct" ? "Benar" : status === "wrong" ? "Salah" : "Tidak dijawab"}</span></header>
    <div className="review-question-layout"><div className="review-question-main">
      {item.presentation_type === "group" && <div className="group-stimulus"><small>Stimulus {item.group_code}</small><RichText content={item.stimulus_text ?? ""}/><QuestionImage src={item.stimulus_image_url} alt="Gambar stimulus soal"/></div>}
      <div className="question-content"><RichText content={item.content_text}/><QuestionImage src={item.question_image_url} alt="Gambar pertanyaan"/></div>
      {item.question_type === "multiple_choice" && <p className="answer-instruction">Soal pilihan ganda kompleks · dapat memiliki lebih dari satu jawaban benar.</p>}
      {item.question_type !== "category" ? <div className="review-answer-list">{item.options.map((option) => {
        const selected = selectedKeys.has(option.key);
        const correct = correctKeys.has(option.key);
        return <div className={`review-option ${selected?"selected":""} ${correct?"correct-key":""} ${selected&&!correct?"wrong-choice":""}`} key={option.key}><span className="option-key">{option.key}</span><span className="review-option-content"><RichText content={option.content}/><QuestionImage src={option.image_url} alt={`Gambar pilihan ${option.key}`}/></span><span className="review-option-markers">{selected&&<small className="your-answer">Jawaban Anda</small>}{correct&&<small className="key-answer">Kunci</small>}</span></div>;
      })}</div> : <div className="review-category-list">{item.options.map((option) => <div className="review-category-row" key={option.key}><div><b>{option.key}</b><span><RichText content={option.content}/><QuestionImage src={option.image_url} alt={`Gambar pernyataan ${option.key}`}/></span></div><div>{(item.category_labels ?? []).map((label) => { const selected = selectedCategories[option.key] === label; const correct = correctCategories[option.key] === label; return <span className={`review-category-choice ${selected?"selected":""} ${correct?"correct-key":""} ${selected&&!correct?"wrong-choice":""}`} key={label}>{label}{selected&&<small>Jawaban Anda</small>}{correct&&<small>Kunci</small>}</span>; })}</div></div>)}</div>}
    </div><aside className="review-analysis"><p className="review-analysis-title">Analisis jawaban</p><dl><div><dt>Hasil</dt><dd className={status}>{status === "correct" ? "Jawaban benar" : status === "wrong" ? "Jawaban belum tepat" : "Belum dijawab"}</dd></div><div><dt>Jawaban Anda</dt><dd>{formatAnswer(item.selected_option)}</dd></div><div><dt>Kunci jawaban</dt><dd>{formatAnswer(item.correct_answer)}</dd></div><div><dt>Bobot soal</dt><dd>{item.score_weight}</dd></div></dl><div className="review-explanation"><h3>Pembahasan</h3><RichText content={item.explanation_text || "Pembahasan belum tersedia."}/>{item.explanation_video_url&&<a className="button secondary" href={item.explanation_video_url} target="_blank" rel="noreferrer">Tonton video pembahasan</a>}</div></aside></div>
  </article>;
}

function parseCategoryAnswer(value?: string) {
  return Object.fromEntries((value ?? "").split(";").map((part) => part.split("=", 2)).filter((part) => part.length === 2));
}

function formatAnswer(value?: string) {
  if (!value) return "—";
  if (value.includes("=")) return value.split(";").map((part) => part.replace("=", ": ")).join(" · ");
  return value.split(",").join(", ");
}

function QuestionImage({ src, alt }: { src?: string; alt: string }) {
  return src ? <Image className="question-media" src={src} alt={alt} width={800} height={500} unoptimized/> : null;
}
