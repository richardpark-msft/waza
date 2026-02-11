#!/bin/bash
# validate-release-flow.sh
#
# Validates the azd extension release flow locally.
# Tests: build → pack → release (dry-run) → publish (dry-run)
#
# Usage: ./validate-release-flow.sh [version]
#   version: optional, defaults to version.txt

set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"
cd "$SCRIPT_DIR"

VERSION="${1:-$(cat version.txt)}"
REPO="${REPO:-spboyer/waza}"

echo "============================================"
echo " azd Extension Release Flow Validation"
echo "============================================"
echo ""
echo "  Version:  $VERSION"
echo "  Repo:     $REPO"
echo "  CWD:      $(pwd)"
echo ""

# --- Step 0: Prerequisites ---
echo "--- Step 0: Checking prerequisites ---"

if ! command -v azd &> /dev/null; then
    echo "ERROR: azd CLI not found. Install from https://aka.ms/azd"
    exit 1
fi
echo "  ✓ azd CLI found: $(azd version 2>/dev/null | head -1)"

if ! command -v gh &> /dev/null; then
    echo "ERROR: gh CLI not found. Install from https://cli.github.com"
    exit 1
fi
echo "  ✓ gh CLI found: $(gh --version | head -1)"

if ! command -v go &> /dev/null; then
    echo "ERROR: go not found"
    exit 1
fi
echo "  ✓ go found: $(go version)"

if [ ! -f "extension.yaml" ]; then
    echo "ERROR: extension.yaml not found in $(pwd)"
    exit 1
fi
echo "  ✓ extension.yaml exists"

# Check azd extensions are enabled
azd config set alpha.extensions on 2>/dev/null || true
echo "  ✓ azd extensions enabled"

# Check azd x is available
if ! azd x build --help &> /dev/null; then
    echo "  Installing microsoft.azd.extensions..."
    azd extension install microsoft.azd.extensions
fi
echo "  ✓ azd x commands available"
echo ""

# --- Step 1: Build ---
echo "--- Step 1: azd x build --all ---"
echo "  Building for all platforms..."
azd x build --all --skip-install
echo ""

echo "  Build output:"
ls -lh bin/ | grep microsoft-azd-waza
echo ""

# --- Step 2: Pack ---
echo "--- Step 2: azd x pack ---"
echo "  Packaging artifacts..."
azd x pack -o ./artifacts
echo ""

echo "  Pack output:"
ls -lh artifacts/ | grep microsoft-azd-waza
echo ""

# --- Step 3: Release (dry-run) ---
echo "--- Step 3: azd x release (DRY RUN) ---"
echo "  This step would create a GitHub release."
echo ""
echo "  Command that would run:"
echo "    azd x release \\"
echo "      --repo $REPO \\"
echo "      --version $VERSION \\"
echo "      --title \"Waza azd Extension v$VERSION\" \\"
echo "      --notes \"Release v$VERSION of the waza azd extension\" \\"
echo "      --artifacts \"./artifacts/*.zip,./artifacts/*.tar.gz\" \\"
echo "      --confirm"
echo ""
echo "  Release tag: azd-ext-microsoft-azd-waza_$VERSION"
echo ""

if [ "${DRY_RUN:-true}" = "false" ]; then
    echo "  Executing release..."
    azd x release \
        --repo "$REPO" \
        --version "$VERSION" \
        --title "Waza azd Extension v$VERSION" \
        --notes "Release v$VERSION of the waza azd extension" \
        --artifacts "./artifacts/*.zip,./artifacts/*.tar.gz" \
        --confirm
    echo "  ✓ Release created"
else
    echo "  SKIPPED (set DRY_RUN=false to execute)"
fi
echo ""

# --- Step 4: Publish (dry-run) ---
echo "--- Step 4: azd x publish (DRY RUN) ---"
echo "  This step would update registry.json with artifact URLs and checksums."
echo ""
echo "  Command that would run:"
echo "    azd x publish \\"
echo "      --repo $REPO \\"
echo "      --version $VERSION \\"
echo "      --artifacts \"./artifacts/*.zip,./artifacts/*.tar.gz\" \\"
echo "      --registry ./registry.json"
echo ""

if [ "${DRY_RUN:-true}" = "false" ]; then
    echo "  Executing publish..."
    azd x publish \
        --repo "$REPO" \
        --version "$VERSION" \
        --artifacts "./artifacts/*.zip,./artifacts/*.tar.gz" \
        --registry ./registry.json
    echo "  ✓ Registry updated"
    echo ""
    echo "  registry.json diff:"
    git diff registry.json || true
else
    echo "  SKIPPED (set DRY_RUN=false to execute)"
fi
echo ""

# --- Summary ---
echo "============================================"
echo " Validation Summary"
echo "============================================"
echo ""
echo "  Flow: build → pack → release → publish"
echo ""
echo "  1. azd x build --all         → bin/ (6 binaries)"
echo "  2. azd x pack -o ./artifacts → artifacts/ (6 archives)"
echo "  3. azd x release --repo $REPO → GitHub release with tag + uploaded archives"
echo "  4. azd x publish --repo $REPO → registry.json updated with URLs + checksums"
echo ""
echo "  After publish, registry.json must be committed back to the repo."
echo "  The GitHub Actions workflow creates a PR with auto-merge for this."
echo ""

# Cleanup
if [ "${CLEANUP:-true}" = "true" ]; then
    echo "  Cleaning up artifacts/ directory..."
    rm -rf artifacts/
    echo "  ✓ Cleaned up"
fi

echo ""
echo "Done."
