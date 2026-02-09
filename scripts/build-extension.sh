#!/usr/bin/env bash
set -euo pipefail

REPO_ROOT="$(cd "$(dirname "$0")/.." && pwd)"
EXT_DIR="$REPO_ROOT/editors/vscode"

echo "==> Building golar binary..."
cd "$REPO_ROOT"
go build -o "$EXT_DIR/lib/tsgo" ./thirdparty/typescript-go/cmd/tsgo

echo "==> Installing extension dependencies..."
cd "$EXT_DIR"
bun install

echo "==> Bundling extension..."
bun run bundle

echo "==> Packaging VSIX..."
npx @vscode/vsce package --no-dependencies

echo ""
echo "Done! Install with:"
echo "  code --install-extension editors/vscode/golar-*.vsix"
