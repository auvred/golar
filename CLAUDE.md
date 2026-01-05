# Golar Language Server

Golar is a native Vue language support implementation for typescript-go (tsgo). The goal is to provide first-class Vue Single File Component (SFC) support directly in the TypeScript compiler, enabling fast, accurate type checking and language server features for Vue projects.

The project is architecturally inspired by Volar.js but implemented in Go. It integrates with typescript-go (a native TypeScript compiler port) to provide type-aware language server features.

The project consists of:

- **Root module** (`github.com/auvred/golar`) - Contains Golar-specific framework code for Vue support
- **typescript-go submodule** - A native port of the TypeScript compiler and language server written in Go
- **Shim layer** - Wrapper packages in `shim/` that expose typescript-go internal APIs for use by Golar

## Repository Structure

```sh
golar/
├── golar/                    # Public API exports
├── internal/
│   ├── golar/               # Core Golar language integration
│   ├── vue/                 # Vue.js specific implementation
│   │   ├── parser/          # Vue template/SFC parser
│   │   ├── codegen/         # TypeScript code generation from Vue templates
│   │   ├── ast/             # Vue AST definitions
│   │   └── tests/           # Vue feature tests (v-if, v-for, diagnostics, etc.)
│   ├── mapping/             # Source-to-service position mapping
│   ├── utils/               # Utilities (overlay VFS, etc.)
│   └── collections/         # Collection utilities
├── shim/                    # Generated wrappers around typescript-go internals
├── typescript-go/           # Submodule: native TypeScript compiler/LSP
│   ├── internal/            # TypeScript compiler internals (ast, checker, binder, etc.)
│   ├── cmd/tsgo/           # Main CLI entry point
│   └── _submodules/TypeScript/  # Reference TypeScript implementation
└── tools/                   # Code generation tools (gen_shims)
```

## Build and Development Commands

This project uses Go modules with a workspace configuration. The typescript-go submodule uses `hereby` (a TypeScript build tool) for builds and tests.

**Use `bun` for all JavaScript/TypeScript package management** (not npm or pnpm).

### Initial Setup

```bash
# Clone submodules and apply patches
git submodule update --init
cd typescript-go
git am --3way --no-gpg-sign ../patches/*.patch
cd ..
```

### Building

```bash
# Build the golar binary (tsgo with Golar extensions)
go build -o golar ./typescript-go/cmd/tsgo

# Or use hereby in typescript-go directory
cd typescript-go
npx hereby build
```

### Testing

```bash
cd typescript-go

# Run all tests
npx hereby test

# Run a specific compiler test
go test -run='TestSubmodule/<test name>' ./internal/testrunner  # For tests in _submodules/TypeScript
go test -run='TestLocal/<test name>' ./internal/testrunner      # For tests in testdata/tests/cases

# Run Vue-specific tests (from root)
cd /Volumes/repos/golar
go test ./internal/vue/tests/...
```

### Code Quality

```bash
cd typescript-go

# Format code (must run before committing)
npx hereby format

# Lint code (must pass before committing)
npx hereby lint

# Accept test baselines after changes
npx hereby baseline-accept
```

## Architecture

### Golar Language Integration

Golar works by transforming framework-specific files (like `.vue`) into TypeScript service code that typescript-go can analyze:

1. **Parser** (`internal/vue/parser`) - Tokenizes and parses Vue SFC files into an AST
2. **Codegen** (`internal/vue/codegen`) - Generates TypeScript code from Vue templates with mappings
3. **Mapping** (`internal/mapping`) - Tracks positions between source (`.vue`) and generated TypeScript
4. **Language Integration** (`internal/golar/golar.go`) - Hooks into typescript-go via `GolarCallbacks`

Key integration points in `internal/golar/golar.go`:

- `compilerHostProxy.GetSourceFile()` - Intercepts `.vue` file reads, parses and generates TS
- `diagnosticProxy` - Maps TypeScript diagnostics back to source positions in `.vue` files
- `WrapFS()` - Overlays virtual files (e.g., `vue-global-types.d.ts`) onto the file system

### Shim Layer

