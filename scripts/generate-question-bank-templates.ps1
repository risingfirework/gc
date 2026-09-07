param([string]$OutputDirectory = (Join-Path $PSScriptRoot "../apps/web/public/templates"))

$ErrorActionPreference = "Stop"
Add-Type -AssemblyName System.IO.Compression
Add-Type -AssemblyName System.IO.Compression.FileSystem
$output = [IO.Path]::GetFullPath($OutputDirectory)
[IO.Directory]::CreateDirectory($output) | Out-Null

function Add-ZipText([IO.Compression.ZipArchive]$Zip, [string]$Path, [string]$Content) {
    $entry = $Zip.CreateEntry($Path, [IO.Compression.CompressionLevel]::Optimal)
    $writer = [IO.StreamWriter]::new($entry.Open(), [Text.UTF8Encoding]::new($false))
    try { $writer.Write($Content) } finally { $writer.Dispose() }
}

function New-Package([string]$Path, [scriptblock]$Build) {
    if ([IO.File]::Exists($Path)) { [IO.File]::Delete($Path) }
    $stream = [IO.File]::Open($Path, [IO.FileMode]::CreateNew)
    $zip = [IO.Compression.ZipArchive]::new($stream, [IO.Compression.ZipArchiveMode]::Create)
    try { & $Build $zip } finally { $zip.Dispose(); $stream.Dispose() }
}

function Xml([string]$Value) { [Security.SecurityElement]::Escape($Value) }
function WordParagraph([string]$Text, [string]$Style = "Normal") {
    '<w:p><w:pPr><w:pStyle w:val="' + $Style + '"/></w:pPr><w:r><w:t xml:space="preserve">' + (Xml $Text) + '</w:t></w:r></w:p>'
}

