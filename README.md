# Golar

> Architecture is inspired by [@johnsoncodehk](https://github.com/johnsoncodehk)'s [Volar.js](https://github.com/volarjs/volar.js).

![Demo: LSP and CLI](./demo.gif)

## Project Scope

Golar is a **type checker and language server** for Vue Single File Components. The goal is to provide fast, accurate TypeScript type checking and IDE features (hover, go-to-definition, completions, etc.) for `.vue` files.

**What Golar does:**
- Parses Vue SFCs and generates TypeScript "service code" for type analysis
- Provides diagnostics (type errors) mapped back to `.vue` source positions
- Powers language server features via typescript-go's LSP

**What Golar does NOT do:**
- Compile/transform Vue SFCs for production (use Vite/Vue CLI for that)
- Bundle or emit JavaScript/TypeScript output
- Handle Vue runtime behavior or reactivity

This focused scope allows Golar to be fast and correct for its primary purpose: catching type errors in Vue code.

## Vue Feature Support Status

Golar is under active development. Below is the current status of Vue SFC feature support:

### Supported

| Feature | Status | Notes |
|---------|--------|-------|
| `<script setup>` basic | Supported | Basic script setup parsing and codegen |
| `<template>` parsing | Supported | HTML template parsing |
| Template interpolations `{{ }}` | Supported | Expression type checking with identifier prefixing |
| `v-if` / `v-else-if` / `v-else` | Supported | Conditional rendering with proper scoping |
| `v-for` | Supported | Loop iteration with destructuring support |
| `v-on` / `@` event handlers | Supported | Both simple (function ref) and compound (inline statements) with proper `$event` typing |
| `defineProps` (type-only) | Supported | `defineProps<{ prop: Type }>()` syntax |
| `withDefaults` | Supported | `withDefaults(defineProps<T>(), {...})` syntax |
| `.vue` module imports | Supported | Default export generated for all components |
| Template-only components | Supported | Components with only `<template>` block |
| Type hoisting | Supported | Type declarations hoisted for forward references |
| Type-only imports | Supported | Correctly handles `import type` and `{ type X }` |
| Ref/computed unwrapping | Supported | Auto-unwraps `Ref<T>` and `ComputedRef<T>` in templates |
| Diagnostic mapping | Supported | Errors map back to source `.vue` positions |
| Imported component resolution | Supported | Direct type reference for components imported in `<script setup>` |
| Global component resolution | Supported | `__VLS_WithComponent` lookup for globally registered components |
| Dynamic components | Supported | `<component :is="expr">` with expression or ternary support |
| `defineEmits` | Supported | Full emit type inference with `const emit = defineEmits<T>()` |
| `defineExpose` | Supported | Component ref type inference with `defineExpose({...})` |
| `defineModel` | Supported | Two-way binding helper for Vue 3.4+ |
| `defineSlots` | Supported | Slot type definitions with `defineSlots<T>()` |
| Component emit type inference | Supported | Full event handler type checking with emit types |
| `v-model` | Supported | Two-way binding on native elements (default and with arguments) |

### Partially Supported

| Feature | Status | Notes |
|---------|--------|-------|
| `v-slot` / `#` slots | Partial | Slot props with type annotations supported via Volar-compatible codegen |
| `v-bind` / `:` bindings | Partial | Basic attribute binding works |
| `defineProps` (runtime) | Partial | `defineProps({...})` syntax extracts prop names |

### Not Yet Supported

| Feature | Status | Notes |
|---------|--------|-------|
| Generic components | Not yet | `<script setup generic="T">` |
| CSS `v-bind()` | Not yet | Style variable binding |
| Pug templates | Not yet | `<template lang="pug">` |

### Known Limitations

- Some complex template expressions may not have accurate source mappings
- Some edge cases in `v-bind:class` with complex expressions may produce false positives

## Plans

* Full Vue support
* Angular
* Svelte
* Astro
* MDX
* Ember
* Type-aware linting + custom JS plugins?

## Quick Start

### VS Code Extension (recommended)

```bash
# Build everything and produce a .vsix (using Makefile)
make build-extension

# Install the extension
make install-extension

# Or manually:
code --install-extension editors/vscode/golar-*.vsix --force
```

Then disable **Vue - Official (Volar)** and **TypeScript Language Features** to avoid conflicts.

See [editors/vscode/README.md](./editors/vscode/README.md) for detailed setup.

### CLI Type Checking

```bash
# Build the golar binary (using Makefile)
make build-binary

# Or manually:
go build -o golar/tsgo ./thirdparty/typescript-go/cmd/tsgo

# Type-check a Vue project
./golar/tsgo -p path/to/tsconfig.json --noEmit
```

### Building from Source

```bash
# Clone and set up submodules
git submodule update --init
cd thirdparty/typescript-go
git am --3way --no-gpg-sign ../../patches/*.patch
cd ../..

# Build
go build -o golar/tsgo ./thirdparty/typescript-go/cmd/tsgo
```

## Testing

```bash
# Run all Vue tests (using Makefile)
make test

# Or manually:
go test ./internal/vue/tests/... -v -count=1

# Run Volar comparison tests (requires .reference/language-tools setup)
go test ./internal/vue/tests/volar_comparison/... -v
```

## Contributing

See [AGENTS.md](./AGENTS.md) for detailed development guidelines and architecture documentation.

## License

[MIT](./LICENSE)
