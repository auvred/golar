#!/bin/bash
# Sync test cases from Volar's test-workspace into our test suite
#
# This copies test files from .reference/language-tools/test-workspace/tsc/
# into internal/vue/tests/volar_comparison/testdata/
#
# Prerequisites:
#   Run: ./scripts/setup-volar-reference.sh

set -e

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(dirname "$SCRIPT_DIR")"
VOLAR_TESTS="$PROJECT_ROOT/.reference/language-tools/test-workspace/tsc"
OUR_TESTS="$PROJECT_ROOT/internal/vue/tests/volar_comparison/testdata"

if [ ! -d "$VOLAR_TESTS" ]; then
    echo "Error: Volar test-workspace not found at $VOLAR_TESTS"
    echo "Run: ./scripts/setup-volar-reference.sh"
    exit 1
fi

echo "============================================"
echo "Syncing Volar test cases"
echo "============================================"
echo ""
echo "Source: $VOLAR_TESTS"
echo "Dest:   $OUR_TESTS"
echo ""

# Create destination directory
mkdir -p "$OUR_TESTS"

# Copy shared utilities
if [ -f "$VOLAR_TESTS/shared.d.ts" ]; then
    cp "$VOLAR_TESTS/shared.d.ts" "$OUR_TESTS/"
    echo "Copied shared.d.ts"
fi

# Count tests
TOTAL=$(ls -d "$VOLAR_TESTS"/*/ 2>/dev/null | wc -l | tr -d ' ')
COPIED=0
SKIPPED=0

# Copy each test case directory
for test_dir in "$VOLAR_TESTS"/*/; do
    test_name=$(basename "$test_dir")
    
    # Skip _failed tests (known failures in Volar)
    if [[ "$test_name" == _failed* ]]; then
        ((SKIPPED++))
        continue
    fi
    
    dest_dir="$OUR_TESTS/$test_name"
    
    # Copy the test directory
    rm -rf "$dest_dir"
    cp -r "$test_dir" "$dest_dir"
    ((COPIED++))
done

echo ""
echo "============================================"
echo "Sync complete!"
echo "============================================"
echo ""
echo "Total test directories: $TOTAL"
echo "Copied: $COPIED"
echo "Skipped (failed tests): $SKIPPED"
echo ""
echo "Test cases are now in:"
echo "  $OUR_TESTS"
echo ""
echo "Run tests with:"
echo "  go test ./internal/vue/tests/volar_comparison/... -v"
echo ""
