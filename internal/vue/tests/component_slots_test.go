package vue_tests

import (
	"testing"

	"github.com/microsoft/typescript-go/shim/fourslash"
	"github.com/microsoft/typescript-go/shim/lsp/lsproto"
)

func TestComponentSlots(t *testing.T) {
	runFourslashTest(t, `// @filename: file.vue
<script setup lang="ts">
	import CompFoo from './file-foo.vue'
</script>

<template>
	<CompFoo v-slot="{ msg }">
		{{ msg/*1*/ }}
	</CompFoo>
	<CompFoo v-slot:default="{ msg }">
		{{ msg/*2*/ }}
	</CompFoo>
	<CompFoo #default="{ msg }">
		{{ msg/*3*/ }}
	</CompFoo>
	<CompFoo v-slot:named-foo="{ msg }">
		{{ msg/*4*/ }}
	</CompFoo>
	<CompFoo #named-foo="{ msg }">
		{{ msg/*5*/ }}
	</CompFoo>
	<CompFoo>
		<template v-slot="{ msg }">
			{{ msg/*6*/ }}
		</template>
	</CompFoo>
	<CompFoo>
		<template v-slot:default="{ msg }">
			{{ msg/*7*/ }}
		</template>
	</CompFoo>
	<CompFoo>
		<template #default="{ msg }">
			{{ msg/*8*/ }}
		</template>
	</CompFoo>
	<CompFoo>
		<template v-slot:named-foo="{ msg }">
			{{ msg/*9*/ }}
		</template>
	</CompFoo>
	<CompFoo>
		<template #named-foo="{ msg }">
			{{ msg/*10*/ }}
		</template>
	</CompFoo>
	<CompFoo>
		<div>
			<template [|v-slot|]></template>
		</div>
	</CompFoo>
	<CompFoo>
		<div>
			<template [|#default|]></template>
		</div>
	</CompFoo>
	<CompFoo>
		<template>
			<template [|v-slot:named-foo|]></template>
		</template>
	</CompFoo>
	<CompFoo>
		<template>
			<template [|#named-foo|]></template>
		</template>
	</CompFoo>
</template>

// @filename: file-foo.vue
<script setup lang="ts">
	defineSlots<{
		default(props: { msg: "hello" }): any
		'named-foo'(props: { msg: "foo" }): any
	}>()
</script>`, func(t *testing.T, f *fourslash.FourslashTest, version vueVersion) {
		if version == vue_3_2 {
			return
		}
		f.VerifyQuickInfoAt(t, "1", `const msg: any`, "")
		f.VerifyQuickInfoAt(t, "2", `const msg: any`, "")
		f.VerifyQuickInfoAt(t, "3", `const msg: any`, "")
		f.VerifyQuickInfoAt(t, "4", `const msg: any`, "")
		f.VerifyQuickInfoAt(t, "5", `const msg: any`, "")
		f.VerifyQuickInfoAt(t, "6", `const msg: any`, "")
		f.VerifyQuickInfoAt(t, "7", `const msg: any`, "")
		f.VerifyQuickInfoAt(t, "8", `const msg: any`, "")
		f.VerifyQuickInfoAt(t, "9", `const msg: any`, "")
		f.VerifyQuickInfoAt(t, "10", `const msg: any`, "")
		f.VerifyNonSuggestionDiagnostics(t, []*lsproto.Diagnostic{
			{
				Code:    &lsproto.IntegerOrString{Integer: ptrTo[int32](1_000_008)},
				Message: "Slot does not belong to the parent component.",
			},
			{
				Code:    &lsproto.IntegerOrString{Integer: ptrTo[int32](1_000_008)},
				Message: "Slot does not belong to the parent component.",
			},
			{
				Code:    &lsproto.IntegerOrString{Integer: ptrTo[int32](1_000_008)},
				Message: "Slot does not belong to the parent component.",
			},
			{
				Code:    &lsproto.IntegerOrString{Integer: ptrTo[int32](1_000_008)},
				Message: "Slot does not belong to the parent component.",
			},
			{
				Range:   lsproto.Range{Start: lsproto.Position{Line: 6, Character: 5}, End: lsproto.Position{Line: 6, Character: 8}},
				Code:    &lsproto.IntegerOrString{Integer: ptrTo[int32](2448)},
				Message: "Block-scoped variable 'msg' used before its declaration.",
			},
			{
				Range:   lsproto.Range{Start: lsproto.Position{Line: 9, Character: 5}, End: lsproto.Position{Line: 9, Character: 8}},
				Code:    &lsproto.IntegerOrString{Integer: ptrTo[int32](2448)},
				Message: "Block-scoped variable 'msg' used before its declaration.",
			},
			{
				Range:   lsproto.Range{Start: lsproto.Position{Line: 12, Character: 5}, End: lsproto.Position{Line: 12, Character: 8}},
				Code:    &lsproto.IntegerOrString{Integer: ptrTo[int32](2448)},
				Message: "Block-scoped variable 'msg' used before its declaration.",
			},
			{
				Range:   lsproto.Range{Start: lsproto.Position{Line: 15, Character: 5}, End: lsproto.Position{Line: 15, Character: 8}},
				Code:    &lsproto.IntegerOrString{Integer: ptrTo[int32](2448)},
				Message: "Block-scoped variable 'msg' used before its declaration.",
			},
			{
				Range:   lsproto.Range{Start: lsproto.Position{Line: 18, Character: 5}, End: lsproto.Position{Line: 18, Character: 8}},
				Code:    &lsproto.IntegerOrString{Integer: ptrTo[int32](2448)},
				Message: "Block-scoped variable 'msg' used before its declaration.",
			},
			{
				Range:   lsproto.Range{Start: lsproto.Position{Line: 22, Character: 6}, End: lsproto.Position{Line: 22, Character: 9}},
				Code:    &lsproto.IntegerOrString{Integer: ptrTo[int32](2448)},
				Message: "Block-scoped variable 'msg' used before its declaration.",
			},
			{
				Range:   lsproto.Range{Start: lsproto.Position{Line: 27, Character: 6}, End: lsproto.Position{Line: 27, Character: 9}},
				Code:    &lsproto.IntegerOrString{Integer: ptrTo[int32](2448)},
				Message: "Block-scoped variable 'msg' used before its declaration.",
			},
			{
				Range:   lsproto.Range{Start: lsproto.Position{Line: 32, Character: 6}, End: lsproto.Position{Line: 32, Character: 9}},
				Code:    &lsproto.IntegerOrString{Integer: ptrTo[int32](2448)},
				Message: "Block-scoped variable 'msg' used before its declaration.",
			},
			{
				Range:   lsproto.Range{Start: lsproto.Position{Line: 37, Character: 6}, End: lsproto.Position{Line: 37, Character: 9}},
				Code:    &lsproto.IntegerOrString{Integer: ptrTo[int32](2448)},
				Message: "Block-scoped variable 'msg' used before its declaration.",
			},
			{
				Range:   lsproto.Range{Start: lsproto.Position{Line: 42, Character: 6}, End: lsproto.Position{Line: 42, Character: 9}},
				Code:    &lsproto.IntegerOrString{Integer: ptrTo[int32](2448)},
				Message: "Block-scoped variable 'msg' used before its declaration.",
			},
		})
	})
}
