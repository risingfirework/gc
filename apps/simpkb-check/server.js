"use strict";

const http = require("http");
const { chromium } = require("playwright");

const PORT = Number(process.env.PORT || 8080);
const SEARCH_URL = process.env.SEARCH_URL || "https://portal.simpkb.id/cari/";
const RESULT_WAIT_MS = 16000;
const TYPING_WAIT_MS = 1500;

let browser = null;

async function getBrowser() {
  if (browser && browser.isConnected()) return browser;
  browser = await chromium.launch({
    headless: true,
    args: ["--no-sandbox", "--disable-dev-shm-usage", "--disable-gpu"],
  });
  return browser;
}

// Interpretasi sederhana halaman hasil pencarian SIMPKB.
function interpretResult(pageText) {
  const normalized = (pageText || "").replace(/\s+/g, " ").toLowerCase();
  if (/ditemukan\s+0\s+data|0\s+data\s+gtk|tidak ditemukan|tidak ada data|tidak ada hasil/.test(normalized)) {
    return { status: "not_found", message: "NIK tidak ditemukan pada portal SIMPKB." };
  }
  if (/ditemukan\s+[1-9]\d*\s+data/.test(normalized)) {
    return { status: "found", message: "NIK ditemukan pada data SIMPKB." };
  }
  if (/data dikerjakan|kepribadian|anda harus login|login dulu/.test(normalized)) {
    return { status: "error", message: "Portal meminta autentikasi atau menampilkan halaman lain." };
  }
  const hasRow = /<tr/i.test(pageText) && /<td/i.test(pageText);
  if (hasRow) {
    return { status: "found", message: "Hasil pencarian SIMPKB ditemukan." };
  }
  return { status: "error", message: "Hasil pencarian tidak dapat dibaca otomatis." };
}

// Buka portal dan tunggu kotak pencarian (aplikasi PrimeReact dirender lewat JS,
// jadi tunggu sampai input benar-benar muncul, bukan hanya domcontentloaded).
async function openSearchPage(page) {
  try {
    await page.goto(SEARCH_URL, { waitUntil: "networkidle", timeout: 30000 });
  } catch {
    await page.goto(SEARCH_URL, { waitUntil: "domcontentloaded", timeout: 20000 });
  }
  await page.waitForTimeout(1200);
}

// Isi kolom pencarian "Nama GTK / No. Peserta UKG / NUPTK / NIK" lalu tekan Cari.
async function fillAndSearch(page, nik) {
  const input = page.locator('input[type="text"]').first();
  await input.waitFor({ state: "visible", timeout: 20000 });
  await input.click();
  await input.fill(nik);
  await input.press("Control+a");
  await input.type(nik, { delay: 50 });
  await page.waitForTimeout(TYPING_WAIT_MS);
  const searchButton = page
    .locator("button:has-text('CARI GTK'), button:has-text('Cari'), button:has-text('cari')")
    .first();
  if (await searchButton.count()) {
    try {
      await searchButton.click({ timeout: 4000 });
    } catch {
      await input.press("Enter");
    }
  } else {
    await input.press("Enter");
  }
}

async function runCheck(nik) {
  const startedAt = Date.now();
  const context = await (await getBrowser()).newContext({
    locale: "id-ID",
    userAgent:
      "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/122.0 Safari/537.36",
  });
  const page = await context.newPage();
  let screenshot = "";
  let lastUrl = SEARCH_URL;
  let pageText = "";
  let status = "error";
  let message = "";

  try {
    await openSearchPage(page);
    await fillAndSearch(page, nik);
    await page.waitForTimeout(RESULT_WAIT_MS);
    lastUrl = page.url();
    pageText = await page.locator("body").innerHTML().catch(() => "");
    const interpretation = interpretResult(pageText);
    status = interpretation.status;
    message = interpretation.message;
  } catch (reason) {
    status = "error";
    message = "Gagal memuat portal SIMPKB: " + String(reason && reason.message ? reason.message : reason).slice(0, 240);
  } finally {
    screenshot = await page.screenshot({ fullPage: true })
      .then((buffer) => "data:image/png;base64," + buffer.toString("base64"))
      .catch(() => "");
    await context.close();
  }
  return {
    nik,
    status,
    message,
    screenshot,
    url: lastUrl,
    took_ms: Date.now() - startedAt,
  };
}

let queue = Promise.resolve();

// Satu browser dipakai semua request agar cold start tidak berulang; setiap
// permintaan diproses bergiliran agar screenshot tidak saling menimpa.
function enqueue(nik, callback) {
  queue = queue.then(() => callback()).catch(() => {});
  return queue;
}

const server = http.createServer((req, res) => {
  const respond = (code, payload) => {
    res.writeHead(code, { "Content-Type": "application/json; charset=utf-8" });
    res.end(JSON.stringify(payload));
  };

  if (req.method === "GET" && req.url === "/health") {
    respond(200, { ok: true });
    return;
  }

  if (req.method !== "POST" || req.url !== "/check") {
    respond(404, { error: "not found" });
    return;
  }

  let raw = "";
  req.on("data", (chunk) => (raw += chunk));
  req.on("end", () => {
    let nik = "";
    try {
      nik = String((JSON.parse(raw) || {}).nik || "").trim();
    } catch { /* body tidak valid */ }
    if (!/^\d{16}$/.test(nik)) {
      respond(400, { error: "NIK must be exactly 16 digits" });
      return;
    }
    enqueue(nik, async () => {
      try {
        respond(200, await runCheck(nik));
      } catch (reason) {
        respond(500, {
          status: "error",
          message: "Gagal menjalankan pengecekan: " + String(reason && reason.message ? reason.message : reason).slice(0, 240),
          screenshot: "",
        });
      }
    });
  });
});

server.listen(PORT, () => {
  console.log("simpkb-check listening on :" + PORT);
  getBrowser().catch((reason) => console.error("browser launch failed: " + reason.message));
});