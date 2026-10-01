[CmdletBinding()]
param(
    [string]$Time = "22:00"
)

$ErrorActionPreference = "Stop"
$projectRoot = Split-Path -Parent $PSScriptRoot
$backupScript = Join-Path $PSScriptRoot "backup-db-local.ps1"
$taskName = "TKA Local - Backup DB"
$currentUser = [System.Security.Principal.WindowsIdentity]::GetCurrent().Name

if (-not (Test-Path -LiteralPath $backupScript)) {
    throw "Skrip backup tidak ditemukan: $backupScript"
}

$action = New-ScheduledTaskAction `
    -Execute "powershell.exe" `
    -Argument "-NoLogo -NoProfile -NonInteractive -WindowStyle Hidden -ExecutionPolicy Bypass -File `"$backupScript`""

$trigger = New-ScheduledTaskTrigger -Daily -At $Time

$settings = New-ScheduledTaskSettingsSet `
    -AllowStartIfOnBatteries `
    -DontStopIfGoingOnBatteries `
    -StartWhenAvailable `
    -ExecutionTimeLimit (New-TimeSpan -Minutes 30)

$principal = New-ScheduledTaskPrincipal -UserId $currentUser -LogonType Interactive -RunLevel Limited

Register-ScheduledTask `
    -TaskName $taskName `
    -Action $action `
    -Trigger $trigger `
    -Settings $settings `
    -Principal $principal `
    -Description "Backup harian PostgreSQL TKA Local ke drive sehat (default C:)." `
    -Force | Out-Null

$backupDir = Join-Path $env:USERPROFILE "tka-backups"
Write-Host "Backup harian TKA berhasil dipasang." -ForegroundColor Green
Write-Host "Task     : $taskName"
Write-Host "Jadwal   : setiap hari $Time"
Write-Host "Tujuan   : $backupDir"
Write-Host "Retensi  : 14 hari"
Write-Host "`nMenjalankan backup sekarang untuk verifikasi..." -ForegroundColor Cyan
Start-ScheduledTask -TaskName $taskName
