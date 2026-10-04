// Generator bank soal contoh: Matematika SD (30 soal).
// Jalankan dari mana saja:
//   node tools/bank-soal/generate-bank-soal-matematika-sd.mjs [output.xlsx]
// Default output: Bank-Soal-Matematika-SD.xlsx di direktori kerja saat ini.
// Hasil .xlsx diimpor lewat Admin Panel > (paket CBT) > Bank Soal > Import Soal.
// Prasyarat: dependensi web (exceljs) sudah terpasang di apps/web/node_modules.

import { createRequire } from "module";
import { fileURLToPath } from "url";
import path from "path";

const scriptDir = path.dirname(fileURLToPath(import.meta.url));
const require = createRequire(path.join(scriptDir, "..", "..", "apps", "web", "package.json"));
const ExcelJS = require("exceljs");

const OPTION_KEYS = ["A", "B", "C", "D", "E"];

function chapterForQuestion(index) {
  if (index < 5) return "Bilangan dan Pecahan";
  if (index < 8) return "Geometri";
  if (index < 10) return "Pengolahan Data";
  if (index < 12) return "Pengukuran";
  return "Bilangan dan Perbandingan";
}

const QUESTIONS = [
  // ================= PILIHAN GANDA (single_choice) 15 =================
  {
    t: "single_choice",
    text: "Hasil dari 2.450 + 1.375 - 925 adalah ...",
    opts: { A: "2.700", B: "2.800", C: "2.900", D: "3.000" },
    ans: "C",
    exp: "2.450 + 1.375 = 3.825. Lalu 3.825 - 925 = 2.900.",
  },
  {
    t: "single_choice",
    text: "KPK (Kelipatan Persekutuan Terkecil) dari 12 dan 18 adalah ...",
    opts: { A: "24", B: "36", C: "48", D: "72" },
    ans: "B",
    exp: "12 = 2^2 x 3 dan 18 = 2 x 3^2. KPK = 2^2 x 3^2 = 4 x 9 = 36.",
  },
  {
    t: "single_choice",
    text: "FPB (Faktor Persekutuan Terbesar) dari 24 dan 36 adalah ...",
    opts: { A: "6", B: "8", C: "12", D: "18" },
    ans: "C",
    exp: "24 = 2^3 x 3 dan 36 = 2^2 x 3^2. FPB = 2^2 x 3 = 4 x 3 = 12.",
  },
  {
    t: "single_choice",
    text: "Bentuk pecahan campuran dari 17/5 adalah ...",
    opts: { A: "3 1/5", B: "3 2/5", C: "4 1/5", D: "4 2/5" },
    ans: "B",
    exp: "17 : 5 = 3 sisa 2, sehingga 17/5 = 3 2/5.",
  },
  {
    t: "single_choice",
    text: "Urutan pecahan 3/5 ; 0,75 ; 2/5 ; 0,5 dari yang terkecil adalah ...",
    opts: {
      A: "2/5 ; 0,5 ; 3/5 ; 0,75",
      B: "0,5 ; 2/5 ; 3/5 ; 0,75",
      C: "2/5 ; 3/5 ; 0,5 ; 0,75",
      D: "0,75 ; 3/5 ; 0,5 ; 2/5",
    },
    ans: "A",
    exp: "Ubah ke desimal: 2/5 = 0,4 ; 0,5 ; 3/5 = 0,6 ; 0,75. Jadi urutannya 2/5 ; 0,5 ; 3/5 ; 0,75.",
  },
  {
    t: "single_choice",
    text: "Sebuah persegi panjang memiliki panjang 15 cm dan lebar 9 cm. Luas persegi panjang tersebut adalah ...",
    opts: { A: "24 cm2", B: "48 cm2", C: "135 cm2", D: "270 cm2" },
    ans: "C",
    exp: "Luas = panjang x lebar = 15 x 9 = 135 cm2.",
  },
  {
    t: "single_choice",
    text: "Keliling sebuah segitiga dengan panjang sisi 8 cm, 10 cm, dan 12 cm adalah ...",
    opts: { A: "20 cm", B: "28 cm", C: "30 cm", D: "32 cm" },
    ans: "C",
    exp: "Keliling segitiga = jumlah semua sisi = 8 + 10 + 12 = 30 cm.",
  },
  {
    t: "single_choice",
    text: "Volume kubus yang panjang rusuknya 6 cm adalah ...",
    opts: { A: "36 cm3", B: "108 cm3", C: "216 cm3", D: "432 cm3" },
    ans: "C",
    exp: "Volume kubus = rusuk x rusuk x rusuk = 6 x 6 x 6 = 216 cm3.",
  },
  {
    t: "single_choice",
    text: "Rata-rata dari data 7, 8, 6, 9, dan 10 adalah ...",
    opts: { A: "7", B: "8", C: "8,5", D: "9" },
    ans: "B",
    exp: "Jumlah data = 7 + 8 + 6 + 9 + 10 = 40. Banyak data 5, rata-rata = 40 : 5 = 8.",
  },
  {
    t: "single_choice",
    text: "Modus dari data 3, 5, 3, 7, 3, 5, 6 adalah ...",
    opts: { A: "3", B: "5", C: "6", D: "7" },
    ans: "A",
    exp: "Modus adalah data yang paling sering muncul. Angka 3 muncul tiga kali, lebih sering daripada yang lain.",
  },
  {
    t: "single_choice",
    text: "Sebuah peta memiliki skala 1 : 500.000. Jika jarak dua kota pada peta 4 cm, jarak sebenarnya kedua kota tersebut adalah ...",
    opts: { A: "2 km", B: "20 km", C: "200 km", D: "2.000 km" },
    ans: "B",
    exp: "Jarak sebenarnya = 4 cm x 500.000 = 2.000.000 cm = 20 km (1 km = 100.000 cm).",
  },
  {
    t: "single_choice",
    text: "Sebuah mobil melaju dengan kecepatan 60 km/jam selama 3 jam. Jarak yang ditempuh mobil tersebut adalah ...",
    opts: { A: "20 km", B: "120 km", C: "180 km", D: "240 km" },
    ans: "C",
    exp: "Jarak = kecepatan x waktu = 60 x 3 = 180 km.",
  },
  {
    t: "single_choice",
    text: "Hasil dari (-8) + 15 - (-3) adalah ...",
    opts: { A: "4", B: "10", C: "20", D: "26" },
    ans: "B",
    exp: "(-8) + 15 = 7. Lalu 7 - (-3) = 7 + 3 = 10.",
  },
  {
    t: "single_choice",
    text: "Nilai 15% dari 240 adalah ...",
    opts: { A: "24", B: "32", C: "36", D: "48" },
    ans: "C",
    exp: "15% = 15/100. Maka 15/100 x 240 = 3.600/100 = 36.",
  },
  {
    t: "single_choice",
    text: "Perbandingan uang Andi dan Budi adalah 3 : 5. Jika uang Budi Rp75.000, uang Andi adalah ...",
    opts: { A: "Rp25.000", B: "Rp45.000", C: "Rp50.000", D: "Rp125.000" },
    ans: "B",
    exp: "Uang Andi = (3/5) x Rp75.000 = 3 x Rp15.000 = Rp45.000.",
  },

  // ================= PILIHAN GANDA MAJEMUK (multiple_choice) 5 =================
  {
    t: "multiple_choice",
    text: "Pilihlah semua bilangan berikut yang habis dibagi 3!",
    opts: { A: "123", B: "145", C: "216", D: "310", E: "414" },
    ans: "A,C,E",
    exp: "Suatu bilangan habis dibagi 3 jika jumlah angkanya kelipatan 3. 123 (1+2+3=6) benar, 216 (2+1+6=9) benar, 414 (4+1+4=9) benar. 145 (10) dan 310 (4) tidak habis dibagi 3.",
  },
  {
    t: "multiple_choice",
    text: "Perhatikan bilangan 4, 9, 15, 16, dan 25. Pilihlah semua bilangan kuadrat sempurna!",
    opts: { A: "4", B: "9", C: "15", D: "16", E: "25" },
    ans: "A,B,D,E",
    exp: "Bilangan kuadrat sempurna adalah hasil perkalian suatu bilangan dengan dirinya sendiri: 4 = 2x2, 9 = 3x3, 16 = 4x4, 25 = 5x5. Sedangkan 15 bukan kuadrat sempurna.",
  },
  {
    t: "multiple_choice",
    text: "Pilihlah semua pecahan yang senilai dengan 2/3!",
    opts: { A: "4/6", B: "6/9", C: "8/10", D: "10/15", E: "12/16" },
    ans: "A,B,D",
    exp: "4/6 = 2/3, 6/9 = 2/3, dan 10/15 = 2/3 karena pembilang dan penyebut dikali bilangan yang sama. 8/10 = 4/5 dan 12/16 = 3/4, tidak senilai dengan 2/3.",
  },
  {
    t: "multiple_choice",
    text: "Pilihlah semua bangun datar yang keempat sisinya sama panjang!",
    opts: { A: "persegi", B: "persegi panjang", C: "belah ketupat", D: "jajar genjang", E: "trapesium sama kaki" },
    ans: "A,C",
    exp: "Persegi dan belah ketupat memiliki empat sisi yang sama panjang. Persegi panjang dan jajar genjang hanya sisi-sisi yang berhadapan yang sama panjang, sedangkan trapesium sama kaki hanya sepasang kakinya.",
  },
  {
    t: "multiple_choice",
    text: "Pilihlah semua bilangan prima dari bilangan berikut!",
    opts: { A: "9", B: "11", C: "15", D: "17", E: "21" },
    ans: "B,D",
    exp: "Bilangan prima hanya memiliki dua faktor, yaitu 1 dan bilangan itu sendiri. 11 dan 17 hanya bisa dibagi 1 dan dirinya sendiri. 9 = 3x3, 15 = 3x5, dan 21 = 3x7.",
  },

  // ================= BENAR / SALAH (category) 5 =================
  {
    t: "category",
    text: "Tentukan Benar atau Salah untuk setiap pernyataan tentang pecahan berikut.",
    opts: {
      A: "1/2 lebih besar dari 1/3.",
      B: "2/4 sama nilainya dengan 1/2.",
      C: "3/5 lebih kecil dari 1/2.",
      D: "5/8 lebih besar dari 3/4.",
      E: "7/10 sama dengan 70%.",
    },
    labels: ["Benar", "Salah"],
    ans: "A=Benar;B=Benar;C=Salah;D=Salah;E=Benar",
    exp: "1/2 = 0,5 lebih besar dari 1/3 = 0,33 (Benar). 2/4 = 1/2 (Benar). 3/5 = 0,6 lebih besar dari 0,5, jadi pernyataannya salah. 5/8 = 0,625 lebih kecil dari 3/4 = 0,75, jadi salah. 7/10 = 0,7 = 70% (Benar).",
  },
  {
    t: "category",
    text: "Tentukan Benar atau Salah untuk setiap pernyataan tentang bilangan bulat berikut.",
    opts: {
      A: "-5 lebih kecil dari -2.",
      B: "-10 lebih besar dari 0.",
      C: "Hasil dari (-3) x (-4) adalah 12.",
      D: "Hasil dari (-20) : 4 adalah -5.",
      E: "Lawan (invers penjumlahan) dari -7 adalah -7.",
    },
    labels: ["Benar", "Salah"],
    ans: "A=Benar;B=Salah;C=Benar;D=Benar;E=Salah",
    exp: "-5 < -2 (Benar). -10 lebih kecil dari 0, jadi pernyataannya salah. (-3) x (-4) = 12 (Benar, negatif x negatif = positif). (-20) : 4 = -5 (Benar). Lawan dari -7 adalah 7, bukan -7 (Salah).",
  },
  {
    t: "category",
    text: "Tentukan Benar atau Salah untuk setiap pernyataan tentang bangun datar berikut.",
    opts: {
      A: "Persegi memiliki empat sudut siku-siku.",
      B: "Luas segitiga = alas x tinggi.",
      C: "Jumlah sudut dalam sebuah segitiga adalah 180 derajat.",
      D: "Panjang diameter lingkaran dua kali panjang jari-jarinya.",
      E: "Semua sisi persegi panjang sama panjang.",
    },
    labels: ["Benar", "Salah"],
    ans: "A=Benar;B=Salah;C=Benar;D=Benar;E=Salah",
    exp: "Persegi punya empat sudut 90 derajat (Benar). Luas segitiga = (alas x tinggi) : 2, jadi pernyataannya salah. Jumlah sudut segitiga 180 derajat (Benar). Diameter = 2 x jari-jari (Benar). Pada persegi panjang hanya sisi berhadapan yang sama, jadi pernyataannya salah.",
  },
  {
    t: "category",
    text: "Tentukan Benar atau Salah untuk setiap pernyataan tentang satuan pengukuran berikut.",
    opts: {
      A: "1 km sama dengan 1.000 m.",
      B: "1 kg sama dengan 100 gram.",
      C: "1 jam sama dengan 3.600 detik.",
      D: "1 liter sama dengan 1.000 mililiter.",
      E: "1 abad sama dengan 10 tahun.",
    },
    labels: ["Benar", "Salah"],
    ans: "A=Benar;B=Salah;C=Benar;D=Benar;E=Salah",
    exp: "1 km = 1.000 m (Benar). 1 kg = 1.000 gram, jadi pernyataannya salah. 1 jam = 60 x 60 = 3.600 detik (Benar). 1 liter = 1.000 mL (Benar). 1 abad = 100 tahun, jadi pernyataannya salah.",
  },
  {
    t: "category",
    text: "Tentukan Benar atau Salah untuk setiap pernyataan tentang KPK dan FPB berikut.",
    opts: {
      A: "KPK dari 4 dan 6 adalah 12.",
      B: "FPB dari 12 dan 18 adalah 6.",
      C: "KPK dari 5 dan 7 adalah 35.",
      D: "FPB dari 8 dan 20 adalah 8.",
      E: "KPK dari 6 dan 9 adalah 18.",
    },
    labels: ["Benar", "Salah"],
    ans: "A=Benar;B=Benar;C=Benar;D=Salah;E=Benar",
    exp: "KPK 4 dan 6 = 12 (Benar). FPB 12 dan 18 = 6 (Benar). KPK 5 dan 7 = 35 karena keduanya prima (Benar). FPB 8 dan 20 = 4, bukan 8 (Salah). KPK 6 dan 9 = 18 (Benar).",
  },

  // ================= ESAI (essay) 5 =================
  {
    t: "essay",
    text: "Hitunglah hasil dari 1.250 + 2.475 - 1.325. Tulislah langkah pengerjaanmu.",
    ans: "1.250 + 2.475 = 3.725, kemudian 3.725 - 1.325 = 2.400. Jadi hasilnya 2.400.",
    exp: "Kerjakan operasi penjumlahan lebih dahulu, lalu pengurangan: 1.250 + 2.475 = 3.725 dan 3.725 - 1.325 = 2.400.",
  },
  {
    t: "essay",
    text: "Ibu membeli 3 kg apel dengan harga Rp18.000 per kg dan 2 kg jeruk dengan harga Rp15.000 per kg. Berapa total uang yang harus dibayar Ibu?",
    ans: "Apel: 3 x 18.000 = 54.000. Jeruk: 2 x 15.000 = 30.000. Total = 54.000 + 30.000 = 84.000. Jadi Ibu harus membayar Rp84.000.",
    exp: "Hitung harga tiap buah terlebih dahulu, lalu jumlahkan: (3 x 18.000) + (2 x 15.000) = 54.000 + 30.000 = 84.000.",
  },
  {
    t: "essay",
    text: "Sebuah kolam berbentuk balok memiliki panjang 8 m, lebar 5 m, dan tinggi 2 m. Hitunglah volume kolam tersebut.",
    ans: "Volume = panjang x lebar x tinggi = 8 x 5 x 2 = 80 m3. Jadi volume kolam adalah 80 m3.",
    exp: "Volume balok = panjang x lebar x tinggi = 8 x 5 x 2 = 80 m3.",
  },
  {
    t: "essay",
    text: "Nilai ulangan Matematika Rina adalah 80, 75, 90, 85, dan 70. Hitunglah rata-rata nilai ulangan Rina.",
    ans: "Jumlah nilai = 80 + 75 + 90 + 85 + 70 = 400. Banyak data = 5. Rata-rata = 400 : 5 = 80. Jadi rata-rata nilainya 80.",
    exp: "Rata-rata = jumlah seluruh data : banyak data = 400 : 5 = 80.",
  },
  {
    t: "essay",
    text: "Sebuah mobil menempuh jarak 180 km dalam waktu 3 jam. Berapa kecepatan rata-rata mobil tersebut?",
    ans: "Kecepatan = jarak : waktu = 180 : 3 = 60 km/jam. Jadi kecepatan rata-rata mobil 60 km/jam.",
    exp: "Kecepatan = jarak : waktu = 180 km : 3 jam = 60 km/jam.",
  },
];

