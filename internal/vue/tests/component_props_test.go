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
		f.VerifyNonSuggestionDiagnostics(t, []*lsproto.Diagnostic{})
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
		switch version {
		case vue_3_2, vue_3_3, vue_3_4:
			f.VerifyNonSuggestionDiagnostics(t, []*lsproto.Diagnostic{})
		default:
			f.VerifyNonSuggestionDiagnostics(t, []*lsproto.Diagnostic{
				{
					Code:    &lsproto.IntegerOrString{Integer: ptrTo[int32](2322)},
					Message: "Type 'string' is not assignable to type 'number'.",
				},
			})
		}
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
		switch version {
		case vue_3_2, vue_3_3, vue_3_4:
			f.VerifyNonSuggestionDiagnostics(t, []*lsproto.Diagnostic{})
		default:
			f.VerifyNonSuggestionDiagnostics(t, []*lsproto.Diagnostic{
				{
					Code:    &lsproto.IntegerOrString{Integer: ptrTo[int32](2322)},
					Message: "Type 'string' is not assignable to type 'number'.",
				},
			})
		}
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
		switch version {
		case vue_3_2, vue_3_3, vue_3_4:
			f.VerifyNonSuggestionDiagnostics(t, []*lsproto.Diagnostic{})
		default:
			f.VerifyNonSuggestionDiagnostics(t, []*lsproto.Diagnostic{
				{
					Code:    &lsproto.IntegerOrString{Integer: ptrTo[int32](2322)},
					Message: "Type 'boolean' is not assignable to type 'number'.",
				},
			})
		}
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
		switch version {
		case vue_3_2, vue_3_3, vue_3_4:
			f.VerifyNonSuggestionDiagnostics(t, []*lsproto.Diagnostic{})
		default:
			f.VerifyNonSuggestionDiagnostics(t, []*lsproto.Diagnostic{
				{
					Code:    &lsproto.IntegerOrString{Integer: ptrTo[int32](2322)},
					Message: "Type 'boolean' is not assignable to type 'number'.",
				},
			})
		}
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
	<CompFoo/>

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
		// Missing required props (TS2345) is not reported because __VLS_FunctionalComponent1
		// uses `& Record<string, unknown>` which makes all props optional.
		// Only type mismatches (TS2322) are reported, matching defineProps behavior.
		switch version {
		case vue_3_2, vue_3_3, vue_3_4:
			f.VerifyNonSuggestionDiagnostics(t, []*lsproto.Diagnostic{})
		default:
			f.VerifyNonSuggestionDiagnostics(t, []*lsproto.Diagnostic{
				isNotAssignable,
			})
		}
	})
}
