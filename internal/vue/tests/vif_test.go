package vue_tests

import (
	"testing"

	"github.com/microsoft/typescript-go/shim/fourslash"
	"github.com/microsoft/typescript-go/shim/lsp/lsproto"
)

func TestVIfEmptyDirective(t *testing.T) {
	runFourslashTest(t, `// @filename: file.vue
<template>
	<div [|v-if=""|]></div>
</template>`, func(t *testing.T, f *fourslash.FourslashTest, version vueVersion) {

		f.VerifyNonSuggestionDiagnostics(t, []*lsproto.Diagnostic{
			{
				Code:    &lsproto.IntegerOrString{Integer: ptrTo[int32](1_000_006)},
				Message: "v-if is missing expression.",
			},
		})
	})
}

func TestVElseIfMissingExpression(t *testing.T) {
	runFourslashTest(t, `// @filename: file.vue
<template>
	<div v-if="true"></div>
	<div [|v-else-if=""|]></div>
</template>`, func(t *testing.T, f *fourslash.FourslashTest, version vueVersion) {

		f.VerifyNonSuggestionDiagnostics(t, []*lsproto.Diagnostic{
			{
				Code:    &lsproto.IntegerOrString{Integer: ptrTo[int32](1_000_006)},
				Message: "v-else-if is missing expression.",
			},
		})
	})
}

func TestVElseIfWithoutAdjacent(t *testing.T) {
	runFourslashTest(t, `// @filename: file.vue
<template>
	<div [|v-else-if|]="true"></div>
</template>`, func(t *testing.T, f *fourslash.FourslashTest, version vueVersion) {

		f.VerifyNonSuggestionDiagnostics(t, []*lsproto.Diagnostic{
			{
				Code:    &lsproto.IntegerOrString{Integer: ptrTo[int32](1_000_003)},
				Message: "v-else-if has no adjacent v-if or v-else-if.",
			},
		})
	})
}

func TestVElseWithoutAdjacent(t *testing.T) {
	runFourslashTest(t, `// @filename: file.vue
<template>
	<div [|v-else|]></div>
</template>`, func(t *testing.T, f *fourslash.FourslashTest, version vueVersion) {

		f.VerifyNonSuggestionDiagnostics(t, []*lsproto.Diagnostic{
			{
				Code:    &lsproto.IntegerOrString{Integer: ptrTo[int32](1_000_003)},
				Message: "v-else has no adjacent v-if or v-else-if.",
			},
		})
	})
}

func TestVElseIfAfterNonConditionalElement(t *testing.T) {
	runFourslashTest(t, `// @filename: file.vue
<template>
	<div v-if="true"></div>
	<div>not part of chain</div>
	<div [|v-else-if|]="true"></div>
</template>`, func(t *testing.T, f *fourslash.FourslashTest, version vueVersion) {

		f.VerifyNonSuggestionDiagnostics(t, []*lsproto.Diagnostic{
			{
				Code:    &lsproto.IntegerOrString{Integer: ptrTo[int32](1_000_003)},
				Message: "v-else-if has no adjacent v-if or v-else-if.",
			},
		})
	})
}

func TestVIfDuplicateConditionalDirective1(t *testing.T) {
	runFourslashTest(t, `// @filename: file.vue
<template>
	<div v-if="true"></div>
	<div v-else-if="true" [|v-if|]="false"></div>
</template>`, func(t *testing.T, f *fourslash.FourslashTest, version vueVersion) {

		f.VerifyNonSuggestionDiagnostics(t, []*lsproto.Diagnostic{
			{
				Code:    &lsproto.IntegerOrString{Integer: ptrTo[int32](1_000_005)},
				Message: "Multiple conditional directives cannot coexist on the same element.",
			},
		})
	})
}

func TestVIfMultipleConditionalDirective2(t *testing.T) {
	runFourslashTest(t, `// @filename: file.vue
<template>
	<div v-if="true" [|v-else-if|]="true"></div>
</template>`, func(t *testing.T, f *fourslash.FourslashTest, version vueVersion) {

		f.VerifyNonSuggestionDiagnostics(t, []*lsproto.Diagnostic{
			{
				Code:    &lsproto.IntegerOrString{Integer: ptrTo[int32](1_000_005)},
				Message: "Multiple conditional directives cannot coexist on the same element.",
			},
		})
	})
}

func TestVIfMultipleConditionalDirective3(t *testing.T) {
	runFourslashTest(t, `// @filename: file.vue
<template>
	<div v-if="true"></div>
	<div v-else-if="true" [|v-else|]></div>
</template>`, func(t *testing.T, f *fourslash.FourslashTest, version vueVersion) {

		f.VerifyNonSuggestionDiagnostics(t, []*lsproto.Diagnostic{
			{
				Code:    &lsproto.IntegerOrString{Integer: ptrTo[int32](1_000_005)},
				Message: "Multiple conditional directives cannot coexist on the same element.",
			},
		})
	})
}

func TestVIfNarrowingAcrossChain(t *testing.T) {
	runFourslashTest(t, `// @filename: file.vue
<script setup lang="ts">
	type Foo = { kind: 'foo'; foo: number }
	type Bar = { kind: 'bar'; bar: boolean }
	const value = {} as unknown as Foo | Bar
</script>

<template>
	<div v-if="value.kind === 'foo'">
		{{ value/*1*/.foo/*2*/ }}
	</div>
	<div v-else-if="value.kind === 'bar'">
		{{ value/*3*/.bar/*4*/ }}
	</div>
	<div v-else>
		{{ value/*5*/ }}
	</div>
</template>`, func(t *testing.T, f *fourslash.FourslashTest, version vueVersion) {

		f.VerifyQuickInfoAt(t, "1", "(property) value: Foo", "")
		f.VerifyQuickInfoAt(t, "2", "(property) foo: number", "")
		f.VerifyQuickInfoAt(t, "3", "(property) value: Bar", "")
		f.VerifyQuickInfoAt(t, "4", "(property) bar: boolean", "")
		f.VerifyQuickInfoAt(t, "5", "(property) value: never", "")

		f.VerifyNonSuggestionDiagnostics(t, []*lsproto.Diagnostic{})
	})
}
