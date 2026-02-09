package vue_tests

import (
	"testing"

	"github.com/microsoft/typescript-go/shim/fourslash"
	"github.com/microsoft/typescript-go/shim/lsp/lsproto"
)

func TestMissingComponentProps(t *testing.T) {
	runFourslashTest(t, `// @filename: file.vue
<script setup lang="ts">
	import CompFoo from './file-foo.vue'
</script>

<template>
	<[|CompFoo|]/>
</template>

// @filename: file-foo.vue
<script setup lang="ts">
	defineProps<{ foo: string }>()
</script>`, func(t *testing.T, f *fourslash.FourslashTest, version vueVersion) {
		switch version {
		case vue_3_2:
			f.VerifyNonSuggestionDiagnostics(t, []*lsproto.Diagnostic{
				{
					Code: &lsproto.IntegerOrString{Integer: ptrTo[int32](2345)},
					Message: `Argument of type '{}' is not assignable to parameter of type 'Partial<{}> & Omit<Readonly<ExtractPropTypes<__VLS_TypePropsToOption<Readonly<Omit<{ foo: string; }, never> & {}>>>> & VNodeProps & AllowedComponentProps & ComponentCustomProps, never>'.
  Property 'foo' is missing in type '{}' but required in type 'Omit<Readonly<ExtractPropTypes<__VLS_TypePropsToOption<Readonly<Omit<{ foo: string; }, never> & {}>>>> & VNodeProps & AllowedComponentProps & ComponentCustomProps, never>'.`,
				},
			})
		case vue_3_3, vue_3_4:
			f.VerifyNonSuggestionDiagnostics(t, []*lsproto.Diagnostic{
				{
					Code: &lsproto.IntegerOrString{Integer: ptrTo[int32](2345)},
					Message: `Argument of type '{}' is not assignable to parameter of type 'Partial<{}> & Omit<{ readonly foo: string; } & VNodeProps & AllowedComponentProps & ComponentCustomProps & Readonly<...>, never>'.
  Property 'foo' is missing in type '{}' but required in type 'Omit<{ readonly foo: string; } & VNodeProps & AllowedComponentProps & ComponentCustomProps & Readonly<ExtractPropTypes<__VLS_TypePropsToOption<DefineProps<LooseRequired<{ foo: string; }>, never>>>>, never>'.`,
				},
			})
		default:
			f.VerifyNonSuggestionDiagnostics(t, []*lsproto.Diagnostic{
				{
					Code: &lsproto.IntegerOrString{Integer: ptrTo[int32](2345)},
					Message: `Argument of type '{}' is not assignable to parameter of type '{ readonly foo: string; } & VNodeProps & AllowedComponentProps & ComponentCustomProps'.
  Property 'foo' is missing in type '{}' but required in type '{ readonly foo: string; }'.`,
				},
			})
		}
	})
}

func TestComponentPropTypeMismatch(t *testing.T) {
	runFourslashTest(t, `// @filename: file.vue
<script setup lang="ts">
	import CompFoo from './file-foo.vue'
</script>

<template>
	<CompFoo [|foo|]="bar" />
</template>

// @filename: file-foo.vue
<script setup lang="ts">
	defineProps<{ foo: number }>()
</script>`, func(t *testing.T, f *fourslash.FourslashTest, version vueVersion) {
		f.VerifyNonSuggestionDiagnostics(t, []*lsproto.Diagnostic{
			{
				Code:    &lsproto.IntegerOrString{Integer: ptrTo[int32](2322)},
				Message: "Type 'string' is not assignable to type 'number'.",
			},
		})
	})
}

func TestComponentPropTypeMismatchDefinePropsVariable(t *testing.T) {
	runFourslashTest(t, `// @filename: file.vue
<script setup lang="ts">
	import CompFoo from './file-foo.vue'
</script>

<template>
	<CompFoo [|foo|]="bar" />
</template>

// @filename: file-foo.vue
<script setup lang="ts">
	const p = defineProps<{ foo: number }>()
</script>`, func(t *testing.T, f *fourslash.FourslashTest, version vueVersion) {
		f.VerifyNonSuggestionDiagnostics(t, []*lsproto.Diagnostic{
			{
				Code:    &lsproto.IntegerOrString{Integer: ptrTo[int32](2322)},
				Message: "Type 'string' is not assignable to type 'number'.",
			},
		})
	})
}

