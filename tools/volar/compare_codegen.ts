#!/usr/bin/env bun
/**
 * Compare Golar codegen output against Volar reference
 *
 * Prerequisites:
 *   Run: ./scripts/setup-volar-reference.sh
 *
 * Usage:
 *   bun run tools/volar/compare_codegen.ts <vue-file>
 *
 * This will:
 *   1. Generate Volar output using the official language-tools
 *   2. Generate Golar output using our Go implementation
 *   3. Show a unified diff of the differences
 *
 * Example:
 *   bun run tools/volar/compare_codegen.ts ./tests/basic-vue-ts/src/App.vue
 */

import * as fs from "fs";
import * as path from "path";
import * as ts from "typescript";
import { execSync, spawnSync } from "child_process";

const repoRoot = path.resolve(__dirname, "../..");
const languageToolsPath = path.join(repoRoot, ".reference/language-tools");
const languageCorePath = path.join(languageToolsPath, "packages/language-core");

async function getVolarOutput(filePath: string): Promise<string> {
  if (!fs.existsSync(languageCorePath)) {
    throw new Error(
      "Volar language-tools not found. Run: ./scripts/setup-volar-reference.sh"
    );
  }

  const languageCore = await import(
    path.join(languageCorePath, "lib/index.js")
  );
  const { forEachEmbeddedCode } = await import("@volar/language-core");

  const content = fs.readFileSync(filePath, "utf-8");

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

  const virtualCode = plugin.createVirtualCode(filePath, "vue", snapshot);

  if (!virtualCode) {
    throw new Error("Failed to create virtual code");
  }

  for (const code of forEachEmbeddedCode(virtualCode)) {
    if (code.id.startsWith("script_")) {
      return code.snapshot.getText(0, code.snapshot.getLength());
    }
  }

  return "";
}

function getGolarOutput(filePath: string): string {
  const result = spawnSync(
    "go",
    ["run", "./cmd/test_codegen", filePath, "--service"],
    {
      cwd: repoRoot,
      encoding: "utf-8",
      maxBuffer: 10 * 1024 * 1024, // 10MB buffer
    }
  );

  if (result.error) {
    throw new Error(`Failed to run Golar: ${result.error.message}`);
  }

  if (result.status !== 0) {
    throw new Error(`Golar exited with code ${result.status}: ${result.stderr}`);
  }

  return result.stdout;
}

async function main() {
  const filePath = process.argv[2];
  if (!filePath) {
    console.error("Usage: bun run tools/volar/compare_codegen.ts <vue-file>");
    process.exit(1);
  }

  const absolutePath = path.resolve(filePath);
  if (!fs.existsSync(absolutePath)) {
    console.error(`File not found: ${absolutePath}`);
    process.exit(1);
  }

  console.log("============================================");
  console.log("Comparing codegen output");
  console.log("============================================");
  console.log(`File: ${absolutePath}`);
  console.log("");

  let volarOutput: string;
  let golarOutput: string;

  try {
    process.stdout.write("Generating Volar output... ");
    volarOutput = await getVolarOutput(absolutePath);
    console.log("done");
  } catch (err) {
    console.log("FAILED");
    console.error(err);
    process.exit(1);
  }

  try {
    process.stdout.write("Generating Golar output... ");
    golarOutput = getGolarOutput(absolutePath);
    console.log("done");
  } catch (err) {
    console.log("FAILED");
    console.error(err);
    process.exit(1);
  }

  // Write to temp files for diff
  const tmpDir = "/tmp/golar-compare";
  fs.mkdirSync(tmpDir, { recursive: true });
  const volarFile = path.join(tmpDir, "volar.ts");
  const golarFile = path.join(tmpDir, "golar.ts");
  fs.writeFileSync(volarFile, volarOutput);
  fs.writeFileSync(golarFile, golarOutput);

  console.log("");
  console.log("--- Diff (Volar vs Golar) ---");
  console.log("");

  const diffResult = spawnSync("diff", ["-u", volarFile, golarFile], {
    encoding: "utf-8",
  });

  if (diffResult.status === 0) {
    console.log("✓ No differences! Golar output matches Volar.");
  } else {
    console.log(diffResult.stdout);
    console.log("");
    console.log("✗ Differences found (see above)");
    console.log("");
    console.log(`Temp files saved to:`);
    console.log(`  Volar: ${volarFile}`);
    console.log(`  Golar: ${golarFile}`);
  }
}

main();
