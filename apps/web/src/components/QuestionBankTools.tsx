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
    chapter_name:string;
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
  "TEMPLATE BANK SOAL TKA — CARA PAKAI",
  "Isi data pada sheet 'Bank Soal'. Satu baris = satu soal. Ujian dan mata pelajaran dipilih saat paket dibuat di aplikasi.",
  "Kolom 'jenis_soal' cukup tulis: Pilihan Ganda, Pilihan Majemuk, Benar/Salah, atau Esai. Boleh dikosongkan (otomatis).",
  "Pilihan Ganda: isi pilihan_1 sampai pilihan_5, lalu tulis kunci_jawaban memakai nomor pilihan. Contoh: 2",
  "Pilihan Majemuk: jawaban benar lebih dari satu, tulis nomornya dipisah koma. Contoh: 1,3",
  "Benar/Salah: tulis kategori tiap pernyataan. Contoh: 1=Benar;2=Salah;3=Benar",
  "Esai: kosongkan pilihan, isi kunci_jawaban dengan jawaban referensi (boleh dikosongkan).",
  "Kolom 'bab' opsional (contoh: Pecahan) untuk analisis kekuatan/kelemahan siswa per bab.",
  "Kolom 'gambar_soal' dan 'gambar_pilihan_1..5' opsional untuk tautan gambar. 'kategori' hanya untuk Benar/Salah.",
  "Kolom 'status': active (tampil) atau inactive (disimpan, tidak ditampilkan). Bila kosong dianggap active.",
  "Huruf A-E juga masih diterima untuk pilihan dan kunci jawaban.",
];

const REF_ROWS=[
  ["Kolom","Nilai yang diterima","Keterangan"],
  ["jenis_soal","Pilihan Ganda | Pilihan Majemuk | Benar/Salah | Esai","Boleh juga: PG, PGK, BS. Kosong = otomatis."],
  ["kunci_jawaban","2 | 1,3 | 1=Benar;2=Salah","Pakai nomor pilihan 1-5 (huruf A-E juga boleh)"],
  ["bab","contoh: Pecahan","Opsional; untuk analisis per bab"],
  ["kategori","Benar|Salah","Khusus Benar/Salah; pemisah tanda |"],
  ["status","active | inactive","Kosong = active"],
];

const OPTION_KEYS=["a","b","c","d","e"];
const LETTERS=["a","b","c","d","e"];
const TYPE_LABEL:Record<string,string>={single_choice:"Pilihan Ganda",multiple_choice:"Pilihan Majemuk",category:"Benar/Salah",essay:"Esai"};
const TYPE_ALIASES:Record<string,string>={
  "pilihan ganda":"single_choice","pg":"single_choice","pg tunggal":"single_choice","pilihan_ganda":"single_choice","single choice":"single_choice","single_choice":"single_choice",
  "pilihan majemuk":"multiple_choice","pg majemuk":"multiple_choice","pgk":"multiple_choice","pilihan ganda majemuk":"multiple_choice","pilihan ganda kompleks":"multiple_choice","multiple choice":"multiple_choice","multiple_choice":"multiple_choice",
  "benar/salah":"category","benar salah":"category","benar-salah":"category","bs":"category","kategori":"category","category":"category","pg kategori":"category","pg benar salah":"category",
  "esai":"essay","essay":"essay","uraian":"essay","isian":"essay","esai/uraian":"essay",
};
const HEADER_ALIASES:Record<string,string>={
  no:"no",nomor:"no","nomor soal":"no",
  jenis_soal:"jenis_soal","jenis soal":"jenis_soal",jenis:"jenis_soal","question type":"jenis_soal","tipe soal":"jenis_soal","bentuk soal":"jenis_soal",bentuk:"jenis_soal",bentukan:"jenis_soal",
  bab:"bab","bab pelajaran":"bab",chapter:"bab","chapter name":"bab",
  soal:"soal",pertanyaan:"soal","question text":"soal",naskah:"soal",question:"soal",content:"soal","content text":"soal",
  kunci:"kunci","kunci jawaban":"kunci",jawaban:"kunci","correct answer":"kunci",answer:"kunci",
  bobot:"bobot",skor:"bobot","score weight":"bobot",score:"bobot","bobot nilai":"bobot",
  pembahasan:"pembahasan",penjelasan:"pembahasan",explanation:"pembahasan","explanation text":"pembahasan",
  status:"status",publikasi:"status",
  "gambar soal":"gambar_soal","question image url":"gambar_soal",gambar:"gambar_soal",image:"gambar_soal","image url":"gambar_soal",image_url:"gambar_soal",
  kategori:"kategori","category labels":"kategori","category label":"kategori",category:"kategori",labels:"kategori",label:"kategori",
};

