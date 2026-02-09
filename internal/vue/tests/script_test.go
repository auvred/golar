package vue_tests

import (
	"testing"

	"github.com/microsoft/typescript-go/shim/fourslash"
	"github.com/microsoft/typescript-go/shim/lsp/lsproto"
)

func TestScriptAndSetup(t *testing.T) {
	runFourslashTest(t, `// @filename: file.vue
<script lang="ts">
	const foo/*1*/ = 'foo'
</script>

<script lang="ts" setup>
	const bar = 'bar'
</script>

<template>
	{{ bar/*2*/ }}
	{{ foo/*3*/ }}
</template>`, func(t *testing.T, f *fourslash.FourslashTest, version vueVersion) {
		f.VerifyQuickInfoAt(t, "1", `const foo: "foo"`, "")
		f.VerifyQuickInfoAt(t, "2", `(property) bar: "bar"`, "")
		f.VerifyQuickInfoAt(t, "3", `(property) foo: "foo"`, "")
		f.VerifyNonSuggestionDiagnostics(t, []*lsproto.Diagnostic{})
	})
}

func TestScriptWithoutSetup(t *testing.T) {
	runFourslashTest(t, `// @filename: file.vue
<script lang="ts">
	import { defineComponent } from 'vue'
	const foo/*1*/ = 'foo'

	export default defineComponent({
		props: {
			bar: String
		}
	})
</script>

<template>
	{{ [|foo|] }}
	{{ bar/*2*/ }}
</template>`, func(t *testing.T, f *fourslash.FourslashTest, version vueVersion) {
		f.VerifyQuickInfoAt(t, "1", `const foo: "foo"`, "")
		f.VerifyQuickInfoAt(t, "2", `(property) bar: string`, "")
		switch version {
		case vue_3_2:
			f.VerifyNonSuggestionDiagnostics(t, []*lsproto.Diagnostic{
				{
					Code:    &lsproto.IntegerOrString{Integer: ptrTo[int32](2339)},
					Message: "Property 'foo' does not exist on type '{ $: ComponentInternalInstance; $data: {}; $props: Partial<{}> & Omit<Readonly<ExtractPropTypes<{ bar: StringConstructor; }>> & VNodeProps & AllowedComponentProps & ComponentCustomProps, never>; ... 11 more ...; bar?: string; }'.",
				},
			})
		case vue_3_3:
			f.VerifyNonSuggestionDiagnostics(t, []*lsproto.Diagnostic{
				{
					Code:    &lsproto.IntegerOrString{Integer: ptrTo[int32](2339)},
					Message: "Property 'foo' does not exist on type '{ $: ComponentInternalInstance; $data: {}; $props: Partial<{}> & Omit<{ readonly bar?: string; } & VNodeProps & AllowedComponentProps & ComponentCustomProps & Readonly<...>, never>; ... 11 more ...; bar?: string; }'.",
				},
			})
		case vue_3_4:
			f.VerifyNonSuggestionDiagnostics(t, []*lsproto.Diagnostic{
				{
					Code:    &lsproto.IntegerOrString{Integer: ptrTo[int32](2339)},
					Message: "Property 'foo' does not exist on type '{ $: ComponentInternalInstance; $data: {}; $props: Partial<{}> & Omit<{ readonly bar?: string; } & VNodeProps & AllowedComponentProps & ComponentCustomProps & Readonly<...>, never>; ... 11 more ...; bar?: string; }'.",
				},
			})
		default:
			f.VerifyNonSuggestionDiagnostics(t, []*lsproto.Diagnostic{
				{
					Code:    &lsproto.IntegerOrString{Integer: ptrTo[int32](2339)},
					Message: "Property 'foo' does not exist on type '{ $: ComponentInternalInstance; $data: {}; $props: Partial<{}> & Omit<{ readonly bar?: string; } & VNodeProps & AllowedComponentProps & ComponentCustomProps, never>; ... 12 more ...; bar?: string; }'.",
				},
			})
		}
	})
}

func TestScriptWithoutDefineComponent(t *testing.T) {
	runFourslashTest(t, `// @filename: file.vue
<script lang="ts">
	export default {
		props: {
			foo: String
		}
	}
</script>

<template>
	{{ foo/*1*/ }}
</template>`, func(t *testing.T, f *fourslash.FourslashTest, version vueVersion) {
		f.VerifyQuickInfoAt(t, "1", `(property) foo: string`, "")
		f.VerifyNonSuggestionDiagnostics(t, []*lsproto.Diagnostic{})
	})
}

func TestScriptWithoutDefineComponentError(t *testing.T) {
	runFourslashTest(t, `// @filename: file.vue
<script lang="ts">
	export default {
		computed: {
			[|foo|]: 123
		}
	}
</script>`, func(t *testing.T, f *fourslash.FourslashTest, version vueVersion) {
		switch version {
		case vue_3_2, vue_3_3, vue_3_4:
			f.VerifyNonSuggestionDiagnostics(t, []*lsproto.Diagnostic{
				{
					Code: &lsproto.IntegerOrString{Integer: ptrTo[int32](2769)},
					Message: `No overload matches this call.
  The last overload gave the following error.
    Type 'number' is not assignable to type 'ComputedGetter<any> | WritableComputedOptions<any>'.`,
				},
			})
		default:
			f.VerifyNonSuggestionDiagnostics(t, []*lsproto.Diagnostic{
				{
					Code: &lsproto.IntegerOrString{Integer: ptrTo[int32](2769)},
					Message: `No overload matches this call.
  The last overload gave the following error.
    Type 'number' is not assignable to type 'ComputedGetter<any> | WritableComputedOptions<any, any>'.`,
				},
			})
		}
	})
}

func TestScriptProps(t *testing.T) {
	runFourslashTest(t, `// @filename: file.vue
<script setup lang="ts">
	import Comp from './file-foo.vue'
</script>

<template>
	<Comp [|foo|]="123"/>
</template>

// @filename: file-foo.vue
<script lang="ts">
	export default {
		props: {
			foo: Number
		}
	}
</script>
`, func(t *testing.T, f *fourslash.FourslashTest, version vueVersion) {
		f.VerifyNonSuggestionDiagnostics(t, []*lsproto.Diagnostic{
			{
				Code:    &lsproto.IntegerOrString{Integer: ptrTo[int32](2322)},
				Message: "Type 'string' is not assignable to type 'number'.",
			},
		})
	})
}
