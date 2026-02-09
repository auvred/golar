package vue_tests

import (
	"testing"

	"github.com/microsoft/typescript-go/shim/fourslash"
	"github.com/microsoft/typescript-go/shim/lsp/lsproto"
)

func TestTSDirectivesSuppressDiagnostics(t *testing.T) {
	runFourslashTest(t, `// @filename: file.vue
<script lang="ts" setup>
	import Comp from './file-foo.vue'

	function foo(arg: string) {}
</script>

<template>
	<!-- @vue-ignore -->
	<Comp @upd="e => e--"/>
	<!-- @vue-expect-error -->
	<Comp @upd="e => e--"/>

	<!-- @vue-ignore -->
	<Comp
		@upd="
			e => e--
		"
	/>
	<!-- @vue-expect-error -->
	<Comp
		@upd="
			e => e--
		"
	/>

	<!-- @vue-ignore -->
	{{ foo(1) }}
	<!-- @vue-expect-error -->
	{{ foo(1) }}

	<!-- @vue-ignore -->
	<Comp @upd="e => e"/>
	[|<!-- @vue-expect-error -->|]
	<Comp @upd="e => e"/>

	<!-- @vue-ignore -->
	{{ foo('1') }}
	[|<!-- @vue-expect-error -->|]
	{{ foo('1') }}
</template>

// @filename: file-foo.vue
<script setup lang="ts">
	defineEmits<{ (e: 'upd', data: string): void }>()
</script>`, func(t *testing.T, f *fourslash.FourslashTest, version vueVersion) {
		f.VerifyNonSuggestionDiagnostics(t, []*lsproto.Diagnostic{
			{
				Code:    &lsproto.IntegerOrString{Integer: ptrTo[int32](1_000_000)},
				Message: "Unused directive.",
			},
			{
				Code:    &lsproto.IntegerOrString{Integer: ptrTo[int32](1_000_000)},
				Message: "Unused directive.",
			},
		})
	})
}

func TestTSDirectivesResetByText(t *testing.T) {
	runFourslashTest(t, `// @filename: file.vue
<script lang="ts" setup>
	import Comp from './file-foo.vue'
</script>

<template>
	<!-- @vue-ignore -->
	text
	<Comp @upd="e => e--"/>
	<!-- @vue-expect-error -->
	text
	<Comp @upd="e => e--"/>
</template>

// @filename: file-foo.vue
<script setup lang="ts">
	defineEmits<{ (e: 'upd', data: string): void }>()
</script>`, func(t *testing.T, f *fourslash.FourslashTest, version vueVersion) {
		f.VerifyNonSuggestionDiagnostics(t, []*lsproto.Diagnostic{
			{
				Range:   lsproto.Range{Start: lsproto.Position{Line: 7, Character: 18}, End: lsproto.Position{Line: 7, Character: 19}},
				Code:    &lsproto.IntegerOrString{Integer: ptrTo[int32](2356)},
				Message: "An arithmetic operand must be of type 'any', 'number', 'bigint' or an enum type.",
			},
			{
				Range:   lsproto.Range{Start: lsproto.Position{Line: 8, Character: 1}, End: lsproto.Position{Line: 8, Character: 27}},
				Code:    &lsproto.IntegerOrString{Integer: ptrTo[int32](1_000_000)},
				Message: "Unused directive.",
			},
			{
				Range:   lsproto.Range{Start: lsproto.Position{Line: 10, Character: 18}, End: lsproto.Position{Line: 10, Character: 19}},
				Code:    &lsproto.IntegerOrString{Integer: ptrTo[int32](2356)},
				Message: "An arithmetic operand must be of type 'any', 'number', 'bigint' or an enum type.",
			},
		})
	})
}