function canonHeader(raw:unknown):string|undefined{
  const key=String(raw??"").trim().toLowerCase().replace(/[_\s]+/g," ");
  if(!key)return undefined;
  if(HEADER_ALIASES[key])return HEADER_ALIASES[key];
  let m=key.match(/^(?:pilihan|opsi|option|jawaban) ([1-5a-e])$/);
  if(m)return "option_"+m[1];
  m=key.match(/^(?:gambar pilihan|gambar opsi|gambar) ([1-5a-e])$/);
  if(m)return "option_image_"+m[1];
  m=key.match(/^(?:pilihan|opsi|option) ([1-5a-e]) (?:image url|image|gambar)$/);
  if(m)return "option_image_"+m[1];
  return undefined;
}

function slotToLetter(token:string):string|undefined{
  const t=token.trim().toLowerCase();
  if(/^[1-9]$/.test(t))return OPTION_KEYS[Number(t)-1]?.toUpperCase();
  if(OPTION_KEYS.includes(t))return t.toUpperCase();
  return undefined;
}

function letterToNumber(ch:string):string{
  const i=OPTION_KEYS.indexOf(ch.trim().toLowerCase());
  return i>=0?String(i+1):ch.trim();
}

function normalizeAnswer(raw:string,type:string):{correct_answer:string;category_labels:string[];error?:string}{
  if(type==="essay")return {correct_answer:raw.trim(),category_labels:[]};
  if(!raw.trim())return {correct_answer:"",category_labels:[],error:"Kunci jawaban belum diisi"};
  if(type==="category"){
    const parts=raw.split(";").map(s=>s.trim()).filter(Boolean);
    const pairs:{key:string;label:string}[]=[];
    for(const part of parts){
      const eq=part.indexOf("=");
      if(eq<0)return {correct_answer:"",category_labels:[],error:'Format Benar/Salah harus seperti "1=Benar;2=Salah"'};
      const key=slotToLetter(part.slice(0,eq));
      if(!key)return {correct_answer:"",category_labels:[],error:`Nomor pilihan "${part.slice(0,eq).trim()}" tidak dikenal`};
      pairs.push({key,label:part.slice(eq+1).trim()});
    }
    pairs.sort((a,b)=>OPTION_KEYS.indexOf(a.key.toLowerCase())-OPTION_KEYS.indexOf(b.key.toLowerCase()));
    const labels=[...new Set(pairs.map(p=>p.label))];
    return {correct_answer:pairs.map(p=>`${p.key}=${p.label}`).join(";"),category_labels:labels};
  }
  const tokens=raw.split(/[,;]/).map(s=>s.trim()).filter(Boolean);
  const letters:string[]=[];
  for(const token of tokens){
    const key=slotToLetter(token);
    if(!key)return {correct_answer:"",category_labels:[],error:`Kunci "${token}" tidak dikenal (gunakan nomor 1-5 atau huruf A-E)`};
    letters.push(key);
  }
  const sorted=[...new Set(letters)].sort((a,b)=>OPTION_KEYS.indexOf(a.toLowerCase())-OPTION_KEYS.indexOf(b.toLowerCase()));
  if(type==="single_choice"&&sorted.length>1)return {correct_answer:"",category_labels:[],error:"Pilihan Ganda hanya boleh satu kunci"};
  return {correct_answer:sorted.join(","),category_labels:[]};
}

