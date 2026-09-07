[CmdletBinding()]
param()

$ErrorActionPreference = "Stop"
$projectRoot = Split-Path -Parent $PSScriptRoot
$startupScript = Join-Path $PSScriptRoot "start-on-login.ps1"
$taskName = "TKA Local - Auto Start"
$currentUser = [System.Security.Principal.WindowsIdentity]::GetCurrent().Name

$action = New-ScheduledTaskAction `
    -Execute "powershell.exe" `
    -Argument "-NoLogo -NoProfile -NonInteractive -WindowStyle Hidden -ExecutionPolicy Bypass -File `"$startupScript`""
$trigger = New-ScheduledTaskTrigger -AtLogOn -User $currentUser
$settings = New-ScheduledTaskSettingsSet `
    -AllowStartIfOnBatteries `
    -DontStopIfGoingOnBatteries `
    -StartWhenAvailable `
    -ExecutionTimeLimit (New-TimeSpan -Minutes 15)
$principal = New-ScheduledTaskPrincipal -UserId $currentUser -LogonType Interactive -RunLevel Limited

Register-ScheduledTask `
    -TaskName $taskName `
    -Action $action `
    -Trigger $trigger `
    -Settings $settings `
    -Principal $principal `
    -Description "Menjalankan aplikasi TKA Local otomatis saat pengguna masuk Windows." `
    -Force | Out-Null

Write-Host "TKA berhasil dipasang agar otomatis aktif saat login Windows." -ForegroundColor Green
Write-Host "Task     : $taskName"
Write-Host "Alamat   : http://localhost:3000"
Write-Host "Log      : $(Join-Path $projectRoot '.cache\autostart.log')"
Write-Host "`nMenjalankan task sekarang..." -ForegroundColor Cyan
Start-ScheduledTask -TaskName $taskName
