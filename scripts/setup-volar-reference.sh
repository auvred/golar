#!/bin/bash
# Setup script for Volar reference implementation
#
# This downloads and builds the official vuejs/language-tools repository
# which is used as a reference for comparing Golar's codegen output.
#
# The reference is stored in .reference/ which is gitignored.
# Only the tooling scripts in tools/volar/ are committed.

set -e

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(dirname "$SCRIPT_DIR")"
REF_DIR="$PROJECT_ROOT/.reference"
LANG_TOOLS_DIR="$REF_DIR/language-tools"

echo "============================================"
echo "Golar - Volar Reference Setup"
echo "============================================"
echo ""

# Check for bun
if ! command -v bun &> /dev/null; then
    echo "Error: bun is required but not installed."
    echo "Install it from: https://bun.sh"
    exit 1
fi

# Create reference directory
mkdir -p "$REF_DIR"

# Clone or update language-tools
if [ -d "$LANG_TOOLS_DIR" ]; then
    echo "Updating existing language-tools..."
    cd "$LANG_TOOLS_DIR"
    git fetch origin
    git reset --hard origin/master
else
    echo "Cloning vuejs/language-tools..."
    git clone --depth 1 https://github.com/vuejs/language-tools.git "$LANG_TOOLS_DIR"
    cd "$LANG_TOOLS_DIR"
fi

echo ""
echo "Installing dependencies with bun..."
bun install

echo ""
echo "Building language-tools..."
bun run build

# Install additional packages needed for our scripts in the .reference directory
cd "$REF_DIR"
echo ""
echo "Installing script dependencies..."
bun add typescript @volar/language-core

echo ""
echo "============================================"
echo "Setup complete!"
echo "============================================"
echo ""
echo "Usage:"
echo ""
echo "  Generate Volar output for a Vue file:"
echo "    bun run tools/volar/generate_volar.ts <vue-file>"
echo ""
echo "  Generate Golar output for a Vue file:"
echo "    go run ./cmd/test_codegen <vue-file> --service"
echo ""
echo "  Compare Golar vs Volar output:"
echo "    bun run tools/volar/compare_codegen.ts <vue-file>"
echo ""
echo "  Run Volar compatibility tests:"
echo "    go test ./internal/vue/tests/volar_comparison/... -v"
echo ""
