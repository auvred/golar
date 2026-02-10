package vue_tests

import (
	"testing"

	"github.com/microsoft/typescript-go/shim/fourslash"
	"github.com/microsoft/typescript-go/shim/lsp/lsproto"
)

func TestSetupGeneric(t *testing.T) {
	runFourslashTest(t, `// @filename: file.vue
// @strict: true
<script lang="ts" setup>
	import Comp from './file-foo.vue'
</script>

<template>
	<Comp
		[|foo|]
	/>
	<Comp
		foo="123"
		@upd="e => e/*1*/"
	/>
</template>

// @filename: file-foo.vue
<script lang="ts" setup generic="T extends string | number">
	defineProps<{
		foo: T
	}>()

	defineEmits<{
		(e: 'upd', data: T): void
	}>()
</script>
`, func(t *testing.T, f *fourslash.FourslashTest, version vueVersion) {
		f.VerifyQuickInfoAt(t, "1", `(parameter) e: any`, "")
		f.VerifyNonSuggestionDiagnostics(t, []*lsproto.Diagnostic{
			{
				Range:   lsproto.Range{Start: lsproto.Position{Line: 10, Character: 8}, End: lsproto.Position{Line: 10, Character: 9}},
				Code:    &lsproto.IntegerOrString{Integer: ptrTo[int32](7006)},
				Message: "Parameter 'e' implicitly has an 'any' type.",
			},
		})
	})
}

func TestSetupGenericDefineModel(t *testing.T) {
	runFourslashTest(t, `// @filename: file.vue
// @strict: true
<script lang="ts" setup>
	import Comp from './file-foo.vue'
</script>

<template>
	<Comp
		model-value="123"
		@update:model-value="e => e/*1*/"
	/>
</template>

// @filename: file-foo.vue
<script lang="ts" setup generic="T extends string | number">
	defineModel<T>()
</script>
`, func(t *testing.T, f *fourslash.FourslashTest, version vueVersion) {
		switch version {
		case vue_3_2, vue_3_3:
			return
		default:
			f.VerifyQuickInfoAt(t, "1", `(parameter) e: any`, "")
		}
		f.VerifyNonSuggestionDiagnostics(t, []*lsproto.Diagnostic{
			{
				Range:   lsproto.Range{Start: lsproto.Position{Line: 7, Character: 23}, End: lsproto.Position{Line: 7, Character: 24}},
				Code:    &lsproto.IntegerOrString{Integer: ptrTo[int32](7006)},
				Message: "Parameter 'e' implicitly has an 'any' type.",
			},
		})
	})
}
