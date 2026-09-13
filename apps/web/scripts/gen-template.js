const ExcelJS = require("exceljs");
const path = require("path");

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

const DATA_HEADERS = [
  ["no", 11],
  ["question_type", 18],
  ["question_text", 30],
  ["question_image_url", 30],
  ["option_a", 28], ["option_a_image_url", 28],
  ["option_b", 28], ["option_b_image_url", 28],
  ["option_c", 28], ["option_c_image_url", 28],
  ["option_d", 28], ["option_d_image_url", 28],
  ["option_e", 28], ["option_e_image_url", 28],
  ["category_labels", 22],
  ["correct_answer", 18],
  ["score_weight", 13],
  ["explanation", 30],
  ["status", 12],
];

const REF_ROWS = [
  ["Field", "Nilai yang diizinkan", "Keterangan"],
  ["question_type", "single_choice | multiple_choice | category | essay", "Empat bentuk soal TKA"],
  ["category_labels", "Benar|Salah", "Pisahkan kategori dengan tanda |"],
  ["correct_answer", "B / A,C / A=Benar;B=Salah", "Sintaks mengikuti bentuk soal"],
  ["status", "active | inactive", "Status publikasi"],
];

async function main() {
  const wb = new ExcelJS.Workbook();
  wb.creator = "TKA";

  const guideSheet = wb.addWorksheet("Panduan", { properties: { tabColor: { argb: "4472C4" } } });
  GUIDE_ROWS.forEach((text, idx) => {
    const row = guideSheet.addRow([text]);
    if (idx === 0) row.getCell(1).font = { bold: true, size: 16 };
  });
  guideSheet.getColumn(1).width = 110;

  const dataSheet = wb.addWorksheet("Bank Soal", { views: [{ state: "frozen", ySplit: 1 }] });
  const headerRow = dataSheet.addRow(DATA_HEADERS.map(([h]) => h));
  headerRow.font = { bold: true, color: { argb: "FFFFFF" } };
  headerRow.fill = { type: "pattern", pattern: "solid", fgColor: { argb: "4472C4" } };
  headerRow.alignment = { horizontal: "center" };
  DATA_HEADERS.forEach(([, width], idx) => { dataSheet.getColumn(idx + 1).width = width; });

  const refSheet = wb.addWorksheet("Referensi", { properties: { tabColor: { argb: "70AD47" } } });
  REF_ROWS.forEach((row) => refSheet.addRow(row));
  refSheet.getRow(1).font = { bold: true, color: { argb: "FFFFFF" } };
  refSheet.getRow(1).fill = { type: "pattern", pattern: "solid", fgColor: { argb: "70AD47" } };
  refSheet.getColumn(1).width = 24;
  refSheet.getColumn(2).width = 48;
  refSheet.getColumn(3).width = 48;

  const out = path.join(__dirname, "..", "public", "templates", "template-bank-soal-tka.xlsx");
  await wb.xlsx.writeFile(out);
  console.log("wrote", out);
}

main().catch((err) => { console.error(err); process.exit(1); });