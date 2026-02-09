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
		switch version {
		case vue_3_2:
			f.VerifyQuickInfoAt(t, "1", `(parameter) e: string | number`, "")
		default:
			f.VerifyQuickInfoAt(t, "1", `(parameter) e: string`, "")
		}
		f.VerifyNonSuggestionDiagnostics(t, []*lsproto.Diagnostic{
			{
				Code:    &lsproto.IntegerOrString{Integer: ptrTo[int32](2322)},
				Message: "Type 'boolean' is not assignable to type 'string | number'.",
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
			f.VerifyQuickInfoAt(t, "1", `(parameter) e: "123" | undefined`, "")
		}
		f.VerifyNonSuggestionDiagnostics(t, []*lsproto.Diagnostic{})
	})
}
