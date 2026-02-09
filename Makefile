.PHONY: help build-binary build-extension install-extension clean test

help:
	@echo "Golar Build Tasks"
	@echo ""
	@echo "  make build-binary      - Build the golar/tsgo binary"
	@echo "  make build-extension   - Build the VS Code extension (.vsix)"
	@echo "  make install-extension - Install the extension in VS Code"
	@echo "  make test              - Run all tests"
	@echo "  make clean             - Clean build artifacts"

build-binary:
	@echo "==> Building golar binary..."
	go build -o golar/tsgo ./thirdparty/typescript-go/cmd/tsgo

build-extension:
	@echo "==> Building golar binary for extension..."
	go build -o editors/vscode/lib/tsgo ./thirdparty/typescript-go/cmd/tsgo
	@echo "==> Installing extension dependencies..."
	cd editors/vscode && bun install
	@echo "==> Bundling extension..."
	cd editors/vscode && bun run bundle
	@echo "==> Packaging VSIX..."
	cd editors/vscode && npx @vscode/vsce package --no-dependencies
	@echo ""
	@echo "Done! Install with: make install-extension"

install-extension:
	@echo "==> Installing VS Code extension..."
	code --install-extension editors/vscode/golar-*.vsix --force

test:
	@echo "==> Running tests..."
	go test ./internal/vue/tests/... -v -count=1

clean:
	@echo "==> Cleaning build artifacts..."
	rm -f golar/tsgo
	rm -f editors/vscode/lib/tsgo
	rm -f editors/vscode/golar-*.vsix
	rm -rf editors/vscode/dist
