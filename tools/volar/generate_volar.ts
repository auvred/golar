#!/usr/bin/env bun
/**
 * Volar codegen reference generator
 *
 * Generates TypeScript service code using the official Volar/Vue language-tools.
 * This is used to compare Golar's output against the reference implementation.
 *
 * Prerequisites:
 *   Run: ./scripts/setup-volar-reference.sh
 *
 * Usage:
 *   bun run tools/volar/generate_volar.ts <vue-file>
 *
 * Example:
 *   bun run tools/volar/generate_volar.ts ./tests/basic-vue-ts/src/App.vue
 */

import * as fs from "fs";
import * as path from "path";
import * as ts from "typescript";

// Resolve paths relative to repo root
const repoRoot = path.resolve(__dirname, "../..");
const languageToolsPath = path.join(repoRoot, ".reference/language-tools");
const languageCorePath = path.join(languageToolsPath, "packages/language-core");

async function main() {
  const filePath = process.argv[2];
  if (!filePath) {
    console.error("Usage: bun run tools/volar/generate_volar.ts <vue-file>");
    process.exit(1);
  }

  const absolutePath = path.resolve(filePath);
  if (!fs.existsSync(absolutePath)) {
    console.error(`File not found: ${absolutePath}`);
    process.exit(1);
  }

  // Check if language-tools is available
  const indexTs = path.join(languageCorePath, "index.ts");
  if (!fs.existsSync(indexTs)) {
    console.error(`
Error: Volar language-tools not found.

Run the setup script first:
  ./scripts/setup-volar-reference.sh
`);
    process.exit(1);
  }

  try {
    // Import from language-tools source (bun can import .ts directly)
    const languageCore = await import(indexTs);
    const { forEachEmbeddedCode } = await import("@volar/language-core");

    const content = fs.readFileSync(absolutePath, "utf-8");

    const snapshot: ts.IScriptSnapshot = {
      getText: (start: number, end: number) => content.slice(start, end),
      getLength: () => content.length,
      getChangeRange: () => undefined,
    };

    const compilerOptions: ts.CompilerOptions = {
      target: ts.ScriptTarget.ESNext,
      module: ts.ModuleKind.ESNext,
      strict: true,
    };

    const vueCompilerOptions = languageCore.getDefaultCompilerOptions();

    const plugin = languageCore.createVueLanguagePlugin(
      ts,
      compilerOptions,
      vueCompilerOptions,
      (id: string) => id
    );

    const virtualCode = plugin.createVirtualCode(absolutePath, "vue", snapshot);

    if (!virtualCode) {
      console.error("Failed to create virtual code");
      process.exit(1);
    }

    // Find the script_ embedded code (main TypeScript output)
    for (const code of forEachEmbeddedCode(virtualCode)) {
      if (code.id.startsWith("script_")) {
        const output = code.snapshot.getText(0, code.snapshot.getLength());
        console.log(output);
        return;
      }
    }

    console.error("No script output found in generated code");
    process.exit(1);
  } catch (err) {
    console.error("Error generating Volar output:", err);
    process.exit(1);
  }
}

main();