const GUIDE_ROWS = [
  "TEMPLATE BANK SOAL TKA",
  "Isi data pada sheet Bank Soal. Ujian dan mata pelajaran dipilih saat paket dibuat di aplikasi.",
  "PG Sederhana: satu kunci, contoh B.",
  "PGK MCMA: minimal dua kunci dipisahkan koma, contoh A,C.",
  "PGK Kategori: seluruh pernyataan diberi kategori, contoh A=Benar;B=Salah;C=Benar.",
  "Esai: correct_answer diisi jawaban referensi, option dibiarkan kosong.",
  "Setiap baris adalah satu soal mandiri. Masukkan stimulus langsung pada question_text.",
  "Gunakan nilai kode pada sheet Referensi agar data konsisten.",
];

const REF_ROWS = [
  ["Field", "Nilai yang diizinkan", "Keterangan"],
  ["question_type", "single_choice | multiple_choice | category | essay", "Empat bentuk soal TKA"],
  ["category_labels", "Benar|Salah", "Pisahkan kategori dengan tanda |"],
  ["correct_answer", "B / A,C / A=Benar;B=Salah", "Sintaks mengikuti bentuk soal"],
  ["status", "active | inactive", "Status publikasi"],
];

function buildWorkbook() {
  const wb = new ExcelJS.Workbook();
  wb.creator = "TKA";
  wb.created = new Date();

  const guideSheet = wb.addWorksheet("Panduan", { properties: { tabColor: { argb: "FF4472C4" } } });
  GUIDE_ROWS.forEach((text, rowIdx) => {
    const row = guideSheet.addRow([text]);
    if (rowIdx === 0) row.getCell(1).font = { bold: true, size: 16 };
  });
  guideSheet.getColumn(1).width = 110;

  const dataSheet = wb.addWorksheet("Bank Soal", { views: [{ state: "frozen", ySplit: 1 }] });
  dataSheet.columns = [
    { header: "no", key: "no", width: 8 },
    { header: "question_type", key: "question_type", width: 18 },
    { header: "chapter_name", key: "chapter_name", width: 24 },
    { header: "question_text", key: "question_text", width: 45 },
    { header: "question_image_url", key: "question_image_url", width: 24 },
    { header: "option_a", key: "option_a", width: 30 }, { header: "option_a_image_url", key: "option_a_image_url", width: 24 },
    { header: "option_b", key: "option_b", width: 30 }, { header: "option_b_image_url", key: "option_b_image_url", width: 24 },
    { header: "option_c", key: "option_c", width: 30 }, { header: "option_c_image_url", key: "option_c_image_url", width: 24 },
    { header: "option_d", key: "option_d", width: 30 }, { header: "option_d_image_url", key: "option_d_image_url", width: 24 },
    { header: "option_e", key: "option_e", width: 30 }, { header: "option_e_image_url", key: "option_e_image_url", width: 24 },
    { header: "category_labels", key: "category_labels", width: 20 },
    { header: "correct_answer", key: "correct_answer", width: 28 },
    { header: "score_weight", key: "score_weight", width: 13 },
    { header: "explanation", key: "explanation", width: 55 },
    { header: "status", key: "status", width: 12 },
  ];
  const headerRow = dataSheet.getRow(1);
  headerRow.font = { bold: true, color: { argb: "FFFFFFFF" } };
  headerRow.fill = { type: "pattern", pattern: "solid", fgColor: { argb: "FF4472C4" } };
  headerRow.alignment = { horizontal: "center", vertical: "middle" };
  headerRow.height = 22;

  QUESTIONS.forEach((item, index) => {
    const row = {
      no: index + 1,
      question_type: item.t,
      chapter_name: item.chapter ?? chapterForQuestion(index),
      question_text: item.text,
      question_image_url: "",
      category_labels: item.labels ? item.labels.join("|") : "",
      correct_answer: item.ans,
      score_weight: 1,
      explanation: item.exp,
      status: "active",
    };
    OPTION_KEYS.forEach((key) => {
      row[`option_${key.toLowerCase()}`] = item.opts && item.opts[key] ? item.opts[key] : "";
      row[`option_${key.toLowerCase()}_image_url`] = "";
    });
    dataSheet.addRow(row);
  });
  dataSheet.eachRow((row, n) => {
    if (n === 1) return;
    row.alignment = { vertical: "top", wrapText: true };
  });

  const refSheet = wb.addWorksheet("Referensi", { properties: { tabColor: { argb: "FF70AD47" } } });
  REF_ROWS.forEach((row) => refSheet.addRow(row));
  refSheet.getRow(1).font = { bold: true, color: { argb: "FFFFFFFF" } };
  refSheet.getRow(1).fill = { type: "pattern", pattern: "solid", fgColor: { argb: "FF70AD47" } };
  refSheet.getColumn(1).width = 24;
  refSheet.getColumn(2).width = 48;
  refSheet.getColumn(3).width = 48;

  return wb;
}

export { QUESTIONS, buildWorkbook };

async function main() {
  const out = process.argv[2] || "Bank-Soal-Matematika-SD.xlsx";
  const wb = buildWorkbook();
  await wb.xlsx.writeFile(out);
  const counts = QUESTIONS.reduce((acc, q) => {
    acc[q.t] = (acc[q.t] || 0) + 1;
    return acc;
  }, {});
  console.log(`wrote: ${path.resolve(out)}`);
  console.log(`total soal: ${QUESTIONS.length}`, counts);
}

if (process.argv[1] && path.resolve(process.argv[1]) === fileURLToPath(import.meta.url)) {
  await main();
}