$wordPath = Join-Path $output "template-bank-soal-tka.docx"
New-Package $wordPath {
    param($zip)
    Add-ZipText $zip "[Content_Types].xml" '<?xml version="1.0" encoding="UTF-8" standalone="yes"?><Types xmlns="http://schemas.openxmlformats.org/package/2006/content-types"><Default Extension="rels" ContentType="application/vnd.openxmlformats-package.relationships+xml"/><Default Extension="xml" ContentType="application/xml"/><Override PartName="/word/document.xml" ContentType="application/vnd.openxmlformats-officedocument.wordprocessingml.document.main+xml"/><Override PartName="/word/styles.xml" ContentType="application/vnd.openxmlformats-officedocument.wordprocessingml.styles+xml"/></Types>'
    Add-ZipText $zip "_rels/.rels" '<?xml version="1.0" encoding="UTF-8" standalone="yes"?><Relationships xmlns="http://schemas.openxmlformats.org/package/2006/relationships"><Relationship Id="rId1" Type="http://schemas.openxmlformats.org/officeDocument/2006/relationships/officeDocument" Target="word/document.xml"/></Relationships>'
    Add-ZipText $zip "word/_rels/document.xml.rels" '<?xml version="1.0" encoding="UTF-8" standalone="yes"?><Relationships xmlns="http://schemas.openxmlformats.org/package/2006/relationships"><Relationship Id="rId1" Type="http://schemas.openxmlformats.org/officeDocument/2006/relationships/styles" Target="styles.xml"/></Relationships>'
    Add-ZipText $zip "word/styles.xml" '<?xml version="1.0" encoding="UTF-8" standalone="yes"?><w:styles xmlns:w="http://schemas.openxmlformats.org/wordprocessingml/2006/main"><w:style w:type="paragraph" w:default="1" w:styleId="Normal"><w:name w:val="Normal"/><w:rPr><w:sz w:val="22"/><w:szCs w:val="22"/></w:rPr></w:style><w:style w:type="paragraph" w:styleId="Title"><w:name w:val="Title"/><w:rPr><w:b/><w:color w:val="173F8A"/><w:sz w:val="36"/></w:rPr></w:style><w:style w:type="paragraph" w:styleId="Heading1"><w:name w:val="heading 1"/><w:rPr><w:b/><w:color w:val="173F8A"/><w:sz w:val="28"/></w:rPr></w:style></w:styles>'
    $lines = @(
        @{T="TEMPLATE BANK SOAL TKA";S="Title"},
        @{T="Format terbaru: Pilihan Ganda Sederhana, PGK MCMA, dan PGK Kategori - satu soal per nomor";S="Normal"},
        @{T="PETUNJUK PENGISIAN";S="Heading1"},
        @{T="1. Isi satu blok untuk setiap nomor soal. Ujian dan mata pelajaran ditentukan saat membuat paket soal di aplikasi.";S="Normal"},
        @{T="2. Bentuk soal: PG Sederhana (tepat satu benar), PGK MCMA (minimal dua benar), atau PGK Kategori (setiap pernyataan wajib diberi kategori).";S="Normal"},
        @{T="3. Setiap nomor harus mandiri. Jika memakai stimulus, masukkan stimulus langsung pada Pertanyaan/Petunjuk nomor tersebut.";S="Normal"},
        @{T="4. Sintaks kunci: Sederhana = B | MCMA = A,C | Kategori = A=Benar;B=Salah;C=Benar";S="Normal"},
        @{T="BLOK SOAL - salin bagian ini sesuai kebutuhan";S="Heading1"},
        @{T="Nomor: [isi]";S="Normal"}, @{T="Bentuk: [PG Sederhana / PGK MCMA / PGK Kategori]";S="Normal"},
        @{T="Pertanyaan/Petunjuk: [isi]";S="Normal"}, @{T="Gambar pertanyaan: [tempel gambar di bawah baris ini]";S="Normal"}, @{T="A. [opsi atau pernyataan + gambar opsional]";S="Normal"}, @{T="B. [opsi atau pernyataan + gambar opsional]";S="Normal"}, @{T="C. [opsi atau pernyataan + gambar opsional]";S="Normal"}, @{T="D. [opsi atau pernyataan + gambar opsional]";S="Normal"},
        @{T="Kategori jawaban: [contoh Benar | Salah; hanya PGK Kategori]";S="Normal"}, @{T="Kunci: [B / A,C / A=Benar;B=Salah;C=Benar]";S="Normal"},
        @{T="Bobot: 1";S="Normal"}, @{T="Pembahasan: [isi]";S="Normal"}, @{T="Status: Aktif";S="Normal"},
        @{T="CONTOH KUNCI BERDASARKAN BENTUK SOAL";S="Heading1"},
        @{T="PG Sederhana | Kunci: B";S="Normal"}, @{T="PGK MCMA | Kunci: A,C";S="Normal"}, @{T="PGK Kategori | Kunci: A=Benar;B=Salah;C=Benar";S="Normal"}
    )
    $body = ($lines | ForEach-Object { WordParagraph $_.T $_.S }) -join ""
    Add-ZipText $zip "word/document.xml" ('<?xml version="1.0" encoding="UTF-8" standalone="yes"?><w:document xmlns:w="http://schemas.openxmlformats.org/wordprocessingml/2006/main"><w:body>' + $body + '<w:sectPr><w:pgSz w:w="11906" w:h="16838"/><w:pgMar w:top="1134" w:right="1134" w:bottom="1134" w:left="1134"/></w:sectPr></w:body></w:document>')
}

function Cell([string]$Reference, [string]$Value, [int]$Style = 0) { '<c r="' + $Reference + '" t="inlineStr" s="' + $Style + '"><is><t xml:space="preserve">' + (Xml $Value) + '</t></is></c>' }
function Row([int]$Number, [string[]]$Values, [int]$Style = 0) {
    $cells = for ($index=0; $index -lt $Values.Count; $index++) { Cell (([char](65+$index)).ToString()+$Number) $Values[$index] $Style }
    '<row r="' + $Number + '">' + ($cells -join '') + '</row>'
}

