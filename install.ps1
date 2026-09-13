param(
    [switch]$Local
)

# ihd PowerShell Installer for Windows
$ErrorActionPreference = 'Stop'

$Repo = "DovahkiinYuzuko/i-hate-decimal-calc"
$IhdHome = "$env:USERPROFILE\.ihd"
$InstallDir = "$IhdHome\bin"
$ExamDir = "$IhdHome\exam"

if (-not (Test-Path $InstallDir)) {
    New-Item -ItemType Directory -Path $InstallDir -Force | Out-Null
}

if ($Local) {
    Write-Host "Installing ihd from local repository..." -ForegroundColor Cyan
    $LocalRoot = $PSScriptRoot
    if (-not $LocalRoot) { $LocalRoot = Get-Location }
    
    Push-Location $LocalRoot
    try {
        go build -ldflags="-s -w" -o (Join-Path $InstallDir "ihd.exe") ./cmd/ihd
    } finally {
        Pop-Location
    }

    if (Test-Path (Join-Path $LocalRoot "exam")) {
        Copy-Item -Path (Join-Path $LocalRoot "exam") -Destination $IhdHome -Recurse -Force
    }
} else {
    # Determine architecture
    $Arch = $env:PROCESSOR_ARCHITECTURE
    if ($Arch -eq "ARM64") {
        $AssetName = "ihd-windows-arm64.zip"
    } else {
        $AssetName = "ihd-windows-amd64.zip"
    }

    Write-Host "Fetching latest release for $Repo..." -ForegroundColor Cyan

    $ReleaseUrl = "https://api.github.com/repos/$Repo/releases/latest"
    try {
        $ReleaseInfo = Invoke-RestMethod -Uri $ReleaseUrl -Headers @{ "User-Agent" = "ihd-installer" }
        $Tag = $ReleaseInfo.tag_name
        Write-Host "Latest release: $Tag" -ForegroundColor Green
    } catch {
        Write-Error "Failed to fetch latest release info from GitHub API: $_"
        exit 1
    }

    $DownloadUrl = "https://github.com/$Repo/releases/download/$Tag/$AssetName"
    $TmpZip = Join-Path $env:TEMP "$AssetName"
    $TmpExtract = Join-Path $env:TEMP "ihd-extract-$([System.Guid]::NewGuid().ToString('N'))"

    try {
        Write-Host "Downloading $DownloadUrl..." -ForegroundColor Cyan
        Invoke-WebRequest -Uri $DownloadUrl -OutFile $TmpZip -UseBasicParsing

        Write-Host "Extracting..." -ForegroundColor Cyan
        Expand-Archive -Path $TmpZip -DestinationPath $TmpExtract -Force

        $ExeSource = Join-Path $TmpExtract "ihd.exe"
        if (-not (Test-Path $ExeSource)) {
            Write-Error "ihd.exe not found in extracted archive."
            exit 1
        }

        Copy-Item -Path $ExeSource -Destination (Join-Path $InstallDir "ihd.exe") -Force

        $ExamSource = Join-Path $TmpExtract "exam"
        if (Test-Path $ExamSource) {
            Copy-Item -Path $ExamSource -Destination $IhdHome -Recurse -Force
        }
    } finally {
        if (Test-Path $TmpZip) { Remove-Item -Path $TmpZip -Force -ErrorAction SilentlyContinue }
        if (Test-Path $TmpExtract) { Remove-Item -Path $TmpExtract -Recurse -Force -ErrorAction SilentlyContinue }
    }
}

# Update User PATH
$UserPath = [Environment]::GetEnvironmentVariable("PATH", [EnvironmentVariableTarget]::User)
$PathEntries = $UserPath -split ';' | Where-Object { $_ -ne '' }

if ($PathEntries -notcontains $InstallDir) {
    Write-Host "Adding $InstallDir to user PATH..." -ForegroundColor Cyan
    $NewUserPath = "$UserPath;$InstallDir"
    [Environment]::SetEnvironmentVariable("PATH", $NewUserPath, [EnvironmentVariableTarget]::User)
}

# Update current session PATH so it works immediately
if (($env:PATH -split ';') -notcontains $InstallDir) {
    $env:PATH = "$env:PATH;$InstallDir"
}

Write-Host "`nSuccessfully installed ihd to $InstallDir\ihd.exe!" -ForegroundColor Green
if (Test-Path $ExamDir) {
    Write-Host "Mathematical Showcase & Cookbook installed to: $ExamDir" -ForegroundColor Cyan
    Write-Host "Run the live showcase: pwsh `"$ExamDir\exam.ps1`"" -ForegroundColor Yellow
}
Write-Host "Run 'ihd --help' or 'ihd' to start calculating." -ForegroundColor Cyan
