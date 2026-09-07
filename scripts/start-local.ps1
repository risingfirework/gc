[CmdletBinding()]
param(
    [switch]$NoBrowser
)

$ErrorActionPreference = "Stop"
$projectRoot = Split-Path -Parent $PSScriptRoot
Set-Location -LiteralPath $projectRoot

Write-Host "`nTKA Local - menyiapkan aplikasi..." -ForegroundColor Cyan

if (-not (Get-Command docker -ErrorAction SilentlyContinue)) {
    throw "Docker belum terpasang. Instal dan jalankan Docker Desktop terlebih dahulu."
}

docker info *> $null
if ($LASTEXITCODE -ne 0) {
    throw "Docker Desktop belum aktif. Buka Docker Desktop, tunggu sampai siap, lalu jalankan kembali MULAI-TKA.cmd."
}

if (-not (Test-Path -LiteralPath ".env")) {
    Copy-Item -LiteralPath ".env.example" -Destination ".env"
    Write-Host "File .env lokal dibuat otomatis." -ForegroundColor DarkGray
}

Write-Host "Menjalankan database, migrasi, seed, API, dan web..." -ForegroundColor Yellow
docker compose up -d --build --force-recreate
if ($LASTEXITCODE -ne 0) {
    throw "Docker Compose gagal menjalankan aplikasi."
}

$url = "http://localhost:3000"
$deadline = (Get-Date).AddMinutes(5)
$ready = $false

while ((Get-Date) -lt $deadline) {
    try {
        $response = Invoke-WebRequest -Uri $url -UseBasicParsing -TimeoutSec 3
        if ($response.StatusCode -ge 200 -and $response.StatusCode -lt 500) {
            $ready = $true
            break
        }
    } catch {
        Start-Sleep -Seconds 2
    }
}

if (-not $ready) {
    Write-Host "`nStatus container:" -ForegroundColor Yellow
    docker compose ps
    Write-Host "`nLog terakhir:" -ForegroundColor Yellow
    docker compose logs --tail 80 backend web migrate seed
    throw "Aplikasi belum siap setelah 5 menit. Periksa log di atas."
}

Write-Host "`nTKA sudah siap." -ForegroundColor Green
Write-Host "Alamat   : $url"
$lanAddresses = Get-NetIPAddress -AddressFamily IPv4 -ErrorAction SilentlyContinue |
    Where-Object {
        $_.IPAddress -notlike "127.*" -and
        $_.IPAddress -notlike "169.254.*" -and
        $_.InterfaceAlias -notmatch "vEthernet|Loopback"
    } |
    Select-Object -ExpandProperty IPAddress -Unique
foreach ($lanAddress in $lanAddresses) {
    Write-Host "Jaringan : http://${lanAddress}:3000"
}
Write-Host "Login    : siswa.sma1@tka.local"
Write-Host "Password : 12345678"
Write-Host "API      : diteruskan otomatis melalui $url/api/v1`n"

if (-not $NoBrowser) {
    Start-Process $url
}
