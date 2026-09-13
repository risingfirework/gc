"use client";

import Image from "next/image";
import { useRef, useState } from "react";
import ExcelJS from "exceljs";
import { AdminQuestion, PresentationType, QuestionType, Status } from "@/services/api";
import RichText from "./RichText";

type Props={
  packageTitle:string;
  packageCode:string;
  level:string;
  questions:AdminQuestion[];
  durationMinutes?:number;
  examId?:string;
  subjectName?:string;
  onImportQuestion?:(input:{
    exam_id:string;
    subject_name:string;
    content_text:string;
    question_type:QuestionType;
    presentation_type:PresentationType;
    group_code:string;
    stimulus_text:string;
    question_image_url:string;
    stimulus_image_url:string;
    category_labels:string[];
    options:{key:string;content:string;image_url?:string}[];
    correct_answer:string;
    score_weight:number;
    explanation_text:string;
    status:Status;
  })=>Promise<unknown>;
};

const GUIDE_ROWS=[
  "TEMPLATE BANK SOAL TKA",
  "Isi data pada sheet Bank Soal. Ujian dan mata pelajaran dipilih saat paket dibuat di aplikasi.",
  "PG Sederhana: satu kunci, contoh B.",
  "PGK MCMA: minimal dua kunci dipisahkan koma, contoh A,C.",
  "PGK Kategori: seluruh pernyataan diberi kategori, contoh A=Benar;B=Salah;C=Benar.",
  "Esai: correct_answer diisi jawaban referensi, option dibiarkan kosong.",
  "Setiap baris adalah satu soal mandiri. Masukkan stimulus langsung pada question_text.",
  "Gunakan nilai kode pada sheet Referensi agar data konsisten.",
];

const REF_ROWS=[
  ["Field","Nilai yang diizinkan","Keterangan"],
  ["question_type","single_choice | multiple_choice | category | essay","Empat bentuk soal TKA"],
  ["category_labels","Benar|Salah","Pisahkan kategori dengan tanda |"],
  ["correct_answer","B / A,C / A=Benar;B=Salah","Sintaks mengikuti bentuk soal"],
  ["status","active | inactive","Status publikasi"],
];

const VALID_TYPES=["single_choice","multiple_choice","category","essay"];
const OPTION_KEYS=["a","b","c","d","e"];

