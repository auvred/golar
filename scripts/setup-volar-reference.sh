#!/bin/bash
# Setup script for Volar reference implementation
# Used for comparing Golar's codegen output against the official Volar implementation

set -e

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(dirname "$SCRIPT_DIR")"
REF_DIR="$PROJECT_ROOT/.reference"

echo "Setting up Volar reference in $REF_DIR..."

mkdir -p "$REF_DIR"
cd "$REF_DIR"

if [ ! -d "language-tools" ]; then
    echo "Cloning vuejs/language-tools..."
    git clone --depth 1 https://github.com/vuejs/language-tools.git
fi

cd language-tools

echo "Installing dependencies..."
bun install

echo "Installing additional required packages..."
bun add typescript @vue/compiler-dom @vue/compiler-sfc alien-signals path-browserify muggle-string

# Create the generator script
cat > ../generate_volar.ts << 'EOF'
/**
 * Volar codegen reference generator
 * 
 * Generates TypeScript service code using the official Volar/Vue language-tools.
 * Usage: bun run generate_volar.ts <vue-file>
 */

import * as fs from 'fs';
import * as ts from 'typescript';
import { createVueLanguagePlugin, getDefaultCompilerOptions } from './language-tools/packages/language-core';
import { forEachEmbeddedCode } from '@volar/language-core';

function getVolarOutput(filePath: string): string {
  const content = fs.readFileSync(filePath, 'utf-8');
  
  const snapshot: ts.IScriptSnapshot = {
    getText: (start, end) => content.slice(start, end),
    getLength: () => content.length,
    getChangeRange: () => undefined,
  };
  
  const compilerOptions: ts.CompilerOptions = {
    target: ts.ScriptTarget.ESNext,
    module: ts.ModuleKind.ESNext,
    strict: true,
  };
  
  const vueCompilerOptions = getDefaultCompilerOptions();
  
  const plugin = createVueLanguagePlugin(
    ts,
    compilerOptions,
    vueCompilerOptions,
    (id: string) => id,
  );
  
  const virtualCode = plugin.createVirtualCode(filePath, 'vue', snapshot);
  
  if (!virtualCode) {
    throw new Error('Failed to create virtual code');
  }
  
  for (const code of forEachEmbeddedCode(virtualCode)) {
    if (code.id.startsWith('script_')) {
      return code.snapshot.getText(0, code.snapshot.getLength());
    }
  }
  
  return '';
}

const filePath = process.argv[2];
if (!filePath) {
  console.error('Usage: bun run generate_volar.ts <vue-file>');
  process.exit(1);
}

const output = getVolarOutput(filePath);
console.log(output);
EOF

echo ""
echo "Setup complete!"
echo ""
echo "To generate Volar output for a Vue file:"
echo "  cd .reference && bun run generate_volar.ts <path-to-vue-file>"
echo ""
echo "To run comparison tests:"
echo "  go test ./internal/vue/tests/volar_comparison/... -v"
