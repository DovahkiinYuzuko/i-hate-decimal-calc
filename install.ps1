param(
    [switch]$Local
)

# ihd PowerShell Installer for Windows
$ErrorActionPreference = 'Stop'
[Net.ServicePointManager]::SecurityProtocol = [Net.ServicePointManager]::SecurityProtocol -bor [Net.SecurityProtocolType]::Tls12

$Repo = "DovahkiinYuzuko/i-hate-decimal-calc"
$IhdHome = "$env:USERPROFILE\.ihd"
$InstallDir = "$IhdHome\bin"
$ExamDir = "$IhdHome\exam"
$ReferencesDir = "$IhdHome\references"
$VerifierSrcDir = "$IhdHome\tools\micro-verifier"

if (-not (Test-Path $InstallDir)) {
    New-Item -ItemType Directory -Path $InstallDir -Force | Out-Null
}

if ($Local) {
    Write-Host "Installing ihd from local repository..." -ForegroundColor Cyan
    $LocalRoot = $PSScriptRoot
    if (-not $LocalRoot) { $LocalRoot = Get-Location }
    
    $PrevCgo = $env:CGO_ENABLED
    $env:CGO_ENABLED = "0"
    Push-Location $LocalRoot
    try {
        $GitTag = (git describe --tags --exact-match 2>$null)
        if (-not $GitTag) {
            $GitTag = (git describe --tags 2>$null)
        }

        $LdFlags = "-s -w"
        if ($GitTag) {
            $LdFlags += " -X github.com/DovahkiinYuzuko/i-hate-decimal-calc/internal/updater.BuildVersion=$GitTag"
        }
        go build -ldflags="$LdFlags" -o (Join-Path $InstallDir "ihd.exe") ./cmd/ihd
        go build -ldflags="-s -w" -o (Join-Path $InstallDir "micro-verifier.exe") ./tools/micro-verifier
    } finally {
        Pop-Location
        if ($null -ne $PrevCgo) { $env:CGO_ENABLED = $PrevCgo } else { Remove-Item env:CGO_ENABLED -ErrorAction SilentlyContinue }
    }

    if (Test-Path (Join-Path $LocalRoot "exam")) {
        Copy-Item -Path (Join-Path $LocalRoot "exam") -Destination $IhdHome -Recurse -Force
    }
    $LocalRefSource = Join-Path $LocalRoot "docs\references"
    if (Test-Path $LocalRefSource) {
        if (-not (Test-Path $ReferencesDir)) {
            New-Item -ItemType Directory -Path $ReferencesDir -Force | Out-Null
        }
        $NestedRef = Join-Path $ReferencesDir "references"
        if (Test-Path $NestedRef) {
            Remove-Item -Path $NestedRef -Recurse -Force -ErrorAction SilentlyContinue
        }
        Copy-Item -Path (Join-Path $LocalRefSource "*") -Destination $ReferencesDir -Recurse -Force
    }
    foreach ($doc in @("LICENSE.MIT", "NOTICE.md", "README.md")) {
        $docPath = Join-Path $LocalRoot $doc
        if (Test-Path $docPath) {
            Copy-Item -Path $docPath -Destination $IhdHome -Force
        }
    }
    $LocalVerifierSrc = Join-Path $LocalRoot "tools\micro-verifier\main.go"
    if (Test-Path $LocalVerifierSrc) {
        if (-not (Test-Path $VerifierSrcDir)) {
            New-Item -ItemType Directory -Path $VerifierSrcDir -Force | Out-Null
        }
        Copy-Item -Path $LocalVerifierSrc -Destination (Join-Path $VerifierSrcDir "main.go") -Force
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
    $TmpSums = Join-Path $env:TEMP "ihd-sums-$([System.Guid]::NewGuid().ToString('N')).txt"
    $TmpExtract = Join-Path $env:TEMP "ihd-extract-$([System.Guid]::NewGuid().ToString('N'))"

    try {
        Write-Host "Downloading $DownloadUrl..." -ForegroundColor Cyan
        Invoke-WebRequest -Uri $DownloadUrl -OutFile $TmpZip -UseBasicParsing

        $ChecksumUrl = "https://github.com/$Repo/releases/download/$Tag/SHA256SUMS"
        Write-Host "Downloading $ChecksumUrl..." -ForegroundColor Cyan
        Invoke-WebRequest -Uri $ChecksumUrl -OutFile $TmpSums -UseBasicParsing

        $ExpectedHash = $null
        foreach ($line in (Get-Content -Path $TmpSums)) {
            if ($line -match "^([a-fA-F0-9]{64})\s+.*$([regex]::Escape($AssetName))`$") {
                $ExpectedHash = $matches[1].ToLowerInvariant()
                break
            }
        }
        if (-not $ExpectedHash) {
            Write-Error "Could not find expected SHA256 checksum for $AssetName in SHA256SUMS manifest."
            exit 1
        }

        Write-Host "Verifying SHA256 checksum..." -ForegroundColor Cyan
        $ActualHash = (Get-FileHash -Path $TmpZip -Algorithm SHA256).Hash.ToLowerInvariant()
        if ($ActualHash -ne $ExpectedHash) {
            Write-Error "SHA256 checksum verification failed for $AssetName!`nExpected: $ExpectedHash`nActual:   $ActualHash"
            exit 1
        }
        Write-Host "SHA256 checksum verified successfully." -ForegroundColor Green

        Write-Host "Extracting..." -ForegroundColor Cyan
        Expand-Archive -Path $TmpZip -DestinationPath $TmpExtract -Force

        $ExeSource = Join-Path $TmpExtract "ihd.exe"
        if (-not (Test-Path $ExeSource)) {
            Write-Error "ihd.exe not found in extracted archive."
            exit 1
        }

        Copy-Item -Path $ExeSource -Destination (Join-Path $InstallDir "ihd.exe") -Force

        $VerifierSource = Join-Path $TmpExtract "micro-verifier.exe"
        if (Test-Path $VerifierSource) {
            Copy-Item -Path $VerifierSource -Destination (Join-Path $InstallDir "micro-verifier.exe") -Force
        }

        $ExamSource = Join-Path $TmpExtract "exam"
        if (Test-Path $ExamSource) {
            Copy-Item -Path $ExamSource -Destination $IhdHome -Recurse -Force
        }

        $RefSource = Join-Path $TmpExtract "docs\references"
        if (Test-Path $RefSource) {
            if (-not (Test-Path $ReferencesDir)) {
                New-Item -ItemType Directory -Path $ReferencesDir -Force | Out-Null
            }
            $NestedRef = Join-Path $ReferencesDir "references"
            if (Test-Path $NestedRef) {
                Remove-Item -Path $NestedRef -Recurse -Force -ErrorAction SilentlyContinue
            }
            Copy-Item -Path (Join-Path $RefSource "*") -Destination $ReferencesDir -Recurse -Force
        }

        foreach ($doc in @("LICENSE.MIT", "NOTICE.md", "README.md")) {
            $docSource = Join-Path $TmpExtract $doc
            if (Test-Path $docSource) {
                Copy-Item -Path $docSource -Destination $IhdHome -Force
            }
        }

        $VerifierGoSource = Join-Path $TmpExtract "tools\micro-verifier\main.go"
        if (Test-Path $VerifierGoSource) {
            if (-not (Test-Path $VerifierSrcDir)) {
                New-Item -ItemType Directory -Path $VerifierSrcDir -Force | Out-Null
            }
            Copy-Item -Path $VerifierGoSource -Destination (Join-Path $VerifierSrcDir "main.go") -Force
        }
    } finally {
        if (Test-Path $TmpZip) { Remove-Item -Path $TmpZip -Force -ErrorAction SilentlyContinue }
        if (Test-Path $TmpSums) { Remove-Item -Path $TmpSums -Force -ErrorAction SilentlyContinue }
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

# Shell Completion Setup
$CompletionsDir = "$IhdHome\completions"
if (-not (Test-Path $CompletionsDir)) {
    New-Item -ItemType Directory -Path $CompletionsDir -Force | Out-Null
}
$PsCompletionFile = Join-Path $CompletionsDir "ihd.ps1"
$IhdExe = Join-Path $InstallDir "ihd.exe"

if (Test-Path $IhdExe) {
    try {
        $prevOutputEncoding = $OutputEncoding
        $prevConsoleEncoding = [Console]::OutputEncoding
        try {
            $OutputEncoding = [System.Text.Encoding]::UTF8
            [Console]::OutputEncoding = [System.Text.Encoding]::UTF8
            & $IhdExe completion powershell | Out-File -FilePath $PsCompletionFile -Encoding utf8
        } finally {
            $OutputEncoding = $prevOutputEncoding
            [Console]::OutputEncoding = $prevConsoleEncoding
        }
        Write-Host "Generated PowerShell completion script: $PsCompletionFile" -ForegroundColor Cyan

        # Register in $PROFILE if not already registered
        if ($PROFILE) {
            $ProfileDir = Split-Path -Path $PROFILE -Parent
            if ($ProfileDir -and -not (Test-Path $ProfileDir)) {
                New-Item -ItemType Directory -Path $ProfileDir -Force | Out-Null
            }
            if (-not (Test-Path $PROFILE)) {
                New-Item -ItemType File -Path $PROFILE -Force | Out-Null
            }

            $ProfileContent = Get-Content -Path $PROFILE -Raw -ErrorAction SilentlyContinue
            if ($null -eq $ProfileContent) { $ProfileContent = "" }
            $CompletionMarker = "# i-hate-decimal-calc shell completion"

            if ($ProfileContent -notmatch [regex]::Escape($CompletionMarker)) {
                $AutoloadSnippet = @"

$CompletionMarker
if (Test-Path "$PsCompletionFile") {
    . "$PsCompletionFile"
}
"@
                Add-Content -Path $PROFILE -Value $AutoloadSnippet -Encoding utf8
                Write-Host "Added completion autoload to PowerShell profile: $PROFILE" -ForegroundColor Cyan
            }
        }

        # Source completion in current session so it works immediately
        if (Test-Path $PsCompletionFile) {
            . $PsCompletionFile
        }
    } catch {
        Write-Warning "Shell completion setup failed: $_"
    }
}

Write-Host "`nSuccessfully installed ihd to $InstallDir\ihd.exe!" -ForegroundColor Green
if (Test-Path $ReferencesDir) {
    Write-Host "Offline Reference Manuals installed to: $ReferencesDir" -ForegroundColor Cyan
    Write-Host "  - Japanese: $ReferencesDir\ja\reference.ja.html" -ForegroundColor Gray
    Write-Host "  - English:  $ReferencesDir\en\reference.en.html" -ForegroundColor Gray
}
if (Test-Path $ExamDir) {
    Write-Host "Mathematical Showcase & Cookbook installed to: $ExamDir" -ForegroundColor Cyan
    Write-Host "Run the live showcase: pwsh `"$ExamDir\exam.ps1`"" -ForegroundColor Yellow
}
if (Test-Path (Join-Path $VerifierSrcDir "main.go")) {
    Write-Host "Standalone Micro-Verifier binary & source installed:" -ForegroundColor Cyan
    Write-Host "  - Binary: $InstallDir\micro-verifier.exe" -ForegroundColor Gray
    Write-Host "  - Source: $VerifierSrcDir\main.go" -ForegroundColor Gray
}
Write-Host "Run 'ihd --help' or 'ihd' to start calculating." -ForegroundColor Cyan
