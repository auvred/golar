# Golar Development TODO

This document contains the roadmap and detailed tasks for Golar development. Tasks are organized by priority and include enough context for delegation to other developers/agents.

## Current Status

Golar can now parse Vue SFCs and generate TypeScript service code for type checking. Basic directives (`v-if`, `v-for`, `v-on`) are supported. The main remaining work is achieving 1:1 compatibility with Volar's codegen output.

### Recent Fixes (Jan 2025)
- Fixed parser `InnerLoc` bug for SFC root elements
- Fixed entity handling (`&&`) in attribute values causing parse failures  
- Fixed ASI issues with interpolation expressions followed by blocks

---

## High Priority: Volar Codegen Compatibility

**Goal**: Achieve 1:1 output compatibility with Volar's codegen so that Golar produces identical TypeScript service code.

### Task: Align Variable Naming

**Current**: Golar uses `__VLS_Ctx`, Volar uses `__VLS_ctx`

**Files to modify**: `internal/vue/codegen/template.go`, `internal/vue/codegen/script.go`

**Steps**:
1. Run comparison: `go test ./internal/vue/tests/volar_comparison/... -v`
2. Identify all `__VLS_*` variable naming differences
3. Update Golar to match Volar's naming conventions
4. Verify tests pass

### Task: Match Type Helper Structure

**Current**: Golar generates simpler type helpers, Volar generates more comprehensive ones

**Files to modify**: `internal/vue/codegen/codegen.go` (GlobalTypes constant)

**Volar generates**:
```typescript
type __VLS_SetupExposed = import('vue').ShallowUnwrapRef<{...}>;
const __VLS_ctx = {
  ...{} as import('vue').ComponentPublicInstance,
  ...{} as __VLS_SetupExposed,
};
type __VLS_LocalComponents = __VLS_SetupExposed;
type __VLS_GlobalComponents = import('vue').GlobalComponents;
let __VLS_components!: __VLS_LocalComponents & __VLS_GlobalComponents;
let __VLS_intrinsics!: import('vue/jsx-runtime').JSX.IntrinsicElements;
```

**Golar currently generates**:
```typescript
type __VLS_SetupExposed = {
  msg: __VLS_UnwrapRef<typeof msg>
}
const __VLS_Ctx = {
  ...{} as unknown as __VLS_SetupExposed,
  ...{} as unknown as import('vue').ComponentPublicInstance,
}
```

**Steps**:
1. Study Volar's `packages/language-core/lib/codegen/script/` directory
2. Update `GlobalTypes` in `codegen.go`
3. Update script codegen to match structure
4. Run comparison tests

### Task: Element/Intrinsic Handling

**Current**: Golar doesn't generate element type checks

**Volar generates**:
```typescript
__VLS_asFunctionalElement1(__VLS_intrinsics.div, __VLS_intrinsics.div)({});
```

**Steps**:
1. Add intrinsic element type generation in template codegen
2. Study Volar's `packages/language-core/lib/codegen/template/element.ts`
3. Implement `__VLS_asFunctionalElement` helper

### Task: Expression Wrapping Style

**Current**: Golar uses `;( expr );`, Volar uses `( expr );`

**Files**: `internal/vue/codegen/template.go`

**Steps**:
1. Change interpolation output from `;( expr );` to `( expr );`
2. Ensure this doesn't reintroduce ASI issues
3. Run all tests

---

## Medium Priority: Missing Vue Features

### Task: Implement `defineEmits` Support

**Status**: Not implemented

**What's needed**:
1. Parse `defineEmits<{...}>()` or `defineEmits(['event1', 'event2'])`
2. Generate emit type definitions
3. Type-check `emit('eventName', payload)` calls in templates

**Reference**: Volar's `packages/language-core/lib/codegen/script/scriptSetup.ts`

### Task: Implement `v-model` Support

**Status**: Not implemented

**What's needed**:
1. Parse `v-model="value"` and `v-model:propName="value"`
2. Generate both getter and setter code
3. Handle modifiers (`.lazy`, `.number`, `.trim`)

