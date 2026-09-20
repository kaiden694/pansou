# PanSou V1 reliable startup

## One-click start

Double-click `Start-PanSou.cmd`, or run:

```powershell
powershell.exe -NoProfile -ExecutionPolicy Bypass -File .\scripts\start-pansou.ps1
```

The launcher starts WSL and Docker, runs Docker Compose, waits for `/api/health`, and opens PanSou only after it is ready.

## Diagnostics

```powershell
powershell.exe -NoProfile -ExecutionPolicy Bypass -File .\scripts\diagnose-pansou.ps1
```

The report includes WSL state, port listeners, HTTP checks, container health, and recent logs.

## Start automatically after Windows sign-in

```powershell
powershell.exe -NoProfile -ExecutionPolicy Bypass -File .\scripts\install-autostart.ps1
```

Remove the scheduled task with:

```powershell
powershell.exe -NoProfile -ExecutionPolicy Bypass -File .\scripts\install-autostart.ps1 -Remove
```
