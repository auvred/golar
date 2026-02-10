# Golar Development TODO

This document contains the roadmap and detailed tasks for Golar development. Tasks are organized by priority and include enough context for delegation to other developers/agents.

## Current Status

**Volar Compatibility: ~95% type checking accuracy, 87% exact codegen match**

- ✅ **Exact Match Tests**: 7/8 passing (simple, component-props, event-handlers, medium-complex, v-for-slots, v-if-else, dynamic-component)
- ✅ **Volar Comparison Tests**: ~120 passing, 3 failing (2 out of scope: Pug/non-TS languages)
- ✅ All core directives work: `v-if`, `v-for`, `v-on`, `v-bind`, `v-slot`
- ✅ Component resolution (imported + global)
- ✅ Dynamic components `<component :is="expr">`
- ✅ Element type checking with `__VLS_asFunctionalElement1`
- ⚠️ **Missing**: `defineEmits`, `defineSlots`, `defineExpose`, `defineModel`, `v-model`

**Path to 100%**: The biggest gap is `defineEmits` type capture, which blocks full event type inference.

### Recent Fixes (Feb 2025)
- **Implemented dynamic component support**: `<component :is="expr">` now works with both simple expressions and ternary conditionals (e.g., `Math.random() > 0.5 ? Foo : Bar`)
- **Optimized export pattern**: Removed unnecessary `__VLS_base` intermediate when components have no slots, matching Volar's output
- **Added Makefile**: Build automation with `make build-binary`, `make build-extension`, `make install-extension`, `make test`, `make clean`
- **Fixed build mode (`-b`) panics**: Build mode's orchestrator now wraps compiler host with Golar callbacks, preventing `ScriptKindUnknown` panics for `.vue` files
- **Fixed empty `<script setup>` panic**: Components with `<script setup lang="ts"></script>` (no content) no longer crash the codegen
- **Fixed template-only component codegen**: Components with no `<script>` tag now properly generate `__VLS_SetupExposed` type for template component resolution
- **Extension registration**: `.vue` extension is now unconditionally registered in `init()`, ensuring the file loader includes `.vue` files in the program regardless of execution mode

### Previous Fixes (Jan 2025)
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

## ~~🎯 Current Sprint: v-model Implementation~~ - COMPLETED ✅

**Goal**: Implement `v-model` directive codegen for two-way binding

**Status**: ✅ IMPLEMENTED and tested across Vue 3.2-3.6!

**What was done**:
1. ✅ Parse `v-model="value"` and `v-model:propName="value"` directives
2. ✅ Generate standalone getter expression for default v-model: `(__VLS_ctx.value);`
3. ✅ Generate prop binding for v-model with argument: `{ value: (__VLS_ctx.value) }`
4. ✅ Matches Volar's exact output pattern
5. ✅ All tests pass (TestVModelBasic, TestVModelWithArg, TestVModelCheckbox, TestVModelMultiple)
6. ✅ Supports native elements (input, textarea, select)

**Note**: Modifiers (`.lazy`, `.number`, `.trim`) affect runtime behavior only and don't change type checking codegen.

**Files modified**:
- `internal/vue/codegen/template.go` - Added `generateVModelGetter()` and skip logic
- `internal/vue/tests/vmodel_test.go` - Added comprehensive tests

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

### ~~Task: Implement `defineEmits` Support~~ - COMPLETED ✅

**Status**: ✅ IMPLEMENTED - Full emit type inference working

**What was done**:
1. ✅ Captures `defineEmits` return value from `const emit = defineEmits<T>()`
2. ✅ Creates `__VLS_EmitProps` type from emit definition
3. ✅ Adds `{ $emit: typeof emit }` to `__VLS_ctx`
4. ✅ Includes `emits: {} as __VLS_NormalizeEmits<typeof emit>` in component export
5. ✅ Enables `__VLS_NormalizeComponentEvent` to properly type-check event handlers
6. ✅ All tests pass across Vue 3.2-3.6

