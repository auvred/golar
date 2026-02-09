package vue_tests

import (
	"testing"

	"github.com/microsoft/typescript-go/shim/fourslash"
	"github.com/microsoft/typescript-go/shim/lsp/lsproto"
)

func TestDefineModel(t *testing.T) {
	runFourslashTest(t, `// @filename: file.vue
// @strict: true
<script lang="ts" setup>
	const model1/*1*/ = defineModel<string>('first')

	const opts = { required: true } as const
	const model2/*3*/ = defineModel<string>('second', {
		...opts,
	})

	const model3/*5*/ = defineModel('third', {
		type: String,
		required: true,
		get(v) {
			return Number.parseInt(v)
		}
	})

	defineModel<string, 'trim'>('fourth')

	defineModel<string, 'capitalize'>('foo-bar', { required: true })

	defineModel<'default model', 'mod'>({ required: true })
</script>

<template>
	{{ first/*2*/ }}
	{{ second/*4*/ }}
	{{ third/*6*/ }}
	{{ $props.third/*7*/ }}
	{{ model3/*8*/ }}
	{{ thirdModifiers/*9*/ }}
	{{ fourthModifiers/*10*/ }}
	{{ fooBar/*11*/ }}
	{{ fooBarModifiers/*12*/ }}
	{{ modelValue/*13*/ }}
	{{ modelModifiers/*14*/ }}
</template>`, func(t *testing.T, f *fourslash.FourslashTest, version vueVersion) {
		switch version {
		case vue_3_2, vue_3_3:
			return
		case vue_3_4:
			f.VerifyQuickInfoAt(t, "1", `const model1: ModelRef<string | undefined, string>`, "")
			f.VerifyQuickInfoAt(t, "3", `const model2: ModelRef<string, string>`, "")
			f.VerifyQuickInfoAt(t, "5", `const model3: ModelRef<string, string>`, "")
			f.VerifyQuickInfoAt(t, "8", `(property) model3: string`, "")
		default:
			f.VerifyQuickInfoAt(t, "1", `const model1: ModelRef<string | undefined, string, string | undefined, string | undefined>`, "")
			f.VerifyQuickInfoAt(t, "3", `const model2: ModelRef<string, string, string, string>`, "")
			f.VerifyQuickInfoAt(t, "5", `const model3: ModelRef<string, string, number, string>`, "")
			f.VerifyQuickInfoAt(t, "8", `(property) model3: number`, "")
		}
		f.VerifyQuickInfoAt(t, "2", `(property) 'first': string | undefined`, "")
		f.VerifyQuickInfoAt(t, "4", `(property) 'second': string`, "")
		f.VerifyQuickInfoAt(t, "6", `(property) 'third': string`, "")
		f.VerifyQuickInfoAt(t, "7", `(property) 'third': string`, "")
		f.VerifyQuickInfoAt(t, "9", `(property) 'thirdModifiers': Partial<Record<string, true>> | undefined`, "")
		f.VerifyQuickInfoAt(t, "10", `(property) 'fourthModifiers': Partial<Record<"trim", true>> | undefined`, "")
		f.VerifyQuickInfoAt(t, "11", `(property) 'fooBar': string`, "")
		f.VerifyQuickInfoAt(t, "12", `(property) 'fooBarModifiers': Partial<Record<"capitalize", true>> | undefined`, "")
		f.VerifyQuickInfoAt(t, "13", `(property) 'modelValue': "default model"`, "")
		f.VerifyQuickInfoAt(t, "14", `(property) 'modelModifiers': Partial<Record<"mod", true>> | undefined`, "")

		f.VerifyNonSuggestionDiagnostics(t, []*lsproto.Diagnostic{})
	})
}

func TestDuplicateDefineModelExplicitName(t *testing.T) {
	runFourslashTest(t, `// @filename: file.vue
<script lang="ts" setup>
	defineModel('foo-bar')

	const model = [|defineModel|]('fooBar')
</script>
`, func(t *testing.T, f *fourslash.FourslashTest, version vueVersion) {
		switch version {
		case vue_3_2, vue_3_3:
			return
		}
		f.VerifyNonSuggestionDiagnostics(t, []*lsproto.Diagnostic{
			{
				Code:    &lsproto.IntegerOrString{Integer: ptrTo[int32](1_000_009)},
				Message: `Duplicate model name "fooBar".`,
			},
		})
	})
}

func TestDuplicateDefineModelDefaultName(t *testing.T) {
	runFourslashTest(t, `// @filename: file.vue
<script lang="ts" setup>
	const model = defineModel()

	[|defineModel|]()

	const anotherModel = [|defineModel|]('model-value')
</script>
`, func(t *testing.T, f *fourslash.FourslashTest, version vueVersion) {
		switch version {
		case vue_3_2, vue_3_3:
			return
		}
		f.VerifyNonSuggestionDiagnostics(t, []*lsproto.Diagnostic{
			{
				Code:    &lsproto.IntegerOrString{Integer: ptrTo[int32](1_000_009)},
				Message: `Duplicate model name "modelValue".`,
			},
			{
				Code:    &lsproto.IntegerOrString{Integer: ptrTo[int32](1_000_009)},
				Message: `Duplicate model name "modelValue".`,
			},
		})
	})
}

func TestDefineModelEmitsDollar(t *testing.T) {
	runFourslashTest(t, `// @filename: file.vue
<script lang="ts" setup>
	const model = defineModel<string>('foo')

</script>

<template>
	{{ $emit('update:foo', [|123|]) }}
</template>
`, func(t *testing.T, f *fourslash.FourslashTest, version vueVersion) {
		switch version {
		case vue_3_2, vue_3_3:
			return
		}
		f.VerifyNonSuggestionDiagnostics(t, []*lsproto.Diagnostic{
			{
				Code:    &lsproto.IntegerOrString{Integer: ptrTo[int32](2345)},
				Message: `Argument of type 'number' is not assignable to parameter of type 'string'.`,
			},
		})
	})
}
