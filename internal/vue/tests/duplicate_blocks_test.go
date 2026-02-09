package vue_tests

import (
	"testing"

	"github.com/microsoft/typescript-go/shim/fourslash"
	"github.com/microsoft/typescript-go/shim/lsp/lsproto"
)

func TestDuplicateScriptSetupDiagnostic(t *testing.T) {
	runFourslashTest(t, `// @filename: file.vue
<script setup lang="ts">
	const foo = 1
</script>
[|<script setup lang="ts">|]
	const bar = 2
</script>`, func(t *testing.T, f *fourslash.FourslashTest, version vueVersion) {

		f.VerifyNonSuggestionDiagnostics(t, []*lsproto.Diagnostic{
			{
				Code:    &lsproto.IntegerOrString{Integer: ptrTo[int32](1_000_000)},
				Message: "Single file component can contain only one <script setup> element.",
			},
		})
	})
}

func TestDuplicateScriptDiagnostic(t *testing.T) {
	runFourslashTest(t, `// @filename: file.vue
<script lang="ts">
	const foo = 1
</script>
[|<script lang="ts">|]
	const bar = 2
</script>`, func(t *testing.T, f *fourslash.FourslashTest, version vueVersion) {

		f.VerifyNonSuggestionDiagnostics(t, []*lsproto.Diagnostic{
			{
				Code:    &lsproto.IntegerOrString{Integer: ptrTo[int32](1_000_002)},
				Message: "Single file component can contain only one <script> element.",
			},
		})
	})
}

func TestDuplicateTemplateDiagnostic(t *testing.T) {
	runFourslashTest(t, `// @filename: file.vue
<template>
	<div>one</div>
</template>
[|<template>|]
	<div>two</div>
</template>`, func(t *testing.T, f *fourslash.FourslashTest, version vueVersion) {

		f.VerifyNonSuggestionDiagnostics(t, []*lsproto.Diagnostic{
			{
				Code:    &lsproto.IntegerOrString{Integer: ptrTo[int32](1_000_001)},
				Message: "Single file component can contain only one <template> element.",
			},
		})
	})
}