**Files**: `internal/vue/codegen/script.go`, `internal/vue/tests/defineemits_test.go`

### ~~Task: Implement `v-model` Support~~ - COMPLETED ✅

**Status**: ✅ IMPLEMENTED - All tests pass

**What was done**:
1. ✅ Parse `v-model="value"` and `v-model:propName="value"`
2. ✅ Generate standalone getter for default v-model
3. ✅ Generate prop binding for v-model with argument
4. ✅ Handles different input types (text, checkbox, radio, select)
5. ✅ Matches Volar's exact codegen output

**Files**: `internal/vue/codegen/template.go`, `internal/vue/tests/vmodel_test.go`

### ~~Task: Implement Component Type Inference~~ - COMPLETED ✅

**Status**: ✅ FULLY IMPLEMENTED

**What's done**:
- ✅ Imported components use direct reference for type inference
- ✅ Global components use `__VLS_WithComponent` lookup
- ✅ Props are passed through `__VLS_asFunctionalComponent1` for type checking
- ✅ Dynamic components (`<component :is="expr">`) work with expression-based component resolution
- ✅ Emit type inference (via `defineEmits`)
- ✅ Component ref types (via `defineExpose`)

### ~~Task: Implement `v-slot` / Slot Props~~ - COMPLETED ✅

**Status**: ✅ WORKING

**What works**:
- ✅ `v-slot="props"` generates proper scope variable
- ✅ `v-slot="{ item }: { item: Type }"` typed slot props work via Volar's callback pattern
- ✅ Default slot content is rendered
- ✅ Slot prop type inference from component's `defineSlots`

### ~~Task: Implement `defineExpose`~~ - COMPLETED ✅

**Status**: ✅ IMPLEMENTED - All tests pass

### ~~Task: Implement `defineSlots`~~ - COMPLETED ✅

**Status**: ✅ IMPLEMENTED - Slot type definitions working

---

## Known Issues / Technical Debt

### Component Emit Type Inference Deferred

Full emit type inference using `__VLS_NormalizeComponentEvent` is currently disabled because:
1. It requires `defineEmits` support to capture emit types
2. Without emit types, `__VLS_ResolveEmits` returns `{}`, causing `keyof Emits` to be `never`
3. This triggers TS2344 constraint errors for any event handler

**Current workaround**: Event handlers are generated as standalone expressions that type-check the handler function but don't verify the event name exists on the component.

**To fix**: Implement `defineEmits` capture in script codegen (see High Priority section).

### ~~Dynamic Components Not Supported~~ - COMPLETED

~~`<component :is="SomeComponent">` is not yet handled. The parser treats `component` as a regular element.~~

**Status**: Implemented (Feb 2025)
- `<component :is="expr">` fully supported
- Works with simple expressions: `<component :is="Foo">`
- Works with ternary expressions: `<component :is="Math.random() > 0.5 ? Foo : Bar">`
- Properly filters `:is` prop from generated component props
- Generates correct `const __VLS_N = (expression);` pattern matching Volar

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
# Build Golar (using Makefile)
make build-binary           # Build golar/tsgo binary
make build-extension        # Build VS Code extension (.vsix)
make install-extension      # Install extension in VS Code
make test                   # Run all tests
make clean                  # Clean build artifacts

# Or manually:
go build -o golar/tsgo ./thirdparty/typescript-go/cmd/tsgo

# Run Vue tests
go test ./internal/vue/tests/... -v -count=1

# Run Volar comparison (requires setup first)
./scripts/setup-volar-reference.sh
go test ./internal/vue/tests/volar_comparison/... -v

# Generate Golar service code for a file
go run ./cmd/test_codegen path/to/file.vue --service

# Generate Volar service code for comparison
cd .reference && bun run generate_volar.ts path/to/file.vue

# Type-check a Vue project with Golar (regular mode)
./golar/tsgo -p path/to/tsconfig.json --noEmit

# Type-check a Vue project with Golar (build mode for monorepos)
./golar/tsgo -b --noEmit

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
