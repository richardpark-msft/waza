# validate-release-flow.ps1
#
# Validates the azd extension release flow locally.
# Tests: build -> pack -> release (dry-run) -> publish (dry-run)
#
# Usage: .\validate-release-flow.ps1 [-Version <ver>] [-DryRun] [-NoDryRun]
#   -Version: optional, defaults to version.txt
#   -NoDryRun: actually execute release and publish steps

param(
    [string]$Version,
    [string]$Repo = "spboyer/waza",
    [switch]$NoDryRun,
    [switch]$SkipCleanup
)

$ErrorActionPreference = 'Stop'

$ScriptDir = Split-Path -Parent $MyInvocation.MyCommand.Path
Set-Location -Path $ScriptDir

if (-not $Version) {
    $Version = (Get-Content version.txt).Trim()
}

Write-Host "============================================"
Write-Host " azd Extension Release Flow Validation"
Write-Host "============================================"
Write-Host ""
Write-Host "  Version:  $Version"
Write-Host "  Repo:     $Repo"
Write-Host "  CWD:      $(Get-Location)"
Write-Host "  DryRun:   $(-not $NoDryRun)"
Write-Host ""

# --- Step 0: Prerequisites ---
Write-Host "--- Step 0: Checking prerequisites ---"

$azdPath = Get-Command azd -ErrorAction SilentlyContinue
if (-not $azdPath) {
    Write-Error "azd CLI not found. Install from https://aka.ms/azd"
    exit 1
}
Write-Host "  OK azd CLI found"

$ghPath = Get-Command gh -ErrorAction SilentlyContinue
if (-not $ghPath) {
    Write-Error "gh CLI not found. Install from https://cli.github.com"
    exit 1
}
Write-Host "  OK gh CLI found"

$goPath = Get-Command go -ErrorAction SilentlyContinue
if (-not $goPath) {
    Write-Error "go not found"
    exit 1
}
Write-Host "  OK go found"

if (-not (Test-Path "extension.yaml")) {
    Write-Error "extension.yaml not found in $(Get-Location)"
    exit 1
}
Write-Host "  OK extension.yaml exists"

azd config set alpha.extensions on 2>$null
Write-Host "  OK azd extensions enabled"

$xHelp = azd x build --help 2>&1
if ($LASTEXITCODE -ne 0) {
    Write-Host "  Installing microsoft.azd.extensions..."
    azd extension install microsoft.azd.extensions
}
Write-Host "  OK azd x commands available"
Write-Host ""

# --- Step 1: Build ---
Write-Host "--- Step 1: azd x build --all ---"
Write-Host "  Building for all platforms..."
azd x build --all --skip-install
if ($LASTEXITCODE -ne 0) { Write-Error "Build failed"; exit 1 }
Write-Host ""

Write-Host "  Build output:"
Get-ChildItem -Path bin -Filter "microsoft-azd-waza*" | ForEach-Object {
    Write-Host ("    {0,-50} {1:N0} bytes" -f $_.Name, $_.Length)
}
Write-Host ""

# --- Step 2: Pack ---
Write-Host "--- Step 2: azd x pack ---"
Write-Host "  Packaging artifacts..."
azd x pack -o ./artifacts
if ($LASTEXITCODE -ne 0) { Write-Error "Pack failed"; exit 1 }
Write-Host ""

Write-Host "  Pack output:"
Get-ChildItem -Path artifacts -Filter "microsoft-azd-waza*" | ForEach-Object {
    Write-Host ("    {0,-50} {1:N0} bytes" -f $_.Name, $_.Length)
}
Write-Host ""

# --- Step 3: Release ---
Write-Host "--- Step 3: azd x release ---"
Write-Host "  This step creates a GitHub release and uploads artifacts."
Write-Host ""
Write-Host "  Command:"
Write-Host "    azd x release ``"
Write-Host "      --repo $Repo ``"
Write-Host "      --version $Version ``"
Write-Host "      --title `"Waza azd Extension v$Version`" ``"
Write-Host "      --notes `"Release v$Version of the waza azd extension`" ``"
Write-Host "      --artifacts `"./artifacts/*.zip,./artifacts/*.tar.gz`" ``"
Write-Host "      --confirm"
Write-Host ""
Write-Host "  Release tag: azd-ext-microsoft-azd-waza_$Version"
Write-Host ""

if ($NoDryRun) {
    Write-Host "  Executing release..."
    azd x release `
        --repo $Repo `
        --version $Version `
        --title "Waza azd Extension v$Version" `
        --notes "Release v$Version of the waza azd extension" `
        --artifacts "./artifacts/*.zip,./artifacts/*.tar.gz" `
        --confirm
    if ($LASTEXITCODE -ne 0) { Write-Error "Release failed"; exit 1 }
    Write-Host "  OK Release created"
} else {
    Write-Host "  SKIPPED (use -NoDryRun to execute)"
}
Write-Host ""

# --- Step 4: Publish ---
Write-Host "--- Step 4: azd x publish ---"
Write-Host "  This step updates registry.json with artifact URLs and checksums."
Write-Host ""
Write-Host "  Command:"
Write-Host "    azd x publish ``"
Write-Host "      --repo $Repo ``"
Write-Host "      --version $Version ``"
Write-Host "      --artifacts `"./artifacts/*.zip,./artifacts/*.tar.gz`" ``"
Write-Host "      --registry ./registry.json"
Write-Host ""

if ($NoDryRun) {
    Write-Host "  Executing publish..."
    azd x publish `
        --repo $Repo `
        --version $Version `
        --artifacts "./artifacts/*.zip,./artifacts/*.tar.gz" `
        --registry ./registry.json
    if ($LASTEXITCODE -ne 0) { Write-Error "Publish failed"; exit 1 }
    Write-Host "  OK Registry updated"
    Write-Host ""
    Write-Host "  registry.json diff:"
    git diff registry.json
} else {
    Write-Host "  SKIPPED (use -NoDryRun to execute)"
}
Write-Host ""

# --- Summary ---
Write-Host "============================================"
Write-Host " Validation Summary"
Write-Host "============================================"
Write-Host ""
Write-Host "  Flow: build -> pack -> release -> publish"
Write-Host ""
Write-Host "  1. azd x build --all         -> bin/ (6 binaries)"
Write-Host "  2. azd x pack -o ./artifacts -> artifacts/ (6 archives)"
Write-Host "  3. azd x release --repo $Repo -> GitHub release with tag + uploaded archives"
Write-Host "  4. azd x publish --repo $Repo -> registry.json updated with URLs + checksums"
Write-Host ""
Write-Host "  After publish, registry.json must be committed back to the repo."
Write-Host "  The GitHub Actions workflow creates a PR with auto-merge for this."
Write-Host ""

# Cleanup
if (-not $SkipCleanup) {
    Write-Host "  Cleaning up artifacts/ directory..."
    Remove-Item -Path artifacts -Recurse -Force -ErrorAction SilentlyContinue
    Write-Host "  OK Cleaned up"
}

Write-Host ""
Write-Host "Done."
