[CmdletBinding()]
param()

$ErrorActionPreference = "Stop"
$projectRoot = Split-Path -Parent $PSScriptRoot
$logDirectory = Join-Path $projectRoot ".cache"
$logFile = Join-Path $logDirectory "autostart.log"

function Test-DockerReady {
    $previousErrorAction = $ErrorActionPreference
    try {
        $ErrorActionPreference = "SilentlyContinue"
        docker info *> $null
        return $LASTEXITCODE -eq 0
    } finally {
        $ErrorActionPreference = $previousErrorAction
    }
}

New-Item -ItemType Directory -Path $logDirectory -Force | Out-Null
Start-Transcript -Path $logFile -Append | Out-Null

try {
    Set-Location -LiteralPath $projectRoot

    if (-not (Get-Command docker -ErrorAction SilentlyContinue)) {
        throw "Docker CLI tidak ditemukan. Instal Docker Desktop terlebih dahulu."
    }

    $dockerReady = $false
    $dockerDesktopPaths = @(
        (Join-Path $env:ProgramFiles "Docker\Docker\Docker Desktop.exe"),
        (Join-Path $env:LOCALAPPDATA "Docker\Docker Desktop.exe")
    )

    if (Test-DockerReady) {
        $dockerReady = $true
    } else {
        $dockerDesktop = $dockerDesktopPaths | Where-Object { Test-Path -LiteralPath $_ } | Select-Object -First 1
        if ($dockerDesktop) {
            Start-Process -FilePath $dockerDesktop -WindowStyle Hidden
        }
    }

    $deadline = (Get-Date).AddMinutes(10)
    while (-not $dockerReady -and (Get-Date) -lt $deadline) {
        Start-Sleep -Seconds 5
        $dockerReady = Test-DockerReady
    }

    if (-not $dockerReady) {
        throw "Docker Desktop belum siap setelah 10 menit."
    }

    if (-not (Test-Path -LiteralPath ".env")) {
        Copy-Item -LiteralPath ".env.example" -Destination ".env"
    }

    docker compose up -d
    if ($LASTEXITCODE -ne 0) {
        throw "Docker Compose gagal menjalankan TKA."
    }

    Write-Output "$(Get-Date -Format s) - TKA aktif di http://localhost:3000"
} catch {
    Write-Error $_
    exit 1
} finally {
    Stop-Transcript | Out-Null
}
