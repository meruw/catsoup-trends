param(
    [string]$GoExe = "go",
    [string]$NodeDir = "",
    [switch]$SkipInstall
)
$ErrorActionPreference = "Stop"
$repo = Split-Path $PSScriptRoot -Parent
if ($NodeDir) { $env:PATH = $NodeDir + ";" + $env:PATH }
$goCommand = (Get-Command $GoExe -ErrorAction Stop).Source
Push-Location (Join-Path $repo "web")
try {
    if (-not $SkipInstall) {
        & npm.cmd ci --no-fund
        if ($LASTEXITCODE -ne 0) { throw "npm ci failed" }
    }
    & npm.cmd run build
    if ($LASTEXITCODE -ne 0) { throw "Frontend build failed" }
} finally { Pop-Location }
Push-Location $repo
try {
    & $goCommand test ./...
    if ($LASTEXITCODE -ne 0) { throw "Go tests failed" }
    & $goCommand vet ./...
    if ($LASTEXITCODE -ne 0) { throw "Go vet failed" }
    New-Item -ItemType Directory -Force -Path (Join-Path $repo "bin") | Out-Null
    & $goCommand build -trimpath "-ldflags=-s -w" -o bin/CATsoupTrends.exe ./cmd/trends
    if ($LASTEXITCODE -ne 0) { throw "Backend build failed" }
    Copy-Item -LiteralPath LICENSE,NOTICE,THIRD_PARTY_NOTICES.txt -Destination bin/
    Write-Host "Built bin/CATsoupTrends.exe. Start it with scripts/launch.ps1."
} finally { Pop-Location }
