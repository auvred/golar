# Golar Development TODO

This document contains the roadmap and detailed tasks for Golar development. Tasks are organized by priority and include enough context for delegation to other developers/agents.

## Current Status

Golar can now parse Vue SFCs and generate TypeScript service code for type checking. Basic directives (`v-if`, `v-for`, `v-on`) are supported. Element type checking with `__VLS_asFunctionalElement1` is now implemented.

### Recent Fixes (Jan 2025)
- Fixed parser `InnerLoc` bug for SFC root elements
- Fixed entity handling (`&&`) in attribute values causing parse failures  
- Fixed ASI issues with interpolation expressions followed by blocks
- **Aligned variable naming**: Changed `__VLS_Ctx` to `__VLS_ctx` (lowercase) to match Volar
- **Updated SetupExposed type**: Now uses `import('vue').ShallowUnwrapRef<{...}>` instead of custom `__VLS_UnwrapRef`
- **Added component/directive declarations**: `__VLS_LocalComponents`, `__VLS_GlobalComponents`, `__VLS_components`, `__VLS_intrinsics`, `__VLS_directives`, `__VLS_StyleScopedClasses`
- **Added element type checking**: `__VLS_asFunctionalElement1(__VLS_intrinsics.TAG, ...)` calls for HTML elements

---

## High Priority: Volar Codegen Compatibility

**Goal**: Achieve 1:1 output compatibility with Volar's codegen so that Golar produces identical TypeScript service code.

### Task: Align Variable Naming - COMPLETED

~~**Current**: Golar uses `__VLS_Ctx`, Volar uses `__VLS_ctx`~~

Updated to use `__VLS_ctx` (lowercase) throughout.

### Task: Match Type Helper Structure - COMPLETED

**Golar now generates**:
```typescript
type __VLS_SetupExposed = import('vue').ShallowUnwrapRef<{
  name: typeof name;
}>;
const __VLS_ctx = {
  ...{} as import('vue').ComponentPublicInstance,
  ...{} as __VLS_SetupExposed,
};
type __VLS_LocalComponents = __VLS_SetupExposed;
type __VLS_GlobalComponents = import('vue').GlobalComponents;
let __VLS_components!: __VLS_LocalComponents & __VLS_GlobalComponents;
let __VLS_intrinsics!: import('vue/jsx-runtime').JSX.IntrinsicElements;
type __VLS_LocalDirectives = __VLS_SetupExposed;
let __VLS_directives!: __VLS_LocalDirectives & import('vue').GlobalDirectives;
type __VLS_StyleScopedClasses = {};
```

### Task: Element/Intrinsic Handling - COMPLETED

**Golar now generates**:
```typescript
__VLS_asFunctionalElement1(__VLS_intrinsics.div, __VLS_intrinsics.div)({
...{ class: "my-class" },
...{ onClick: (handler) },
});
```

### Task: StyleScopedClasses Comments (Low Priority)

**Current**: Golar doesn't generate `/** @type {__VLS_StyleScopedClasses['className']} */` comments

**Volar generates** these comments after each element for CSS class type checking. This is a nice-to-have feature but not critical for type checking functionality.

### Task: Expression Wrapping Style (Low Priority)

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

This difference is intentional - the leading semicolon prevents ASI issues when an expression is followed by a block. The output is functionally equivalent.

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