**Reference**: Volar's `packages/language-core/lib/codegen/template/vModel.ts`

### Task: Implement Component Type Inference

**Status**: Not implemented

**What's needed**:
1. When using `<MyComponent :prop="value">`, infer prop types from component
2. Check that props match component's `defineProps`
3. Check slot content types

**This is complex** - requires:
- Resolving component imports
- Extracting prop/emit/slot types from component definitions
- Generating proper type constraints

### Task: Implement `v-slot` / Slot Props

**Status**: Partial (syntax parsed, types not inferred)

**What's needed**:
1. `<template #default="{ item }">` should type `item` from slot definition
2. Requires component type inference first

---

## Low Priority: Additional Features

### Task: Generic Components

**Syntax**: `<script setup generic="T extends string">`

**What's needed**:
1. Parse generic parameter from script setup
2. Pass generic to component type definition
3. Allow generic usage in props/template

### Task: `defineExpose` Support

**What's needed**:
1. Parse `defineExpose({ method, value })`
2. Generate exposed type for component refs

### Task: CSS `v-bind()` Support

**What's needed**:
1. Parse `v-bind(variable)` in `<style>` blocks
2. Generate type check for the bound variable

---

## Testing Tasks

### Task: Expand Volar Comparison Test Cases

**File**: `internal/vue/tests/volar_comparison/compare_test.go`

**Add test cases for**:
- [ ] Nested v-for loops
- [ ] v-if/v-else chains
- [ ] Multiple script blocks
- [ ] Component imports
- [ ] defineProps with defaults
- [ ] Complex template expressions

### Task: Add Baseline Tests

Create snapshot/baseline tests similar to TypeScript's test infrastructure:
1. Input `.vue` files in `testdata/`
2. Expected service code output in `testdata/baselines/`
3. Test runner compares actual vs expected

---

## Commands Reference

```bash
# Build Golar
go build -o golar/tsgo ./typescript-go/cmd/tsgo

# Run Vue tests
go test ./internal/vue/tests/... -v

# Run Volar comparison (requires setup first)
./scripts/setup-volar-reference.sh
go test ./internal/vue/tests/volar_comparison/... -v

# Generate Golar service code for a file
go run ./cmd/test_codegen path/to/file.vue --service

# Generate Volar service code for comparison
cd .reference && bun run generate_volar.ts path/to/file.vue

# Type-check a Vue project with Golar
./golar/tsgo -p path/to/tsconfig.json --noEmit

# Compare error counts
./golar/tsgo -p tsconfig.json --noEmit 2>&1 | grep -c "error TS"
bunx tsgo -p tsconfig.json --noEmit 2>&1 | grep -c "error TS"
```

---

## Architecture Notes for New Contributors

### How Golar Works

1. **Parser** (`internal/vue/parser/`) tokenizes Vue SFCs into AST
2. **Codegen** (`internal/vue/codegen/`) transforms AST to TypeScript "service code"
3. **Mapping** (`internal/mapping/`) tracks source-to-service position mappings
4. **Integration** (`internal/golar/golar.go`) hooks into typescript-go via `GolarCallbacks`

### Key Files

- `internal/vue/parser/parser.go` - Main parser, handles SFC structure
- `internal/vue/parser/tokenizer.go` - HTML tokenizer with Vue extensions
- `internal/vue/codegen/template.go` - Template expression codegen
- `internal/vue/codegen/script.go` - Script block codegen
- `internal/golar/golar.go` - Integration with typescript-go

### Debugging Tips

1. Use `go run ./cmd/test_codegen file.vue` to see parsed AST
2. Use `go run ./cmd/test_codegen file.vue --service` to see generated TS
3. Compare with `cd .reference && bun run generate_volar.ts file.vue`
4. Add `println()` statements in parser/codegen for tracing (remove before commit)

### Common Gotchas

1. **Entity handling**: `&&` in attributes was breaking parsing - entity decoder is disabled
2. **ASI issues**: Always add explicit `;` after generated expressions
3. **Stack state**: When `onCloseTag` is called, element is already removed from stack
4. **Syntax errors block semantic errors**: Fix codegen syntax errors first
