[CmdletBinding()]
param(
    [string]$BackupDir = (Join-Path $env:USERPROFILE "tka-backups"),
    [int]$RetentionDays = 14
)

$ErrorActionPreference = "Stop"
$projectRoot = Split-Path -Parent $PSScriptRoot
Set-Location -LiteralPath $projectRoot

# Backup PostgreSQL lokal TKA untuk pengembangan di Windows.
# Dump format custom + terkompresi, divalidasi dengan pg_restore --list.
# Simpan di drive sehat (default C:) agar tidak bergantung pada disk data Docker.

if (-not (Get-Command docker -ErrorAction SilentlyContinue)) {
    throw "Docker belum terpasang. Instal dan jalankan Docker Desktop terlebih dahulu."
}

docker info *> $null
if ($LASTEXITCODE -ne 0) {
    throw "Docker Desktop belum aktif. Buka Docker Desktop, lalu jalankan ulang skrip ini."
}

$dbUser = "tka"
$dbName = "tka"
$envFile = Join-Path $projectRoot ".env"
if (Test-Path -LiteralPath $envFile) {
    foreach ($line in Get-Content -LiteralPath $envFile) {
        if ($line -match '^\s*POSTGRES_USER\s*=\s*(.+?)\s*$') { $dbUser = $Matches[1].Trim('"').Trim("'") }
        if ($line -match '^\s*POSTGRES_DB\s*=\s*(.+?)\s*$') { $dbName = $Matches[1].Trim('"').Trim("'") }
    }
}

New-Item -ItemType Directory -Force -Path $BackupDir | Out-Null

$timestamp = (Get-Date).ToUniversalTime().ToString("yyyyMMddTHHmmssZ")
$backupName = "tka-$timestamp.dump"
$backupFile = Join-Path $BackupDir $backupName
$partialFile = Join-Path $BackupDir ".$backupName.partial"
$errorFile = Join-Path $BackupDir ".$backupName.err"

Write-Host "[$(Get-Date -Format o)] Mulai backup PostgreSQL lokal -> $backupFile"

$dumpCommand = "docker compose exec -T postgres pg_dump " +
    "--username=`"$dbUser`" --dbname=`"$dbName`" " +
    "--format=custom --compress=zstd:9 --no-owner --no-privileges"

try {
    cmd /c "$dumpCommand > `"$partialFile`" 2> `"$errorFile`""
    if ($LASTEXITCODE -ne 0) {
        $detail = if (Test-Path -LiteralPath $errorFile) { Get-Content -LiteralPath $errorFile -Raw } else { "" }
        throw "pg_dump gagal (exit $LASTEXITCODE). $detail"
    }

    if (-not (Test-Path -LiteralPath $partialFile) -or (Get-Item -LiteralPath $partialFile).Length -eq 0) {
        throw "Dump kosong atau tidak terbentuk."
    }

    cmd /c "docker compose exec -T postgres pg_restore --list < `"$partialFile`" > nul 2> `"$errorFile`""
    if ($LASTEXITCODE -ne 0) {
        $detail = if (Test-Path -LiteralPath $errorFile) { Get-Content -LiteralPath $errorFile -Raw } else { "" }
        throw "Validasi pg_restore gagal (exit $LASTEXITCODE). $detail"
    }

    Move-Item -LiteralPath $partialFile -Destination $backupFile -Force
    $hash = (Get-FileHash -LiteralPath $backupFile -Algorithm SHA256).Hash
    "$hash  $backupName" | Out-File -LiteralPath "$backupFile.sha256" -Encoding ascii

    $cutoff = (Get-Date).AddDays(-$RetentionDays)
    Get-ChildItem -LiteralPath $BackupDir -Filter "tka-*.dump*" -File |
        Where-Object { $_.LastWriteTime -lt $cutoff } |
        Remove-Item -Force -ErrorAction SilentlyContinue

    $sizeMb = [math]::Round((Get-Item -LiteralPath $backupFile).Length / 1MB, 2)
    Write-Host "[$(Get-Date -Format o)] Backup OK: $backupFile ($sizeMb MB)"
} finally {
    Remove-Item -LiteralPath $partialFile, $errorFile -Force -ErrorAction SilentlyContinue
}
