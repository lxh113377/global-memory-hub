# 构建 fenjue-agent: 先把 ../web/dist 拷到 embed 目录, 再编译 (仅标准库)。
$ErrorActionPreference = "Stop"
$agentDir = $PSScriptRoot
$distSrc = Join-Path $agentDir "..\web\dist"
$distDst = Join-Path $agentDir "cmd\fenjue-agent\dist"
if (-not (Test-Path $distSrc)) { throw "missing $distSrc" }
New-Item -ItemType Directory -Force -Path $distDst | Out-Null
Copy-Item -Path (Join-Path $distSrc "*") -Destination $distDst -Recurse -Force

$goEntry = Get-Command go -ErrorAction SilentlyContinue
$go = if ($goEntry) { $goEntry.Source } else { "C:\Program Files\Go\bin\go.exe" }
if (-not (Test-Path $go)) { throw "go not found in PATH or C:\Program Files\Go\bin\go.exe" }

Push-Location $agentDir
try {
    & $go build -o fenjue-agent.exe .\cmd\fenjue-agent
    if ($LASTEXITCODE -ne 0) { throw "go build failed with exit code $LASTEXITCODE" }
} finally {
    Pop-Location
}
Write-Host "built: $(Join-Path $agentDir 'fenjue-agent.exe')"
