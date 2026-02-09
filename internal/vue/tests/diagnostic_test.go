package vue_tests

import (
	"testing"

	"github.com/microsoft/typescript-go/shim/fourslash"
	"github.com/microsoft/typescript-go/shim/lsp/lsproto"
)

func TestDiagnostic(t *testing.T) {
	runFourslashTest(t, `// @filename: file.vue
<script lang="ts" setup>
	const [|foo|]: string = 5
</script>`, func(t *testing.T, f *fourslash.FourslashTest, version vueVersion) {
		f.VerifyNonSuggestionDiagnostics(t, []*lsproto.Diagnostic{
			{
				Code:    &lsproto.IntegerOrString{Integer: ptrTo[int32](2322)},
				Message: "Type 'number' is not assignable to type 'string'.",
			},
		})
	})
}

func TestVueSyntaxError(t *testing.T) {
	// TODO:
	t.Skip()
	runFourslashTest(t, `// @filename: file.vue
<template>
	<div>
</template>`, func(t *testing.T, f *fourslash.FourslashTest, version vueVersion) {
		f.VerifyNonSuggestionDiagnostics(t, []*lsproto.Diagnostic{
			{
				Code:    &lsproto.IntegerOrString{Integer: ptrTo[int32](2322)},
				Message: "Type 'number' is not assignable to type 'string'.",
			},
		})
	})
}
