$ErrorActionPreference = 'Continue'
$ProjectWslPath = '/mnt/e/ppp/pansou'

Write-Host '=== PanSou V1 diagnostics ===' -ForegroundColor Cyan
Write-Host ('Timestamp: ' + (Get-Date -Format o))
Write-Host ('PowerShell: ' + $PSVersionTable.PSVersion)

Write-Host "`n--- WSL distributions ---"
wsl.exe --list --verbose

Write-Host "`n--- Port 8888 listeners ---"
Get-NetTCPConnection -State Listen -LocalPort 8888 -ErrorAction SilentlyContinue |
    Select-Object LocalAddress, LocalPort, OwningProcess |
    Format-Table -AutoSize

Write-Host "`n--- HTTP checks ---"
foreach ($target in @('http://127.0.0.1:8888/', 'http://127.0.0.1:8888/api/health')) {
    try {
        $watch = [Diagnostics.Stopwatch]::StartNew()
        $response = Invoke-WebRequest -Uri $target -UseBasicParsing -TimeoutSec 8
        $watch.Stop()
        Write-Host ("{0} status={1} elapsed_ms={2}" -f $target, $response.StatusCode, $watch.ElapsedMilliseconds)
    } catch {
        Write-Host ("{0} error={1}" -f $target, $_.Exception.Message) -ForegroundColor Red
    }
}

Write-Host "`n--- Docker Compose status ---"
wsl.exe -d Ubuntu -- bash -lc "cd '$ProjectWslPath' && docker compose ps"

Write-Host "`n--- Container health ---"
wsl.exe -d Ubuntu -- bash -lc "docker inspect pansou --format 'status={{.State.Status}} health={{if .State.Health}}{{.State.Health.Status}}{{else}}none{{end}} restart={{.HostConfig.RestartPolicy.Name}}' 2>/dev/null || echo 'container not found'"

Write-Host "`n--- Recent logs ---"
wsl.exe -d Ubuntu -- bash -lc "cd '$ProjectWslPath' && docker compose logs --tail=80 pansou"