$excelPath = Join-Path $output "template-bank-soal-tka.xlsx"
New-Package $excelPath {
    param($zip)
    Add-ZipText $zip "[Content_Types].xml" '<?xml version="1.0" encoding="UTF-8" standalone="yes"?><Types xmlns="http://schemas.openxmlformats.org/package/2006/content-types"><Default Extension="rels" ContentType="application/vnd.openxmlformats-package.relationships+xml"/><Default Extension="xml" ContentType="application/xml"/><Override PartName="/xl/workbook.xml" ContentType="application/vnd.openxmlformats-officedocument.spreadsheetml.sheet.main+xml"/><Override PartName="/xl/styles.xml" ContentType="application/vnd.openxmlformats-officedocument.spreadsheetml.styles+xml"/><Override PartName="/xl/worksheets/sheet1.xml" ContentType="application/vnd.openxmlformats-officedocument.spreadsheetml.worksheet+xml"/><Override PartName="/xl/worksheets/sheet2.xml" ContentType="application/vnd.openxmlformats-officedocument.spreadsheetml.worksheet+xml"/><Override PartName="/xl/worksheets/sheet3.xml" ContentType="application/vnd.openxmlformats-officedocument.spreadsheetml.worksheet+xml"/></Types>'
    Add-ZipText $zip "_rels/.rels" '<?xml version="1.0" encoding="UTF-8" standalone="yes"?><Relationships xmlns="http://schemas.openxmlformats.org/package/2006/relationships"><Relationship Id="rId1" Type="http://schemas.openxmlformats.org/officeDocument/2006/relationships/officeDocument" Target="xl/workbook.xml"/></Relationships>'
    Add-ZipText $zip "xl/workbook.xml" '<?xml version="1.0" encoding="UTF-8" standalone="yes"?><workbook xmlns="http://schemas.openxmlformats.org/spreadsheetml/2006/main" xmlns:r="http://schemas.openxmlformats.org/officeDocument/2006/relationships"><sheets><sheet name="Petunjuk" sheetId="1" r:id="rId1"/><sheet name="Bank Soal" sheetId="2" r:id="rId2"/><sheet name="Referensi" sheetId="3" r:id="rId3"/></sheets></workbook>'
    Add-ZipText $zip "xl/_rels/workbook.xml.rels" '<?xml version="1.0" encoding="UTF-8" standalone="yes"?><Relationships xmlns="http://schemas.openxmlformats.org/package/2006/relationships"><Relationship Id="rId1" Type="http://schemas.openxmlformats.org/officeDocument/2006/relationships/worksheet" Target="worksheets/sheet1.xml"/><Relationship Id="rId2" Type="http://schemas.openxmlformats.org/officeDocument/2006/relationships/worksheet" Target="worksheets/sheet2.xml"/><Relationship Id="rId3" Type="http://schemas.openxmlformats.org/officeDocument/2006/relationships/worksheet" Target="worksheets/sheet3.xml"/><Relationship Id="rId4" Type="http://schemas.openxmlformats.org/officeDocument/2006/relationships/styles" Target="styles.xml"/></Relationships>'
    Add-ZipText $zip "xl/styles.xml" '<?xml version="1.0" encoding="UTF-8" standalone="yes"?><styleSheet xmlns="http://schemas.openxmlformats.org/spreadsheetml/2006/main"><fonts count="2"><font><sz val="11"/><name val="Calibri"/></font><font><b/><color rgb="FFFFFFFF"/><sz val="11"/><name val="Calibri"/></font></fonts><fills count="3"><fill><patternFill patternType="none"/></fill><fill><patternFill patternType="gray125"/></fill><fill><patternFill patternType="solid"><fgColor rgb="FF173F8A"/><bgColor indexed="64"/></patternFill></fill></fills><borders count="1"><border/></borders><cellStyleXfs count="1"><xf/></cellStyleXfs><cellXfs count="2"><xf fontId="0" fillId="0" borderId="0" xfId="0"/><xf fontId="1" fillId="2" borderId="0" xfId="0" applyFont="1" applyFill="1"><alignment wrapText="1"/></xf></cellXfs></styleSheet>'
    $instructions = @(
        (Row 1 @("TEMPLATE BANK SOAL TKA") 1),
        (Row 2 @("Isi data pada sheet Bank Soal. Ujian dan mata pelajaran dipilih saat paket dibuat di aplikasi.")),
        (Row 3 @("PG Sederhana: satu kunci, contoh B.")),
        (Row 4 @("PGK MCMA: minimal dua kunci dipisahkan koma, contoh A,C.")),
        (Row 5 @("PGK Kategori: seluruh pernyataan diberi kategori, contoh A=Benar;B=Salah;C=Benar.")),
        (Row 6 @("Setiap baris adalah satu soal mandiri. Masukkan stimulus langsung pada question_text.")),
        (Row 7 @("Gunakan nilai kode pada sheet Referensi agar data konsisten."))
    ) -join ""
    Add-ZipText $zip "xl/worksheets/sheet1.xml" ('<?xml version="1.0" encoding="UTF-8" standalone="yes"?><worksheet xmlns="http://schemas.openxmlformats.org/spreadsheetml/2006/main"><cols><col min="1" max="1" width="110" customWidth="1"/></cols><sheetData>'+$instructions+'</sheetData></worksheet>')
    $headers = @("no","question_type","question_text","question_image_url","option_a","option_a_image_url","option_b","option_b_image_url","option_c","option_c_image_url","option_d","option_d_image_url","option_e","option_e_image_url","category_labels","correct_answer","score_weight","explanation","status")
    $samples = @(
        @("1","single_choice","Hasil 12 + 8 adalah ...","","18","","20","","22","","24","","","","","B","1","12 + 8 = 20","active"),
        @("2","multiple_choice","Pilih semua bilangan prima.","","2","","4","","5","","9","","","","","A,C","1","2 dan 5 adalah prima.","active"),
        @("3","category","Tentukan Benar atau Salah.","","Pernyataan pertama","","Pernyataan kedua","","Pernyataan ketiga","","","","","","Benar|Salah","A=Benar;B=Salah;C=Benar","1","Uraikan alasan setiap pernyataan.","active")
    )
    $rows = Row 1 $headers 1
    for ($index=0; $index -lt $samples.Count; $index++) { $rows += Row ($index+2) $samples[$index] }
    $validations = '<dataValidations count="2"><dataValidation type="list" allowBlank="0" sqref="B2:B1000"><formula1>"single_choice,multiple_choice,category"</formula1></dataValidation><dataValidation type="list" allowBlank="0" sqref="S2:S1000"><formula1>"active,inactive"</formula1></dataValidation></dataValidations>'
    Add-ZipText $zip "xl/worksheets/sheet2.xml" ('<?xml version="1.0" encoding="UTF-8" standalone="yes"?><worksheet xmlns="http://schemas.openxmlformats.org/spreadsheetml/2006/main"><sheetViews><sheetView workbookViewId="0"><pane ySplit="1" topLeftCell="A2" activePane="bottomLeft" state="frozen"/></sheetView></sheetViews><cols><col min="1" max="2" width="18" customWidth="1"/><col min="3" max="19" width="28" customWidth="1"/></cols><sheetData>'+$rows+'</sheetData><autoFilter ref="A1:S1000"/>'+$validations+'</worksheet>')
    $referenceRows = (Row 1 @("Field","Nilai yang diizinkan","Keterangan") 1) + (Row 2 @("question_type","single_choice | multiple_choice | category","Tiga bentuk soal TKA")) + (Row 3 @("category_labels","Benar|Salah","Pisahkan kategori dengan tanda |")) + (Row 4 @("correct_answer","B / A,C / A=Benar;B=Salah","Sintaks mengikuti bentuk soal")) + (Row 5 @("status","active | inactive","Status publikasi"))
    Add-ZipText $zip "xl/worksheets/sheet3.xml" ('<?xml version="1.0" encoding="UTF-8" standalone="yes"?><worksheet xmlns="http://schemas.openxmlformats.org/spreadsheetml/2006/main"><cols><col min="1" max="1" width="24" customWidth="1"/><col min="2" max="3" width="48" customWidth="1"/></cols><sheetData>'+$referenceRows+'</sheetData></worksheet>')
}

Write-Host "Generated $wordPath"
Write-Host "Generated $excelPath"