export default function QuestionBankTools({
  packageTitle,packageCode,level,questions,durationMinutes=60,
  examId,subjectName,onImportQuestion,
}:Props){
  const [preview,setPreview]=useState(false);
  const [index,setIndex]=useState(0);
  const [answers,setAnswers]=useState<Record<string,string>>({});
  const [importing,setImporting]=useState(false);
  const [importProgress,setImportProgress]=useState<{total:number;done:number;errors:string[]} | null>(null);
  const fileRef=useRef<HTMLInputElement>(null);
  const question=questions[index];

  function choose(key:string){
    if(!question)return;
    if(question.question_type==="multiple_choice"){
      const selected=new Set((answers[question.id]??"").split(",").filter(Boolean));
      if(selected.has(key))selected.delete(key);else selected.add(key);
      setAnswers(current=>({...current,[question.id]:[...selected].sort().join(",")}));
      return;
    }
    setAnswers(current=>({...current,[question.id]:key}));
  }

  function chooseCategory(optionKey:string,label:string){
    if(!question)return;
    const values=Object.fromEntries((answers[question.id]??"").split(";").filter(Boolean).map(part=>part.split("=",2)));
    values[optionKey]=label;
    setAnswers(current=>({...current,[question.id]:Object.entries(values).map(([key,value])=>`${key}=${value}`).join(";")}));
  }

  function printPDF(){
    const oldTitle=document.title;
    document.title=`${packageCode}-${packageTitle}`;
    window.print();
    window.setTimeout(()=>{document.title=oldTitle},500);
  }

  async function downloadExcel(){
    const wb=new ExcelJS.Workbook();
    wb.creator="TKA";
    wb.created=new Date();

    const guideSheet=wb.addWorksheet("Panduan",{properties:{tabColor:{argb:"4472C4"}}});
    GUIDE_ROWS.forEach((text,rowIdx)=>{
      const row=guideSheet.addRow([text]);
      if(rowIdx===0)row.getCell(1).font={bold:true,size:16};
    });
    guideSheet.getColumn(1).width=110;

    const dataSheet=wb.addWorksheet("Bank Soal",{views:[{state:"frozen",ySplit:1}]});
    dataSheet.columns=[
      {header:"no",key:"no",width:11},
      {header:"question_type",key:"question_type",width:18},
      {header:"question_text",key:"question_text",width:30},
      {header:"question_image_url",key:"question_image_url",width:30},
      {header:"option_a",key:"option_a",width:28},{header:"option_a_image_url",key:"option_a_image_url",width:28},
      {header:"option_b",key:"option_b",width:28},{header:"option_b_image_url",key:"option_b_image_url",width:28},
      {header:"option_c",key:"option_c",width:28},{header:"option_c_image_url",key:"option_c_image_url",width:28},
      {header:"option_d",key:"option_d",width:28},{header:"option_d_image_url",key:"option_d_image_url",width:28},
      {header:"option_e",key:"option_e",width:28},{header:"option_e_image_url",key:"option_e_image_url",width:28},
      {header:"category_labels",key:"category_labels",width:22},
      {header:"correct_answer",key:"correct_answer",width:18},
      {header:"score_weight",key:"score_weight",width:13},
      {header:"explanation",key:"explanation",width:30},
      {header:"status",key:"status",width:12},
    ];
    const headerRow=dataSheet.getRow(1);
    headerRow.font={bold:true,color:{argb:"FFFFFF"}};
    headerRow.fill={type:"pattern",pattern:"solid",fgColor:{argb:"4472C4"}};
    headerRow.alignment={horizontal:"center"};

    questions.forEach((item,itemIndex)=>{
      const byKey=Object.fromEntries((item.options??[]).map(option=>[option.key,option]));
      dataSheet.addRow({
        no:itemIndex+1,
        question_type:item.question_type,
        question_text:item.content_text??"",
        question_image_url:item.question_image_url??"",
        option_a:byKey.A?.content??"",option_a_image_url:byKey.A?.image_url??"",
        option_b:byKey.B?.content??"",option_b_image_url:byKey.B?.image_url??"",
        option_c:byKey.C?.content??"",option_c_image_url:byKey.C?.image_url??"",
        option_d:byKey.D?.content??"",option_d_image_url:byKey.D?.image_url??"",
        option_e:byKey.E?.content??"",option_e_image_url:byKey.E?.image_url??"",
        category_labels:(item.category_labels??[]).join("|"),
        correct_answer:item.correct_answer,
        score_weight:item.score_weight??"",
        explanation:item.explanation_text??"",
        status:item.status,
      });
    });

    const refSheet=wb.addWorksheet("Referensi",{properties:{tabColor:{argb:"70AD47"}}});
    REF_ROWS.forEach(row=>refSheet.addRow(row));
    refSheet.getRow(1).font={bold:true,color:{argb:"FFFFFF"}};
    refSheet.getRow(1).fill={type:"pattern",pattern:"solid",fgColor:{argb:"70AD47"}};
    refSheet.getColumn(1).width=24;
    refSheet.getColumn(2).width=48;
    refSheet.getColumn(3).width=48;

    const buffer=await wb.xlsx.writeBuffer();
    const blob=new Blob([buffer],{type:"application/vnd.openxmlformats-officedocument.spreadsheetml.sheet"});
    const url=URL.createObjectURL(blob);
    const anchor=document.createElement("a");
    anchor.href=url;
    anchor.download=`${packageCode}-${packageTitle}-soal.xlsx`;
    document.body.appendChild(anchor);
    anchor.click();
    document.body.removeChild(anchor);
    URL.revokeObjectURL(url);
  }

  async function handleImportFile(event:React.ChangeEvent<HTMLInputElement>){
    const file=event.target.files?.[0];
    if(!file||!examId||!onImportQuestion)return;
    setImporting(true);
    setImportProgress({total:0,done:0,errors:[]});
    try{
      const wb=new ExcelJS.Workbook();
      const buffer=await file.arrayBuffer();
      await wb.xlsx.load(buffer);
      const sheet=wb.getWorksheet("Bank Soal")??wb.worksheets[1];
      if(!sheet){setImportProgress(p=>p?{...p,errors:["Sheet 'Bank Soal' tidak ditemukan"]}:null);return}

      const headerRow=sheet.getRow(1);
      const colMap:Record<string,number>={};
      headerRow.eachCell((cell,colNumber)=>{
        const val=String(cell.value??"").trim().toLowerCase();
        if(val)colMap[val]=colNumber;
      });

      const parseErrors:string[]=[];
      const parsed:{
        row:number;
        question_type:string;
        content_text:string;
        question_image_url:string;
        options:{key:string;content:string;image_url?:string}[];
        category_labels:string[];
        correct_answer:string;
        score_weight:number;
        explanation_text:string;
        status:string;
      }[]=[];

      const val=(row:ExcelJS.Row,col:number)=>col>0?String(row.getCell(col).value??"").trim():"";
      sheet.eachRow((row,rowNumber)=>{
        if(rowNumber<=1)return;
        const qType=val(row,colMap["question_type"]??3).toLowerCase();
        if(!qType)return;
        if(!VALID_TYPES.includes(qType)){
          parseErrors.push(`Baris ${rowNumber}: bentukan soal "${qType}" tidak dikenal`);
          return;
        }
        const contentText=val(row,colMap["question_text"]??4);
        if(!contentText){parseErrors.push(`Baris ${rowNumber}: question_text kosong`);return}

        const options:{key:string;content:string;image_url?:string}[]=[];
        if(qType!=="essay"){
          OPTION_KEYS.forEach(k=>{
            const content=val(row,colMap[`option_${k}`]??0);
            const image_url=val(row,colMap[`option_${k}_image_url`]??0);
            if(content||image_url)options.push({key:k.toUpperCase(),content,image_url:image_url||undefined});
          });
          if(options.length<2){parseErrors.push(`Baris ${rowNumber}: butuh minimal 2 pilihan jawaban`);return}
        }

        const catRaw=val(row,colMap["category_labels"]??0);
        const category_labels=catRaw?catRaw.split("|").map(s=>s.trim()).filter(Boolean):[];

        parsed.push({
          row:rowNumber,
          question_type:qType,
          content_text:contentText,
          question_image_url:val(row,colMap["question_image_url"]??0),
          options,
          category_labels,
          correct_answer:val(row,colMap["correct_answer"]??0),
          score_weight:Number(val(row,colMap["score_weight"]??0))||1,
          explanation_text:val(row,colMap["explanation"]??0),
          status:val(row,colMap["status"]??0).toLowerCase()==="inactive"?"inactive":"active",
        });
      });

      setImportProgress({total:parsed.length,done:0,errors:[...parseErrors]});
      const errors=[...parseErrors];
      for(let i=0;i<parsed.length;i++){
        const row=parsed[i];
        try{
          await onImportQuestion({
            exam_id:examId,
            subject_name:subjectName??"",
            content_text:row.content_text,
            question_type:row.question_type as QuestionType,
            presentation_type:"single",
            group_code:"",
            stimulus_text:"",
            question_image_url:row.question_image_url,
            stimulus_image_url:"",
            category_labels:row.category_labels,
            options:row.options,
            correct_answer:row.correct_answer,
            score_weight:row.score_weight,
            explanation_text:row.explanation_text,
            status:row.status as Status,
          });
        }catch(e){errors.push(`Baris ${row.row}: ${String(e).slice(0,80)}`)}
        setImportProgress(prev=>prev?{...prev,done:i+1,errors:[...errors]}:null);
      }
      setImportProgress(prev=>prev?{...prev,errors}:null);
    }catch(e){
      setImportProgress(p=>p?{...p,errors:[...p.errors,"Gagal membaca file: "+String(e).slice(0,100)]}:null);
    }finally{
      setImporting(false);
      if(fileRef.current)fileRef.current.value="";
    }
  }

  const answered=(item:AdminQuestion)=>Boolean(answers[item.id]);

  return <>
    <div className="question-bank-tools">
      <button className="button secondary" type="button" disabled={!questions.length} onClick={()=>{setIndex(0);setPreview(true)}}>Preview Tes</button>
      <button className="button secondary" type="button" disabled={!questions.length} onClick={printPDF}>Cetak / Simpan PDF</button>
      <button className="button secondary" type="button" disabled={!questions.length} onClick={downloadExcel}>Download Soal</button>
      <input ref={fileRef} type="file" accept=".xlsx" className="hidden-file-input" onChange={handleImportFile}/>
      <button className="button secondary" type="button" disabled={!examId||!onImportQuestion||importing} onClick={()=>fileRef.current?.click()}>{importing?"Mengimpor...":"Import Soal"}</button>
      <span>{questions.length} soal siap ditampilkan</span>
    </div>
    {importProgress&&<div className="import-progress-card">
      <div className="import-progress-header"><strong>Import Soal</strong><button type="button" className="import-progress-close" onClick={()=>setImportProgress(null)}>×</button></div>
      {importProgress.total>0&&<div className="import-progress-bar-wrap"><div className="import-progress-bar" style={{width:`${Math.round((importProgress.done/importProgress.total)*100)}%`}}/></div>}
      <p className="import-progress-status">{importProgress.done}/{importProgress.total} soal terimpor{importProgress.errors.length?` · ${importProgress.errors.length} galat`:importProgress.done===importProgress.total?" · Selesai":""}</p>
      {importProgress.errors.length>0&&<ul className="import-progress-errors">{importProgress.errors.map((e,i)=><li key={i}>{e}</li>)}</ul>}
    </div>}
    {preview&&question&&<div className="preview-backdrop" role="presentation" onMouseDown={()=>setPreview(false)}><section className="preview-shell" role="dialog" aria-modal="true" aria-label={`Preview tes ${packageTitle}`} onMouseDown={event=>event.stopPropagation()}>
      <header className="preview-header"><div><small>MODE PREVIEW · Tidak menyimpan hasil</small><strong>{packageTitle}</strong><span>{packageCode} · {level}</span></div><div className="timer-box"><small>Durasi tes</small><strong>{durationMinutes}:00</strong></div><button className="preview-close" type="button" aria-label="Tutup preview" onClick={()=>setPreview(false)}>×</button></header>
      <div className="preview-layout"><main className="card preview-question"><div className="question-meta"><span>{question.subject_name}</span><span>Soal {index+1} dari {questions.length}</span></div><div className="question-content"><RichText content={question.content_text}/><QuestionImage src={question.question_image_url} alt="Gambar soal"/></div>
      {question.question_type==="essay"?<div className="essay-answer-preview"><p className="answer-instruction">Soal esai · jawaban uraian ditulis peserta pada lembar jawaban.</p><textarea readOnly disabled rows={6} placeholder="Ruang jawaban esai"/></div>:question.question_type==="category"?<div className="category-answer-table">{question.options.map(option=><div className="category-answer-row" key={option.key}><div><b>{option.key}</b><span><RichText content={option.content}/><QuestionImage src={option.image_url} alt={`Gambar ${option.key}`}/></span></div><div>{question.category_labels.map(label=>{const chosen=(answers[question.id]??"").split(";").includes(`${option.key}=${label}`);return <button type="button" className={chosen?"selected":""} onClick={()=>chooseCategory(option.key,label)} key={label}>{label}</button>})}</div></div>)}</div>:<div className="answer-list">{question.options.map(option=>{const selected=question.question_type==="multiple_choice"?(answers[question.id]??"").split(",").includes(option.key):answers[question.id]===option.key;return <button type="button" className={`option ${selected?"selected":""}`} onClick={()=>choose(option.key)} key={option.key}><span className="option-key">{selected&&question.question_type==="multiple_choice"?"✓":option.key}</span><span><RichText content={option.content}/><QuestionImage src={option.image_url} alt={`Gambar pilihan ${option.key}`}/></span></button>})}</div>}
      <footer className="exam-actions"><button className="button secondary" type="button" disabled={index===0} onClick={()=>setIndex(current=>current-1)}>Sebelumnya</button>{index<questions.length-1?<button className="button" type="button" onClick={()=>setIndex(current=>current+1)}>Selanjutnya</button>:<button className="button submit-button" type="button" onClick={()=>setPreview(false)}>Selesai preview</button>}</footer></main>
      <aside className="card question-panel preview-panel"><div className="panel-summary"><strong>{questions.filter(answered).length}/{questions.length}</strong><span>soal dicoba</span></div><div className="numbers">{questions.map((item,itemIndex)=><button type="button" className={`number ${answered(item)?"done":""} ${itemIndex===index?"current":""}`} onClick={()=>setIndex(itemIndex)} key={item.id}>{itemIndex+1}</button>)}</div><div className="legend"><span><i className="done"/>Dicoba</span><span><i/>Belum</span></div><p className="preview-note">Jawaban dalam preview hanya simulasi dan tidak masuk ke hasil siswa.</p></aside></div>
    </section></div>}
    <section className="question-bank-print"><header><p>PAKET SOAL · {level}</p><h1>{packageTitle}</h1><div><span>Kode: <b>{packageCode}</b></span><span>Jumlah soal: <b>{questions.length}</b></span><span>Durasi: <b>{durationMinutes} menit</b></span></div></header><div className="print-student"><span>Nama: __________________________________</span><span>Kelas: ______________</span><span>Tanggal: ______________</span></div>{questions.map((item,itemIndex)=><article className="print-question" key={item.id}><div className="print-question-number">{itemIndex+1}</div><div><small>{item.subject_name}</small><p><RichText content={item.content_text}/></p><QuestionImage src={item.question_image_url} alt="Gambar soal"/>{item.question_type==="essay"?<div className="print-essay-space"><p className="print-essay-label">Jawaban:</p><div className="print-essay-lines"/></div>:<ol type="A">{item.options.map(option=><li key={option.key}><RichText content={option.content}/><QuestionImage src={option.image_url} alt={`Gambar ${option.key}`}/></li>)}</ol>}</div></article>)}<section className="print-answer-key"><h2>Kunci Jawaban dan Pembahasan</h2>{questions.map((item,index)=><div key={item.id}><b>{index+1}. {item.correct_answer}</b>{item.explanation_text&&<p><RichText content={item.explanation_text}/></p>}</div>)}</section><footer>Dicetak dari sistem bank soal · {packageCode}</footer></section>
  </>;
}

function QuestionImage({src,alt}:{src?:string;alt:string}){return src?<Image className="question-media" src={src} alt={alt} width={800} height={500} unoptimized/>:null}
