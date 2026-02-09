module github.com/auvred/golar

go 1.25.0

replace (
	github.com/microsoft/typescript-go/shim/ast => ./shim/typescript-go/ast
	github.com/microsoft/typescript-go/shim/binder => ./shim/typescript-go/binder
	github.com/microsoft/typescript-go/shim/bundled => ./shim/typescript-go/bundled
	github.com/microsoft/typescript-go/shim/checker => ./shim/typescript-go/checker
	github.com/microsoft/typescript-go/shim/compiler => ./shim/typescript-go/compiler
	github.com/microsoft/typescript-go/shim/core => ./shim/typescript-go/core
	github.com/microsoft/typescript-go/shim/diagnostics => ./shim/typescript-go/diagnostics
	github.com/microsoft/typescript-go/shim/diagnosticwriter => ./shim/typescript-go/diagnosticwriter
	github.com/microsoft/typescript-go/shim/fourslash => ./shim/typescript-go/fourslash
	github.com/microsoft/typescript-go/shim/golarext => ./shim/typescript-go/golarext
	github.com/microsoft/typescript-go/shim/lsp/lsproto => ./shim/typescript-go/lsp/lsproto
	github.com/microsoft/typescript-go/shim/parser => ./shim/typescript-go/parser
	github.com/microsoft/typescript-go/shim/project => ./shim/typescript-go/project
	github.com/microsoft/typescript-go/shim/scanner => ./shim/typescript-go/scanner
	github.com/microsoft/typescript-go/shim/sourcemap => ./shim/typescript-go/sourcemap
	github.com/microsoft/typescript-go/shim/testutil => ./shim/typescript-go/testutil
	github.com/microsoft/typescript-go/shim/tsoptions => ./shim/typescript-go/tsoptions
	github.com/microsoft/typescript-go/shim/tspath => ./shim/typescript-go/tspath
	github.com/microsoft/typescript-go/shim/vfs => ./shim/typescript-go/vfs
	github.com/microsoft/typescript-go/shim/vfs/cachedvfs => ./shim/typescript-go/vfs/cachedvfs
	github.com/microsoft/typescript-go/shim/vfs/osvfs => ./shim/typescript-go/vfs/osvfs
)

require (
	github.com/google/go-cmp v0.7.0
	github.com/microsoft/typescript-go/shim/ast v0.0.0
	github.com/microsoft/typescript-go/shim/binder v0.0.0
	github.com/microsoft/typescript-go/shim/bundled v0.0.0
	github.com/microsoft/typescript-go/shim/checker v0.0.0
	github.com/microsoft/typescript-go/shim/compiler v0.0.0
	github.com/microsoft/typescript-go/shim/core v0.0.0
	github.com/microsoft/typescript-go/shim/lsp/lsproto v0.0.0
	github.com/microsoft/typescript-go/shim/project v0.0.0
	github.com/microsoft/typescript-go/shim/scanner v0.0.0
	github.com/microsoft/typescript-go/shim/diagnostics v0.0.0
	github.com/microsoft/typescript-go/shim/diagnosticwriter v0.0.0
	github.com/microsoft/typescript-go/shim/fourslash v0.0.0
	github.com/microsoft/typescript-go/shim/golarext v0.0.0
	github.com/microsoft/typescript-go/shim/lsp/lsproto v0.0.0
	github.com/microsoft/typescript-go/shim/parser v0.0.0
	github.com/microsoft/typescript-go/shim/tsoptions v0.0.0
	github.com/microsoft/typescript-go/shim/sourcemap v0.0.0
	github.com/microsoft/typescript-go/shim/testutil v0.0.0
	github.com/microsoft/typescript-go/shim/tspath v0.0.0
	github.com/microsoft/typescript-go/shim/vfs v0.0.0
	github.com/microsoft/typescript-go/shim/vfs/cachedvfs v0.0.0
	github.com/microsoft/typescript-go/shim/vfs/osvfs v0.0.0
	golang.org/x/text v0.31.0
	golang.org/x/tools v0.38.0
	gotest.tools/v3 v3.5.2
)

require (
	github.com/dlclark/regexp2 v1.11.5 // indirect
	github.com/go-json-experiment/json v0.0.0-20251027170946-4849db3c2f7e // indirect
	github.com/klauspost/cpuid/v2 v2.0.9 // indirect
	github.com/microsoft/typescript-go v0.0.0-20251204215308-2ae410164f65 // indirect
	github.com/peter-evans/patience v0.3.0 // indirect
	github.com/zeebo/xxh3 v1.0.2 // indirect
	golang.org/x/mod v0.29.0 // indirect
	golang.org/x/sync v0.18.0 // indirect
	golang.org/x/sys v0.38.0 // indirect
)
