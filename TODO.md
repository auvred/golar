# Golar Development TODO

This document contains the roadmap and detailed tasks for Golar development. Tasks are organized by priority and include enough context for delegation to other developers/agents.

## Current Status

Golar can now parse Vue SFCs and generate TypeScript service code for type checking. Basic directives (`v-if`, `v-for`, `v-on`) are supported. Element type checking with `__VLS_asFunctionalElement1` is now implemented. Component resolution distinguishes between imported and global components.

### Recent Fixes (Jan 2025)
- Fixed parser `InnerLoc` bug for SFC root elements
- Fixed entity handling (`&&`) in attribute values causing parse failures  
- Fixed ASI issues with interpolation expressions followed by blocks
- **Aligned variable naming**: Changed `__VLS_Ctx` to `__VLS_ctx` (lowercase) to match Volar
- **Updated SetupExposed type**: Now uses `import('vue').ShallowUnwrapRef<{...}>` instead of custom `__VLS_UnwrapRef`
- **Added component/directive declarations**: `__VLS_LocalComponents`, `__VLS_GlobalComponents`, `__VLS_components`, `__VLS_intrinsics`, `__VLS_directives`, `__VLS_StyleScopedClasses`
- **Added element type checking**: `__VLS_asFunctionalElement1(__VLS_intrinsics.TAG, ...)` calls for HTML elements
- **Added typed slot props**: `v-slot="{ item }: { item: Type }"` syntax now works via Volar's callback pattern
- **Fixed event handler typing**: Compound event handlers now have `[any]` type annotation for `$event`
- **Distinguished imported vs global components**: 
  - Imported components use direct reference: `const __VLS_0 = ComponentName || ComponentName`
  - Global components use type lookup: `let __VLS_0!: __VLS_WithComponent<...>`
- **Fixed setupConsts tracking**: Use AST `Text` field instead of position slicing to avoid trivia

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

### Task: Implement `defineEmits` Support - HIGH PRIORITY

**Status**: Not implemented - blocking component emit type inference

**What's needed**:
1. Capture `defineEmits` return value: `const __VLS_emit = defineEmits(['event1', 'event2'])`
2. Create `__VLS_EmitProps` type from emit definition
3. Add `{ $emit: typeof __VLS_emit }` to `__VLS_ctx`
4. Include `emits: {} as __VLS_NormalizeEmits<typeof __VLS_emit>` in component export
5. This enables `__VLS_NormalizeComponentEvent` to properly type-check event handlers

**Why it matters**: Without this, components that don't declare emits cause TS2344 errors when using `__VLS_NormalizeComponentEvent` because `keyof Emits` resolves to `never`.

**Reference**: 
- Volar's `packages/language-core/lib/codegen/script/scriptSetup.ts`
- See child.vue in `.reference/language-tools/test-workspace/tsc/#3100/`

### Task: Implement `v-model` Support

**Status**: Not implemented

**What's needed**:
1. Parse `v-model="value"` and `v-model:propName="value"`
2. Generate both getter and setter code
3. Handle modifiers (`.lazy`, `.number`, `.trim`)

**Reference**: Volar's `packages/language-core/lib/codegen/template/vModel.ts`

### Task: Implement Component Type Inference

**Status**: Partially implemented

**What's done**:
- Imported components use direct reference for type inference
- Global components use `__VLS_WithComponent` lookup
- Props are passed through `__VLS_asFunctionalComponent1` for type checking

**What's still needed**:
1. Emit type inference (requires `defineEmits` support)
2. Slot content type checking
3. Component ref types (`defineExpose`)

**This is complex** - requires:
- Extracting emit types from component definitions
- Generating proper type constraints for slots

### Task: Implement `v-slot` / Slot Props

**Status**: Partial - basic slot props work, type inference from component doesn't

**What works**:
- `v-slot="props"` generates proper scope variable
- `v-slot="{ item }: { item: Type }"` typed slot props work via Volar's callback pattern
- Default slot content is rendered

**What's needed**:
1. Infer slot prop types from component's `defineSlots` or slot definition
2. Type-check slot content against component's expected slot structure
3. Requires component type inference first

---

## Known Issues / Technical Debt

### Component Emit Type Inference Deferred

Full emit type inference using `__VLS_NormalizeComponentEvent` is currently disabled because:
1. It requires `defineEmits` support to capture emit types
2. Without emit types, `__VLS_ResolveEmits` returns `{}`, causing `keyof Emits` to be `never`
3. This triggers TS2344 constraint errors for any event handler

**Current workaround**: Event handlers are generated as standalone expressions that type-check the handler function but don't verify the event name exists on the component.

**To fix**: Implement `defineEmits` capture in script codegen (see High Priority section).

### Dynamic Components Not Supported

`<component :is="SomeComponent">` is not yet handled. The parser treats `component` as a regular element.

### Pug Templates Not Supported

`<template lang="pug">` parses but codegen doesn't handle pug syntax.

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
