# Anti2Verge Windows Build Script

param (
    [string]$Mode = "all"
)

$ErrorActionPreference = "Stop"

$GoCmd = "go"
if (-not (Get-Command "go" -ErrorAction SilentlyContinue)) {
    if (Test-Path "D:\go\bin\go.exe") {
        $GoCmd = "D:\go\bin\go.exe"
    } elseif (Test-Path "C:\Program Files\Go\bin\go.exe") {
        $GoCmd = "C:\Program Files\Go\bin\go.exe"
    } else {
        Write-Error "Go compiler not found. Please install Go or set PATH."
        exit 1
    }
}

Write-Host "Using Go compiler: $GoCmd"
& $GoCmd version

if (-not (Test-Path "bin")) {
    New-Item -ItemType Directory -Path "bin" | Out-Null
}

Write-Host "Step 1: Generating Windows icon and PE resource (.syso)..."
& $GoCmd run ./scripts/build_res.go

Write-Host "Step 2: Resolving dependencies..."
& $GoCmd mod tidy

# 1. Build CLI binary
if ($Mode -eq "all" -or $Mode -eq "cli") {
    Write-Host "Step 3: Building CLI version: bin/anti2verge-cli.exe ..."
    & $GoCmd build -ldflags "-s -w" -o bin/anti2verge-cli.exe ./cmd/anti2verge
    Write-Host "CLI version built successfully!"
}

# 2. Build GUI tray binary (no black cmd window)
if ($Mode -eq "all" -or $Mode -eq "gui") {
    Write-Host "Step 4: Building GUI/Tray version: bin/Anti2Verge.exe ..."
    & $GoCmd build -ldflags "-H windowsgui -s -w" -o bin/Anti2Verge.exe ./cmd/anti2verge
    Copy-Item "bin/Anti2Verge.exe" "Anti2Verge.exe" -Force
    Write-Host "GUI Tray version built successfully (copied to root directory as well)!"
}

Write-Host "=========================================="
Write-Host "Build complete! Output files in bin/:"
Get-ChildItem bin\*.exe | Select-Object Name, Length, LastWriteTime | Format-Table -AutoSize
Write-Host "=========================================="
