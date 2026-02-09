package vue_tests

import (
	"testing"

	"github.com/microsoft/typescript-go/shim/fourslash"
	"github.com/microsoft/typescript-go/shim/lsp/lsproto"
)

func TestDefineEmitsVariableDecl(t *testing.T) {
	runFourslashTest(t, `// @filename: file.vue
<script lang="ts" setup>
const emit = defineEmits<{ (e: 'foo', id: number): void  }>()
</script>

<template>
	{{ emit/*1*/ }}
	{{ $emit/*2*/ }}
</template>`, func(t *testing.T, f *fourslash.FourslashTest, version vueVersion) {
		f.VerifyQuickInfoAt(t, "1", `(property) emit: (e: "foo", id: number) => void`, "")
		f.VerifyQuickInfoAt(t, "2", `(property) $emit: (e: "foo", id: number) => void`, "")
		f.VerifyNonSuggestionDiagnostics(t, []*lsproto.Diagnostic{})
	})
}

func TestDefineEmitsTypeMismatch(t *testing.T) {
	runFourslashTest(t, `// @filename: file.vue
<script lang="ts" setup>
const emit = defineEmits<{ (e: 'foo', id: number): void  }>()
</script>

<template>
	{{ emit('foo', [|'2'|]) }}
</template>`, func(t *testing.T, f *fourslash.FourslashTest, version vueVersion) {
		f.VerifyNonSuggestionDiagnostics(t, []*lsproto.Diagnostic{
			{
				Code:    &lsproto.IntegerOrString{Integer: ptrTo[int32](2345)},
				Message: "Argument of type 'string' is not assignable to parameter of type 'number'.",
			},
		})
	})
}
func TestDuplicateDefineEmitsCallExpression(t *testing.T) {
	runFourslashTest(t, `// @filename: file.vue
<script lang="ts" setup>
	const e = defineEmits<{ (e: 'foo'): void }>()
	[|defineEmits|]<{ (e: 'foo'): void }>()
</script>`, func(t *testing.T, f *fourslash.FourslashTest, version vueVersion) {

		f.VerifyNonSuggestionDiagnostics(t, []*lsproto.Diagnostic{
			{
				Code:    &lsproto.IntegerOrString{Integer: ptrTo[int32](1_000_007)},
				Message: "Duplicate defineEmits call.",
			},
		})
	})
}

func TestDuplicateDefineEmitsVariableDecl(t *testing.T) {
	runFourslashTest(t, `// @filename: file.vue
<script lang="ts" setup>
	defineEmits<{ (e: 'foo'): void }>()
	const e = [|defineEmits|]<{ (e: 'foo'): void }>()
</script>`, func(t *testing.T, f *fourslash.FourslashTest, version vueVersion) {

		f.VerifyNonSuggestionDiagnostics(t, []*lsproto.Diagnostic{
			{
				Code:    &lsproto.IntegerOrString{Integer: ptrTo[int32](1_000_007)},
				Message: "Duplicate defineEmits call.",
			},
		})
	})
}
