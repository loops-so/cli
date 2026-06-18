# install.ps1
param(
    [string]$Tag = "latest",
    [string]$InstallDir = "$env:LOCALAPPDATA\Programs\loops"
)

$ErrorActionPreference = "Stop"

$GHRepo = "loops-so/cli"
$GHAssetsUrl = "https://github.com/$GHRepo/releases/download"
$ProjName = "loops_cli"
$BinName = "loops.exe"

$Arch = switch ($env:PROCESSOR_ARCHITECTURE) {
    "AMD64"  { "x86_64" }
    "ARM64"  { "arm64" }
    "x86"    { "i386" }
    default  { throw "Unsupported architecture: $env:PROCESSOR_ARCHITECTURE" }
}

function Get-GithubRelease {
    param([string]$Repo, [string]$Version)
    $url = if ($Version -eq "latest") {
        "https://api.github.com/repos/$Repo/releases/latest"
    } else {
        "https://api.github.com/repos/$Repo/releases/tags/$Version"
    }
    $response = Invoke-RestMethod -Uri $url
    return $response.tag_name
}

function Confirm-Checksum {
    param([string]$FilePath, [string]$ChecksumsPath)
    $filename = Split-Path $FilePath -Leaf
    $line = Get-Content $ChecksumsPath | Where-Object { $_ -match [regex]::Escape($filename) }
    if (-not $line) {
        throw "Could not find checksum for $filename"
    }
    $want = ($line -split '\s+')[0].ToLower()
    $got = (Get-FileHash -Algorithm SHA256 -Path $FilePath).Hash.ToLower()
    if ($want -ne $got) {
        throw "Checksum mismatch for $filename`: expected $want, got $got"
    }
}

$release = Get-GithubRelease -Repo $GHRepo -Version $Tag
$versionNoV = $release -replace '^v', ''
$archiveName = "${ProjName}_windows_${Arch}.zip"
$checksumsName = "${ProjName}_${versionNoV}_checksums.txt"
$downloadUrl = "$GHAssetsUrl/$release/$archiveName"
$checksumsUrl = "$GHAssetsUrl/$release/$checksumsName"

Write-Host "Installing $ProjName $release for windows/$Arch..."

$tmpDir = Join-Path $env:TEMP ([System.IO.Path]::GetRandomFileName())
New-Item -ItemType Directory -Path $tmpDir | Out-Null

try {
    Invoke-WebRequest -Uri $downloadUrl -OutFile "$tmpDir\$archiveName"
    Invoke-WebRequest -Uri $checksumsUrl -OutFile "$tmpDir\$checksumsName"

    Confirm-Checksum -FilePath "$tmpDir\$archiveName" -ChecksumsPath "$tmpDir\$checksumsName"

    Expand-Archive -Path "$tmpDir\$archiveName" -DestinationPath $tmpDir -Force

    if (-not (Test-Path $InstallDir)) {
        New-Item -ItemType Directory -Path $InstallDir | Out-Null
    }

    Copy-Item "$tmpDir\$BinName" "$InstallDir\$BinName" -Force

    $userPath = [System.Environment]::GetEnvironmentVariable("Path", "User")
    if ($userPath -notlike "*$InstallDir*") {
        [System.Environment]::SetEnvironmentVariable("Path", "$userPath;$InstallDir", "User")
        $env:PATH = "$env:PATH;$InstallDir"
        Write-Host "Added $InstallDir to your PATH"
    }
} finally {
    Remove-Item -Recurse -Force $tmpDir -ErrorAction SilentlyContinue
}

Write-Host "Done!"
Write-Host "Installed to $InstallDir\$BinName"
& "$InstallDir\$BinName" --version
