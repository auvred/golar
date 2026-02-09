package vue_tests

import (
	"testing"

	"github.com/microsoft/typescript-go/shim/fourslash"
	"github.com/microsoft/typescript-go/shim/lsp/lsproto"
)

func TestElementProps(t *testing.T) {
	runFourslashTest(t, `// @filename: file.vue
<script setup lang="ts">
	const foo = '123'
	const id = 123
</script>

<template>
	<div [|foo|]="123"></div>
	<div id="123"></div>
	<div [|:id="123"|]></div>
	<[|div|] v-bind="{ id: 123 }"></div>
	<input v-model="foo">
	<div [|:id|]></div>
	<div [|:dir|]></div>
	<div @click="e => e/*1*/"></div>
</template>`, func(t *testing.T, f *fourslash.FourslashTest, version vueVersion) {
	switch version {
	case vue_3_2, vue_3_3, vue_3_4:
		f.VerifyQuickInfoAt(t, "1", `(parameter) e: MouseEvent`, "")
	default:
		f.VerifyQuickInfoAt(t, "1", `(parameter) e: PointerEvent`, "")

	}
	common1 := &lsproto.Diagnostic{
			Code: &lsproto.IntegerOrString{Integer: ptrTo[int32](2322)},
			Message:"Type 'number' is not assignable to type 'string'.",
	}
	switch version {
	case vue_3_2:
			f.VerifyNonSuggestionDiagnostics(t, []*lsproto.Diagnostic{
				{
					Code: &lsproto.IntegerOrString{Integer: ptrTo[int32](2353)},
					Message: "Object literal may only specify known properties, and ''foo'' does not exist in type 'ElementAttrs<HTMLAttributes>'.",
				},
				common1,
				{
			Code: &lsproto.IntegerOrString{Integer: ptrTo[int32](2345)},
			Message: `Argument of type '{ id: number; }' is not assignable to parameter of type 'ElementAttrs<HTMLAttributes>'.
  Type '{ id: number; }' is not assignable to type 'HTMLAttributes'.
    Types of property 'id' are incompatible.
      Type 'number' is not assignable to type 'string'.`,
	},
				common1,
				{
					Code: &lsproto.IntegerOrString{Integer: ptrTo[int32](2339)},
					Message: "Property 'dir' does not exist on type '{ $: ComponentInternalInstance; $data: {}; $props: {}; $attrs: Data; $refs: Data; $slots: Readonly<InternalSlots>; $root: ComponentPublicInstance<...>; ... 8 more ...; id: 123; }'.",
				},
			})
		case vue_3_3, vue_3_4:
			f.VerifyNonSuggestionDiagnostics(t, []*lsproto.Diagnostic{
				{
					Code: &lsproto.IntegerOrString{Integer: ptrTo[int32](2353)},
					Message: "Object literal may only specify known properties, and ''foo'' does not exist in type 'HTMLAttributes & ReservedProps'.",
				},
				common1,
				{
			Code: &lsproto.IntegerOrString{Integer: ptrTo[int32](2345)},
			Message: `Argument of type '{ id: number; }' is not assignable to parameter of type 'HTMLAttributes & ReservedProps'.
  Type '{ id: number; }' is not assignable to type 'HTMLAttributes'.
    Types of property 'id' are incompatible.
      Type 'number' is not assignable to type 'string'.`,
			},
				common1,
				{
					Code: &lsproto.IntegerOrString{Integer: ptrTo[int32](2339)},
					Message: "Property 'dir' does not exist on type '{ $: ComponentInternalInstance; $data: {}; $props: {}; $attrs: Data; $refs: Data; $slots: Readonly<InternalSlots>; $root: ComponentPublicInstance<...>; ... 8 more ...; id: 123; }'.",
				},
			})
		default:
			f.VerifyNonSuggestionDiagnostics(t, []*lsproto.Diagnostic{
				{
					Code: &lsproto.IntegerOrString{Integer: ptrTo[int32](2353)},
					Message: "Object literal may only specify known properties, and ''foo'' does not exist in type 'HTMLAttributes & ReservedProps'.",
				},
				common1,
				{
			Code: &lsproto.IntegerOrString{Integer: ptrTo[int32](2345)},
			Message: `Argument of type '{ id: number; }' is not assignable to parameter of type 'HTMLAttributes & ReservedProps'.
  Type '{ id: number; }' is not assignable to type 'HTMLAttributes'.
    Types of property 'id' are incompatible.
      Type 'number' is not assignable to type 'string'.`,
			},
				common1,
				{
					Code: &lsproto.IntegerOrString{Integer: ptrTo[int32](2339)},
					Message: "Property 'dir' does not exist on type '{ $: ComponentInternalInstance; $data: {}; $props: {}; $attrs: Data; $refs: Data; $slots: Readonly<InternalSlots>; $root: ComponentPublicInstance<...>; ... 9 more ...; id: 123; }'.",
				},
			})
		}
	})
}