function answerToNumbers(answer:string,type:string):string{
  if(type==="essay"||!answer)return answer??"";
  if(type==="category")return answer.split(";").map(part=>{const eq=part.indexOf("=");if(eq<0)return part;return `${letterToNumber(part.slice(0,eq))}=${part.slice(eq+1)}`}).join(";");
  return answer.split(",").map(a=>letterToNumber(a)).join(",");
}

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
      {header:"no",key:"no",width:8},
      {header:"jenis_soal",key:"jenis_soal",width:18},
      {header:"bab",key:"bab",width:20},
      {header:"soal",key:"soal",width:44},
      {header:"pilihan_1",key:"pilihan_1",width:26},{header:"pilihan_2",key:"pilihan_2",width:26},{header:"pilihan_3",key:"pilihan_3",width:26},{header:"pilihan_4",key:"pilihan_4",width:26},{header:"pilihan_5",key:"pilihan_5",width:26},
      {header:"kunci_jawaban",key:"kunci_jawaban",width:20},
      {header:"bobot",key:"bobot",width:10},
      {header:"pembahasan",key:"pembahasan",width:36},
      {header:"status",key:"status",width:12},
      {header:"gambar_soal",key:"gambar_soal",width:28},
      {header:"kategori",key:"kategori",width:16},
      {header:"gambar_pilihan_1",key:"gambar_pilihan_1",width:26},{header:"gambar_pilihan_2",key:"gambar_pilihan_2",width:26},{header:"gambar_pilihan_3",key:"gambar_pilihan_3",width:26},{header:"gambar_pilihan_4",key:"gambar_pilihan_4",width:26},{header:"gambar_pilihan_5",key:"gambar_pilihan_5",width:26},
    ];
    const headerRow=dataSheet.getRow(1);
    headerRow.font={bold:true,color:{argb:"FFFFFF"}};
    headerRow.fill={type:"pattern",pattern:"solid",fgColor:{argb:"4472C4"}};
    headerRow.alignment={horizontal:"center"};

    questions.forEach((item,itemIndex)=>{
      const byKey=Object.fromEntries((item.options??[]).map(option=>[option.key.toUpperCase(),option]));
      const type=item.question_type||"single_choice";
      dataSheet.addRow({
        no:itemIndex+1,
        jenis_soal:TYPE_LABEL[type]??type,
        bab:item.chapter_name??"",
        soal:item.content_text??"",
        pilihan_1:byKey.A?.content??"",pilihan_2:byKey.B?.content??"",pilihan_3:byKey.C?.content??"",pilihan_4:byKey.D?.content??"",pilihan_5:byKey.E?.content??"",
        kunci_jawaban:answerToNumbers(item.correct_answer??"",type),
        bobot:item.score_weight??"",
        pembahasan:item.explanation_text??"",
        status:item.status,
        gambar_soal:item.question_image_url??"",
        kategori:(item.category_labels??[]).join("|"),
        gambar_pilihan_1:byKey.A?.image_url??"",gambar_pilihan_2:byKey.B?.image_url??"",gambar_pilihan_3:byKey.C?.image_url??"",gambar_pilihan_4:byKey.D?.image_url??"",gambar_pilihan_5:byKey.E?.image_url??"",
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
        const field=canonHeader(cell.value);
        if(field&&!colMap[field])colMap[field]=colNumber;
      });
      if(!colMap["soal"]&&!colMap["jenis_soal"]){
        setImportProgress(p=>p?{...p,errors:["Header sheet 'Bank Soal' tidak dikenali. Silakan unduh ulang template terbaru."]}:null);
        return;
      }

      const parseErrors:string[]=[];
      const parsed:{
        row:number;
        question_type:string;
        chapter_name:string;
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
        const contentText=val(row,colMap["soal"]??0);
        const rawType=val(row,colMap["jenis_soal"]??0).toLowerCase().replace(/\s+/g," ").trim();
        const rawAnswer=val(row,colMap["kunci"]??0);
        const rawLabels=val(row,colMap["kategori"]??0);
        if(!contentText&&!rawType&&!rawAnswer)return;

        const options:{key:string;content:string;image_url?:string}[]=[];
        LETTERS.forEach((letter,slot)=>{
          const content=val(row,colMap[`option_${slot+1}`]??0)||val(row,colMap[`option_${letter}`]??0);
          const image_url=val(row,colMap[`option_image_${slot+1}`]??0)||val(row,colMap[`option_image_${letter}`]??0);
          if(content||image_url)options.push({key:letter.toUpperCase(),content,image_url:image_url||undefined});
        });

        let qType=rawType?TYPE_ALIASES[rawType]:undefined;
        if(rawType&&!qType){
          parseErrors.push(`Baris ${rowNumber}: jenis soal "${rawType}" tidak dikenal. Gunakan: Pilihan Ganda, Pilihan Majemuk, Benar/Salah, atau Esai.`);
          return;
        }
        if(!qType){
          if(rawAnswer.includes("="))qType="category";
          else if(!options.length)qType="essay";
          else if(/[,;]/.test(rawAnswer))qType="multiple_choice";
          else qType="single_choice";
        }
        if(!contentText){parseErrors.push(`Baris ${rowNumber}: kolom "soal" kosong`);return}
        if(qType!=="essay"&&options.length<2){parseErrors.push(`Baris ${rowNumber}: butuh minimal 2 pilihan jawaban (isi pilihan_1 dan seterusnya)`);return}

        const normalized=normalizeAnswer(rawAnswer,qType);
        if(normalized.error){parseErrors.push(`Baris ${rowNumber}: ${normalized.error}`);return}
        const category_labels=qType==="category"&&rawLabels?rawLabels.split(/[|,]/).map(s=>s.trim()).filter(Boolean):normalized.category_labels;
        if(qType==="category"&&!category_labels.length){parseErrors.push(`Baris ${rowNumber}: kategori Benar/Salah belum diisi`);return}

        parsed.push({
          row:rowNumber,
          question_type:qType,
          chapter_name:val(row,colMap["bab"]??0),
          content_text:contentText,
          question_image_url:val(row,colMap["gambar_soal"]??0),
          options,
          category_labels,
          correct_answer:normalized.correct_answer,
          score_weight:Number(val(row,colMap["bobot"]??0))||1,
          explanation_text:val(row,colMap["pembahasan"]??0),
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
            chapter_name:row.chapter_name,
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