The `shim/` directory contains generated wrapper packages that expose typescript-go's internal APIs. These are created by `tools/gen_shims` and should not be manually edited. To update:

```bash
./tools/update-typescript-go-shims.sh
```

### TypeScript-Go Reference

When implementing features or fixing bugs, `_submodules/TypeScript` serves as the reference implementation. The code in `typescript-go/internal/` is a Go port of the TypeScript codebase. Always consult the TypeScript source when the Go behavior differs or is incomplete.

## Testing Framework

### Compiler Tests

TypeScript compiler tests are written as `.ts`/`.tsx` files in `testdata/tests/cases/compiler/` with special comment directives:

```typescript
// @target: esnext
// @module: preserve
// @strict: true

// @filename: file1.ts
export interface Person {
    name: string;
}

// @filename: file2.ts
import { Person } from "./file1";
```

**Always enable `@strict: true` for new tests unless testing non-strict behavior.**

Test outputs are generated in `testdata/baselines/local/` and compared against `testdata/baselines/reference/`. Use `npx hereby baseline-accept` to accept new baselines.

### Vue Tests

Vue-specific tests use the fourslash harness (see `internal/vue/tests/`). These test language service features like:

- Diagnostics in templates (`diagnostic_test.go`)
- Quick info/hover (`quickinfo_test.go`)
- Directives like `v-if` and `v-for` (`vif_test.go`, `vfor_test.go`)

## Development Workflow

### For TypeScript-Go Work

1. Write minimal test case in `testdata/tests/cases/compiler/`
2. Run test to verify failure (or baseline)
3. Implement fix in relevant `internal/` package
4. Run tests and accept baselines
5. Format and lint before committing

### For Golar/Vue Work

1. Add test in `internal/vue/tests/`
2. Implement parser/codegen changes
3. Update mappings if needed
4. Run tests: `go test ./internal/vue/tests/...`
5. Verify end-to-end with `go build -o golar ./typescript-go/cmd/tsgo`

## Important Constraints

- **Do not remove debug assertions or panic calls** - Existing assertions are correct
- **Do not add/change dependencies** unless explicitly requested
- **Shim files are auto-generated** - Do not manually edit files in `shim/`
- **Always run format and lint** - CI will reject PRs if these fail
- **Reference TypeScript source** - `_submodules/TypeScript` is the behavioral reference
- **Always choose correctness over shortcuts** - This is an open source tool used by many developers. When faced with a choice between a quick/lazy solution and the proper/correct approach, always implement it the proper way. Correctness and maintainability are paramount.

## Go Workspace

This project uses Go 1.25 workspaces (`go.work`):

- Root module: `github.com/auvred/golar`
- Submodule: `github.com/microsoft/typescript-go`

Replace directives in `go.mod` map typescript-go shim imports to local `./shim/` directories.

## Patching typescript-go

The `patches/` directory contains Git patches that extend typescript-go with Golar integration points. When making changes to files in `typescript-go/`:

1. Make changes in the `typescript-go/` submodule
2. Amend the existing Golar patch commit: `git commit --amend`
3. Regenerate the patch: `git format-patch -1 HEAD -o ../patches/`

The patch adds hooks for:
- `GolarCallbacks` interface in `internal/golarext/golarext.go`
- Compiler host wrapping for custom file parsing
- Diagnostic adjustment for source mapping
- LSP integration points

**All typescript-go execution modes must use Golar callbacks** - This includes regular compilation (`tsc.go`), build mode (`build/orchestrator.go`), watch mode, and LSP. Missing integration in any mode will cause `.vue` files to be parsed as raw TypeScript.

## Volar.js as Reference Implementation

When implementing Vue language features, **always consult Volar.js source code** for the correct behavior:
- Repository: https://github.com/vuejs/language-tools
- Key packages: `packages/language-core/lib/codegen/`

Volar's approach should be followed for:
- Template expression handling (interpolations, directives)
- Event handler codegen (`v-on`/`@` - compound vs simple expressions)
- Slot handling (`v-slot`/`#`)
- Two-way binding (`v-model`)
- Component type inference

## Codegen Architecture

The codegen transforms `.vue` files into TypeScript "service code" that preserves:

