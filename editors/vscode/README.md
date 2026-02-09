# Golar - VS Code Extension

Fast, native Vue language server for VS Code, powered by [typescript-go](https://github.com/nicolo-ribaudo/tc39-proposal-type-annotations).

## Features

- Type checking for `.vue` Single File Components
- Hover information (types, documentation)
- Go-to-definition for imports and symbols
- Diagnostics mapped to source positions in `.vue` files
- Syntax highlighting for Vue templates (directives, interpolations)
- Supports `.ts`, `.tsx`, `.js`, `.jsx` alongside `.vue`

## Installation

### From Source (local build)

```bash
# From the repository root:
./scripts/build-extension.sh

# Install the produced .vsix:
code --install-extension editors/vscode/golar-*.vsix
```

### Using a Pre-built Binary

If you have a pre-built `tsgo` binary, point the extension at it:

1. Install the `.vsix` as above
2. In VS Code settings, set `golar.tsdk` to the directory containing your `tsgo` binary

## Setup

### Disable Conflicting Extensions

Golar replaces both Volar and the built-in TypeScript language features. To avoid conflicts:

1. **Disable "Vue - Official" (Volar)** - Golar provides Vue language features
2. **Disable "TypeScript Language Features"** - Golar provides TS/JS language features

You can use a [VS Code Profile](https://code.visualstudio.com/docs/editor/profiles) to keep a separate configuration for Golar without affecting your default setup.

### Verify It Works

1. Open a `.vue` file
2. Check the status bar for the Golar indicator
3. Hover over a variable in `<script setup>` to see type info
4. Check the "Golar" output channel for server logs

## Settings

| Setting | Description | Default |
|---------|-------------|---------|
| `golar.tsdk` | Path to directory containing the `tsgo` binary | Bundled binary |
| `golar.trace.server` | LSP trace level (`off`, `messages`, `verbose`) | `verbose` |
| `golar.pprofDir` | Directory to write pprof profiles to | — |
| `golar.goMemLimit` | GOMEMLIMIT for the language server (e.g., `4GiB`) | — |

## Commands

| Command | Description |
|---------|-------------|
| `Golar: Restart Language Server` | Restart the LSP server |
| `Golar: Show Output` | Open the Golar output channel |
| `Golar: Show LSP Trace` | Open the LSP trace output |

## Troubleshooting

### No diagnostics or language features

- Check the "Golar" output channel for errors
- Ensure your project has a `tsconfig.json` with `.vue` in the `include` array
- Ensure conflicting extensions are disabled

### Binary not found

- Set `golar.tsdk` to the directory containing the `tsgo` binary
- Or rebuild: `go build -o editors/vscode/lib/tsgo ./thirdparty/typescript-go/cmd/tsgo`

## Publishing

See the [Publishing Guide](#publishing-guide) section below for instructions on distributing the extension and binary.

### Publishing Guide

#### VS Code Marketplace

1. Create a publisher account at https://marketplace.visualstudio.com/manage
2. Generate a Personal Access Token (PAT) at https://dev.azure.com with **Marketplace > Manage** scope
3. Login and publish:

```bash
cd editors/vscode
npx @vscode/vsce login nonfx
npx @vscode/vsce publish
```

#### Open VSX (open-source marketplace)

```bash
cd editors/vscode
npx ovsx publish golar-*.vsix -p <open-vsx-token>
```

#### Cross-platform Binaries

The extension bundles a platform-specific `tsgo` binary. For multi-platform distribution, build for each target:

```bash
# macOS ARM
GOOS=darwin GOARCH=arm64 go build -o tsgo-darwin-arm64 ./thirdparty/typescript-go/cmd/tsgo

# macOS Intel
GOOS=darwin GOARCH=amd64 go build -o tsgo-darwin-amd64 ./thirdparty/typescript-go/cmd/tsgo

# Linux
GOOS=linux GOARCH=amd64 go build -o tsgo-linux-amd64 ./thirdparty/typescript-go/cmd/tsgo

# Windows
GOOS=windows GOARCH=amd64 go build -o tsgo-windows-amd64.exe ./thirdparty/typescript-go/cmd/tsgo
```

For Marketplace distribution, use [platform-specific extensions](https://code.visualstudio.com/api/working-with-extensions/publishing-extension#platformspecific-extensions) to publish separate `.vsix` packages per platform, each containing the matching binary.

## Development

```bash
# Build just the extension JS (fast iteration)
cd editors/vscode && bun run bundle

# Build everything (binary + extension + vsix)
./scripts/build-extension.sh
```
