param(
    [int]$Port = 8080
)

$ErrorActionPreference = "Stop"

$connections = Get-NetTCPConnection -LocalPort $Port -ErrorAction SilentlyContinue
if (-not $connections) {
    Write-Host "No process is listening on port $Port"
    exit 0
}

$stopped = @{}
foreach ($conn in $connections) {
    $pid = $conn.OwningProcess
    if (-not $stopped.ContainsKey($pid)) {
        Stop-Process -Id $pid -Force -ErrorAction SilentlyContinue
        $stopped[$pid] = $true
        Write-Host "Stopped PID $pid on port $Port"
    }
}