func TestComponentPropTypeMismatchBoolean(t *testing.T) {
	runFourslashTest(t, `// @filename: file.vue
<script setup lang="ts">
	import CompFoo from './file-foo.vue'
</script>

<template>
	<CompFoo [|foo|] />
</template>

// @filename: file-foo.vue
<script setup lang="ts">
	defineProps<{ foo: number }>()
</script>`, func(t *testing.T, f *fourslash.FourslashTest, version vueVersion) {
		f.VerifyNonSuggestionDiagnostics(t, []*lsproto.Diagnostic{
			{
				Code:    &lsproto.IntegerOrString{Integer: ptrTo[int32](2322)},
				Message: "Type 'boolean' is not assignable to type 'number'.",
			},
		})
	})
}

func TestComponentKebabCasePropTypeMismatch(t *testing.T) {
	runFourslashTest(t, `// @filename: file.vue
<script setup lang="ts">
	import CompFoo from './file-foo.vue'
</script>

<template>
	<CompFoo [|foo-bar|] />
</template>

// @filename: file-foo.vue
<script setup lang="ts">
	defineProps<{ fooBar: number }>()
</script>`, func(t *testing.T, f *fourslash.FourslashTest, version vueVersion) {
		f.VerifyNonSuggestionDiagnostics(t, []*lsproto.Diagnostic{
			{
				Code:    &lsproto.IntegerOrString{Integer: ptrTo[int32](2322)},
				Message: "Type 'boolean' is not assignable to type 'number'.",
			},
		})
	})
}

func TestMultilineProps(t *testing.T) {
	runFourslashTest(t, `// @filename: file.vue
<script setup lang="ts">
	import CompFoo from './file-foo.vue'
</script>

<template>
	<CompFoo foo="
		multiline!
	"/>
</template>

// @filename: file-foo.vue
<script setup lang="ts">
	defineProps<{ foo: string }>()
</script>`, func(t *testing.T, f *fourslash.FourslashTest, version vueVersion) {
		f.VerifyNonSuggestionDiagnostics(t, []*lsproto.Diagnostic{})
	})
}

func TestRequiredDefineModelProps(t *testing.T) {
	runFourslashTest(t, `// @filename: file.vue
// @strict: true
<script setup lang="ts">
	import CompFoo from './file-foo.vue'
</script>

<template>
	<[|CompFoo|]/>

	<CompFoo [|model-value|]="123"/>
</template>

// @filename: file-foo.vue
<script setup lang="ts">
	defineModel<number>({ required: true })
</script>`, func(t *testing.T, f *fourslash.FourslashTest, version vueVersion) {
		isNotAssignable := &lsproto.Diagnostic{
			Code:    &lsproto.IntegerOrString{Integer: ptrTo[int32](2322)},
			Message: "Type 'string' is not assignable to type 'number'.",
		}
		switch version {
		case vue_3_2, vue_3_3:
			return
		case vue_3_4:
			f.VerifyNonSuggestionDiagnostics(t, []*lsproto.Diagnostic{
				{
					Code: &lsproto.IntegerOrString{Integer: ptrTo[int32](2345)},
					Message: `Argument of type '{}' is not assignable to parameter of type 'Partial<{}> & Omit<{ readonly modelValue: number; readonly modelModifiers?: Partial<Record<string, true>> | undefined; "onUpdate:modelValue"?: ((value: number) => any) | undefined; } & ... 5 more ... & { ...; }, never>'.
  Property ''modelValue'' is missing in type '{}' but required in type 'Omit<{ readonly modelValue: number; readonly modelModifiers?: Partial<Record<string, true>> | undefined; "onUpdate:modelValue"?: ((value: number) => any) | undefined; } & ... 5 more ... & { ...; }, never>'.`,
				},
				isNotAssignable,
			})
		default:
			f.VerifyNonSuggestionDiagnostics(t, []*lsproto.Diagnostic{
				{
					Code: &lsproto.IntegerOrString{Integer: ptrTo[int32](2345)},
					Message: `Argument of type '{}' is not assignable to parameter of type '{ readonly modelValue: number; readonly modelModifiers?: Partial<Record<string, true>> | undefined; readonly "onUpdate:modelValue"?: ((value: number) => any) | undefined; } & VNodeProps & AllowedComponentProps & ComponentCustomProps'.
  Property ''modelValue'' is missing in type '{}' but required in type '{ readonly modelValue: number; readonly modelModifiers?: Partial<Record<string, true>> | undefined; readonly "onUpdate:modelValue"?: ((value: number) => any) | undefined; }'.`,
				},
				isNotAssignable,
			})
		}
	})
}
