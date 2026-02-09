package vue_tests

import (
	"testing"

	"github.com/microsoft/typescript-go/shim/fourslash"
	"github.com/microsoft/typescript-go/shim/lsp/lsproto"
)

func TestDefinePropsCallExpression(t *testing.T) {
	runFourslashTest(t, `// @filename: file.vue
<script lang="ts" setup>
	defineProps<{ foo: string }>()
</script>

<template>
	{{ foo/*1*/ }}
	{{ $props/*2*/ }}
</template>`, func(t *testing.T, f *fourslash.FourslashTest, version vueVersion) {
		f.VerifyQuickInfoAt(t, "1", "(property) foo: string", "")
		switch version {
		case vue_3_2:
			f.VerifyQuickInfoAt(t, "2", "(property) $props: Readonly<Omit<{ foo: string; }, never> & {}>", "")
		default:
			f.VerifyQuickInfoAt(t, "2", "(property) $props: DefineProps<LooseRequired<{ foo: string; }>, never>", "")
		}
		f.VerifyNonSuggestionDiagnostics(t, []*lsproto.Diagnostic{})
	})
}

func TestDefinePropsVariableDecl(t *testing.T) {
	runFourslashTest(t, `// @filename: file.vue
<script lang="ts" setup>
	const p = defineProps<{ foo: string }>()
	p.foo/*1*/
</script>

<template>
	{{ foo/*2*/ }}
	{{ $props/*3*/ }}
</template>`, func(t *testing.T, f *fourslash.FourslashTest, version vueVersion) {
		f.VerifyQuickInfoAt(t, "1", "(property) foo: string", "")
		f.VerifyQuickInfoAt(t, "2", "(property) foo: string", "")
		switch version {
		case vue_3_2:
			f.VerifyQuickInfoAt(t, "3", "(property) $props: Readonly<Omit<{ foo: string; }, never> & {}>", "")
		default:
			f.VerifyQuickInfoAt(t, "3", "(property) $props: DefineProps<LooseRequired<{ foo: string; }>, never>", "")
		}
		f.VerifyNonSuggestionDiagnostics(t, []*lsproto.Diagnostic{})
	})
}

func TestDuplicateDefinePropsCallExpression(t *testing.T) {
	runFourslashTest(t, `// @filename: file.vue
<script lang="ts" setup>
	const p = defineProps<{ foo: string }>()
	[|defineProps|]<{ foo: string }>()
</script>`, func(t *testing.T, f *fourslash.FourslashTest, version vueVersion) {

		f.VerifyNonSuggestionDiagnostics(t, []*lsproto.Diagnostic{
			{
				Code:    &lsproto.IntegerOrString{Integer: ptrTo[int32](1_000_007)},
				Message: "Duplicate defineProps call.",
			},
		})
	})
}

func TestDuplicateDefinePropsVariableDecl(t *testing.T) {
	runFourslashTest(t, `// @filename: file.vue
<script lang="ts" setup>
	defineProps<{ foo: string }>()
	const p = [|defineProps|]<{ foo: string }>()
</script>`, func(t *testing.T, f *fourslash.FourslashTest, version vueVersion) {

		f.VerifyNonSuggestionDiagnostics(t, []*lsproto.Diagnostic{
			{
				Code:    &lsproto.IntegerOrString{Integer: ptrTo[int32](1_000_007)},
				Message: "Duplicate defineProps call.",
			},
		})
	})
}

func TestDefinePropsWithDefaults(t *testing.T) {
	runFourslashTest(t, `// @filename: file.vue
<script lang="ts" setup>
	const p = withDefaults(defineProps<{ foo?: string }>(), { foo: 'bar' })
	p.foo/*1*/
</script>

<template>
	{{ foo/*2*/ }}
</template>`, func(t *testing.T, f *fourslash.FourslashTest, version vueVersion) {

		f.VerifyQuickInfoAt(t, "1", "(property) foo: string", "")
		f.VerifyQuickInfoAt(t, "2", "(property) foo: string", "")
		f.VerifyNonSuggestionDiagnostics(t, []*lsproto.Diagnostic{})
	})
}

func TestDefinePropsWithDefaultsCallExpr(t *testing.T) {
	runFourslashTest(t, `// @filename: file.vue
<script lang="ts" setup>
	withDefaults(defineProps<{ foo?: string }>(), { foo: 'bar' })
</script>

<template>
	{{ foo/*1*/ }}
</template>`, func(t *testing.T, f *fourslash.FourslashTest, version vueVersion) {

		f.VerifyQuickInfoAt(t, "1", "(property) foo: string", "")
		f.VerifyNonSuggestionDiagnostics(t, []*lsproto.Diagnostic{})
	})
}
