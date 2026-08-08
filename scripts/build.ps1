[CmdletBinding()]
param(
    [string]$Output = "bin/repolens.exe",
    [string]$Version = "0.1.0-dev",
    [switch]$SkipTests
)

$ErrorActionPreference = "Stop"
$Root = Split-Path -Parent $PSScriptRoot
$Cache = Join-Path $Root ".cache"
$env:GOCACHE = Join-Path $Cache "go-build"
$env:GOMODCACHE = Join-Path $Cache "go-mod"

Push-Location (Join-Path $Root "web")
try {
    npm.cmd ci
    npm.cmd run check
    npm.cmd run build
}
finally {
    Pop-Location
}

if (-not $SkipTests) {
    & go test ./...
    if ($LASTEXITCODE -ne 0) { throw "Go tests failed" }
}

$OutputPath = Join-Path $Root $Output
New-Item -ItemType Directory -Force -Path (Split-Path -Parent $OutputPath) | Out-Null
& go build -trimpath -ldflags="-s -w -X main.version=$Version" -o $OutputPath ./cmd/repolens
if ($LASTEXITCODE -ne 0) { throw "Go build failed" }

Write-Host "Built $OutputPath"
