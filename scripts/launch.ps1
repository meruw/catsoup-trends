$ErrorActionPreference = "Stop"
$repo = Split-Path $PSScriptRoot -Parent
$exe = Join-Path $repo "bin/CATsoupTrends.exe"
if (-not (Test-Path -LiteralPath $exe -PathType Leaf)) {
    throw "Build Trends first using scripts/build.ps1."
}
Start-Process -FilePath $exe -WorkingDirectory $repo -WindowStyle Hidden
