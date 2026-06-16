param(
    [int]$Port = 8080,
    [string]$BaseUrl = "",
    [switch]$StopFirst
)

$ErrorActionPreference = "Stop"
$RepoRoot = Split-Path -Parent $PSScriptRoot
Set-Location $RepoRoot

if (-not (Test-Path ".env")) {
    throw ".env not found. Copy .env.example to .env first."
}

if (-not (Test-Path "data")) {
    New-Item -ItemType Directory -Path "data" | Out-Null
}

$goCmd = "go"
if (-not (Get-Command $goCmd -ErrorAction SilentlyContinue)) {
    throw "go not found in PATH"
}

if ($StopFirst) {
    Get-NetTCPConnection -LocalPort $Port -ErrorAction SilentlyContinue | ForEach-Object {
        Stop-Process -Id $_.OwningProcess -Force -ErrorAction SilentlyContinue
    }
}

& $goCmd build -o bridge.exe ./cmd/bridge
if ($LASTEXITCODE -ne 0) {
    throw "go build failed"
}

$envMap = @{}
Get-Content ".env" | ForEach-Object {
    if ($_ -match '^\s*#' -or $_ -match '^\s*$') { return }
    $parts = $_ -split '=', 2
    if ($parts.Count -eq 2) {
        $envMap[$parts[0].Trim()] = $parts[1].Trim()
    }
}

$env:PORT = "$Port"
if ($BaseUrl -ne "") { $env:BASE_URL = $BaseUrl }
foreach ($key in $envMap.Keys) {
    if (-not (Test-Path "Env:$key") -or $key -notin @("PORT","BASE_URL")) {
        Set-Item -Path "Env:$key" -Value $envMap[$key]
    }
}

$proc = Start-Process -FilePath "$RepoRoot\bridge.exe" -WorkingDirectory $RepoRoot -WindowStyle Hidden -PassThru
Start-Sleep -Seconds 2

Write-Host "OIDC Bridge started"
Write-Host "PID: $($proc.Id)"
Write-Host "Port: $Port"
if ($env:BASE_URL) { Write-Host "Base URL: $($env:BASE_URL)" }
