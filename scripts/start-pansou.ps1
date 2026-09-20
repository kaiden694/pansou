param(
    [switch]$NoBrowser,
    [int]$TimeoutSeconds = 90
)

$ErrorActionPreference = 'Stop'
$ProjectWslPath = '/mnt/e/ppp/pansou'
$Url = 'http://127.0.0.1:8888/'
$HealthUrl = 'http://127.0.0.1:8888/api/health'

function Test-Http {
    param([string]$Target)
    try {
        $response = Invoke-WebRequest -Uri $Target -UseBasicParsing -TimeoutSec 5
        return $response.StatusCode -ge 200 -and $response.StatusCode -lt 400
    } catch {
        return $false
    }
}

Write-Host '[PanSou] Starting Ubuntu and Docker...' -ForegroundColor Cyan
wsl.exe -d Ubuntu -- bash -lc 'sudo systemctl start docker >/dev/null 2>&1 || true'

Write-Host '[PanSou] Starting containers...' -ForegroundColor Cyan
wsl.exe -d Ubuntu -- bash -lc "cd '$ProjectWslPath' && docker compose up -d"
if ($LASTEXITCODE -ne 0) {
    throw 'docker compose up failed. Run scripts/diagnose-pansou.ps1 for details.'
}

Write-Host '[PanSou] Waiting for the health endpoint...' -ForegroundColor Cyan
$deadline = (Get-Date).AddSeconds($TimeoutSeconds)
do {
    if (Test-Http -Target $HealthUrl) {
        Write-Host '[PanSou] Ready: http://127.0.0.1:8888/' -ForegroundColor Green
        if (-not $NoBrowser) {
            Start-Process -FilePath $Url
        }
        exit 0
    }
    Start-Sleep -Seconds 2
} while ((Get-Date) -lt $deadline)

Write-Host '[PanSou] Startup timed out. Collecting diagnostics...' -ForegroundColor Yellow
& (Join-Path $PSScriptRoot 'diagnose-pansou.ps1')
exit 1
