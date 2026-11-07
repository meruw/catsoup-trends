param(
    [string]$GoExe = "go",
    [string]$NodeDir = "",
    [string]$DataDir = ""
)
$ErrorActionPreference = "Stop"
$repo = Split-Path $PSScriptRoot -Parent
if ($NodeDir) { $env:PATH = $NodeDir + ";" + $env:PATH }
$previousGo = $env:TRENDS_GO_EXE
$previousData = $env:TRENDS_DEV_DATA_DIR
$env:TRENDS_GO_EXE = (Get-Command $GoExe -ErrorAction Stop).Source
$env:TRENDS_DEV_DATA_DIR = $DataDir
Push-Location (Join-Path $repo "web")
try {
    if (-not (Test-Path -LiteralPath node_modules)) { throw "Run npm ci in web before starting development." }
    & node dev.mjs
    if ($LASTEXITCODE -ne 0) { throw "Development server stopped with an error." }
} finally {
    Pop-Location
    $env:TRENDS_GO_EXE = $previousGo
    $env:TRENDS_DEV_DATA_DIR = $previousData
}
