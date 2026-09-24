# nui installer — https://nui.plmbr.dev/install.ps1
# Usage: irm https://nui.plmbr.dev/install.ps1 | iex
#
# Environment:
#   $env:NUI_VERSION      Release tag (default: latest), e.g. v0.1.0
#   $env:NUI_INSTALL_DIR  Install directory (default: %LOCALAPPDATA%\nui)
#   $env:GITHUB_REPO      GitHub owner/repo (default: plmbr/nui)
#   $env:GITHUB_TOKEN     Optional; only used if the latest tag can't be resolved via redirect

$ErrorActionPreference = "Stop"

$GithubRepo = if ($env:GITHUB_REPO) { $env:GITHUB_REPO } else { "plmbr/nui" }
$InstallDir = if ($env:NUI_INSTALL_DIR) { $env:NUI_INSTALL_DIR } else { Join-Path $env:LOCALAPPDATA "nui" }
$Version = if ($env:NUI_VERSION) { $env:NUI_VERSION } else { "latest" }
$BinaryName = "nui.exe"

function Say($Message) { Write-Host $Message }
function Fail($Message) { Write-Error $Message; exit 1 }

function Get-PlatformArch {
    $arch = $env:PROCESSOR_ARCHITECTURE
    if ($env:PROCESSOR_ARCHITEW6432) {
        $arch = $env:PROCESSOR_ARCHITEW6432
    }
    switch ($arch) {
        "AMD64" { return "amd64" }
        "ARM64" { return "arm64" }
        default {
            Fail @"
unsupported architecture: $arch

Download a release manually:
  https://github.com/$GithubRepo/releases
"@
        }
    }
}

# Resolves "latest" via the github.com redirect (/releases/latest -> /releases/tag/<tag>)
# rather than api.github.com, which is limited to 60 unauthenticated requests/hour per IP.
function Resolve-Version {
    if ($Version -ne "latest") { return }

    try {
        $resp = Invoke-WebRequest -Uri "https://github.com/$GithubRepo/releases/latest" -UseBasicParsing -ErrorAction Stop
        $finalUrl = $null
        if ($resp.BaseResponse.ResponseUri) {
            # Windows PowerShell 5.1 (HttpWebResponse)
            $finalUrl = $resp.BaseResponse.ResponseUri.AbsoluteUri
        } elseif ($resp.BaseResponse.RequestMessage -and $resp.BaseResponse.RequestMessage.RequestUri) {
            # PowerShell 7+ (HttpResponseMessage)
            $finalUrl = $resp.BaseResponse.RequestMessage.RequestUri.AbsoluteUri
        }
        if ($finalUrl -match '/releases/tag/(.+)$') {
            $script:Version = [uri]::UnescapeDataString($Matches[1])
            return
        }
    } catch {
        # fall through to the API fallback below
    }

    $headers = @{}
    if ($env:GITHUB_TOKEN) { $headers["Authorization"] = "Bearer $($env:GITHUB_TOKEN)" }
    try {
        $release = Invoke-RestMethod -Uri "https://api.github.com/repos/$GithubRepo/releases/latest" -Headers $headers -ErrorAction Stop
    } catch {
        Fail @"
could not resolve latest release from GitHub: $($_.Exception.Message)

Set `$env:NUI_VERSION explicitly (e.g. `$env:NUI_VERSION = "v0.1.0"), or set `$env:GITHUB_TOKEN to avoid API rate limits.
"@
    }
    if (-not $release.tag_name) {
        Fail 'could not resolve latest release from GitHub. Set $env:NUI_VERSION explicitly (e.g. $env:NUI_VERSION = "v0.1.0").'
    }
    $script:Version = $release.tag_name
}

function Get-FileSha256([string]$Path) {
    return (Get-FileHash -Algorithm SHA256 -Path $Path).Hash.ToLower()
}

function Verify-Checksum([string]$ArchiveName, [string]$ArchivePath, [string]$ChecksumsPath) {
    $line = Get-Content $ChecksumsPath | Where-Object { $_ -match " $($ArchiveName)$" } | Select-Object -First 1
    if (-not $line) {
        Fail "checksums.txt does not list $ArchiveName"
    }
    $expected = ($line -split '\s+')[0].ToLower()
    $actual = Get-FileSha256 $ArchivePath
    if ($expected -ne $actual) {
        Fail "checksum mismatch for $ArchiveName"
    }
}

function Test-PathInUserPath([string]$Dir) {
    $userPath = [Environment]::GetEnvironmentVariable("Path", "User")
    if (-not $userPath) { return $false }
    foreach ($entry in $userPath.Split(';')) {
        if ($entry.TrimEnd('\') -ieq $Dir.TrimEnd('\')) { return $true }
    }
    return $false
}

function Add-InstallDirToUserPath([string]$Dir) {
    if (Test-PathInUserPath $Dir) { return }
    $userPath = [Environment]::GetEnvironmentVariable("Path", "User")
    if ($userPath) {
        [Environment]::SetEnvironmentVariable("Path", "$userPath;$Dir", "User")
    } else {
        [Environment]::SetEnvironmentVariable("Path", $Dir, "User")
    }
    $env:Path = "$env:Path;$Dir"
    Say ""
    Say "Added $Dir to your user PATH (open a new terminal if nui is not found)."
}

$PlatformArch = Get-PlatformArch
Resolve-Version

$ArchiveName = "nui_${Version}_windows_${PlatformArch}.zip"
$BaseUrl = "https://github.com/$GithubRepo/releases/download/$Version"
$ArchiveUrl = "$BaseUrl/$ArchiveName"
$ChecksumsUrl = "$BaseUrl/checksums.txt"

$TempDir = Join-Path ([System.IO.Path]::GetTempPath()) ("nui-install-" + [guid]::NewGuid().ToString())
New-Item -ItemType Directory -Path $TempDir -Force | Out-Null

try {
    Say "Installing nui $Version (windows/$PlatformArch)"

    $ChecksumsPath = Join-Path $TempDir "checksums.txt"
    $ArchivePath = Join-Path $TempDir $ArchiveName
    Invoke-WebRequest -Uri $ChecksumsUrl -OutFile $ChecksumsPath
    Invoke-WebRequest -Uri $ArchiveUrl -OutFile $ArchivePath
    Verify-Checksum $ArchiveName $ArchivePath $ChecksumsPath

    $ExtractDir = Join-Path $TempDir "extract"
    Expand-Archive -Path $ArchivePath -DestinationPath $ExtractDir -Force
    $BinaryPath = Join-Path $ExtractDir $BinaryName
    if (-not (Test-Path $BinaryPath)) {
        Fail "archive did not contain $BinaryName"
    }

    New-Item -ItemType Directory -Path $InstallDir -Force | Out-Null
    $Dest = Join-Path $InstallDir $BinaryName
    Copy-Item -Path $BinaryPath -Destination $Dest -Force

    Say "Installed $BinaryName to $Dest"
    Add-InstallDirToUserPath $InstallDir

    $installedVersion = & $Dest --version 2>$null
    if ($installedVersion) {
        Say "nui version: $installedVersion"
    }
}
finally {
    Remove-Item -Path $TempDir -Recurse -Force -ErrorAction SilentlyContinue
}