1. **Line correspondence** - Service code maintains same line count as source for accurate error positions
2. **Source mappings** - Character-level mappings between source and service text
3. **Scope tracking** - Template scopes for `v-for` variables, event `$event`, etc.

Key codegen patterns:
- **Interpolations** `{{ expr }}` → `;( expr )` with identifier prefixing
- **Conditionals** `v-if`/`v-else-if`/`v-else` → `if/else if/else` blocks
- **Loops** `v-for` → block scope with destructuring from `__VLS_vFor()` helper
- **Events** `@click="handler"` → `;(handler)` or `;((...[$event]) => { ... })` for compound expressions

### Compound vs Simple Event Expressions

Event handlers require special handling based on expression type:
- **Simple**: Function reference or property access (`handleClick`, `obj.method`) → use directly
- **Compound**: Inline statement (`count++`, `emit('event')`) → wrap in arrow function with `$event` parameter

## Common Pitfalls

1. **Go string literals** - Use double quotes `"...\n"` for strings with escapes, not backticks `` `...\n` `` which treat `\n` as literal characters

2. **Multiple execution modes** - typescript-go has several entry points (regular compile, build mode `-b`, watch mode `-w`, LSP). Each must integrate Golar callbacks.

3. **Diagnostic position mapping** - Errors from generated code must map back to source positions. Unmapped positions will show incorrect locations.

4. **AST field additions** - When adding fields to Vue AST nodes (like `DirectiveNode.Arg`), update both the struct definition and the parser that populates it.

5. **Parser `InnerLoc` for SFC elements** - When closing SFC root elements (script, template, style), the element has already been removed from the stack when `onCloseTag()` is called. Check `len(p.stack) == 0` not `inSFCRoot()` which checks stack length incorrectly.

6. **Entity handling in attribute values** - The tokenizer's entity decoder (`stateInEntity`) is not fully implemented. When it sees `&` in attribute values (like `v-if="a && b"`), it enters `StateInEntity` but never exits, causing parsing to fail. Entity handling is currently disabled for attribute values since Vue templates use JavaScript operators like `&&`.

7. **ASI (Automatic Semicolon Insertion) in codegen** - Generated expressions like `;(expr)` followed by `{` on the next line can be misinterpreted by TypeScript as arrow functions. Always add explicit semicolons: `;(expr);` not `;(expr)`.

8. **Syntax errors block semantic diagnostics** - TypeScript's `GetDiagnosticsOfAnyProgram()` exits early if there are syntax errors (TS1xxx), preventing semantic errors (TS2xxx) from being reported. Fix syntax errors in codegen first.

## Volar Comparison Testing

The `.reference/` directory (gitignored) contains the official Volar/Vue language-tools for comparing Golar's codegen output against the reference implementation.

### Setup

```bash
./scripts/setup-volar-reference.sh
```

This script:
1. Clones vuejs/language-tools into `.reference/language-tools/`
2. Installs dependencies with bun
3. Creates `generate_volar.ts` script for generating reference output

### Running Comparison Tests

```bash
go test ./internal/vue/tests/volar_comparison/... -v
```

### Generating Volar Output Manually

```bash
cd .reference
bun run generate_volar.ts <path-to-vue-file>
```

### Generating Golar Output Manually

```bash
go run ./cmd/test_codegen <path-to-vue-file> --service
```

The goal is 1:1 compatibility with Volar's codegen output. Key differences to address:
- Variable naming (`__VLS_ctx` vs `__VLS_Ctx`)
- Type helper structure and imports
- Element/component intrinsic handling
- Expression wrapping style

## Debug Tools

### test_codegen

A CLI tool for inspecting parser and codegen output:

```bash
# Show AST structure
go run ./cmd/test_codegen <vue-file>

# Show generated TypeScript service code  
go run ./cmd/test_codegen <vue-file> --service
```

### Comparing with Official tsgo

```bash
# Build Golar
go build -o golar/tsgo ./typescript-go/cmd/tsgo

# Run Golar on a project
./golar/tsgo -p <tsconfig-path> --noEmit

# Compare error count with official tsgo
bunx tsgo -p <tsconfig-path> --noEmit 2>&1 | grep -c "error TS"
```
