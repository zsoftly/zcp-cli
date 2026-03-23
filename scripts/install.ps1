<#
.SYNOPSIS
    ZCP CLI installer for Windows
.DESCRIPTION
    Downloads and installs the zcp binary from GitHub releases.
.PARAMETER InstallDir
    Directory to install zcp.exe (default: $env:LOCALAPPDATA\Programs\zcp)
#>
# Write-Host is intentional: this is an interactive installer that writes to the console.
[Diagnostics.CodeAnalysis.SuppressMessageAttribute('PSAvoidUsingWriteHost', '', Scope = 'Function', Target = '*')]
param(
    [string]$InstallDir = "$env:LOCALAPPDATA\Programs\zcp"
)

$ErrorActionPreference = "Stop"
$Repo = "zsoftly/zcp-cli"
$BinaryName = "zcp"

function Write-Info  { Write-Host "  $args" -ForegroundColor Cyan }
function Write-Ok    { Write-Host "  [OK] $args" -ForegroundColor Green }
function Write-Err   { Write-Host "  [ERROR] $args" -ForegroundColor Red; exit 1 }

Write-Host ""
Write-Host "  ZCP CLI Installer" -ForegroundColor Cyan
Write-Host "  -----------------" -ForegroundColor Cyan
Write-Host ""

# Detect arch
$Arch = if ([System.Runtime.InteropServices.RuntimeInformation]::OSArchitecture -eq "Arm64") { "arm64" } else { "amd64" }
$AssetName = "${BinaryName}-windows-${Arch}.exe"
$DownloadUrl = "https://github.com/${Repo}/releases/latest/download/${AssetName}"

Write-Info "Downloading ${AssetName}..."

if (-not (Test-Path $InstallDir)) {
    New-Item -ItemType Directory -Path $InstallDir -Force | Out-Null
}

$TmpFile = [System.IO.Path]::GetTempFileName() + ".exe"
try {
    Invoke-WebRequest -Uri $DownloadUrl -OutFile $TmpFile -UseBasicParsing
    Move-Item -Path $TmpFile -Destination "$InstallDir\${BinaryName}.exe" -Force
} catch {
    Write-Err "Download failed: $_"
}

# Add to PATH if not already there
$CurrentPath = [System.Environment]::GetEnvironmentVariable("PATH", "User")
if ($CurrentPath -notlike "*$InstallDir*") {
    [System.Environment]::SetEnvironmentVariable("PATH", "$CurrentPath;$InstallDir", "User")
    Write-Info "Added $InstallDir to user PATH"
}

Write-Ok "Installed ${BinaryName} to $InstallDir\${BinaryName}.exe"
Write-Host ""
Write-Info "Restart your terminal, then run: ${BinaryName} version"
Write-Host ""
