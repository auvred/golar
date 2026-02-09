module github.com/auvred/golar/astro

go 1.25.0

replace (
	github.com/withastro/compiler/shim => ../../shim/astro-compiler/internal
	github.com/withastro/compiler/shim/transform => ../../shim/astro-compiler/transform
	github.com/withastro/compiler/shim/handler => ../../shim/astro-compiler/handler
	github.com/withastro/compiler/shim/printer => ../../shim/astro-compiler/printer

	// TODO:
	github.com/auvred/golar/plugin => ../..
)

require (
	github.com/withastro/compiler/shim v0.0.0
	github.com/withastro/compiler/shim/transform v0.0.0
	github.com/withastro/compiler/shim/handler v0.0.0
	github.com/withastro/compiler/shim/printer v0.0.0

	github.com/auvred/golar/plugin v0.0.0
)

